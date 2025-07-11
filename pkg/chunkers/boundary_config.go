package chunkers

import (
	"fmt"
	"time"
)

// SemanticBoundaryConfig configures semantic boundary detection methods
type SemanticBoundaryConfig struct {
	// Method for boundary detection
	Method BoundaryDetectionMethod `json:"method"`
	
	// ThresholdMethod for calculating optimal thresholds
	ThresholdMethod ThresholdMethod `json:"threshold_method"`
	
	// PercentileThreshold for percentile-based detection (0.0-1.0)
	PercentileThreshold float64 `json:"percentile_threshold"`
	
	// InterquartileMultiplier for IQR-based detection
	InterquartileMultiplier float64 `json:"interquartile_multiplier"`
	
	// GradientSensitivity for gradient-based detection
	GradientSensitivity float64 `json:"gradient_sensitivity"`
	
	// AdaptiveWindowSize for adaptive threshold calculation
	AdaptiveWindowSize int `json:"adaptive_window_size"`
	
	// MinBoundaryDistance minimum distance between boundaries
	MinBoundaryDistance int `json:"min_boundary_distance"`
	
	// EnableSmoothing applies smoothing to similarity scores
	EnableSmoothing bool `json:"enable_smoothing"`
	
	// SmoothingWindow size for smoothing
	SmoothingWindow int `json:"smoothing_window"`
	
	// EnablePostProcessing applies post-processing to detected boundaries
	EnablePostProcessing bool `json:"enable_post_processing"`
	
	// QualityThreshold minimum quality score for boundaries
	QualityThreshold float64 `json:"quality_threshold"`
}

// DefaultSemanticBoundaryConfig returns default boundary detection configuration
func DefaultSemanticBoundaryConfig() *SemanticBoundaryConfig {
	return &SemanticBoundaryConfig{
		Method:                  BoundaryMethodPercentile,
		ThresholdMethod:         ThresholdMethodStdDev,
		PercentileThreshold:     0.7,
		InterquartileMultiplier: 1.5,
		GradientSensitivity:     0.5,
		AdaptiveWindowSize:      5,
		MinBoundaryDistance:     2,
		EnableSmoothing:         true,
		SmoothingWindow:         3,
		EnablePostProcessing:    true,
		QualityThreshold:        0.6,
	}
}

// Validate validates the boundary configuration
func (sbc *SemanticBoundaryConfig) Validate() error {
	if sbc.PercentileThreshold < 0 || sbc.PercentileThreshold > 1 {
		return fmt.Errorf("percentile threshold must be between 0 and 1")
	}
	
	if sbc.InterquartileMultiplier < 0 {
		return fmt.Errorf("interquartile multiplier must be non-negative")
	}
	
	if sbc.GradientSensitivity < 0 || sbc.GradientSensitivity > 1 {
		return fmt.Errorf("gradient sensitivity must be between 0 and 1")
	}
	
	if sbc.AdaptiveWindowSize < 1 {
		return fmt.Errorf("adaptive window size must be positive")
	}
	
	if sbc.MinBoundaryDistance < 0 {
		return fmt.Errorf("minimum boundary distance must be non-negative")
	}
	
	if sbc.SmoothingWindow < 1 {
		return fmt.Errorf("smoothing window must be positive")
	}
	
	if sbc.QualityThreshold < 0 || sbc.QualityThreshold > 1 {
		return fmt.Errorf("quality threshold must be between 0 and 1")
	}
	
	return nil
}

// Enhanced Semantic Chunker Configuration
type EnhancedSemanticConfig struct {
	// Base chunker configuration
	*ChunkerConfig
	
	// Boundary detection configuration
	BoundaryConfig *SemanticBoundaryConfig `json:"boundary_config"`
	
	// EnableEmbeddingAnalysis uses embeddings for semantic analysis
	EnableEmbeddingAnalysis bool `json:"enable_embedding_analysis"`
	
	// EmbeddingBatchSize for batch processing embeddings
	EmbeddingBatchSize int `json:"embedding_batch_size"`
	
	// SimilarityThreshold for semantic similarity
	SimilarityThreshold float64 `json:"similarity_threshold"`
	
	// ContextWindow size for maintaining coherence
	ContextWindow int `json:"context_window"`
	
	// EnableQualityScoring calculates quality scores for chunks
	EnableQualityScoring bool `json:"enable_quality_scoring"`
	
	// QualityWeights for different quality metrics
	QualityWeights map[string]float64 `json:"quality_weights"`
	
	// EnableOptimization optimizes chunk boundaries
	EnableOptimization bool `json:"enable_optimization"`
	
	// OptimizationIterations maximum iterations for optimization
	OptimizationIterations int `json:"optimization_iterations"`
	
	// CacheEmbeddings enables embedding caching
	CacheEmbeddings bool `json:"cache_embeddings"`
	
	// CacheTTL time-to-live for cached embeddings
	CacheTTL time.Duration `json:"cache_ttl"`
}

// DefaultEnhancedSemanticConfig returns default enhanced semantic configuration
func DefaultEnhancedSemanticConfig() *EnhancedSemanticConfig {
	return &EnhancedSemanticConfig{
		ChunkerConfig:           DefaultChunkerConfig(),
		BoundaryConfig:          DefaultSemanticBoundaryConfig(),
		EnableEmbeddingAnalysis: true,
		EmbeddingBatchSize:      10,
		SimilarityThreshold:     0.7,
		ContextWindow:          3,
		EnableQualityScoring:   true,
		QualityWeights: map[string]float64{
			"coherence":    0.4,
			"completeness": 0.3,
			"size_balance": 0.2,
			"boundary_quality": 0.1,
		},
		EnableOptimization:     false, // Disabled by default for performance
		OptimizationIterations: 3,
		CacheEmbeddings:       true,
		CacheTTL:              1 * time.Hour,
	}
}

// Validate validates the enhanced semantic configuration
func (esc *EnhancedSemanticConfig) Validate() error {
	if esc.ChunkerConfig != nil {
		// Use factory to validate base config
		factory := NewChunkerFactory()
		if err := factory.ValidateConfig(esc.ChunkerConfig); err != nil {
			return fmt.Errorf("invalid base config: %w", err)
		}
	}
	
	if esc.BoundaryConfig != nil {
		if err := esc.BoundaryConfig.Validate(); err != nil {
			return fmt.Errorf("invalid boundary config: %w", err)
		}
	}
	
	if esc.EmbeddingBatchSize < 1 {
		return fmt.Errorf("embedding batch size must be positive")
	}
	
	if esc.SimilarityThreshold < 0 || esc.SimilarityThreshold > 1 {
		return fmt.Errorf("similarity threshold must be between 0 and 1")
	}
	
	if esc.ContextWindow < 1 {
		return fmt.Errorf("context window must be positive")
	}
	
	if esc.OptimizationIterations < 1 {
		return fmt.Errorf("optimization iterations must be positive")
	}
	
	if esc.CacheTTL < 0 {
		return fmt.Errorf("cache TTL must be non-negative")
	}
	
	// Validate quality weights
	if esc.QualityWeights != nil {
		totalWeight := 0.0
		for _, weight := range esc.QualityWeights {
			if weight < 0 || weight > 1 {
				return fmt.Errorf("quality weights must be between 0 and 1")
			}
			totalWeight += weight
		}
		
		if totalWeight > 1.01 { // Allow small floating point errors
			return fmt.Errorf("total quality weights cannot exceed 1.0")
		}
	}
	
	return nil
}

// ChunkQualityMetrics contains quality metrics for chunks
type ChunkQualityMetrics struct {
	// CoherenceScore measures semantic coherence within the chunk
	CoherenceScore float64 `json:"coherence_score"`
	
	// CompletenessScore measures if the chunk contains complete ideas
	CompletenessScore float64 `json:"completeness_score"`
	
	// SizeBalanceScore measures if the chunk size is appropriate
	SizeBalanceScore float64 `json:"size_balance_score"`
	
	// BoundaryQualityScore measures the quality of chunk boundaries
	BoundaryQualityScore float64 `json:"boundary_quality_score"`
	
	// OverallQualityScore weighted combination of all metrics
	OverallQualityScore float64 `json:"overall_quality_score"`
	
	// SemanticDensity measures the density of meaningful content
	SemanticDensity float64 `json:"semantic_density"`
	
	// TopicConsistency measures consistency of topics within chunk
	TopicConsistency float64 `json:"topic_consistency"`
	
	// InformationValue measures the information content
	InformationValue float64 `json:"information_value"`
}

// CalculateQualityScore calculates overall quality score from individual metrics
func (cqm *ChunkQualityMetrics) CalculateQualityScore(weights map[string]float64) {
	if weights == nil {
		// Default equal weights
		cqm.OverallQualityScore = (cqm.CoherenceScore + cqm.CompletenessScore + 
			cqm.SizeBalanceScore + cqm.BoundaryQualityScore) / 4.0
		return
	}
	
	score := 0.0
	totalWeight := 0.0
	
	if weight, exists := weights["coherence"]; exists {
		score += weight * cqm.CoherenceScore
		totalWeight += weight
	}
	
	if weight, exists := weights["completeness"]; exists {
		score += weight * cqm.CompletenessScore
		totalWeight += weight
	}
	
	if weight, exists := weights["size_balance"]; exists {
		score += weight * cqm.SizeBalanceScore
		totalWeight += weight
	}
	
	if weight, exists := weights["boundary_quality"]; exists {
		score += weight * cqm.BoundaryQualityScore
		totalWeight += weight
	}
	
	if totalWeight > 0 {
		cqm.OverallQualityScore = score / totalWeight
	} else {
		cqm.OverallQualityScore = (cqm.CoherenceScore + cqm.CompletenessScore + 
			cqm.SizeBalanceScore + cqm.BoundaryQualityScore) / 4.0
	}
}

// Performance Configuration for Chunking
type ChunkingPerformanceConfig struct {
	// EnableParallelProcessing processes chunks in parallel
	EnableParallelProcessing bool `json:"enable_parallel_processing"`
	
	// MaxWorkers for parallel processing
	MaxWorkers int `json:"max_workers"`
	
	// BatchSize for processing chunks in batches
	BatchSize int `json:"batch_size"`
	
	// EnableProfiling collects performance metrics
	EnableProfiling bool `json:"enable_profiling"`
	
	// MemoryLimit maximum memory usage in MB
	MemoryLimit int `json:"memory_limit"`
	
	// TimeoutDuration maximum time for processing
	TimeoutDuration time.Duration `json:"timeout_duration"`
	
	// EnableProgressive processes text progressively
	EnableProgressive bool `json:"enable_progressive"`
	
	// ProgressiveChunkSize for progressive processing
	ProgressiveChunkSize int `json:"progressive_chunk_size"`
	
	// EnableStreaming enables streaming chunk processing
	EnableStreaming bool `json:"enable_streaming"`
	
	// StreamBufferSize for streaming
	StreamBufferSize int `json:"stream_buffer_size"`
}

// DefaultChunkingPerformanceConfig returns default performance configuration
func DefaultChunkingPerformanceConfig() *ChunkingPerformanceConfig {
	return &ChunkingPerformanceConfig{
		EnableParallelProcessing: true,
		MaxWorkers:              4,
		BatchSize:               50,
		EnableProfiling:         false,
		MemoryLimit:            1024, // 1GB
		TimeoutDuration:        30 * time.Second,
		EnableProgressive:      false,
		ProgressiveChunkSize:   10000, // characters
		EnableStreaming:        false,
		StreamBufferSize:       100,
	}
}

// Validate validates the performance configuration
func (cpc *ChunkingPerformanceConfig) Validate() error {
	if cpc.MaxWorkers < 1 {
		return fmt.Errorf("max workers must be positive")
	}
	
	if cpc.BatchSize < 1 {
		return fmt.Errorf("batch size must be positive")
	}
	
	if cpc.MemoryLimit < 1 {
		return fmt.Errorf("memory limit must be positive")
	}
	
	if cpc.TimeoutDuration < 0 {
		return fmt.Errorf("timeout duration must be non-negative")
	}
	
	if cpc.ProgressiveChunkSize < 1 {
		return fmt.Errorf("progressive chunk size must be positive")
	}
	
	if cpc.StreamBufferSize < 1 {
		return fmt.Errorf("stream buffer size must be positive")
	}
	
	return nil
}

// Advanced Chunking Configuration that combines all config types
type AdvancedChunkingConfig struct {
	// Base chunker configuration
	*ChunkerConfig
	
	// Enhanced semantic configuration
	SemanticConfig *EnhancedSemanticConfig `json:"semantic_config,omitempty"`
	
	// Contextual configuration
	ContextualConfig *ContextualConfig `json:"contextual_config,omitempty"`
	
	// Propositionalization configuration
	PropositionConfig *PropositionConfig `json:"proposition_config,omitempty"`
	
	// Performance configuration
	PerformanceConfig *ChunkingPerformanceConfig `json:"performance_config,omitempty"`
	
	// Global settings
	EnableMetrics     bool `json:"enable_metrics"`
	EnableDebugLogs   bool `json:"enable_debug_logs"`
	LogLevel         string `json:"log_level"`
}

// DefaultAdvancedChunkingConfig returns comprehensive default configuration
func DefaultAdvancedChunkingConfig() *AdvancedChunkingConfig {
	return &AdvancedChunkingConfig{
		ChunkerConfig:     DefaultChunkerConfig(),
		SemanticConfig:    DefaultEnhancedSemanticConfig(),
		ContextualConfig:  DefaultContextualConfig(),
		PropositionConfig: DefaultPropositionConfig(),
		PerformanceConfig: DefaultChunkingPerformanceConfig(),
		EnableMetrics:     true,
		EnableDebugLogs:   false,
		LogLevel:         "INFO",
	}
}

// Validate validates the complete advanced configuration
func (acc *AdvancedChunkingConfig) Validate() error {
	// Validate base config
	if acc.ChunkerConfig != nil {
		factory := NewChunkerFactory()
		if err := factory.ValidateConfig(acc.ChunkerConfig); err != nil {
			return fmt.Errorf("invalid base config: %w", err)
		}
	}
	
	// Validate semantic config
	if acc.SemanticConfig != nil {
		if err := acc.SemanticConfig.Validate(); err != nil {
			return fmt.Errorf("invalid semantic config: %w", err)
		}
	}
	
	// Validate performance config
	if acc.PerformanceConfig != nil {
		if err := acc.PerformanceConfig.Validate(); err != nil {
			return fmt.Errorf("invalid performance config: %w", err)
		}
	}
	
	// Validate log level
	validLogLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	}
	
	if !validLogLevels[acc.LogLevel] {
		return fmt.Errorf("invalid log level: %s", acc.LogLevel)
	}
	
	return nil
}