# VectorDB Package

The `vectordb` package provides comprehensive vector database integration for MemGOS, with full support for Qdrant and a flexible factory pattern for adding additional vector database backends.

## Features

- **Complete Qdrant Integration**: Full-featured Qdrant client with collection management, vector operations, and advanced search capabilities
- **Factory Pattern**: Pluggable architecture for multiple vector database backends
- **Connection Management**: Connection pooling, health monitoring, and automatic reconnection
- **Batch Operations**: Efficient bulk operations with configurable batch sizes
- **Vector Analysis**: Built-in similarity calculations, vector statistics, and analysis tools
- **Performance Monitoring**: Comprehensive metrics and performance tracking
- **In-Memory Indexing**: Fast in-memory vector indexing for prototyping and testing
- **Configuration Management**: Flexible configuration with validation and environment variable support

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/memtensor/memgos/pkg/vectordb"
)

func main() {
    // Create configuration
    config := vectordb.DefaultVectorDBConfig()
    config.Qdrant.Host = "localhost"
    config.Qdrant.Port = 6333
    config.Qdrant.CollectionName = "my_vectors"
    config.Qdrant.VectorSize = 768

    // Create and connect to database
    db, err := vectordb.CreateVectorDB(config)
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer db.Disconnect(ctx)

    // Create collection
    collectionConfig := &vectordb.CollectionConfig{
        Name:       "my_vectors",
        VectorSize: 768,
        Distance:   "cosine",
    }
    
    if err := db.CreateCollection(ctx, collectionConfig); err != nil {
        log.Fatal(err)
    }

    // Add vectors
    items := vectordb.VectorDBItemList{
        {
            ID:      "doc1",
            Vector:  []float32{0.1, 0.2, 0.3, /* ... more dimensions */},
            Payload: map[string]interface{}{"text": "document content"},
        },
    }
    
    if err := db.Add(ctx, "my_vectors", items); err != nil {
        log.Fatal(err)
    }

    // Search for similar vectors
    query := []float32{0.1, 0.2, 0.3 /* ... */}
    results, err := db.Search(ctx, "my_vectors", query, 10, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Found %d similar vectors", len(results))
}
```

### Using the Registry

```go
// Register databases with different configurations
registry := vectordb.GetRegistry()

// Production database
prodConfig := vectordb.DefaultVectorDBConfig()
prodConfig.Qdrant.Host = "prod-qdrant.example.com"
registry.Register("production", prodConfig)

// Development database
devConfig := vectordb.DefaultVectorDBConfig()
devConfig.Qdrant.Host = "localhost"
registry.Register("development", devConfig)

// Get database by name
db, err := registry.Get(ctx, "production")
if err != nil {
    log.Fatal(err)
}
```

## Architecture

### Core Components

1. **BaseVectorDB Interface**: Common interface for all vector database implementations
2. **QdrantVectorDB**: Complete Qdrant implementation with gRPC client
3. **VectorDBFactory**: Factory pattern for creating database instances
4. **ConnectionManager**: Connection pooling and lifecycle management
5. **VectorDBRegistry**: Named database configurations and instances
6. **VectorOperations**: High-level operations across multiple databases

### Data Models

#### VectorDBItem
Represents a vector with metadata:

```go
type VectorDBItem struct {
    ID      string                 `json:"id"`      // UUID
    Vector  []float32              `json:"vector"`  // Embedding vector
    Payload map[string]interface{} `json:"payload"` // Metadata
    Score   *float32               `json:"score"`   // Similarity score (for search results)
}
```

#### Configuration
Hierarchical configuration with backend-specific settings:

```go
type VectorDBConfig struct {
    Backend types.BackendType `json:"backend"`
    Qdrant  *QdrantConfig     `json:"qdrant"`
    // ... other settings
}

type QdrantConfig struct {
    Host           string        `json:"host"`
    Port           int           `json:"port"`
    CollectionName string        `json:"collection_name"`
    VectorSize     int           `json:"vector_size"`
    Distance       string        `json:"distance"`    // "cosine", "euclidean", "dot"
    // ... performance and connection settings
}
```

## Operations

### Collection Management

```go
// Create collection
config := &vectordb.CollectionConfig{
    Name:       "embeddings",
    VectorSize: 768,
    Distance:   "cosine",
    OnDiskPayload: false,
}
err := db.CreateCollection(ctx, config)

// List collections
collections, err := db.ListCollections(ctx)

// Check if collection exists
exists, err := db.CollectionExists(ctx, "embeddings")

// Get collection info
info, err := db.GetCollectionInfo(ctx, "embeddings")

// Delete collection
err = db.DeleteCollection(ctx, "embeddings")
```

### Vector Operations

```go
// Add vectors
items := vectordb.VectorDBItemList{...}
err := db.Add(ctx, "collection", items)

// Upsert (add or update)
err := db.Upsert(ctx, "collection", items)

// Search for similar vectors
query := []float32{...}
results, err := db.Search(ctx, "collection", query, 10, filters)

// Get by ID
item, err := db.GetByID(ctx, "collection", "vector-id")

// Get by multiple IDs
items, err := db.GetByIDs(ctx, "collection", []string{"id1", "id2"})

// Delete vectors
err := db.Delete(ctx, "collection", []string{"id1", "id2"})
```

### Batch Operations

```go
// Batch add with custom batch size
err := db.BatchAdd(ctx, "collection", items, 100)

// Batch update
err := db.BatchUpdate(ctx, "collection", items, 50)

// Batch delete
err := db.BatchDelete(ctx, "collection", ids, 100)
```

### Advanced Search with Filters

```go
filters := map[string]interface{}{
    "category": "document",
    "author":   "john_doe",
}

results, err := db.Search(ctx, "collection", query, 10, filters)
```

## Vector Analysis Tools

### Similarity Calculations

```go
calc := vectordb.NewSimilarityCalculator()

// Cosine similarity
similarity, err := calc.CosineSimilarity(vectorA, vectorB)

// Euclidean distance
distance, err := calc.EuclideanDistance(vectorA, vectorB)

// Dot product
product, err := calc.DotProduct(vectorA, vectorB)

// Vector normalization
normalized := calc.NormalizeVector(vector)
```

### Vector Statistics

```go
analyzer := vectordb.NewVectorAnalyzer()

stats := analyzer.AnalyzeVector(vector)
// Returns: dimensions, mean, std, min, max, norm
```

### In-Memory Index

```go
index := vectordb.NewVectorIndex()

// Add vectors
index.Add(vector, metadata)

// Search
results, err := index.Search(query, 10, "cosine")
```

## Performance Monitoring

```go
monitor := vectordb.NewPerformanceMonitor()

// Record operations
monitor.RecordOperation("search", duration, success)

// Get metrics
metrics := monitor.GetMetrics("search")
// Returns: total ops, success/error counts, avg/min/max duration
```

## Configuration

### Environment Variables

The package supports configuration through environment variables:

```bash
QDRANT_HOST=localhost
QDRANT_PORT=6333
QDRANT_COLLECTION=vectors
QDRANT_VECTOR_SIZE=768
QDRANT_DISTANCE=cosine
QDRANT_API_KEY=your-api-key
QDRANT_USE_HTTPS=false
```

### Advanced Configuration

```go
config := &vectordb.QdrantConfig{
    Host:              "localhost",
    Port:              6333,
    CollectionName:    "vectors",
    VectorSize:        768,
    Distance:          "cosine",
    UseHTTPS:          false,
    ConnectionTimeout: 30 * time.Second,
    RequestTimeout:    60 * time.Second,
    MaxRetries:        3,
    RetryInterval:     time.Second,
    BatchSize:         100,
    SearchLimit:       1000,
    DefaultTopK:       10,
    ScoreThreshold:    0.0,
    OnDiskPayload:     false,
}
```

### Quantization Support

```go
config.QuantizationConfig = &vectordb.QuantizationConfig{
    Scalar: &vectordb.ScalarQuantization{
        Type:      "int8",
        Quantile:  0.99,
        AlwaysRam: false,
    },
}
```

## Error Handling

The package uses structured error types:

```go
import "github.com/memtensor/memgos/pkg/types"

// Check error type
if memErr, ok := err.(*types.MemGOSError); ok {
    switch memErr.Type {
    case types.ErrorTypeValidation:
        // Handle validation errors
    case types.ErrorTypeNotFound:
        // Handle not found errors
    case types.ErrorTypeExternal:
        // Handle external service errors
    }
}
```

## Testing

### Unit Tests

```bash
go test ./pkg/vectordb -v
```

### Benchmark Tests

```bash
go test ./pkg/vectordb -bench=. -benchmem
```

### Integration Tests

Integration tests require a running Qdrant instance:

```bash
# Start Qdrant with Docker
docker run -p 6333:6333 qdrant/qdrant

# Run integration tests
go test ./pkg/vectordb -tags=integration -v
```

## Extension Points

### Adding New Vector Database Backends

1. Implement the `BaseVectorDB` interface
2. Create a provider implementing `VectorDBProvider`
3. Register the provider with the factory

```go
type MyVectorDB struct {
    *vectordb.BaseVectorDBImpl
    // backend-specific fields
}

func (m *MyVectorDB) Search(ctx context.Context, collection string, vector []float32, limit int, filters map[string]interface{}) (vectordb.VectorDBItemList, error) {
    // Implementation
}

// Implement other required methods...

type MyProvider struct{}

func (p *MyProvider) Create(config *vectordb.VectorDBConfig) (vectordb.BaseVectorDB, error) {
    return NewMyVectorDB(config.MyBackend)
}

// Register the provider
factory := vectordb.GetFactory()
factory.RegisterProvider(&MyProvider{})
```

## Performance Considerations

### Batch Sizes
- Use larger batch sizes (100-1000) for bulk operations
- Smaller batches (10-50) for real-time operations
- Monitor memory usage with large batches

### Connection Pooling
- Use the ConnectionManager for multiple databases
- Set appropriate connection timeouts
- Enable health checks for production

### Vector Dimensions
- Higher dimensions require more memory and compute
- Consider quantization for memory-constrained environments
- Use appropriate distance metrics for your use case

### Indexing
- Use on-disk payload for large metadata
- Enable quantization for reduced memory usage
- Monitor index optimization status

## Examples

See `examples_test.go` for comprehensive usage examples including:
- Basic CRUD operations
- Batch processing
- Similarity search
- Performance monitoring
- Configuration management
- Error handling patterns

## Dependencies

- `github.com/qdrant/go-client`: Official Qdrant Go client
- `google.golang.org/grpc`: gRPC for Qdrant communication
- `github.com/cenkalti/backoff/v4`: Retry logic with exponential backoff
- `github.com/google/uuid`: UUID generation and validation

## License

This package is part of the MemGOS project and follows the same licensing terms.