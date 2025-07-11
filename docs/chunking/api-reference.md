# API Reference - Enhanced Chunking System

## Core Interfaces

### Chunker Interface
The primary interface for all chunking strategies.

```go
type Chunker interface {
    // Chunk processes text and returns chunks
    Chunk(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error)
    
    // Name returns the strategy name
    Name() string
    
    // Configure updates the chunker configuration
    Configure(config interface{}) error
}
```

### QualityMetric Interface
Interface for implementing quality assessment metrics.

```go
type QualityMetric interface {
    // Name returns the metric name
    Name() string
    
    // Description returns a detailed description
    Description() string
    
    // Evaluate assesses chunk quality
    Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error)
}
```

### EmbeddingProvider Interface
Abstraction for embedding services.

```go
type EmbeddingProvider interface {
    // GetEmbedding returns embedding for text
    GetEmbedding(ctx context.Context, text string) ([]float64, error)
    
    // GetEmbeddings returns embeddings for multiple texts
    GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
    
    // Dimensions returns embedding dimensions
    Dimensions() int
    
    // Name returns provider name
    Name() string
}
```

## Core Data Structures

### Chunk
Represents a processed text chunk with metadata.

```go
type Chunk struct {
    ID           string                 `json:"id"`
    Text         string                 `json:"text"`
    TokenCount   int                    `json:"token_count"`
    StartIndex   int                    `json:"start_index"`
    EndIndex     int                    `json:"end_index"`
    Metadata     map[string]interface{} `json:"metadata"`
    
    // Semantic information
    Embedding    []float64              `json:"embedding,omitempty"`
    Sentences    []string               `json:"sentences"`
    Keywords     []string               `json:"keywords"`
    
    // Quality metrics
    QualityScore float64                `json:"quality_score"`
    Confidence   float64                `json:"confidence"`
    
    // Hierarchical information
    ParentID     string                 `json:"parent_id,omitempty"`
    ChildIDs     []string               `json:"child_ids,omitempty"`
    Level        int                    `json:"level"`
    
    // Context information
    Context      string                 `json:"context,omitempty"`
    
    // Multi-modal information
    ContentType  string                 `json:"content_type"`
    CodeLanguage string                 `json:"code_language,omitempty"`
    
    // Processing metadata
    Strategy     string                 `json:"strategy"`
    ProcessedAt  time.Time              `json:"processed_at"`
}
```

### PipelineResult
Result from chunking pipeline processing.

```go
type PipelineResult struct {
    Chunks            []*Chunk                `json:"chunks"`
    QualityScore      float64                 `json:"quality_score"`
    ProcessingTime    time.Duration           `json:"processing_time"`
    Strategy          string                  `json:"strategy"`
    TokenCount        int                     `json:"token_count"`
    Metadata          map[string]interface{}  `json:"metadata"`
    QualityAssessment *QualityAssessment      `json:"quality_assessment,omitempty"`
}
```

### QualityAssessment
Comprehensive quality assessment results.

```go
type QualityAssessment struct {
    OverallScore      float64               `json:"overall_score"`
    Coherence         float64               `json:"coherence"`
    Completeness      float64               `json:"completeness"`
    Relevance         float64               `json:"relevance"`
    InformationDensity float64              `json:"information_density"`
    Readability       float64               `json:"readability"`
    SemanticIntegrity float64               `json:"semantic_integrity"`
    StructurePreservation float64           `json:"structure_preservation"`
    OverlapOptimization float64             `json:"overlap_optimization"`
    
    Issues            []QualityIssue        `json:"issues"`
    Suggestions       []QualitySuggestion   `json:"suggestions"`
    ProcessingTime    time.Duration         `json:"processing_time"`
    Metadata          map[string]interface{} `json:"metadata"`
}
```

## Chunking Pipeline

### ChunkingPipeline
Main orchestration component for chunking operations.

```go
type ChunkingPipeline struct {
    strategies map[string]Chunker
    selector   StrategySelector
    config     *ChunkerConfig
    monitor    *RealTimeMonitor
    mutex      sync.RWMutex
}

// NewChunkingPipeline creates a new pipeline
func NewChunkingPipeline(config *ChunkerConfig) *ChunkingPipeline

// AddStrategy adds a chunking strategy
func (cp *ChunkingPipeline) AddStrategy(name string, chunker Chunker) error

// Process processes text through the pipeline
func (cp *ChunkingPipeline) Process(ctx context.Context, text string, metadata map[string]interface{}) (*PipelineResult, error)

// SetStrategySelector sets custom strategy selection logic
func (cp *ChunkingPipeline) SetStrategySelector(selector StrategySelector)

// GetStrategies returns available strategies
func (cp *ChunkingPipeline) GetStrategies() []string

// RemoveStrategy removes a strategy
func (cp *ChunkingPipeline) RemoveStrategy(name string) error
```

### StrategySelector
Function type for custom strategy selection.

```go
type StrategySelector func(ctx context.Context, text string, metadata map[string]interface{}) string
```

## Chunking Strategies

### EmbeddingBasedChunker
Semantic similarity-based chunking.

```go
// NewEmbeddingBasedChunker creates a new embedding-based chunker
func NewEmbeddingBasedChunker(config *EmbeddingChunkerConfig) *EmbeddingBasedChunker

type EmbeddingChunkerConfig struct {
    SimilarityThreshold float64 `json:"similarity_threshold"`
    EmbeddingProvider   string  `json:"embedding_provider"`
    ModelName          string  `json:"model_name"`
    WindowSize         int     `json:"window_size"`
    MinChunkSize       int     `json:"min_chunk_size"`
    MaxChunkSize       int     `json:"max_chunk_size"`
    EnableBatching     bool    `json:"enable_batching"`
    BatchSize          int     `json:"batch_size"`
}
```

### ContextualChunker
LLM-enhanced context generation.

```go
// NewContextualChunker creates a new contextual chunker
func NewContextualChunker(config *ContextualChunkerConfig) *ContextualChunker

type ContextualChunkerConfig struct {
    LLMProvider       string          `json:"llm_provider"`
    ModelName        string          `json:"model_name"`
    ContextLength    int             `json:"context_length"`
    ContextPosition  ContextPosition `json:"context_position"`
    QualityThreshold float64         `json:"quality_threshold"`
    MaxContextCalls  int             `json:"max_context_calls"`
    ContextTemplate  string          `json:"context_template"`
}
```

### AgenticChunker
AI-driven reasoning for chunking decisions.

```go
// NewAgenticChunker creates a new agentic chunker
func NewAgenticChunker(config *AgenticChunkerConfig) *AgenticChunker

type AgenticChunkerConfig struct {
    LLMProvider          string        `json:"llm_provider"`
    ModelName           string        `json:"model_name"`
    AnalysisDepth       AnalysisDepth `json:"analysis_depth"`
    ReasoningSteps      int           `json:"reasoning_steps"`
    ConfidenceThreshold float64       `json:"confidence_threshold"`
    MaxLLMCalls         int           `json:"max_llm_calls"`
    EnableChainOfThought bool          `json:"enable_chain_of_thought"`
    Temperature         float64       `json:"temperature"`
}
```

### MultiModalChunker
Multi-format content handling.

```go
// NewMultiModalChunker creates a new multi-modal chunker
func NewMultiModalChunker(config *MultiModalChunkerConfig) *MultiModalChunker

type MultiModalChunkerConfig struct {
    PreserveCodeBlocks bool     `json:"preserve_code_blocks"`
    PreserveTables     bool     `json:"preserve_tables"`
    HandleImages       bool     `json:"handle_images"`
    CodeLanguages      []string `json:"code_languages"`
    ImageFormats       []string `json:"image_formats"`
    TableDetection     bool     `json:"table_detection"`
    CodeBlockMinLines  int      `json:"code_block_min_lines"`
    PreserveFormatting bool     `json:"preserve_formatting"`
}
```

### HierarchicalChunker
Parent-child relationships with summaries.

```go
// NewHierarchicalChunker creates a new hierarchical chunker
func NewHierarchicalChunker(config *HierarchicalChunkerConfig) *HierarchicalChunker

type HierarchicalChunkerConfig struct {
    MaxLevels              int         `json:"max_levels"`
    SummaryMode           SummaryMode `json:"summary_mode"`
    PreserveSectionHeaders bool        `json:"preserve_section_headers"`
    CrossReferenceTracking bool        `json:"cross_reference_tracking"`
    ParentChunkSize        int         `json:"parent_chunk_size"`
    ChildChunkSize         int         `json:"child_chunk_size"`
    SummaryLength          int         `json:"summary_length"`
    EnableNavigationLinks  bool        `json:"enable_navigation_links"`
}
```

## Quality Assessment

### QualityMetricsCalculator
Real-time quality assessment engine.

```go
// NewQualityMetricsCalculator creates a new quality calculator
func NewQualityMetricsCalculator(config *QualityMetricsConfig) *QualityMetricsCalculator

// AssessQuality performs comprehensive quality assessment
func (qmc *QualityMetricsCalculator) AssessQuality(ctx context.Context, chunks []*Chunk, originalText string) (*QualityAssessment, error)

// GetMetricHistory returns quality history
func (qmc *QualityMetricsCalculator) GetMetricHistory(metric QualityMetricType, timeRange time.Duration) ([]*QualityDataPoint, error)

// RegisterCustomMetric adds a custom quality metric
func (qmc *QualityMetricsCalculator) RegisterCustomMetric(metric QualityMetric) error
```

### Built-in Quality Metrics

```go
// CoverageMetric evaluates text coverage
type CoverageMetric struct{}
func (cm *CoverageMetric) Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error)

// CoherenceMetric evaluates semantic coherence
type CoherenceMetric struct{}
func (cm *CoherenceMetric) Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error)

// BoundaryQualityMetric evaluates boundary quality
type BoundaryQualityMetric struct{}
func (bqm *BoundaryQualityMetric) Evaluate(chunks []*Chunk, groundTruth []*GroundTruthChunk, originalText string) (float64, error)
```

## Production Features

### ProductionOptimizer
Production-grade optimizations.

```go
// NewProductionOptimizer creates a new optimizer
func NewProductionOptimizer(config *ProductionConfig) *ProductionOptimizer

// OptimizeChunking applies production optimizations
func (po *ProductionOptimizer) OptimizeChunking(ctx context.Context, pipeline *ChunkingPipeline, text string, metadata map[string]interface{}) (*PipelineResult, error)

// GetMetrics returns production metrics
func (po *ProductionOptimizer) GetMetrics() *ProductionMetrics

// IsHealthy returns health status
func (po *ProductionOptimizer) IsHealthy() bool
```

### RealTimeMonitor
Comprehensive monitoring and alerting.

```go
// NewRealTimeMonitor creates a new monitor
func NewRealTimeMonitor(config *MonitoringConfig) *RealTimeMonitor

// Start begins monitoring
func (rtm *RealTimeMonitor) Start()

// Stop stops monitoring
func (rtm *RealTimeMonitor) Stop()

// RecordRequest records request metrics
func (rtm *RealTimeMonitor) RecordRequest(duration time.Duration, success bool)

// GetHealthStatus returns health status
func (rtm *RealTimeMonitor) GetHealthStatus() *MonitoringHealthStatus

// SendAlert sends monitoring alert
func (rtm *RealTimeMonitor) SendAlert(alert Alert)
```

### CircuitBreaker
Circuit breaker pattern implementation.

```go
// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker

// CanExecute checks if execution is allowed
func (cb *CircuitBreaker) CanExecute() bool

// RecordSuccess records successful execution
func (cb *CircuitBreaker) RecordSuccess()

// RecordFailure records failed execution
func (cb *CircuitBreaker) RecordFailure()

// GetState returns current state
func (cb *CircuitBreaker) GetState() string
```

## Evaluation Framework

### EvaluationFramework
A/B testing and benchmarking.

```go
// NewEvaluationFramework creates a new framework
func NewEvaluationFramework(config *EvaluationConfig) *EvaluationFramework

// EvaluateStrategy evaluates a single strategy
func (ef *EvaluationFramework) EvaluateStrategy(ctx context.Context, strategyName string, pipeline *ChunkingPipeline) (*StrategyEvaluation, error)

// RunComparison runs A/B test comparison
func (ef *EvaluationFramework) RunComparison(ctx context.Context, strategyA, strategyB string, pipeline *ChunkingPipeline) (*ComparisonResult, error)

// RunBenchmark runs performance benchmark
func (ef *EvaluationFramework) RunBenchmark(ctx context.Context, benchmarkName string, strategy string, pipeline *ChunkingPipeline) (*BenchmarkSummary, error)

// LoadStandardDatasets loads evaluation datasets
func (ef *EvaluationFramework) LoadStandardDatasets() error
```

## Error Handling

### Common Errors

```go
var (
    ErrInvalidChunkSize    = errors.New("invalid chunk size")
    ErrInvalidOverlap      = errors.New("invalid overlap size")
    ErrStrategyNotFound    = errors.New("strategy not found")
    ErrQualityTooLow       = errors.New("quality below threshold")
    ErrRateLimitExceeded   = errors.New("rate limit exceeded")
    ErrCircuitBreakerOpen  = errors.New("circuit breaker is open")
    ErrResourceLimitReached = errors.New("resource limit reached")
)

// ChunkingError represents chunking-specific errors
type ChunkingError struct {
    Type    string
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e *ChunkingError) Error() string
func (e *ChunkingError) Unwrap() error
```

## Utilities

### TokenEstimator
Token counting abstraction.

```go
type TokenEstimator interface {
    EstimateTokens(text string) int
    CountTokens(text string) int
    TruncateToTokens(text string, maxTokens int) string
}

// NewTikTokenEstimator creates TikToken-based estimator
func NewTikTokenEstimator(model string) (TokenEstimator, error)

// NewSimpleTokenEstimator creates simple word-based estimator
func NewSimpleTokenEstimator() TokenEstimator
```

### SemanticAnalyzer
Linguistic analysis engine.

```go
// NewSemanticAnalyzer creates a new analyzer
func NewSemanticAnalyzer(config *SemanticAnalyzerConfig) *SemanticAnalyzer

// AnalyzeSentences extracts sentences from text
func (sa *SemanticAnalyzer) AnalyzeSentences(text string) ([]string, error)

// ExtractKeywords extracts key terms
func (sa *SemanticAnalyzer) ExtractKeywords(text string) ([]string, error)

// DetectLanguage identifies text language
func (sa *SemanticAnalyzer) DetectLanguage(text string) (string, float64, error)

// CalculateSimilarity computes semantic similarity
func (sa *SemanticAnalyzer) CalculateSimilarity(text1, text2 string) (float64, error)
```

## Configuration Management

### ConfigValidator
Configuration validation utilities.

```go
// NewConfigValidator creates a new validator
func NewConfigValidator() *ConfigValidator

// ValidateConfig validates configuration
func (cv *ConfigValidator) ValidateConfig(config *AdvancedChunkerConfig) error

// ValidateBaseConfig validates base configuration
func (cv *ConfigValidator) validateBaseConfig(config *ChunkerConfig) error
```

### ConfigSerializer
Configuration serialization utilities.

```go
// NewConfigSerializer creates a new serializer
func NewConfigSerializer() *ConfigSerializer

// ToJSON serializes to JSON
func (cs *ConfigSerializer) ToJSON(config *AdvancedChunkerConfig) ([]byte, error)

// FromJSON deserializes from JSON
func (cs *ConfigSerializer) FromJSON(data []byte) (*AdvancedChunkerConfig, error)
```

## Constants and Enums

### Quality Metrics

```go
const (
    MetricCoherence         QualityMetricType = "coherence"
    MetricCompleteness      QualityMetricType = "completeness"
    MetricRelevance         QualityMetricType = "relevance"
    MetricInformation       QualityMetricType = "information_density"
    MetricReadability       QualityMetricType = "readability"
    MetricSemanticIntegrity QualityMetricType = "semantic_integrity"
    MetricStructure         QualityMetricType = "structure_preservation"
    MetricOverlap           QualityMetricType = "overlap_optimization"
)
```

### Analysis Depth

```go
const (
    AnalysisDepthBasic        AnalysisDepth = "basic"
    AnalysisDepthIntermediate AnalysisDepth = "intermediate"
    AnalysisDepthExpert       AnalysisDepth = "expert"
)
```

### Configuration Presets

```go
const (
    PresetDefault        ConfigurationPreset = "default"
    PresetHighQuality    ConfigurationPreset = "high_quality"
    PresetHighPerformance ConfigurationPreset = "high_performance"
    PresetBalanced       ConfigurationPreset = "balanced"
    PresetRAG            ConfigurationPreset = "rag"
    PresetResearchPaper  ConfigurationPreset = "research_paper"
    PresetTechnicalDoc   ConfigurationPreset = "technical_doc"
    PresetChatbot        ConfigurationPreset = "chatbot"
)
```

### Circuit Breaker States

```go
const (
    StateClosed   CircuitState = iota // Allowing requests
    StateOpen                         // Blocking requests
    StateHalfOpen                     // Testing recovery
)
```

### Alert Severities

```go
const (
    AlertSeverityInfo AlertSeverity = iota
    AlertSeverityWarning
    AlertSeverityError
    AlertSeverityCritical
)
```

## Examples

### Basic Usage

```go
// Create pipeline
config := chunkers.DefaultChunkerConfig()
pipeline := chunkers.NewChunkingPipeline(config)

// Add strategy
chunker := chunkers.NewEmbeddingBasedChunker(&chunkers.EmbeddingChunkerConfig{
    SimilarityThreshold: 0.8,
})
pipeline.AddStrategy("semantic", chunker)

// Process text
result, err := pipeline.Process(ctx, text, nil)
if err != nil {
    return err
}

// Check quality
if result.QualityScore < 0.7 {
    log.Println("Quality below threshold")
}
```

### Production Usage

```go
// Production configuration
config := chunkers.GetPresetConfig(chunkers.PresetHighPerformance)

// Create optimizer
optimizer := chunkers.NewProductionOptimizer(chunkers.DefaultProductionConfig())
optimizer.Start()

// Process with optimizations
result, err := optimizer.OptimizeChunking(ctx, pipeline, text, metadata)
if err != nil {
    return err
}

// Monitor health
if !optimizer.IsHealthy() {
    log.Println("System unhealthy")
}
```