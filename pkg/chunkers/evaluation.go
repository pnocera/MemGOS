package chunkers

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// EvaluationFramework provides comprehensive evaluation and benchmarking for chunking strategies
type EvaluationFramework struct {
	datasets       map[string]*EvaluationDataset
	benchmarks     map[string]*Benchmark
	qualityMetrics []QualityMetric
	config         *EvaluationConfig
	results        *EvaluationResults
	mutex          sync.RWMutex
}

// EvaluationConfig configures the evaluation framework
type EvaluationConfig struct {
	// Test configuration
	TimeoutPerTest     time.Duration `json:"timeout_per_test"`
	MaxConcurrentTests int           `json:"max_concurrent_tests"`
	RetryAttempts      int           `json:"retry_attempts"`

	// Quality thresholds
	MinQualityScore float64       `json:"min_quality_score"`
	MaxResponseTime time.Duration `json:"max_response_time"`
	MinSuccessRate  float64       `json:"min_success_rate"`

	// Benchmark configuration
	WarmupRuns     int  `json:"warmup_runs"`
	BenchmarkRuns  int  `json:"benchmark_runs"`
	CollectMetrics bool `json:"collect_metrics"`

	// Comparison settings
	EnableABTesting         bool    `json:"enable_ab_testing"`
	ABTestSampleSize        int     `json:"ab_test_sample_size"`
	StatisticalSignificance float64 `json:"statistical_significance"`
}

// DefaultEvaluationConfig returns sensible defaults for evaluation
func DefaultEvaluationConfig() *EvaluationConfig {
	return &EvaluationConfig{
		TimeoutPerTest:          30 * time.Second,
		MaxConcurrentTests:      4,
		RetryAttempts:           3,
		MinQualityScore:         0.7,
		MaxResponseTime:         5 * time.Second,
		MinSuccessRate:          0.95,
		WarmupRuns:              3,
		BenchmarkRuns:           10,
		CollectMetrics:          true,
		EnableABTesting:         true,
		ABTestSampleSize:        100,
		StatisticalSignificance: 0.05,
	}
}

// EvaluationDataset represents a dataset for evaluation
type EvaluationDataset struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Documents   []*EvaluationDocument  `json:"documents"`
	GroundTruth []*GroundTruthChunk    `json:"ground_truth,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// EvaluationDocument represents a document in an evaluation dataset
type EvaluationDocument struct {
	ID             string                 `json:"id"`
	Text           string                 `json:"text"`
	ContentType    string                 `json:"content_type"`
	Language       string                 `json:"language"`
	Metadata       map[string]interface{} `json:"metadata"`
	ExpectedChunks int                    `json:"expected_chunks,omitempty"`
}

// GroundTruthChunk represents expected chunking results for evaluation
type GroundTruthChunk struct {
	DocumentID string                 `json:"document_id"`
	ChunkIndex int                    `json:"chunk_index"`
	StartIndex int                    `json:"start_index"`
	EndIndex   int                    `json:"end_index"`
	Text       string                 `json:"text"`
	Quality    float64                `json:"quality"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Benchmark represents a performance benchmark
type Benchmark struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Dataset     string             `json:"dataset"`
	Strategy    string             `json:"strategy"`
	Config      *ChunkerConfig     `json:"config"`
	Results     []*BenchmarkResult `json:"results"`
	CreatedAt   time.Time          `json:"created_at"`
}

// BenchmarkResult represents the result of a benchmark run
type BenchmarkResult struct {
	RunID          string             `json:"run_id"`
	Timestamp      time.Time          `json:"timestamp"`
	ProcessingTime time.Duration      `json:"processing_time"`
	MemoryUsage    int64              `json:"memory_usage"`
	QualityScore   float64            `json:"quality_score"`
	ChunkCount     int                `json:"chunk_count"`
	ErrorCount     int                `json:"error_count"`
	Metrics        map[string]float64 `json:"metrics"`
}

// QualityMetric defines a quality evaluation metric
type QualityMetric interface {
	Name() string
	Description() string
	Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error)
}

// EvaluationResults stores comprehensive evaluation results
type EvaluationResults struct {
	OverallScore     float64                        `json:"overall_score"`
	StrategyScores   map[string]*StrategyEvaluation `json:"strategy_scores"`
	DatasetResults   map[string]*DatasetEvaluation  `json:"dataset_results"`
	BenchmarkResults map[string]*BenchmarkSummary   `json:"benchmark_results"`
	QualityMetrics   map[string]float64             `json:"quality_metrics"`
	PerformanceStats *PerformanceStats              `json:"performance_stats"`
	ComparisonMatrix map[string]map[string]float64  `json:"comparison_matrix"`
	Recommendations  []string                       `json:"recommendations"`
	GeneratedAt      time.Time                      `json:"generated_at"`
	mutex            sync.RWMutex                   `json:"-"`
}

// StrategyEvaluation contains evaluation results for a specific strategy
type StrategyEvaluation struct {
	StrategyName     string             `json:"strategy_name"`
	OverallScore     float64            `json:"overall_score"`
	QualityScore     float64            `json:"quality_score"`
	PerformanceScore float64            `json:"performance_score"`
	ReliabilityScore float64            `json:"reliability_score"`
	DetailedMetrics  map[string]float64 `json:"detailed_metrics"`
	TestResults      []*TestResult      `json:"test_results"`
	Strengths        []string           `json:"strengths"`
	Weaknesses       []string           `json:"weaknesses"`
}

// DatasetEvaluation contains evaluation results for a specific dataset
type DatasetEvaluation struct {
	DatasetName      string            `json:"dataset_name"`
	BestStrategy     string            `json:"best_strategy"`
	BestScore        float64           `json:"best_score"`
	StrategyRankings []StrategyRanking `json:"strategy_rankings"`
	Insights         []string          `json:"insights"`
}

// StrategyRanking represents the ranking of a strategy on a dataset
type StrategyRanking struct {
	StrategyName string  `json:"strategy_name"`
	Score        float64 `json:"score"`
	Rank         int     `json:"rank"`
}

// BenchmarkSummary provides a summary of benchmark results
type BenchmarkSummary struct {
	BenchmarkName     string        `json:"benchmark_name"`
	RunCount          int           `json:"run_count"`
	AverageTime       time.Duration `json:"average_time"`
	MedianTime        time.Duration `json:"median_time"`
	MinTime           time.Duration `json:"min_time"`
	MaxTime           time.Duration `json:"max_time"`
	StandardDeviation time.Duration `json:"standard_deviation"`
	ThroughputQPS     float64       `json:"throughput_qps"`
	MemoryEfficiency  float64       `json:"memory_efficiency"`
}

// PerformanceStats contains overall performance statistics
type PerformanceStats struct {
	TotalTests  int           `json:"total_tests"`
	PassedTests int           `json:"passed_tests"`
	FailedTests int           `json:"failed_tests"`
	AverageTime time.Duration `json:"average_time"`
	TotalTime   time.Duration `json:"total_time"`
	MemoryUsage int64         `json:"memory_usage"`
	ErrorRate   float64       `json:"error_rate"`
}

// TestResult represents the result of a single test
type TestResult struct {
	TestName       string             `json:"test_name"`
	Strategy       string             `json:"strategy"`
	Dataset        string             `json:"dataset"`
	Document       string             `json:"document"`
	Passed         bool               `json:"passed"`
	Score          float64            `json:"score"`
	ProcessingTime time.Duration      `json:"processing_time"`
	Error          string             `json:"error,omitempty"`
	Metrics        map[string]float64 `json:"metrics"`
	Timestamp      time.Time          `json:"timestamp"`
}

// NewEvaluationFramework creates a new evaluation framework
func NewEvaluationFramework(config *EvaluationConfig) *EvaluationFramework {
	if config == nil {
		config = DefaultEvaluationConfig()
	}

	framework := &EvaluationFramework{
		datasets:   make(map[string]*EvaluationDataset),
		benchmarks: make(map[string]*Benchmark),
		qualityMetrics: []QualityMetric{
			&CoverageMetric{},
			&CoherenceMetric{},
			&BoundaryQualityMetric{},
			&SizeConsistencyMetric{},
			&SemanticSimilarityMetric{},
		},
		config: config,
		results: &EvaluationResults{
			StrategyScores:   make(map[string]*StrategyEvaluation),
			DatasetResults:   make(map[string]*DatasetEvaluation),
			BenchmarkResults: make(map[string]*BenchmarkSummary),
			QualityMetrics:   make(map[string]float64),
			ComparisonMatrix: make(map[string]map[string]float64),
			GeneratedAt:      time.Now(),
		},
	}

	return framework
}

// AddDataset adds an evaluation dataset
func (ef *EvaluationFramework) AddDataset(dataset *EvaluationDataset) {
	ef.mutex.Lock()
	defer ef.mutex.Unlock()

	ef.datasets[dataset.Name] = dataset
}

// LoadStandardDatasets loads common evaluation datasets
func (ef *EvaluationFramework) LoadStandardDatasets() error {
	// Load common chunking evaluation datasets
	datasets := []*EvaluationDataset{
		{
			Name:        "scientific_papers",
			Description: "Collection of scientific papers for evaluation",
			Documents:   generateScientificPapers(),
			CreatedAt:   time.Now(),
		},
		{
			Name:        "news_articles",
			Description: "Collection of news articles for evaluation",
			Documents:   generateNewsArticles(),
			CreatedAt:   time.Now(),
		},
		{
			Name:        "technical_documentation",
			Description: "Technical documentation with code and examples",
			Documents:   generateTechnicalDocs(),
			CreatedAt:   time.Now(),
		},
		{
			Name:        "mixed_content",
			Description: "Mixed content with tables, lists, and formatting",
			Documents:   generateMixedContent(),
			CreatedAt:   time.Now(),
		},
	}

	for _, dataset := range datasets {
		ef.AddDataset(dataset)
	}

	return nil
}

// EvaluateStrategy evaluates a single chunking strategy across all datasets
func (ef *EvaluationFramework) EvaluateStrategy(ctx context.Context, strategyName string, pipeline *ChunkingPipeline) (*StrategyEvaluation, error) {
	ef.mutex.RLock()
	datasets := make([]*EvaluationDataset, 0, len(ef.datasets))
	for _, dataset := range ef.datasets {
		datasets = append(datasets, dataset)
	}
	ef.mutex.RUnlock()

	evaluation := &StrategyEvaluation{
		StrategyName:    strategyName,
		DetailedMetrics: make(map[string]float64),
		TestResults:     make([]*TestResult, 0),
		Strengths:       make([]string, 0),
		Weaknesses:      make([]string, 0),
	}

	totalScore := 0.0
	totalTests := 0

	for _, dataset := range datasets {
		datasetScore, testResults, err := ef.evaluateStrategyOnDataset(ctx, strategyName, pipeline, dataset)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate strategy %s on dataset %s: %w", strategyName, dataset.Name, err)
		}

		totalScore += datasetScore
		totalTests++
		evaluation.TestResults = append(evaluation.TestResults, testResults...)
		evaluation.DetailedMetrics[fmt.Sprintf("dataset_%s_score", dataset.Name)] = datasetScore
	}

	if totalTests > 0 {
		evaluation.OverallScore = totalScore / float64(totalTests)
	}

	// Calculate component scores
	evaluation.QualityScore = ef.calculateQualityScore(evaluation.TestResults)
	evaluation.PerformanceScore = ef.calculatePerformanceScore(evaluation.TestResults)
	evaluation.ReliabilityScore = ef.calculateReliabilityScore(evaluation.TestResults)

	// Generate insights
	evaluation.Strengths, evaluation.Weaknesses = ef.generateInsights(evaluation)

	return evaluation, nil
}

// RunComparison runs A/B testing comparison between strategies
func (ef *EvaluationFramework) RunComparison(ctx context.Context, strategyA, strategyB string, pipeline *ChunkingPipeline) (*ComparisonResult, error) {
	if !ef.config.EnableABTesting {
		return nil, fmt.Errorf("A/B testing is disabled")
	}

	// Run evaluations for both strategies
	evalA, err := ef.EvaluateStrategy(ctx, strategyA, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate strategy A (%s): %w", strategyA, err)
	}

	evalB, err := ef.EvaluateStrategy(ctx, strategyB, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate strategy B (%s): %w", strategyB, err)
	}

	// Perform statistical analysis
	comparison := &ComparisonResult{
		StrategyA:  strategyA,
		StrategyB:  strategyB,
		ScoreA:     evalA.OverallScore,
		ScoreB:     evalB.OverallScore,
		SampleSize: ef.config.ABTestSampleSize,
		Timestamp:  time.Now(),
	}

	// Calculate statistical significance
	comparison.PValue = ef.calculatePValue(evalA.TestResults, evalB.TestResults)
	comparison.IsSignificant = comparison.PValue < ef.config.StatisticalSignificance

	// Determine winner
	if comparison.ScoreA > comparison.ScoreB {
		comparison.Winner = strategyA
		comparison.Improvement = ((comparison.ScoreA - comparison.ScoreB) / comparison.ScoreB) * 100
	} else {
		comparison.Winner = strategyB
		comparison.Improvement = ((comparison.ScoreB - comparison.ScoreA) / comparison.ScoreA) * 100
	}

	comparison.Confidence = 1.0 - comparison.PValue

	return comparison, nil
}

// RunBenchmark runs performance benchmarks for a strategy
func (ef *EvaluationFramework) RunBenchmark(ctx context.Context, benchmarkName string, strategy string, pipeline *ChunkingPipeline) (*BenchmarkSummary, error) {
	ef.mutex.RLock()
	dataset, exists := ef.datasets[benchmarkName]
	ef.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("dataset %s not found for benchmark", benchmarkName)
	}

	benchmark := &Benchmark{
		Name:        benchmarkName,
		Description: fmt.Sprintf("Performance benchmark for %s strategy", strategy),
		Dataset:     benchmarkName,
		Strategy:    strategy,
		Results:     make([]*BenchmarkResult, 0),
		CreatedAt:   time.Now(),
	}

	// Warmup runs
	for i := 0; i < ef.config.WarmupRuns; i++ {
		_, err := ef.runBenchmarkIteration(ctx, benchmark, dataset, pipeline)
		if err != nil {
			return nil, fmt.Errorf("warmup run %d failed: %w", i+1, err)
		}
	}

	// Actual benchmark runs
	results := make([]*BenchmarkResult, 0, ef.config.BenchmarkRuns)
	for i := 0; i < ef.config.BenchmarkRuns; i++ {
		result, err := ef.runBenchmarkIteration(ctx, benchmark, dataset, pipeline)
		if err != nil {
			return nil, fmt.Errorf("benchmark run %d failed: %w", i+1, err)
		}
		results = append(results, result)
	}

	// Calculate summary statistics
	summary := ef.calculateBenchmarkSummary(benchmarkName, results)

	// Store benchmark
	ef.mutex.Lock()
	ef.benchmarks[benchmarkName] = benchmark
	ef.results.BenchmarkResults[benchmarkName] = &summary
	ef.mutex.Unlock()

	return &summary, nil
}

// ComparisonResult represents the result of an A/B test comparison
type ComparisonResult struct {
	StrategyA     string    `json:"strategy_a"`
	StrategyB     string    `json:"strategy_b"`
	ScoreA        float64   `json:"score_a"`
	ScoreB        float64   `json:"score_b"`
	Winner        string    `json:"winner"`
	Improvement   float64   `json:"improvement_percent"`
	PValue        float64   `json:"p_value"`
	IsSignificant bool      `json:"is_significant"`
	Confidence    float64   `json:"confidence"`
	SampleSize    int       `json:"sample_size"`
	Timestamp     time.Time `json:"timestamp"`
}

// Quality Metrics Implementation

// CoverageMetric evaluates how well chunks cover the original text
type CoverageMetric struct{}

func (cm *CoverageMetric) Name() string { return "coverage" }
func (cm *CoverageMetric) Description() string {
	return "Measures how well chunks cover the original text"
}

func (cm *CoverageMetric) Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error) {
	if len(chunks) == 0 {
		return 0.0, nil
	}

	totalCoverage := 0
	for _, chunk := range chunks {
		totalCoverage += len(chunk.Text)
	}

	coverage := float64(totalCoverage) / float64(len(originalText))
	if coverage > 1.0 {
		coverage = 1.0 // Account for overlap
	}

	return coverage, nil
}

// CoherenceMetric evaluates semantic coherence of chunks
type CoherenceMetric struct{}

func (cm *CoherenceMetric) Name() string        { return "coherence" }
func (cm *CoherenceMetric) Description() string { return "Measures semantic coherence within chunks" }

func (cm *CoherenceMetric) Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error) {
	if len(chunks) == 0 {
		return 0.0, nil
	}

	coherentChunks := 0
	for _, chunk := range chunks {
		if isCoherent(chunk) {
			coherentChunks++
		}
	}

	return float64(coherentChunks) / float64(len(chunks)), nil
}

// BoundaryQualityMetric evaluates the quality of chunk boundaries
type BoundaryQualityMetric struct{}

func (bqm *BoundaryQualityMetric) Name() string { return "boundary_quality" }
func (bqm *BoundaryQualityMetric) Description() string {
	return "Measures the quality of chunk boundaries"
}

func (bqm *BoundaryQualityMetric) Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error) {
	if len(chunks) == 0 {
		return 0.0, nil
	}

	goodBoundaries := 0
	for _, chunk := range chunks {
		if hasGoodBoundary(chunk) {
			goodBoundaries++
		}
	}

	return float64(goodBoundaries) / float64(len(chunks)), nil
}

// SizeConsistencyMetric evaluates consistency of chunk sizes
type SizeConsistencyMetric struct{}

func (scm *SizeConsistencyMetric) Name() string        { return "size_consistency" }
func (scm *SizeConsistencyMetric) Description() string { return "Measures consistency of chunk sizes" }

func (scm *SizeConsistencyMetric) Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error) {
	if len(chunks) <= 1 {
		return 1.0, nil
	}

	sizes := make([]int, len(chunks))
	for i, chunk := range chunks {
		sizes[i] = chunk.TokenCount
	}

	return calculateSizeConsistency(sizes), nil
}

// calculateSizeConsistency calculates consistency of chunk sizes
func calculateSizeConsistency(sizes []int) float64 {
	if len(sizes) <= 1 {
		return 1.0
	}
	
	// Calculate mean
	sum := 0
	for _, size := range sizes {
		sum += size
	}
	mean := float64(sum) / float64(len(sizes))
	
	// Calculate variance
	variance := 0.0
	for _, size := range sizes {
		diff := float64(size) - mean
		variance += diff * diff
	}
	variance /= float64(len(sizes))
	
	// Return consistency score (lower variance = higher consistency)
	// Use coefficient of variation inverted
	if mean == 0 {
		return 1.0
	}
	cv := math.Sqrt(variance) / mean
	return math.Max(0.0, 1.0 - cv)
}

// SemanticSimilarityMetric evaluates semantic similarity within chunks
type SemanticSimilarityMetric struct{}

func (ssm *SemanticSimilarityMetric) Name() string { return "semantic_similarity" }
func (ssm *SemanticSimilarityMetric) Description() string {
	return "Measures semantic similarity within chunks"
}

func (ssm *SemanticSimilarityMetric) Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error) {
	// This would require embedding calculations - simplified for now
	if len(chunks) == 0 {
		return 0.0, nil
	}

	// Simplified semantic similarity based on sentence structure
	semanticScore := 0.0
	for _, chunk := range chunks {
		if len(chunk.Sentences) > 0 {
			semanticScore += calculateSemanticCoherence(chunk.Sentences)
		}
	}

	return semanticScore / float64(len(chunks)), nil
}

// Helper functions for evaluation framework

func (ef *EvaluationFramework) evaluateStrategyOnDataset(ctx context.Context, strategyName string, pipeline *ChunkingPipeline, dataset *EvaluationDataset) (float64, []*TestResult, error) {
	testResults := make([]*TestResult, 0)
	totalScore := 0.0

	for _, document := range dataset.Documents {
		result, err := ef.evaluateDocument(ctx, strategyName, pipeline, document, dataset)
		if err != nil {
			// Continue with other documents but record the error
			result = &TestResult{
				TestName:  fmt.Sprintf("%s_%s", dataset.Name, document.ID),
				Strategy:  strategyName,
				Dataset:   dataset.Name,
				Document:  document.ID,
				Passed:    false,
				Score:     0.0,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
		}

		testResults = append(testResults, result)
		totalScore += result.Score
	}

	averageScore := 0.0
	if len(dataset.Documents) > 0 {
		averageScore = totalScore / float64(len(dataset.Documents))
	}

	return averageScore, testResults, nil
}

func (ef *EvaluationFramework) evaluateDocument(ctx context.Context, strategyName string, pipeline *ChunkingPipeline, document *EvaluationDocument, dataset *EvaluationDataset) (*TestResult, error) {
	start := time.Now()

	// Process document through pipeline
	result, err := pipeline.Process(ctx, document.Text, document.Metadata)
	processingTime := time.Since(start)

	testResult := &TestResult{
		TestName:       fmt.Sprintf("%s_%s", dataset.Name, document.ID),
		Strategy:       strategyName,
		Dataset:        dataset.Name,
		Document:       document.ID,
		ProcessingTime: processingTime,
		Metrics:        make(map[string]float64),
		Timestamp:      time.Now(),
	}

	if err != nil {
		testResult.Passed = false
		testResult.Score = 0.0
		testResult.Error = err.Error()
		return testResult, err
	}

	// Get ground truth for this document
	groundTruth := ef.getGroundTruth(document.ID, dataset)

	// Evaluate using quality metrics
	qualityScores := make(map[string]float64)
	totalQuality := 0.0

	for _, metric := range ef.qualityMetrics {
		score, err := metric.Evaluate(result.Chunks, groundTruth, document.Text)
		if err != nil {
			continue // Skip failed metrics
		}

		qualityScores[metric.Name()] = score
		totalQuality += score
		testResult.Metrics[metric.Name()] = score
	}

	// Calculate overall score
	testResult.Score = totalQuality / float64(len(ef.qualityMetrics))
	testResult.Passed = testResult.Score >= ef.config.MinQualityScore &&
		processingTime <= ef.config.MaxResponseTime

	return testResult, nil
}

func (ef *EvaluationFramework) getGroundTruth(documentID string, dataset *EvaluationDataset) []*GroundTruthChunk {
	var groundTruth []*GroundTruthChunk
	for _, gt := range dataset.GroundTruth {
		if gt.DocumentID == documentID {
			groundTruth = append(groundTruth, gt)
		}
	}
	return groundTruth
}

// Additional helper functions would be implemented here...
// This includes functions for calculating various metrics, generating datasets,
// statistical analysis, and benchmark operations.

// Example implementations of some helper functions:

func isCoherent(chunk *Chunk) bool {
	// Simple coherence check based on sentence structure
	if len(chunk.Sentences) == 0 {
		return false
	}

	// Check if sentences flow logically (simplified)
	return len(chunk.Sentences) >= 1 && strings.TrimSpace(chunk.Text) != ""
}

func hasGoodBoundary(chunk *Chunk) bool {
	text := strings.TrimSpace(chunk.Text)
	if len(text) == 0 {
		return false
	}

	// Check if chunk ends with proper punctuation
	lastChar := text[len(text)-1]
	return lastChar == '.' || lastChar == '!' || lastChar == '?' || lastChar == '\n'
}

func calculateSemanticCoherence(sentences []string) float64 {
	if len(sentences) <= 1 {
		return 1.0
	}

	// Simplified coherence calculation
	coherentPairs := 0
	for i := 0; i < len(sentences)-1; i++ {
		if hasSemanticConnection(sentences[i], sentences[i+1]) {
			coherentPairs++
		}
	}

	return float64(coherentPairs) / float64(len(sentences)-1)
}

func hasSemanticConnection(sentence1, sentence2 string) bool {
	// Very simplified semantic connection check
	words1 := strings.Fields(strings.ToLower(sentence1))
	words2 := strings.Fields(strings.ToLower(sentence2))

	commonWords := 0
	for _, word1 := range words1 {
		for _, word2 := range words2 {
			if word1 == word2 && len(word1) > 3 { // Only count meaningful words
				commonWords++
			}
		}
	}

	// If they share at least one meaningful word, consider them connected
	return commonWords > 0
}

// Dataset generation functions would be implemented here...
func generateScientificPapers() []*EvaluationDocument {
	return []*EvaluationDocument{
		{
			ID:          "paper_1",
			Text:        `Abstract: This paper presents a novel approach to machine learning...`,
			ContentType: "scientific_paper",
			Language:    "en",
			Metadata:    map[string]interface{}{"type": "research_paper"},
		},
		// More papers would be added here
	}
}

func generateNewsArticles() []*EvaluationDocument {
	// Implementation for news articles dataset
	return []*EvaluationDocument{}
}

func generateTechnicalDocs() []*EvaluationDocument {
	// Implementation for technical documentation dataset
	return []*EvaluationDocument{}
}

func generateMixedContent() []*EvaluationDocument {
	// Implementation for mixed content dataset
	return []*EvaluationDocument{}
}

// Missing method implementations

func (ef *EvaluationFramework) calculateQualityScore(testResults []*TestResult) float64 {
	if len(testResults) == 0 {
		return 0.0
	}
	
	totalScore := 0.0
	for _, result := range testResults {
		if result.Passed {
			totalScore += result.Score
		}
	}
	
	return totalScore / float64(len(testResults))
}

func (ef *EvaluationFramework) calculatePerformanceScore(testResults []*TestResult) float64 {
	if len(testResults) == 0 {
		return 0.0
	}
	
	totalScore := 0.0
	baselineTime := 100 * time.Millisecond // baseline processing time
	
	for _, result := range testResults {
		// Score based on processing time - faster is better
		timeScore := 1.0 - (float64(result.ProcessingTime) / float64(baselineTime))
		if timeScore < 0 {
			timeScore = 0
		}
		totalScore += timeScore
	}
	
	return totalScore / float64(len(testResults))
}

func (ef *EvaluationFramework) calculateReliabilityScore(testResults []*TestResult) float64 {
	if len(testResults) == 0 {
		return 0.0
	}
	
	passedTests := 0
	for _, result := range testResults {
		if result.Passed {
			passedTests++
		}
	}
	
	return float64(passedTests) / float64(len(testResults))
}

func (ef *EvaluationFramework) generateInsights(evaluation *StrategyEvaluation) ([]string, []string) {
	strengths := []string{}
	weaknesses := []string{}
	
	if evaluation.QualityScore > 0.8 {
		strengths = append(strengths, "High quality chunk generation")
	} else if evaluation.QualityScore < 0.6 {
		weaknesses = append(weaknesses, "Below average chunk quality")
	}
	
	if evaluation.PerformanceScore > 0.8 {
		strengths = append(strengths, "Excellent processing performance")
	} else if evaluation.PerformanceScore < 0.6 {
		weaknesses = append(weaknesses, "Performance optimization needed")
	}
	
	if evaluation.ReliabilityScore > 0.9 {
		strengths = append(strengths, "Very reliable processing")
	} else if evaluation.ReliabilityScore < 0.8 {
		weaknesses = append(weaknesses, "Reliability concerns")
	}
	
	return strengths, weaknesses
}

func (ef *EvaluationFramework) calculatePValue(resultsA, resultsB []*TestResult) float64 {
	// Simplified p-value calculation - in production would use proper statistical test
	if len(resultsA) == 0 || len(resultsB) == 0 {
		return 1.0
	}
	
	// Calculate means
	sumA, sumB := 0.0, 0.0
	for _, r := range resultsA {
		sumA += r.Score
	}
	for _, r := range resultsB {
		sumB += r.Score
	}
	
	meanA := sumA / float64(len(resultsA))
	meanB := sumB / float64(len(resultsB))
	
	// Simple heuristic: larger difference = lower p-value
	diff := meanA - meanB
	if diff < 0 {
		diff = -diff
	}
	
	// Scale to [0, 1] where larger differences have lower p-values
	pValue := 1.0 - (diff * 2.0) // Simplified
	if pValue < 0 {
		pValue = 0.001 // Minimum p-value
	}
	
	return pValue
}

func (ef *EvaluationFramework) runBenchmarkIteration(ctx context.Context, benchmark *Benchmark, dataset *EvaluationDataset, pipeline *ChunkingPipeline) (*BenchmarkResult, error) {
	start := time.Now()
	
	// Process a sample document from the dataset
	if len(dataset.Documents) == 0 {
		return nil, fmt.Errorf("no documents in dataset")
	}
	
	doc := dataset.Documents[0]
	_, err := pipeline.Process(ctx, doc.Text, doc.Metadata)
	processingTime := time.Since(start)
	
	result := &BenchmarkResult{
		RunID:          fmt.Sprintf("%s-%d", benchmark.Name, time.Now().UnixNano()),
		Timestamp:      time.Now(),
		ProcessingTime: processingTime,
		MemoryUsage:    1024 * 1024, // Placeholder: 1MB
		QualityScore:   0.8,         // Placeholder
		ChunkCount:     10,          // Placeholder
		ErrorCount:     0,
		Metrics:        make(map[string]float64),
	}
	
	if err != nil {
		result.ErrorCount = 1
		result.QualityScore = 0.0
	}
	
	return result, nil
}

func (ef *EvaluationFramework) calculateBenchmarkSummary(benchmarkName string, results []*BenchmarkResult) BenchmarkSummary {
	if len(results) == 0 {
		return BenchmarkSummary{
			BenchmarkName: benchmarkName,
			RunCount:      0,
		}
	}
	
	totalTime := time.Duration(0)
	minTime := results[0].ProcessingTime
	maxTime := results[0].ProcessingTime
	
	for _, result := range results {
		totalTime += result.ProcessingTime
		if result.ProcessingTime < minTime {
			minTime = result.ProcessingTime
		}
		if result.ProcessingTime > maxTime {
			maxTime = result.ProcessingTime
		}
	}
	
	avgTime := totalTime / time.Duration(len(results))
	
	return BenchmarkSummary{
		BenchmarkName:    benchmarkName,
		RunCount:         len(results),
		AverageTime:      avgTime,
		MedianTime:       avgTime, // Simplified
		MinTime:          minTime,
		MaxTime:          maxTime,
		ThroughputQPS:    1000.0 / float64(avgTime.Milliseconds()), // Simplified
		MemoryEfficiency: 0.85, // Placeholder
	}
}
