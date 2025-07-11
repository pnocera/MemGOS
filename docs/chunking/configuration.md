# Configuration Guide - Enhanced Chunking System

## Overview

The Enhanced Chunking System provides comprehensive configuration options through presets and custom configurations. This guide covers all configuration aspects from basic settings to advanced production parameters.

## Configuration Presets

### High Quality (Research Papers)
Optimized for maximum chunk quality with advanced AI analysis.

```go
config := chunkers.GetPresetConfig(chunkers.PresetHighQuality)
```

**Features:**
- Small chunk size (400 tokens) for precision
- High overlap (100 tokens) for context preservation
- Expert-level agentic analysis (5 reasoning steps)
- LLM-driven summaries for hierarchical chunking
- All 8 quality metrics enabled
- High confidence threshold (0.9)

**Best for:** Academic papers, research documents, legal texts

### High Performance (Production)
Optimized for speed and throughput with minimal overhead.

```go
config := chunkers.GetPresetConfig(chunkers.PresetHighPerformance)
```

**Features:**
- Large chunk size (800 tokens) for efficiency
- Low overlap (50 tokens) for speed
- Basic analysis depth (1 reasoning step)
- Extract-based summaries (no LLM calls)
- Minimal quality metrics (coherence, completeness)
- Aggressive parallel processing (8 workers)

**Best for:** Real-time APIs, large document batches, production systems

### RAG Systems
Optimized for Retrieval-Augmented Generation applications.

```go
config := chunkers.GetPresetConfig(chunkers.PresetRAG)
```

**Features:**
- Optimal embedding size (384 tokens)
- Moderate overlap (96 tokens)
- Semantic integrity focus
- Cross-reference tracking
- Information density optimization
- Balanced performance/quality

**Best for:** Question answering, chatbots, information retrieval

### Research Papers
Specialized for academic and research documents.

```go
config := chunkers.GetPresetConfig(chunkers.PresetResearchPaper)
```

**Features:**
- Small chunks (300 tokens) for precision
- Section header preservation
- Cross-reference tracking
- Table and image handling
- Citation preservation
- High quality thresholds

**Best for:** Academic research, scientific papers, literature review

### Technical Documentation
Optimized for technical documents with code and diagrams.

```go
config := chunkers.GetPresetConfig(chunkers.PresetTechnicalDoc)
```

**Features:**
- Code block preservation
- Multi-language code support
- Table structure retention
- Section header preservation
- Balanced chunk size (400-600 tokens)
- Multi-modal content handling

**Best for:** API documentation, software manuals, technical guides

### Chatbot Applications
Optimized for conversational AI applications.

```go
config := chunkers.GetPresetConfig(chunkers.PresetChatbot)
```

**Features:**
- Small chunks (200 tokens) for quick responses
- Low overlap (30 tokens) for speed
- Readability and relevance focus
- Fast processing optimizations
- Simple quality metrics

**Best for:** Conversational AI, customer support, interactive systems

## Custom Configuration

### Basic Configuration

```go
config := &chunkers.ChunkerConfig{
    ChunkSize:    512,     // Target chunk size in tokens
    ChunkOverlap: 80,      // Overlap between chunks
    MinChunkSize: 100,     // Minimum chunk size
    MaxChunkSize: 1000,    // Maximum chunk size
    Language:     "en",    // Document language
    Separators:   []string{"\n\n", "\n", ". ", " "}, // Boundary separators
}
```

### Advanced Configuration

```go
config := &chunkers.AdvancedChunkerConfig{
    BaseConfig: &chunkers.ChunkerConfig{
        ChunkSize:    512,
        ChunkOverlap: 80,
    },
    
    // Quality configuration
    QualityConfig: &chunkers.QualityMetricsConfig{
        EnableQualityAssessment:  true,
        QualityThreshold:         0.7,
        MetricsToCollect: []chunkers.QualityMetricType{
            chunkers.MetricCoherence,
            chunkers.MetricCompleteness,
            chunkers.MetricRelevance,
            chunkers.MetricInformation,
        },
        SamplingRate:             1.0,
        EnableRealTimeMonitoring: true,
        AlertThresholds: map[string]float64{
            "coherence":    0.6,
            "completeness": 0.5,
        },
        HistoryRetention: 24 * time.Hour,
    },
    
    // Performance configuration
    PerformanceConfig: &chunkers.PerformanceConfig{
        EnableParallelProcessing: true,
        MaxConcurrency:          4,
        EnableCaching:           true,
        CacheSize:               1000,
        CacheTTL:                1 * time.Hour,
        EnableProfiling:         false,
        MemoryLimits: &chunkers.MemoryLimits{
            MaxHeapSize:      1024 * 1024 * 1024, // 1GB
            MaxCacheSize:     256 * 1024 * 1024,  // 256MB
            MaxEmbeddingSize: 512 * 1024 * 1024,  // 512MB
        },
        OptimizationLevel: chunkers.OptimizationIntermediate,
    },
    
    // Integration configuration
    IntegrationConfig: &chunkers.IntegrationConfig{
        LLMProviders: map[string]*chunkers.LLMProviderConfig{
            "openai": {
                Provider:  "openai",
                ModelName: "gpt-4",
                Timeout:   30 * time.Second,
                RetryPolicy: &chunkers.RetryPolicy{
                    MaxRetries:    3,
                    InitialDelay:  1 * time.Second,
                    MaxDelay:      10 * time.Second,
                    BackoffFactor: 2.0,
                },
            },
        },
        EmbeddingProviders: map[string]*chunkers.EmbeddingProviderConfig{
            "openai": {
                Provider:   "openai",
                ModelName:  "text-embedding-3-small",
                Dimensions: 1536,
                BatchSize:  100,
                Timeout:    30 * time.Second,
            },
        },
    },
}
```

## Strategy-Specific Configuration

### Embedding-Based Chunking

```go
config := &chunkers.EmbeddingChunkerConfig{
    SimilarityThreshold: 0.8,           // Semantic similarity threshold
    EmbeddingProvider:   "openai",      // Embedding service
    ModelName:          "text-embedding-3-small",
    WindowSize:         3,              // Sentence window for comparison
    MinChunkSize:       100,            // Minimum chunk size
    MaxChunkSize:       800,            // Maximum chunk size
    EnableBatching:     true,           // Batch embeddings for efficiency
    BatchSize:          50,             // Embedding batch size
}
```

### Contextual Chunking

```go
config := &chunkers.ContextualChunkerConfig{
    LLMProvider:        "openai",       // LLM service
    ModelName:         "gpt-4",         // Model name
    ContextLength:     200,             // Context summary length
    ContextPosition:   chunkers.ContextPositionBefore, // Context placement
    QualityThreshold:  0.7,             // Minimum quality score
    MaxContextCalls:   5,               // Limit LLM calls
    ContextTemplate:   "Summarize this text: {text}", // Custom template
}
```

### Agentic Chunking

```go
config := &chunkers.AgenticChunkerConfig{
    LLMProvider:         "openai",
    ModelName:          "gpt-4",
    AnalysisDepth:      chunkers.AnalysisDepthExpert,  // Analysis complexity
    ReasoningSteps:     5,                             // Number of reasoning steps
    ConfidenceThreshold: 0.9,                          // Decision confidence
    MaxLLMCalls:        10,                            // Limit API calls
    EnableChainOfThought: true,                        // Enable reasoning chains
    Temperature:        0.1,                           // LLM temperature
}
```

### Multi-Modal Chunking

```go
config := &chunkers.MultiModalChunkerConfig{
    PreserveCodeBlocks: true,           // Keep code blocks intact
    PreserveTables:     true,           // Preserve table structure
    HandleImages:       true,           // Process image metadata
    CodeLanguages: []string{            // Supported code languages
        "go", "python", "javascript", "typescript", "java", "cpp",
    },
    ImageFormats: []string{             // Supported image formats
        "png", "jpg", "jpeg", "gif", "svg", "webp",
    },
    TableDetection:     true,           // Auto-detect tables
    CodeBlockMinLines:  3,              // Minimum code block size
    PreserveFormatting: true,           // Keep original formatting
}
```

### Hierarchical Chunking

```go
config := &chunkers.HierarchicalChunkerConfig{
    MaxLevels:              3,          // Maximum hierarchy levels
    SummaryMode:           chunkers.SummaryModeLLM,     // Summary generation
    PreserveSectionHeaders: true,       // Keep section headers
    CrossReferenceTracking: true,       // Track cross-references
    ParentChunkSize:        1000,       // Parent chunk size
    ChildChunkSize:         300,        // Child chunk size
    SummaryLength:          150,        // Summary length
    EnableNavigationLinks: true,        // Add navigation
}
```

## Production Configuration

### Caching Configuration

```go
cacheConfig := &chunkers.CacheConfig{
    Size:               10000,          // Maximum cache entries
    TTL:                time.Hour,      // Time to live
    Policy:             chunkers.PolicyLRU,    // Eviction policy
    EnableTiered:       true,           // Enable tiered caching
    CompressionEnabled: true,           // Compress cache entries
}
```

### Monitoring Configuration

```go
monitoringConfig := &chunkers.AdvancedMonitoringConfig{
    EnableMetrics:    true,             // Enable metrics collection
    EnableTracing:    false,            // Enable request tracing
    EnableLogging:    true,             // Enable logging
    LogLevel:         "INFO",           // Log level
    SamplingRate:     0.1,              // Sampling rate for traces
    CustomMetrics:    []string{         // Custom metrics to collect
        "chunk_quality", "processing_time", "cache_hit_rate",
    },
}
```

### Resource Management

```go
resourceConfig := &chunkers.ResourceConfig{
    MemoryLimit:  1024 * 1024 * 1024,  // 1GB memory limit
    RequestLimit: 100,                  // Concurrent request limit
    GCInterval:   10 * time.Second,     // GC check interval
    GCThreshold:  0.8,                  // GC trigger threshold
    EnableAutoGC: true,                 // Enable automatic GC
}
```

### Circuit Breaker Configuration

```go
circuitConfig := &chunkers.CircuitBreakerConfig{
    FailureThreshold: 5,                // Failures before opening
    RecoveryTimeout:  30 * time.Second, // Recovery wait time
    SuccessThreshold: 3,                // Successes to close
}
```

## Configuration Validation

```go
// Validate configuration
validator := chunkers.NewConfigValidator()
if err := validator.ValidateConfig(config); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

### Common Validation Rules

- Chunk size must be positive
- Chunk overlap must be less than chunk size
- Quality thresholds must be between 0 and 1
- Memory limits must be positive
- Concurrency settings must be reasonable

## Environment Variables

```bash
# LLM Configuration
export OPENAI_API_KEY="your-api-key"
export ANTHROPIC_API_KEY="your-api-key"

# Embedding Configuration
export EMBEDDING_PROVIDER="openai"
export EMBEDDING_MODEL="text-embedding-3-small"

# Performance Settings
export CHUNKER_MAX_CONCURRENCY="8"
export CHUNKER_CACHE_SIZE="10000"
export CHUNKER_ENABLE_PROFILING="false"

# Quality Settings
export CHUNKER_QUALITY_THRESHOLD="0.7"
export CHUNKER_ENABLE_MONITORING="true"
```

## Configuration Examples

### Development Configuration

```go
config := &chunkers.AdvancedChunkerConfig{
    BaseConfig: &chunkers.ChunkerConfig{
        ChunkSize:    256,
        ChunkOverlap: 50,
    },
    QualityConfig: &chunkers.QualityMetricsConfig{
        EnableQualityAssessment: false,  // Disable for speed
    },
    PerformanceConfig: &chunkers.PerformanceConfig{
        EnableParallelProcessing: false, // Simpler debugging
        MaxConcurrency:          1,
        EnableCaching:           false,
        EnableProfiling:         true,   // Enable for development
    },
}
```

### Production Configuration

```go
config := &chunkers.AdvancedChunkerConfig{
    BaseConfig: &chunkers.ChunkerConfig{
        ChunkSize:    512,
        ChunkOverlap: 80,
    },
    QualityConfig: &chunkers.QualityMetricsConfig{
        EnableQualityAssessment:  true,
        QualityThreshold:         0.8,
        EnableRealTimeMonitoring: true,
        SamplingRate:             0.1,   // Sample for performance
    },
    PerformanceConfig: &chunkers.PerformanceConfig{
        EnableParallelProcessing: true,
        MaxConcurrency:          runtime.NumCPU() * 2,
        EnableCaching:           true,
        CacheSize:               50000,
        OptimizationLevel:       chunkers.OptimizationAggressive,
        MemoryLimits: &chunkers.MemoryLimits{
            MaxHeapSize:  4 * 1024 * 1024 * 1024, // 4GB
            MaxCacheSize: 1 * 1024 * 1024 * 1024, // 1GB
        },
    },
}
```

## Best Practices

### Performance Optimization

1. **Use appropriate presets** for your use case
2. **Enable caching** for repeated operations
3. **Tune concurrency** based on your hardware
4. **Monitor memory usage** and set appropriate limits
5. **Use sampling** for quality assessment in production

### Quality Optimization

1. **Enable all quality metrics** for critical applications
2. **Set appropriate thresholds** based on your requirements
3. **Use real-time monitoring** to detect quality degradation
4. **Implement A/B testing** to compare strategies
5. **Monitor and tune** based on actual performance

### Configuration Management

1. **Use version control** for configuration files
2. **Implement configuration validation** in your pipeline
3. **Use environment variables** for secrets and deployment-specific settings
4. **Document configuration changes** and their impact
5. **Test configuration changes** in staging before production

## Troubleshooting

### Common Issues

1. **High memory usage**: Reduce cache size or enable compression
2. **Slow performance**: Increase concurrency or reduce quality metrics
3. **Poor quality**: Lower thresholds or enable more metrics
4. **API rate limits**: Implement retry policies and reduce concurrency

### Configuration Debugging

```go
// Enable debug logging
config.IntegrationConfig.MonitoringConfig.LogLevel = "DEBUG"

// Enable profiling
config.PerformanceConfig.EnableProfiling = true

// Check configuration
fmt.Printf("Configuration: %+v\n", config)
```