# MemGOS Embedders Package

The embedders package provides a comprehensive embedding system for MemGOS, supporting multiple embedding providers with a unified interface.

## Features

- **Multiple Providers**: Support for Sentence Transformers, OpenAI, and Ollama embeddings
- **Factory Pattern**: Easy creation and management of embedder instances
- **Batch Processing**: Efficient batch embedding for large datasets
- **Streaming Support**: Real-time embedding processing capabilities
- **Auto-Detection**: Intelligent model selection based on task requirements
- **Performance Monitoring**: Built-in metrics and performance tracking
- **Error Handling**: Comprehensive error handling with retry mechanisms
- **Configuration Management**: Flexible configuration system with validation

## Supported Providers

### 1. Sentence Transformers
- **ONNX Runtime**: Local inference with optimized performance
- **Popular Models**: Support for all-MiniLM-L6-v2, all-mpnet-base-v2, and more
- **Custom Pooling**: Mean, max, CLS token, and mean-max pooling strategies
- **Local Deployment**: No external API dependencies

### 2. OpenAI Embeddings
- **Latest Models**: text-embedding-3-large, text-embedding-3-small, ada-002
- **High Dimensions**: Up to 3072 dimensions for enhanced accuracy
- **Rate Limiting**: Built-in rate limiting and retry logic
- **Cost Optimization**: Batch processing for cost efficiency

### 3. Ollama Embeddings
- **Local Models**: Deploy embedding models locally via Ollama
- **Model Management**: Pull, delete, and manage models programmatically
- **Streaming**: Support for streaming embeddings
- **Health Monitoring**: Built-in health checks and server monitoring

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/memtensor/memgos/pkg/embedders"
)

func main() {
    // Create embedder configuration
    config := &embedders.EmbedderConfig{
        Provider:  "sentence-transformer",
        Model:     "all-MiniLM-L6-v2",
        Dimension: 384,
        MaxLength: 512,
        BatchSize: 32,
        Normalize: true,
    }
    
    // Create embedder from factory
    embedder, err := embedders.CreateEmbedder(config)
    if err != nil {
        log.Fatal(err)
    }
    defer embedder.Close()
    
    // Generate embedding
    ctx := context.Background()
    text := "This is a sample text for embedding"
    
    embedding, err := embedder.Embed(ctx, text)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated embedding with %d dimensions\n", len(embedding))
}
```

### Batch Processing

```go
// Batch embedding for multiple texts
texts := []string{
    "First document text",
    "Second document text", 
    "Third document text",
}

embeddings, err := embedder.EmbedBatch(ctx, texts)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated %d embeddings\n", len(embeddings))
```

### Auto-Detection

```go
// Auto-detect best model for task
factory := embedders.GetGlobalFactory()
model, err := factory.AutoDetectModel("semantic-similarity", "en", "")
if err != nil {
    log.Fatal(err)
}

config := &embedders.EmbedderConfig{
    Provider:  model.Provider,
    Model:     model.Name,
    Dimension: model.Dimension,
    MaxLength: model.MaxLength,
    BatchSize: 32,
    Normalize: true,
}
```

## Provider-Specific Examples

### OpenAI Embeddings

```go
config := &embedders.EmbedderConfig{
    Provider:  "openai",
    Model:     "text-embedding-ada-002",
    APIKey:    "your-api-key",
    Dimension: 1536,
    MaxLength: 8191,
    BatchSize: 100,
    Normalize: true,
}

embedder, err := embedders.CreateEmbedder(config)
if err != nil {
    log.Fatal(err)
}
defer embedder.Close()

// Cost estimation
if oaiEmbedder, ok := embedder.(*embedders.OpenAIEmbedder); ok {
    cost := oaiEmbedder.EstimateCost(texts)
    fmt.Printf("Estimated cost: $%.4f\n", cost["estimated_cost"])
}
```

### Ollama Embeddings

```go
config := &embedders.EmbedderConfig{
    Provider:  "ollama",
    Model:     "nomic-embed-text",
    BaseURL:   "http://localhost:11434",
    Dimension: 768,
    MaxLength: 2048,
    BatchSize: 10,
    Normalize: true,
}

embedder, err := embedders.CreateEmbedder(config)
if err != nil {
    log.Fatal(err)
}
defer embedder.Close()

// Ensure model exists
if ollamaEmbedder, ok := embedder.(*embedders.OllamaEmbedder); ok {
    err := ollamaEmbedder.EnsureModelExists(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Advanced Features

### Streaming Embeddings

```go
streaming := embedders.NewStreamingEmbedder("all-MiniLM-L6-v2", 384, 5)
defer streaming.Close()

streaming.StartStreaming()

// Add texts to stream
go func() {
    for _, text := range texts {
        streaming.AddToStream(text)
    }
    streaming.StopStreaming()
}()

// Process batches
for {
    select {
    case <-streaming.GetFlushSignal():
        batch := streaming.FlushBuffer()
        if len(batch) > 0 {
            // Process batch
            fmt.Printf("Processing batch of %d texts\n", len(batch))
        }
    case <-time.After(10 * time.Second):
        return // Timeout
    }
}
```

### Custom Pooling Strategies

```go
// For sentence transformers
if stEmbedder, ok := embedder.(*embedders.SentenceTransformerEmbedder); ok {
    err := stEmbedder.SetPoolingStrategy("mean_max")
    if err != nil {
        log.Fatal(err)
    }
}
```

### Health Monitoring

```go
// Health check for all providers
factory := embedders.GetGlobalFactory()
results := factory.HealthCheck(ctx)

for provider, err := range results {
    if err != nil {
        fmt.Printf("Provider %s is unhealthy: %v\n", provider, err)
    } else {
        fmt.Printf("Provider %s is healthy\n", provider)
    }
}

// Individual embedder health check
err = embedder.HealthCheck(ctx)
if err != nil {
    fmt.Printf("Embedder health check failed: %v\n", err)
}
```

### Performance Metrics

```go
// Get usage statistics
if baseEmbedder, ok := embedder.(interface{ GetMetrics() map[string]interface{} }); ok {
    metrics := baseEmbedder.GetMetrics()
    fmt.Printf("Total embed calls: %v\n", metrics["embed_calls"])
    fmt.Printf("Total duration: %v\n", metrics["embed_duration"])
}

// Provider-specific statistics
if oaiEmbedder, ok := embedder.(*embedders.OpenAIEmbedder); ok {
    stats := oaiEmbedder.GetUsageStatistics()
    fmt.Printf("Total tokens used: %v\n", stats["total_tokens_used"])
}
```

## Configuration

### Environment Variables

```bash
# Sentence Transformers
export SENTENCE_TRANSFORMERS_HOME=/path/to/models

# OpenAI
export OPENAI_API_KEY=your-api-key

# Ollama
export OLLAMA_BASE_URL=http://localhost:11434
```

### Configuration File Example

```yaml
embedder:
  provider: sentence-transformer
  model: all-MiniLM-L6-v2
  dimension: 384
  max_length: 512
  batch_size: 32
  normalize: true
  timeout: 30s
  
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    base_url: https://api.openai.com/v1
    
  ollama:
    base_url: ${OLLAMA_BASE_URL:-http://localhost:11434}
    
  sentence-transformer:
    model_path: ${SENTENCE_TRANSFORMERS_HOME}
```

## Error Handling

```go
// Custom error handling
_, err := embedder.Embed(ctx, text)
if err != nil {
    if embedErr, ok := err.(*embedders.EmbedderError); ok {
        switch embedErr.Type {
        case "validation":
            fmt.Printf("Validation error: %s\n", embedErr.Message)
        case "api":
            fmt.Printf("API error: %s\n", embedErr.Message)
        case "timeout":
            fmt.Printf("Timeout error: %s\n", embedErr.Message)
        }
    }
}
```

## Best Practices

### 1. Model Selection
- Use `all-MiniLM-L6-v2` for general-purpose embeddings (fast, good quality)
- Use `all-mpnet-base-v2` for higher quality embeddings (slower, better accuracy)
- Use OpenAI models for production applications requiring highest quality
- Use Ollama for local deployment and privacy requirements

### 2. Batch Processing
- Always use batch processing for multiple texts
- Adjust batch size based on memory constraints and model type
- Monitor performance metrics to optimize batch sizes

### 3. Text Preprocessing
- The embedders automatically preprocess text, but consider:
  - Removing HTML tags and special characters
  - Normalizing unicode characters
  - Splitting very long documents into chunks

### 4. Caching
- Implement caching for frequently used embeddings
- Consider using the built-in metrics to identify cache opportunities

### 5. Error Handling
- Implement retry logic for API failures
- Use health checks to monitor provider availability
- Have fallback providers for critical applications

## Performance Tuning

### Memory Usage
```go
// Optimize memory usage
config.BatchSize = 16  // Smaller batches for limited memory
config.MaxLength = 256 // Shorter max length if possible
```

### Speed Optimization
```go
// Optimize for speed
config.BatchSize = 64  // Larger batches for throughput
config.Normalize = false // Skip normalization if not needed
```

### Quality Optimization
```go
// Optimize for quality
config.Model = "all-mpnet-base-v2" // Higher quality model
config.Normalize = true  // Enable normalization
```

## Testing

```bash
# Run all tests
go test ./pkg/embedders

# Run benchmarks
go test -bench=. ./pkg/embedders

# Run examples
go test -run=Example ./pkg/embedders
```

## Dependencies

The embedders package requires:
- `github.com/sashabaranov/go-openai` for OpenAI integration
- `github.com/go-resty/resty/v2` for HTTP clients
- `github.com/avast/retry-go` for retry logic
- `github.com/stretchr/testify` for testing

For ONNX runtime support (sentence transformers), additional dependencies may be required:
- ONNX Runtime Go bindings
- Model files in ONNX format

## Contributing

When contributing to the embedders package:

1. Add tests for new providers or features
2. Update documentation and examples
3. Follow the existing error handling patterns
4. Include benchmarks for performance-critical code
5. Ensure thread safety for concurrent usage

## License

This package is part of MemGOS and follows the same license terms.