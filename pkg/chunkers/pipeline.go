package chunkers

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// ChunkingPipeline orchestrates multiple chunking strategies with fallback mechanisms
type ChunkingPipeline struct {
	strategies      []PipelineStrategy
	config          *PipelineConfig
	metrics         *PipelineMetrics
	healthChecker   *HealthChecker
	loadBalancer    *LoadBalancer
	mutex           sync.RWMutex
}

// PipelineConfig configures the chunking pipeline
type PipelineConfig struct {
	// Processing configuration
	MaxConcurrency     int           `json:"max_concurrency"`
	Timeout           time.Duration `json:"timeout"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	
	// Strategy configuration
	EnableFallback    bool          `json:"enable_fallback"`
	FallbackThreshold float64       `json:"fallback_threshold"`
	QualityThreshold  float64       `json:"quality_threshold"`
	
	// Performance configuration
	EnableParallel    bool          `json:"enable_parallel"`
	BatchSize         int           `json:"batch_size"`
	BufferSize        int           `json:"buffer_size"`
	
	// Monitoring configuration
	EnableMetrics     bool          `json:"enable_metrics"`
	MetricsInterval   time.Duration `json:"metrics_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// DefaultPipelineConfig returns sensible defaults for pipeline configuration
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		MaxConcurrency:      4,
		Timeout:            30 * time.Second,
		RetryAttempts:      3,
		RetryDelay:         1 * time.Second,
		EnableFallback:     true,
		FallbackThreshold:  0.5,
		QualityThreshold:   0.7,
		EnableParallel:     true,
		BatchSize:          10,
		BufferSize:         100,
		EnableMetrics:      true,
		MetricsInterval:    10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
}

// PipelineStrategy represents a chunking strategy with priority and configuration
type PipelineStrategy struct {
	Name          string          `json:"name"`
	ChunkerType   ChunkerType     `json:"chunker_type"`
	Config        *ChunkerConfig  `json:"config"`
	Priority      int             `json:"priority"`
	Weight        float64         `json:"weight"`
	HealthScore   float64         `json:"health_score"`
	IsEnabled     bool            `json:"is_enabled"`
	LLMProvider   interfaces.LLM  `json:"-"`
	chunker       Chunker         `json:"-"`
	lastUsed      time.Time       `json:"-"`
	errorCount    int             `json:"-"`
	successCount  int             `json:"-"`
}

// PipelineMetrics tracks performance and usage statistics
type PipelineMetrics struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	StrategyUsage       map[string]int64 `json:"strategy_usage"`
	ErrorCounts         map[string]int64 `json:"error_counts"`
	QualityScores       []float64     `json:"quality_scores"`
	LastUpdated         time.Time     `json:"last_updated"`
	mutex               sync.RWMutex  `json:"-"`
}

// NewChunkingPipeline creates a new chunking pipeline
func NewChunkingPipeline(config *PipelineConfig) *ChunkingPipeline {
	if config == nil {
		config = DefaultPipelineConfig()
	}
	
	pipeline := &ChunkingPipeline{
		strategies:    make([]PipelineStrategy, 0),
		config:        config,
		metrics:       NewPipelineMetrics(),
		healthChecker: NewHealthChecker(config.HealthCheckInterval),
		loadBalancer:  NewLoadBalancer(),
	}
	
	// Start monitoring if enabled
	if config.EnableMetrics {
		go pipeline.startMetricsCollector()
	}
	
	return pipeline
}

// AddStrategy adds a chunking strategy to the pipeline
func (cp *ChunkingPipeline) AddStrategy(strategy PipelineStrategy) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	// Validate strategy
	if strategy.Name == "" {
		return fmt.Errorf("strategy name cannot be empty")
	}
	
	if strategy.Config == nil {
		strategy.Config = DefaultChunkerConfig()
	}
	
	// Create chunker instance
	factory := NewChunkerFactory()
	var chunker Chunker
	var err error
	
	// Handle LLM-requiring chunkers
	if requiresLLM(strategy.ChunkerType) {
		if strategy.LLMProvider == nil {
			return fmt.Errorf("LLM provider required for chunker type %s", strategy.ChunkerType)
		}
		chunker, err = factory.CreateAdvancedChunker(strategy.ChunkerType, strategy.Config, strategy.LLMProvider)
	} else {
		chunker, err = factory.CreateChunker(strategy.ChunkerType, strategy.Config)
	}
	
	if err != nil {
		return fmt.Errorf("failed to create chunker for strategy %s: %w", strategy.Name, err)
	}
	
	strategy.chunker = chunker
	strategy.HealthScore = 1.0
	strategy.IsEnabled = true
	strategy.lastUsed = time.Now()
	
	cp.strategies = append(cp.strategies, strategy)
	
	// Sort strategies by priority (higher priority first)
	sort.Slice(cp.strategies, func(i, j int) bool {
		return cp.strategies[i].Priority > cp.strategies[j].Priority
	})
	
	return nil
}

// Process processes text through the pipeline with strategy selection and fallback
func (cp *ChunkingPipeline) Process(ctx context.Context, text string, metadata map[string]interface{}) (*PipelineResult, error) {
	start := time.Now()
	
	cp.metrics.mutex.Lock()
	cp.metrics.TotalRequests++
	cp.metrics.mutex.Unlock()
	
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, cp.config.Timeout)
	defer cancel()
	
	// Select strategy
	strategy, err := cp.selectStrategy(timeoutCtx, text, metadata)
	if err != nil {
		cp.recordError("strategy_selection", err)
		return nil, fmt.Errorf("failed to select strategy: %w", err)
	}
	
	// Process with selected strategy
	result, err := cp.processWithStrategy(timeoutCtx, strategy, text, metadata)
	if err != nil {
		// Try fallback if enabled
		if cp.config.EnableFallback {
			fallbackResult, fallbackErr := cp.processFallback(timeoutCtx, text, metadata, strategy)
			if fallbackErr == nil {
				result = fallbackResult
				result.UsedFallback = true
				err = nil
			}
		}
		
		if err != nil {
			cp.recordError(strategy.Name, err)
			cp.metrics.mutex.Lock()
			cp.metrics.FailedRequests++
			cp.metrics.mutex.Unlock()
			return nil, fmt.Errorf("processing failed: %w", err)
		}
	}
	
	// Record success metrics
	processingTime := time.Since(start)
	cp.recordSuccess(strategy.Name, processingTime)
	
	result.ProcessingTime = processingTime
	result.StrategyUsed = strategy.Name
	
	return result, nil
}

// ProcessBatch processes multiple texts in parallel or sequential mode
func (cp *ChunkingPipeline) ProcessBatch(ctx context.Context, texts []string, metadata []map[string]interface{}) ([]*PipelineResult, error) {
	if len(texts) == 0 {
		return []*PipelineResult{}, nil
	}
	
	// Ensure metadata slice has same length as texts
	if metadata == nil {
		metadata = make([]map[string]interface{}, len(texts))
	}
	for len(metadata) < len(texts) {
		metadata = append(metadata, nil)
	}
	
	if cp.config.EnableParallel {
		return cp.processBatchParallel(ctx, texts, metadata)
	}
	return cp.processBatchSequential(ctx, texts, metadata)
}

// selectStrategy selects the best strategy based on content analysis and health scores
func (cp *ChunkingPipeline) selectStrategy(ctx context.Context, text string, metadata map[string]interface{}) (*PipelineStrategy, error) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	
	if len(cp.strategies) == 0 {
		return nil, fmt.Errorf("no strategies available")
	}
	
	// Analyze content to determine best strategy
	analysis := cp.analyzeContent(text, metadata)
	
	// Score strategies based on content analysis and health
	bestStrategy := cp.strategies[0]
	bestScore := cp.scoreStrategy(&bestStrategy, analysis)
	
	for i := 1; i < len(cp.strategies); i++ {
		strategy := &cp.strategies[i]
		if !strategy.IsEnabled {
			continue
		}
		
		score := cp.scoreStrategy(strategy, analysis)
		if score > bestScore {
			bestScore = score
			bestStrategy = *strategy
		}
	}
	
	return &bestStrategy, nil
}

// processWithStrategy processes text with a specific strategy
func (cp *ChunkingPipeline) processWithStrategy(ctx context.Context, strategy *PipelineStrategy, text string, metadata map[string]interface{}) (*PipelineResult, error) {
	// Create timeout context for this strategy
	strategyCtx, cancel := context.WithTimeout(ctx, cp.config.Timeout)
	defer cancel()
	
	// Process with retry logic
	var chunks []*Chunk
	var err error
	
	for attempt := 0; attempt < cp.config.RetryAttempts; attempt++ {
		select {
		case <-strategyCtx.Done():
			return nil, strategyCtx.Err()
		default:
		}
		
		chunks, err = strategy.chunker.ChunkWithMetadata(strategyCtx, text, metadata)
		if err == nil {
			break
		}
		
		if attempt < cp.config.RetryAttempts-1 {
			time.Sleep(cp.config.RetryDelay)
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("strategy %s failed after %d attempts: %w", strategy.Name, cp.config.RetryAttempts, err)
	}
	
	// Evaluate quality
	qualityScore := cp.evaluateQuality(chunks, text)
	
	result := &PipelineResult{
		Chunks:       chunks,
		StrategyUsed: strategy.Name,
		QualityScore: qualityScore,
		UsedFallback: false,
		Metadata:     metadata,
	}
	
	// Record quality score
	cp.metrics.mutex.Lock()
	cp.metrics.QualityScores = append(cp.metrics.QualityScores, qualityScore)
	// Keep only last 1000 scores
	if len(cp.metrics.QualityScores) > 1000 {
		cp.metrics.QualityScores = cp.metrics.QualityScores[len(cp.metrics.QualityScores)-1000:]
	}
	cp.metrics.mutex.Unlock()
	
	return result, nil
}

// processFallback attempts fallback processing when primary strategy fails
func (cp *ChunkingPipeline) processFallback(ctx context.Context, text string, metadata map[string]interface{}, failedStrategy *PipelineStrategy) (*PipelineResult, error) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	
	// Find alternative strategies (enabled and different from failed)
	for _, strategy := range cp.strategies {
		if !strategy.IsEnabled || strategy.Name == failedStrategy.Name {
			continue
		}
		
		// Try with lower health threshold
		if strategy.HealthScore >= cp.config.FallbackThreshold {
			result, err := cp.processWithStrategy(ctx, &strategy, text, metadata)
			if err == nil {
				return result, nil
			}
		}
	}
	
	return nil, fmt.Errorf("all fallback strategies failed")
}

// processBatchParallel processes texts in parallel
func (cp *ChunkingPipeline) processBatchParallel(ctx context.Context, texts []string, metadata []map[string]interface{}) ([]*PipelineResult, error) {
	results := make([]*PipelineResult, len(texts))
	errors := make([]error, len(texts))
	
	// Create worker pool
	maxWorkers := cp.config.MaxConcurrency
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	
	type job struct {
		index int
		text  string
		meta  map[string]interface{}
	}
	
	jobs := make(chan job, len(texts))
	var wg sync.WaitGroup
	
	// Start workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result, err := cp.Process(ctx, job.text, job.meta)
				results[job.index] = result
				errors[job.index] = err
			}
		}()
	}
	
	// Send jobs
	for i, text := range texts {
		jobs <- job{index: i, text: text, meta: metadata[i]}
	}
	close(jobs)
	
	// Wait for completion
	wg.Wait()
	
	// Check for errors
	var errorCount int
	for _, err := range errors {
		if err != nil {
			errorCount++
		}
	}
	
	// Return results if most succeeded
	if errorCount < len(texts)/2 {
		return results, nil
	}
	
	return results, fmt.Errorf("batch processing failed: %d out of %d texts failed", errorCount, len(texts))
}

// processBatchSequential processes texts sequentially
func (cp *ChunkingPipeline) processBatchSequential(ctx context.Context, texts []string, metadata []map[string]interface{}) ([]*PipelineResult, error) {
	results := make([]*PipelineResult, len(texts))
	
	for i, text := range texts {
		result, err := cp.Process(ctx, text, metadata[i])
		if err != nil {
			return results, fmt.Errorf("failed to process text %d: %w", i, err)
		}
		results[i] = result
	}
	
	return results, nil
}

// PipelineResult represents the result of pipeline processing
type PipelineResult struct {
	Chunks         []*Chunk                 `json:"chunks"`
	StrategyUsed   string                   `json:"strategy_used"`
	QualityScore   float64                  `json:"quality_score"`
	ProcessingTime time.Duration            `json:"processing_time"`
	UsedFallback   bool                     `json:"used_fallback"`
	Metadata       map[string]interface{}   `json:"metadata"`
}

// ContentAnalysis provides analysis of input content to guide strategy selection
type ContentAnalysis struct {
	Length            int                    `json:"length"`
	Complexity        float64                `json:"complexity"`
	Language          string                 `json:"language"`
	ContentType       string                 `json:"content_type"`
	HasCode           bool                   `json:"has_code"`
	HasTables         bool                   `json:"has_tables"`
	SentenceCount     int                    `json:"sentence_count"`
	ParagraphCount    int                    `json:"paragraph_count"`
	EstimatedTokens   int                    `json:"estimated_tokens"`
}

// analyzeContent analyzes content characteristics to guide strategy selection
func (cp *ChunkingPipeline) analyzeContent(text string, metadata map[string]interface{}) *ContentAnalysis {
	analysis := &ContentAnalysis{
		Length:      len(text),
		Language:    "en", // Default, could be detected
		ContentType: "text",
	}
	
	// Estimate tokens using simple method
	factory := NewTokenizerFactory()
	if tokenizer, err := factory.CreateTokenizer(DefaultTokenizerConfig()); err == nil {
		if tokens, err := tokenizer.CountTokens(text); err == nil {
			analysis.EstimatedTokens = tokens
		}
	}
	
	// Detect content characteristics
	analysis.HasCode = containsCode(text)
	analysis.HasTables = containsTables(text)
	analysis.SentenceCount = countSentences(text)
	analysis.ParagraphCount = countParagraphs(text)
	analysis.Complexity = calculateComplexity(text)
	
	// Use metadata if available
	if metadata != nil {
		if lang, ok := metadata["language"].(string); ok {
			analysis.Language = lang
		}
		if contentType, ok := metadata["content_type"].(string); ok {
			analysis.ContentType = contentType
		}
	}
	
	return analysis
}

// scoreStrategy scores a strategy based on content analysis and health
func (cp *ChunkingPipeline) scoreStrategy(strategy *PipelineStrategy, analysis *ContentAnalysis) float64 {
	baseScore := strategy.HealthScore * strategy.Weight
	
	// Adjust score based on content characteristics
	switch strategy.ChunkerType {
	case ChunkerTypeSentence:
		if analysis.Complexity < 0.5 {
			baseScore *= 1.2 // Good for simple content
		}
	case ChunkerTypeSemantic:
		if analysis.Complexity > 0.5 {
			baseScore *= 1.3 // Good for complex content
		}
	case ChunkerTypeMultiModal:
		if analysis.HasCode || analysis.HasTables {
			baseScore *= 1.5 // Excellent for mixed content
		}
	case ChunkerTypeContextual:
		if analysis.Length > 1000 {
			baseScore *= 1.2 // Good for long documents
		}
	case ChunkerTypeAgentic:
		if analysis.Complexity > 0.7 {
			baseScore *= 1.4 // Excellent for complex content
		}
	}
	
	// Penalize recently failed strategies
	if strategy.errorCount > strategy.successCount {
		baseScore *= 0.8
	}
	
	// Favor recently successful strategies
	if time.Since(strategy.lastUsed) < time.Hour && strategy.successCount > 0 {
		baseScore *= 1.1
	}
	
	return baseScore
}

// evaluateQuality evaluates the quality of chunking results
func (cp *ChunkingPipeline) evaluateQuality(chunks []*Chunk, originalText string) float64 {
	if len(chunks) == 0 {
		return 0.0
	}
	
	var qualityScore float64
	
	// Factor 1: Coverage (how much of original text is covered)
	totalCoverage := 0
	for _, chunk := range chunks {
		totalCoverage += len(chunk.Text)
	}
	coverageRatio := float64(totalCoverage) / float64(len(originalText))
	qualityScore += coverageRatio * 0.3
	
	// Factor 2: Size consistency (chunks should be reasonably sized)
	var sizes []int
	for _, chunk := range chunks {
		sizes = append(sizes, chunk.TokenCount)
	}
	sizeConsistency := calculateSizeConsistency(sizes)
	qualityScore += sizeConsistency * 0.2
	
	// Factor 3: Semantic coherence (if chunks have sentence structure)
	coherenceScore := calculateCoherence(chunks)
	qualityScore += coherenceScore * 0.3
	
	// Factor 4: Boundary quality (clean sentence/paragraph boundaries)
	boundaryScore := calculateBoundaryQuality(chunks)
	qualityScore += boundaryScore * 0.2
	
	// Normalize to 0-1 range
	if qualityScore > 1.0 {
		qualityScore = 1.0
	}
	
	return qualityScore
}

// Utility functions for content analysis and quality evaluation

func requiresLLM(chunkerType ChunkerType) bool {
	return chunkerType == ChunkerTypeContextual ||
		chunkerType == ChunkerTypePropositionalization ||
		chunkerType == ChunkerTypeAgentic
}

func containsCode(text string) bool {
	// Simple heuristics for code detection
	codeIndicators := []string{"function", "class", "import", "def ", "var ", "const ", "let ", "public ", "private "}
	for _, indicator := range codeIndicators {
		if len(text) > 0 && strings.Contains(text, indicator) {
			return true
		}
	}
	return false
}

func containsTables(text string) bool {
	// Simple heuristics for table detection
	tableIndicators := []string{"|", "\t", "Table", "Column", "Row"}
	count := 0
	for _, indicator := range tableIndicators {
		if strings.Contains(text, indicator) {
			count++
		}
	}
	return count >= 2 // Need multiple indicators
}

func countSentences(text string) int {
	sentenceEnders := []string{".", "!", "?"}
	count := 0
	for _, ender := range sentenceEnders {
		count += strings.Count(text, ender)
	}
	return count
}

func countParagraphs(text string) int {
	paragraphs := strings.Split(text, "\n\n")
	nonEmpty := 0
	for _, p := range paragraphs {
		if strings.TrimSpace(p) != "" {
			nonEmpty++
		}
	}
	return nonEmpty
}

func calculateComplexity(text string) float64 {
	// Simple complexity calculation based on sentence length, vocabulary, etc.
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0.0
	}
	
	sentences := countSentences(text)
	if sentences == 0 {
		sentences = 1
	}
	
	avgWordsPerSentence := float64(len(words)) / float64(sentences)
	
	// Vocabulary diversity
	uniqueWords := make(map[string]bool)
	for _, word := range words {
		uniqueWords[strings.ToLower(word)] = true
	}
	
	vocabularyDiversity := float64(len(uniqueWords)) / float64(len(words))
	
	// Combine factors (normalize to 0-1)
	complexity := (avgWordsPerSentence/20.0)*0.5 + vocabularyDiversity*0.5
	if complexity > 1.0 {
		complexity = 1.0
	}
	
	return complexity
}


func calculateCoherence(chunks []*Chunk) float64 {
	// Simple coherence calculation based on sentence structure
	coherentChunks := 0
	
	for _, chunk := range chunks {
		if len(chunk.Sentences) > 0 {
			// Check if chunk starts and ends with complete sentences
			firstSentence := chunk.Sentences[0]
			lastSentence := chunk.Sentences[len(chunk.Sentences)-1]
			
			if strings.TrimSpace(firstSentence) != "" && strings.TrimSpace(lastSentence) != "" {
				coherentChunks++
			}
		}
	}
	
	return float64(coherentChunks) / float64(len(chunks))
}

func calculateBoundaryQuality(chunks []*Chunk) float64 {
	// Check if chunks have clean boundaries (end with sentence terminators)
	cleanBoundaries := 0
	
	for _, chunk := range chunks {
		text := strings.TrimSpace(chunk.Text)
		if text != "" {
			lastChar := text[len(text)-1]
			if lastChar == '.' || lastChar == '!' || lastChar == '?' || lastChar == '\n' {
				cleanBoundaries++
			}
		}
	}
	
	return float64(cleanBoundaries) / float64(len(chunks))
}

// recordSuccess records successful processing metrics
func (cp *ChunkingPipeline) recordSuccess(strategyName string, processingTime time.Duration) {
	cp.metrics.mutex.Lock()
	defer cp.metrics.mutex.Unlock()
	
	cp.metrics.SuccessfulRequests++
	cp.metrics.TotalProcessingTime += processingTime
	cp.metrics.AverageProcessingTime = cp.metrics.TotalProcessingTime / time.Duration(cp.metrics.SuccessfulRequests)
	
	if cp.metrics.StrategyUsage == nil {
		cp.metrics.StrategyUsage = make(map[string]int64)
	}
	cp.metrics.StrategyUsage[strategyName]++
	cp.metrics.LastUpdated = time.Now()
	
	// Update strategy success count
	cp.mutex.Lock()
	for i := range cp.strategies {
		if cp.strategies[i].Name == strategyName {
			cp.strategies[i].successCount++
			cp.strategies[i].lastUsed = time.Now()
			break
		}
	}
	cp.mutex.Unlock()
}

// recordError records error metrics
func (cp *ChunkingPipeline) recordError(strategyName string, err error) {
	cp.metrics.mutex.Lock()
	defer cp.metrics.mutex.Unlock()
	
	if cp.metrics.ErrorCounts == nil {
		cp.metrics.ErrorCounts = make(map[string]int64)
	}
	cp.metrics.ErrorCounts[strategyName]++
	cp.metrics.LastUpdated = time.Now()
	
	// Update strategy error count
	cp.mutex.Lock()
	for i := range cp.strategies {
		if cp.strategies[i].Name == strategyName {
			cp.strategies[i].errorCount++
			// Reduce health score on repeated errors
			if cp.strategies[i].errorCount > cp.strategies[i].successCount {
				cp.strategies[i].HealthScore *= 0.9
				if cp.strategies[i].HealthScore < 0.1 {
					cp.strategies[i].IsEnabled = false
				}
			}
			break
		}
	}
	cp.mutex.Unlock()
}

// GetMetrics returns current pipeline metrics
func (cp *ChunkingPipeline) GetMetrics() *PipelineMetrics {
	cp.metrics.mutex.RLock()
	defer cp.metrics.mutex.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := &PipelineMetrics{
		TotalRequests:         cp.metrics.TotalRequests,
		SuccessfulRequests:    cp.metrics.SuccessfulRequests,
		FailedRequests:        cp.metrics.FailedRequests,
		TotalProcessingTime:   cp.metrics.TotalProcessingTime,
		AverageProcessingTime: cp.metrics.AverageProcessingTime,
		LastUpdated:           cp.metrics.LastUpdated,
		StrategyUsage:         make(map[string]int64),
		ErrorCounts:           make(map[string]int64),
		QualityScores:         make([]float64, len(cp.metrics.QualityScores)),
	}
	
	// Copy maps
	for k, v := range cp.metrics.StrategyUsage {
		metrics.StrategyUsage[k] = v
	}
	for k, v := range cp.metrics.ErrorCounts {
		metrics.ErrorCounts[k] = v
	}
	copy(metrics.QualityScores, cp.metrics.QualityScores)
	
	return metrics
}

// GetStrategies returns current strategies configuration
func (cp *ChunkingPipeline) GetStrategies() []PipelineStrategy {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	
	strategies := make([]PipelineStrategy, len(cp.strategies))
	copy(strategies, cp.strategies)
	return strategies
}

// UpdateStrategyHealth updates health score for a strategy
func (cp *ChunkingPipeline) UpdateStrategyHealth(strategyName string, healthScore float64) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	for i := range cp.strategies {
		if cp.strategies[i].Name == strategyName {
			cp.strategies[i].HealthScore = healthScore
			cp.strategies[i].IsEnabled = healthScore > 0.1
			return nil
		}
	}
	
	return fmt.Errorf("strategy %s not found", strategyName)
}

// startMetricsCollector starts background metrics collection
func (cp *ChunkingPipeline) startMetricsCollector() {
	ticker := time.NewTicker(cp.config.MetricsInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		cp.collectMetrics()
	}
}

// collectMetrics collects periodic metrics
func (cp *ChunkingPipeline) collectMetrics() {
	// Update strategy health based on recent performance
	cp.mutex.Lock()
	for i := range cp.strategies {
		strategy := &cp.strategies[i]
		
		// Calculate recent success rate
		totalAttempts := strategy.successCount + strategy.errorCount
		if totalAttempts > 0 {
			successRate := float64(strategy.successCount) / float64(totalAttempts)
			
			// Adjust health score based on success rate
			if successRate > 0.8 {
				strategy.HealthScore = math.Min(1.0, strategy.HealthScore*1.05)
			} else if successRate < 0.5 {
				strategy.HealthScore *= 0.95
			}
			
			// Enable/disable based on health
			strategy.IsEnabled = strategy.HealthScore > 0.1
		}
	}
	cp.mutex.Unlock()
}

// NewPipelineMetrics creates a new metrics instance
func NewPipelineMetrics() *PipelineMetrics {
	return &PipelineMetrics{
		StrategyUsage: make(map[string]int64),
		ErrorCounts:   make(map[string]int64),
		QualityScores: make([]float64, 0),
		LastUpdated:   time.Now(),
	}
}