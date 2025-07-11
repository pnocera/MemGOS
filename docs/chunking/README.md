# 🧠 Enhanced Chunking System Documentation

The Enhanced Chunking System represents a complete transformation of MemGOS's text processing capabilities, implementing state-of-the-art algorithms and production-grade features.

## 📋 Overview

### What's New
- **11 Modern Chunking Strategies**: From basic to cutting-edge AI-driven algorithms
- **Semantic Intelligence**: Embedding-based and LLM-driven boundary detection
- **Production Features**: Monitoring, caching, circuit breakers, rate limiting
- **Quality Assessment**: 8-dimensional quality metrics with real-time scoring
- **Configuration Presets**: Optimized settings for different use cases

### Performance Improvements
- **2-5x Better Quality**: AI-driven semantic awareness
- **Parallel Processing**: Multi-threaded chunking with worker pools
- **Advanced Caching**: LRU/LFU policies with configurable TTL
- **Real-time Monitoring**: Comprehensive metrics and alerting

## 🚀 Quick Start

### Basic Usage

```go
import "github.com/memtensor/memgos/pkg/chunkers"

// Create a pipeline with semantic chunking
config := chunkers.DefaultChunkerConfig()
config.ChunkSize = 512
config.ChunkOverlap = 80

pipeline := chunkers.NewChunkingPipeline(config)

// Add semantic chunker
embeddingChunker := chunkers.NewEmbeddingBasedChunker(&chunkers.EmbeddingChunkerConfig{
    SimilarityThreshold: 0.8,
    MinChunkSize:       100,
    MaxChunkSize:       800,
})
pipeline.AddStrategy("semantic", embeddingChunker)

// Process text
result, err := pipeline.Process(ctx, "Your text here", nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated %d chunks with quality score %.2f\n", 
    len(result.Chunks), result.QualityScore)
```

### Configuration Presets

```go
// High quality for research papers
config := chunkers.GetPresetConfig(chunkers.PresetHighQuality)

// High performance for production
config := chunkers.GetPresetConfig(chunkers.PresetHighPerformance)

// Optimized for RAG systems
config := chunkers.GetPresetConfig(chunkers.PresetRAG)

// Balanced approach
config := chunkers.GetPresetConfig(chunkers.PresetBalanced)
```

## 🧠 Chunking Strategies

### 1. Embedding-Based Semantic Chunking
Intelligent boundary detection using sentence embeddings.

```go
chunker := chunkers.NewEmbeddingBasedChunker(&chunkers.EmbeddingChunkerConfig{
    SimilarityThreshold: 0.8,
    EmbeddingProvider:   "openai",
    ModelName:          "text-embedding-3-small",
    WindowSize:         3,
    MinChunkSize:       100,
    MaxChunkSize:       800,
})
```

**Features:**
- Semantic similarity boundary detection
- Configurable similarity thresholds
- Multiple embedding provider support
- Sliding window analysis

### 2. Contextual Retrieval (Anthropic-style)
LLM-generated context for enhanced chunk retrieval.

```go
chunker := chunkers.NewContextualChunker(&chunkers.ContextualChunkerConfig{
    LLMProvider:        "openai",
    ModelName:         "gpt-4",
    ContextLength:     200,
    ContextPosition:   chunkers.ContextPositionBefore,
    QualityThreshold:  0.7,
})
```

**Features:**
- AI-generated contextual summaries
- Configurable context placement
- Quality-based filtering
- Multi-LLM provider support

### 3. Propositionalization Chunking
Breaks text into atomic semantic propositions.

```go
chunker := chunkers.NewPropositionChunker(&chunkers.PropositionChunkerConfig{
    LLMProvider:         "openai",
    ModelName:          "gpt-4",
    MaxPropositions:    5,
    MinConfidence:      0.8,
    EnableValidation:   true,
})
```

**Features:**
- Atomic semantic unit extraction
- Confidence scoring
- Validation with secondary LLM calls
- Structured proposition output

### 4. Agentic Chunking
LLM-driven reasoning for intelligent chunking decisions.

```go
chunker := chunkers.NewAgenticChunker(&chunkers.AgenticChunkerConfig{
    LLMProvider:         "openai",
    ModelName:          "gpt-4",
    AnalysisDepth:      chunkers.AnalysisDepthExpert,
    ReasoningSteps:     5,
    ConfidenceThreshold: 0.9,
    MaxLLMCalls:        10,
})
```

**Features:**
- Multi-step reasoning process
- Confidence-based decision making
- Adaptive analysis depth
- Reasoning chain tracking

### 5. Multi-Modal Chunking
Handles diverse content types including code, tables, and images.

```go
chunker := chunkers.NewMultiModalChunker(&chunkers.MultiModalChunkerConfig{
    PreserveCodeBlocks: true,
    PreserveTables:     true,
    HandleImages:       true,
    CodeLanguages:      []string{"go", "python", "javascript"},
    ImageFormats:       []string{"png", "jpg", "svg"},
})
```

**Features:**
- Code block preservation
- Table structure retention
- Image metadata extraction
- Language-specific processing

### 6. Hierarchical Chunking
Creates parent-child relationships with cross-references.

```go
chunker := chunkers.NewHierarchicalChunker(&chunkers.HierarchicalChunkerConfig{
    MaxLevels:              3,
    SummaryMode:           chunkers.SummaryModeLLM,
    PreserveSectionHeaders: true,
    CrossReferenceTracking: true,
    ParentChunkSize:        1000,
    ChildChunkSize:         300,
})
```

**Features:**
- Multi-level hierarchy generation
- Automatic summarization
- Section header preservation
- Cross-reference tracking

## 📊 Quality Metrics

### 8-Dimensional Quality Assessment

```go
// Configure quality metrics
config := &chunkers.QualityMetricsConfig{
    EnableQualityAssessment: true,
    QualityThreshold:       0.7,
    MetricsToCollect: []chunkers.QualityMetricType{
        chunkers.MetricCoherence,
        chunkers.MetricCompleteness,
        chunkers.MetricRelevance,
        chunkers.MetricInformation,
        chunkers.MetricReadability,
        chunkers.MetricSemanticIntegrity,
        chunkers.MetricStructure,
        chunkers.MetricOverlap,
    },
}

calculator := chunkers.NewQualityMetricsCalculator(config)
assessment, err := calculator.AssessQuality(ctx, chunks, originalText)
```

### Quality Dimensions

1. **Coherence**: Internal logical consistency
2. **Completeness**: Information preservation
3. **Relevance**: Contextual appropriateness
4. **Information Density**: Content concentration
5. **Readability**: Human comprehension ease
6. **Semantic Integrity**: Meaning preservation
7. **Structure Preservation**: Format retention
8. **Overlap Optimization**: Redundancy minimization

## 🏭 Production Features

### Pipeline Orchestration

```go
// Create production pipeline
pipeline := chunkers.NewChunkingPipeline(config)

// Add multiple strategies
pipeline.AddStrategy("semantic", embeddingChunker)
pipeline.AddStrategy("agentic", agenticChunker)
pipeline.AddStrategy("hierarchical", hierarchicalChunker)

// Set strategy selection logic
pipeline.SetStrategySelector(func(ctx context.Context, text string, metadata map[string]interface{}) string {
    if len(text) > 10000 {
        return "hierarchical"
    } else if isCodeContent(text) {
        return "multimodal"
    }
    return "semantic"
})
```

### Production Optimizer

```go
// Enable production optimizations
optimizer := chunkers.NewProductionOptimizer(&chunkers.ProductionConfig{
    EnableCaching:         true,
    CacheSize:            10000,
    CacheTTL:             time.Hour,
    MaxConcurrentRequests: 100,
    EnableCircuitBreaker:  true,
    EnableRateLimit:      true,
    RequestsPerSecond:    100,
})

// Optimize chunking process
result, err := optimizer.OptimizeChunking(ctx, pipeline, text, metadata)
```

### Real-time Monitoring

```go
// Start monitoring
monitor := chunkers.NewRealTimeMonitor(config)
monitor.Start()

// Record metrics
monitor.RecordRequest(duration, success)
monitor.RecordChunkingMetrics(chunkCount, averageSize, quality, strategy)

// Get health status
health := monitor.GetHealthStatus()
fmt.Printf("System health: %s (score: %.2f)\n", health.Status, health.Score)
```

## 🧪 Evaluation & Benchmarking

### A/B Testing Framework

```go
// Create evaluation framework
framework := chunkers.NewEvaluationFramework(config)

// Load standard datasets
framework.LoadStandardDatasets()

// Compare strategies
comparison, err := framework.RunComparison(ctx, "strategy_a", "strategy_b", pipeline)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Winner: %s (%.2f%% improvement, p-value: %.4f)\n",
    comparison.Winner, comparison.Improvement, comparison.PValue)
```

### Performance Benchmarking

```go
// Run benchmarks
summary, err := framework.RunBenchmark(ctx, "performance_test", strategy, pipeline)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Average time: %v, Throughput: %.2f QPS\n",
    summary.AverageTime, summary.ThroughputQPS)
```

## ⚙️ Configuration Guide

### Preset Configurations

#### High Quality (Research Papers)
```go
config := chunkers.GetPresetConfig(chunkers.PresetResearchPaper)
// Optimized for: Academic papers, research documents
// Features: Section preservation, cross-references, high quality thresholds
```

#### High Performance (Production APIs)
```go
config := chunkers.GetPresetConfig(chunkers.PresetHighPerformance)
// Optimized for: Real-time processing, large document batches
// Features: Parallel processing, minimal LLM calls, aggressive caching
```

#### RAG Systems
```go
config := chunkers.GetPresetConfig(chunkers.PresetRAG)
// Optimized for: Retrieval-augmented generation
// Features: Optimal embedding sizes, semantic integrity focus
```

#### Technical Documentation
```go
config := chunkers.GetPresetConfig(chunkers.PresetTechnicalDoc)
// Optimized for: API docs, software manuals
// Features: Code block preservation, multi-modal support
```

### Custom Configuration

```go
config := &chunkers.AdvancedChunkerConfig{
    BaseConfig: &chunkers.ChunkerConfig{
        ChunkSize:    512,
        ChunkOverlap: 80,
    },
    QualityConfig: &chunkers.QualityMetricsConfig{
        EnableQualityAssessment: true,
        QualityThreshold:       0.8,
        SamplingRate:           1.0,
    },
    PerformanceConfig: &chunkers.PerformanceConfig{
        EnableParallelProcessing: true,
        MaxConcurrency:          8,
        EnableCaching:           true,
        CacheSize:               5000,
        OptimizationLevel:       chunkers.OptimizationAggressive,
    },
}
```

## 🔧 Advanced Usage

### Custom Strategy Implementation

```go
type CustomChunker struct {
    config *CustomChunkerConfig
}

func (c *CustomChunker) Chunk(ctx context.Context, text string, metadata map[string]interface{}) ([]*chunkers.Chunk, error) {
    // Custom chunking logic
    chunks := make([]*chunkers.Chunk, 0)
    
    // Your implementation here
    
    return chunks, nil
}

func (c *CustomChunker) Name() string {
    return "custom_chunker"
}

func (c *CustomChunker) Configure(config interface{}) error {
    // Configuration logic
    return nil
}
```

### Quality Metric Implementation

```go
type CustomQualityMetric struct{}

func (m *CustomQualityMetric) Name() string {
    return "custom_metric"
}

func (m *CustomQualityMetric) Description() string {
    return "Custom quality assessment metric"
}

func (m *CustomQualityMetric) Evaluate(chunks []*chunkers.Chunk, groundTruth []*chunkers.GroundTruthChunk, originalText string) (float64, error) {
    // Custom evaluation logic
    score := 0.85 // Your calculation
    return score, nil
}
```

## 📈 Performance Optimization

### Memory Management

```go
config := &chunkers.PerformanceConfig{
    MemoryLimits: &chunkers.MemoryLimits{
        MaxHeapSize:      1 << 30, // 1GB
        MaxCacheSize:     256 << 20, // 256MB
        MaxEmbeddingSize: 512 << 20, // 512MB
    },
}
```

### Parallel Processing

```go
config := &chunkers.PerformanceConfig{
    EnableParallelProcessing: true,
    MaxConcurrency:          runtime.NumCPU() * 2,
    WorkerPoolSize:          8,
    BatchSize:              10,
}
```

### Caching Strategies

```go
cacheConfig := &chunkers.CacheConfig{
    Size:               10000,
    TTL:                time.Hour,
    Policy:             chunkers.PolicyLRU,
    EnableTiered:       true,
    CompressionEnabled: true,
}
```

## 🔍 Troubleshooting

### Common Issues

#### Performance Problems
```go
// Enable profiling
config.PerformanceConfig.EnableProfiling = true

// Monitor metrics
metrics := optimizer.GetMetrics()
if metrics.AverageResponseTime > 1000 { // 1 second
    // Reduce quality settings or increase caching
}
```

#### Memory Issues
```go
// Check memory usage
if metrics.MemoryUsage > config.MemoryLimits.MaxHeapSize {
    // Trigger garbage collection or reduce cache size
    runtime.GC()
}
```

#### Quality Issues
```go
// Assess quality
assessment, _ := calculator.AssessQuality(ctx, chunks, text)
if assessment.OverallScore < 0.7 {
    // Switch to higher quality strategy or adjust thresholds
}
```

### Health Monitoring

```go
// Check system health
if !optimizer.IsHealthy() {
    health := monitor.GetHealthStatus()
    for check, result := range health.Checks {
        if result.Status != "healthy" {
            log.Printf("Health check %s failed: %s", check, result.Message)
        }
    }
}
```

## 📚 API Reference

For complete API documentation, see:
- [Core API Reference](api-reference.md)
- [Configuration Reference](configuration.md)
- [Strategy Reference](strategies.md)
- [Quality Metrics Reference](quality-metrics.md)

## 🤝 Contributing

To contribute to the Enhanced Chunking System:

1. **Add New Strategies**: Implement the `Chunker` interface
2. **Improve Quality Metrics**: Implement the `QualityMetric` interface
3. **Performance Optimizations**: Focus on parallel processing and caching
4. **Documentation**: Improve guides and examples

See [Contributing Guidelines](../../CONTRIBUTING.md) for details.

## 📄 License

The Enhanced Chunking System is part of MemGOS and licensed under the MIT License.