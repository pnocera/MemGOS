# MemGOS Memory Readers

Advanced memory reading and analysis system for MemGOS, providing intelligent memory extraction, pattern detection, and comprehensive analytics.

## Overview

The Memory Reader System provides sophisticated capabilities for analyzing and extracting insights from memory content. It supports multiple reading strategies, from simple fast processing to advanced semantic analysis with machine learning.

## Features

### Core Capabilities
- **Multi-Strategy Reading**: Simple, Advanced, Semantic, and Hybrid approaches
- **Pattern Detection**: Identify patterns, sequences, and anomalies in memory content
- **Quality Assessment**: Multi-dimensional quality analysis with detailed metrics
- **Duplicate Detection**: Advanced similarity-based duplicate identification
- **Sentiment Analysis**: Emotional tone and subjectivity analysis
- **Smart Summarization**: Extractive, abstractive, and hybrid summarization
- **Semantic Clustering**: Group related memories using advanced algorithms

### Advanced Analytics
- **Cross-Memory Relationships**: Detect connections between memory items
- **Temporal Pattern Analysis**: Identify time-based patterns and trends
- **Theme Extraction**: Automatic topic and theme identification
- **Entity Recognition**: Extract and categorize named entities
- **Complexity Analysis**: Measure content complexity and readability

## Architecture

```
├── base.go          # Core interfaces and base implementations
├── simple.go        # Simple/fast reader implementation
├── memory.go        # Advanced reader with semantic capabilities
├── factory.go       # Reader factory and management
├── config.go        # Configuration management system
└── readers_test.go  # Comprehensive test suite
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/memtensor/memgos/pkg/readers"
)

func main() {
    // Create a simple reader
    reader := readers.NewSimpleMemReader(nil)
    defer reader.Close()
    
    // Create a read query
    query := &readers.ReadQuery{
        Query: "software development",
        TopK:  10,
    }
    
    // Read and analyze memories
    ctx := context.Background()
    result, err := reader.Read(ctx, query)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d memories\n", result.TotalFound)
    fmt.Printf("Processing time: %v\n", result.ProcessTime)
}
```

### Advanced Usage with Factory

```go
package main

import (
    "context"
    "github.com/memtensor/memgos/pkg/readers"
)

func main() {
    // Create factory with components
    factory := readers.CreateFactoryWithComponents(memCube, embedder, vectorDB)
    
    // Define requirements
    requirements := &readers.ReaderRequirements{
        NeedsSemanticSearch:     true,
        NeedsAdvancedAnalytics:  true,
        MaxMemoryCount:          1000,
        PerformancePriority:     5,
        AccuracyPriority:        8,
    }
    
    // Get recommended reader
    reader, err := factory.CreateAutoReader(requirements)
    if err != nil {
        panic(err)
    }
    defer reader.Close()
    
    // Use the reader...
}
```

## Reader Types

### Simple Reader
- **Best for**: High-performance, basic analysis
- **Features**: Fast processing, basic pattern detection
- **Use cases**: Real-time applications, resource-constrained environments

```go
reader := readers.NewSimpleMemReader(config)
```

### Advanced Reader
- **Best for**: Comprehensive analysis with ML capabilities
- **Features**: Semantic analysis, clustering, anomaly detection
- **Use cases**: Deep analysis, research, detailed insights

```go
reader := readers.NewAdvancedMemReader(config, memCube, embedder, vectorDB)
```

### Auto Reader (Recommended)
- **Best for**: Automatic selection based on requirements
- **Features**: Intelligent type selection, optimal performance
- **Use cases**: Most applications, adaptive processing

```go
reader, err := factory.CreateAutoReader(requirements)
```

## Configuration

### Predefined Profiles

```go
cm := readers.NewConfigManager("")

// High performance - fast processing
config := cm.CreateProfiledConfig(readers.ProfileHighPerformance)

// High accuracy - comprehensive analysis  
config := cm.CreateProfiledConfig(readers.ProfileHighAccuracy)

// Balanced - good performance and accuracy
config := cm.CreateProfiledConfig(readers.ProfileBalanced)

// Low resource - minimal processing
config := cm.CreateProfiledConfig(readers.ProfileLowResource)
```

### Custom Configuration

```go
config := &readers.ReaderConfig{
    Strategy:           readers.ReadStrategyAdvanced,
    AnalysisDepth:      "deep",
    PatternDetection:   true,
    QualityAssessment:  true,
    DuplicateDetection: true,
    SentimentAnalysis:  true,
    Summarization:      true,
    PatternConfig: &readers.PatternConfig{
        MinSupport:    0.05,
        MinConfidence: 0.7,
        MaxPatterns:   100,
        PatternTypes:  []string{"sequential", "frequent", "anomaly"},
    },
    SummarizationConfig: &readers.SummarizationConfig{
        MaxLength:        500,
        Strategy:         "hybrid",
        KeyPoints:        5,
        IncludeThemes:    true,
        IncludeEntities:  true,
        IncludeSentiment: true,
    },
}
```

## Analysis Results

### Memory Analysis
```go
analysis, err := reader.AnalyzeMemory(ctx, memory)
// Returns:
// - Keywords and themes
// - Sentiment analysis
// - Quality assessment
// - Complexity metrics
// - Generated insights
```

### Collection Analysis
```go
analysis, err := reader.AnalyzeMemories(ctx, memories)
// Returns:
// - Memory statistics
// - Relationship mapping
// - Semantic clusters
// - Timeline analysis
// - Smart recommendations
```

### Pattern Extraction
```go
patterns, err := reader.ExtractPatterns(ctx, memories, config)
// Returns:
// - Detected patterns
// - Frequency analysis
// - Pattern sequences
// - Anomaly detection
```

### Summarization
```go
summary, err := reader.SummarizeMemories(ctx, memories, config)
// Returns:
// - Generated summary
// - Key points
// - Extracted themes
// - Named entities
// - Quality metrics
```

## Performance Optimization

### Reader Selection
- **Simple**: 1000+ memories/sec, basic analysis
- **Advanced**: 100+ memories/sec, comprehensive analysis
- **Semantic**: 10+ memories/sec, ML-powered insights

### Best Practices

1. **Choose the Right Reader**
   ```go
   // For real-time processing
   reader := factory.CreateReader(readers.ReaderTypeSimple)
   
   // For comprehensive analysis
   reader := factory.CreateReader(readers.ReaderTypeAdvanced)
   ```

2. **Use Reader Pools for Concurrency**
   ```go
   pool := readers.NewReaderPool(factory, readers.ReaderTypeSimple, 10)
   reader, err := pool.GetReader()
   defer pool.ReturnReader(reader)
   ```

3. **Configure Analysis Depth**
   ```go
   config.AnalysisDepth = "basic"    // Fast
   config.AnalysisDepth = "medium"   // Balanced
   config.AnalysisDepth = "deep"     // Comprehensive
   ```

4. **Optimize for Your Use Case**
   ```go
   // For pattern-heavy analysis
   config.PatternDetection = true
   config.PatternConfig.MaxPatterns = 200
   
   // For quality-focused analysis
   config.QualityAssessment = true
   config.DuplicateDetection = true
   ```

## Integration Examples

### With Memory Cubes
```go
// Connect to memory cube
textMem := memCube.GetTextualMemory()

// Create reader with cube integration
reader := readers.NewAdvancedMemReader(config, memCube, embedder, vectorDB)

// Perform semantic search
query := &readers.ReadQuery{
    Query:    "machine learning algorithms",
    TopK:     20,
    Strategy: readers.ReadStrategySemantic,
}

result, err := reader.Read(ctx, query)
```

### With Schedulers
```go
// Schedule background analysis
task := &types.ScheduledTask{
    TaskType: "memory_analysis",
    Payload:  analysisRequest,
    Priority: types.TaskPriorityMedium,
}

scheduler.Schedule(ctx, task)
```

### Batch Processing
```go
// Process multiple memory sets
strategy := readers.NewMultiReaderStrategy(factory)
strategy.AddReader("fast", readers.ReaderTypeSimple, 0.3)
strategy.AddReader("accurate", readers.ReaderTypeAdvanced, 0.7)

// Use weighted results for optimal processing
```

## Error Handling

```go
result, err := reader.Read(ctx, query)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "timeout"):
        // Handle timeout
        fmt.Println("Analysis timed out, try simpler strategy")
    case strings.Contains(err.Error(), "memory limit"):
        // Handle memory constraints
        fmt.Println("Reduce batch size or use low-resource profile")
    default:
        // Handle other errors
        fmt.Printf("Analysis failed: %v\n", err)
    }
}
```

## Testing

Run the test suite:

```bash
# Run all tests
go test ./pkg/readers

# Run with verbose output
go test -v ./pkg/readers

# Run benchmarks
go test -bench=. ./pkg/readers

# Run specific tests
go test -run TestSimpleMemReader ./pkg/readers
```

## Configuration Files

### JSON Configuration
```json
{
  "strategy": "advanced",
  "analysis_depth": "deep",
  "pattern_detection": true,
  "quality_assessment": true,
  "duplicate_detection": true,
  "sentiment_analysis": true,
  "summarization": true,
  "pattern_config": {
    "min_support": 0.05,
    "min_confidence": 0.7,
    "max_patterns": 100,
    "pattern_types": ["sequential", "frequent", "anomaly"]
  },
  "summarization_config": {
    "max_length": 500,
    "min_length": 50,
    "strategy": "hybrid",
    "key_points": 5,
    "include_themes": true,
    "include_entities": true,
    "include_sentiment": true
  }
}
```

### Loading Configuration
```go
cm := readers.NewConfigManager("/path/to/configs")
err := cm.LoadConfig("production", "prod_config.json")
if err != nil {
    panic(err)
}

config, err := cm.GetConfig("production")
reader := readers.NewAdvancedMemReader(config, memCube, embedder, vectorDB)
```

## Monitoring and Metrics

### Performance Metrics
```go
// Monitor processing time
start := time.Now()
result, err := reader.Read(ctx, query)
processingTime := time.Since(start)

// Track memory usage
var m runtime.MemStats
runtime.ReadMemStats(&m)
fmt.Printf("Memory usage: %d KB\n", m.Alloc/1024)
```

### Quality Metrics
```go
// Assess result quality
if result.Quality != nil {
    fmt.Printf("Overall quality: %.2f\n", result.Quality.Score)
    fmt.Printf("Quality dimensions: %+v\n", result.Quality.Dimensions)
}

// Monitor pattern detection success
if result.Patterns != nil {
    fmt.Printf("Patterns found: %d\n", len(result.Patterns.Patterns))
    fmt.Printf("Pattern confidence: %.2f\n", result.Patterns.Confidence)
}
```

## Troubleshooting

### Common Issues

1. **Memory Not Found**
   ```go
   // Ensure memory cube is properly connected
   if memCube == nil {
       return fmt.Errorf("memory cube not initialized")
   }
   ```

2. **Semantic Search Failing**
   ```go
   // Check embedder and vector DB
   if embedder == nil || vectorDB == nil {
       // Fallback to textual search
       config.Strategy = readers.ReadStrategyAdvanced
   }
   ```

3. **Performance Issues**
   ```go
   // Use appropriate reader type
   capabilities, _ := factory.GetReaderCapabilities(readerType)
   if memoryCount > capabilities.RecommendedMemoryLimit {
       // Switch to high-performance reader
       reader = factory.CreateReader(readers.ReaderTypeSimple)
   }
   ```

### Debug Mode
```go
config.Options["debug_mode"] = true
config.Options["verbose_logging"] = true
reader := readers.NewAdvancedMemReader(config, memCube, embedder, vectorDB)
```

## Contributing

When adding new features:

1. Implement the interface methods in the appropriate reader
2. Add comprehensive tests
3. Update configuration options if needed
4. Document performance characteristics
5. Add examples for new functionality

## License

This package is part of the MemGOS project and follows the same licensing terms.