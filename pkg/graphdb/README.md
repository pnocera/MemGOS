# Graph Database Package

This package provides a unified interface for graph database operations in MemGOS, supporting both **KuzuDB** (embedded) and **Neo4j** (server-based) implementations.

## 🚀 Overview

The graph database package offers a factory pattern with dual provider support, allowing you to choose between high-performance embedded KuzuDB or distributed Neo4j based on your deployment needs.

## 📊 Provider Comparison

| Feature | KuzuDB | Neo4j |
|---------|--------|-------|
| **Architecture** | Embedded | Server-based |
| **Performance** | 10-100x faster | Standard |
| **Deployment** | Single binary | Requires server |
| **Concurrency** | Process-local | Multi-client |
| **Memory Usage** | Optimized | Higher overhead |
| **Network Latency** | Zero | TCP/HTTP |
| **Best For** | High-performance, single-node | Distributed, multi-client |

## 🏗️ Architecture

### Core Components

- **Factory Pattern**: `GraphDBFactory` manages provider registration and instance creation
- **Base Implementation**: `BaseGraphDB` provides common functionality (health, metrics, stats)
- **Provider Implementations**: 
  - `KuzuGraphDB`: Embedded KuzuDB with columnar storage
  - `Neo4jGraphDB`: Server-based Neo4j with connection pooling

### Interface Hierarchy

```go
// Basic interface for all graph databases
type GraphDB interface {
    CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error)
    CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error
    Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
    GetNode(ctx context.Context, id string) (map[string]interface{}, error)
    Close() error
}

// Extended interface with advanced features
type ExtendedGraphDB interface {
    GraphDB
    
    // Batch operations
    CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error)
    CreateRelations(ctx context.Context, relations []GraphRelation) error
    ExecuteBatch(ctx context.Context, operations []BatchOperation) error
    
    // Graph traversal
    Traverse(ctx context.Context, startID string, options *GraphTraversalOptions) ([]GraphNode, error)
    FindPaths(ctx context.Context, startID, endID string, maxDepth int) ([]GraphPath, error)
    RunAlgorithm(ctx context.Context, options *GraphAlgorithmOptions) (map[string]interface{}, error)
    
    // Schema management
    CreateIndex(ctx context.Context, label, property string) error
    CreateConstraint(ctx context.Context, label, property, constraintType string) error
    GetSchema(ctx context.Context) (map[string]interface{}, error)
    
    // Health and monitoring
    HealthCheck(ctx context.Context) error
    GetMetrics(ctx context.Context) (map[string]interface{}, error)
    
    // Transactions
    BeginTransaction(ctx context.Context) (GraphTransaction, error)
}
```

## 🔧 Configuration

### KuzuDB Configuration (Default)

```yaml
graph_db:
  provider: "kuzu"
  kuzu_config:
    database_path: "./data/kuzu_db"        # Database file location
    read_only: false                       # Read-only mode
    buffer_pool_size: 1073741824          # 1GB buffer pool
    max_num_threads: 4                     # CPU threads
    enable_compression: true               # Data compression
    timeout_seconds: 30                    # Query timeout
```

### Neo4j Configuration

```yaml
graph_db:
  provider: "neo4j"
  uri: "bolt://localhost:7687"
  username: "neo4j"
  password: "password"
  database: "neo4j"
  max_conn_pool: 50
  conn_timeout: 30s
  read_timeout: 15s
  write_timeout: 15s
  retry_attempts: 3
  retry_delay: 1s
  ssl_mode: "disable"
```

## 📝 Usage Examples

### Basic Usage

```go
import "github.com/memtensor/memgos/pkg/graphdb"

// Create KuzuDB instance
config := graphdb.DefaultKuzuDBGraphDBConfig()
config.KuzuConfig.DatabasePath = "./my_graph.db"

db, err := graphdb.NewKuzuGraphDB(config, logger, metrics)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Create a node
nodeID, err := db.CreateNode(ctx, []string{"Person"}, map[string]interface{}{
    "name": "Alice",
    "age":  30,
})

// Create relationship
err = db.CreateRelation(ctx, nodeID, otherNodeID, "KNOWS", map[string]interface{}{
    "since": "2023",
})

// Query the graph
results, err := db.Query(ctx, "MATCH (p:Person) WHERE p.age > $age RETURN p", map[string]interface{}{
    "age": 25,
})
```

### Factory Pattern

```go
// Using the factory
factory := graphdb.NewGraphDBFactory(logger, metrics)

// Create instance based on configuration
db, err := factory.Create(config)
if err != nil {
    log.Fatal(err)
}

// Get or create (reuses existing instances)
db, err = factory.GetOrCreate(config)
```

### Advanced Operations

```go
// Batch operations
nodes := []graphdb.GraphNode{
    {Labels: []string{"Person"}, Properties: map[string]interface{}{"name": "Bob"}},
    {Labels: []string{"Person"}, Properties: map[string]interface{}{"name": "Carol"}},
}
nodeIDs, err := db.CreateNodes(ctx, nodes)

// Graph traversal
options := &graphdb.GraphTraversalOptions{
    MaxDepth:  3,
    Direction: "BOTH",
    Limit:     100,
}
neighbors, err := db.Traverse(ctx, nodeID, options)

// Schema operations
err = db.CreateIndex(ctx, "Person", "name")
err = db.CreateConstraint(ctx, "Person", "email", "UNIQUE")
```

## 🚀 Performance Optimization

### KuzuDB Optimizations

1. **Buffer Pool Sizing**: Allocate 60-80% of available RAM
```go
config.KuzuConfig.BufferPoolSize = 8 * 1024 * 1024 * 1024 // 8GB
```

2. **Thread Configuration**: Match CPU cores
```go
config.KuzuConfig.MaxNumThreads = runtime.NumCPU()
```

3. **Compression**: Enable for storage efficiency
```go
config.KuzuConfig.EnableCompression = true
```

### Neo4j Optimizations

1. **Connection Pooling**: Size based on concurrent users
```go
config.MaxConnPool = 100  // For high-concurrency
```

2. **Timeout Tuning**: Balance responsiveness vs. complex queries
```go
config.ReadTimeout = 30 * time.Second
config.WriteTimeout = 60 * time.Second
```

## 🧪 Testing

### Unit Tests

```bash
# Run all graph database tests
go test ./pkg/graphdb/...

# Run with race detection
go test -race ./pkg/graphdb/...

# Run benchmarks
go test -bench=. ./pkg/graphdb/...
```

### Integration Tests

```bash
# Test with real KuzuDB
go test -tags=integration ./pkg/graphdb/...

# Test with Neo4j (requires running instance)
NEO4J_URI=bolt://localhost:7687 go test -tags=neo4j ./pkg/graphdb/...
```

## 🔍 Monitoring

### Health Checks

```go
// Check database health
err := db.HealthCheck(ctx)
if err != nil {
    log.Printf("Database unhealthy: %v", err)
}

// Get metrics
metrics, err := db.GetMetrics(ctx)
fmt.Printf("Query count: %v", metrics["total_queries"])
```

### Built-in Metrics

- **Performance**: Query latency, throughput, error rates
- **Usage**: Node/relationship counts, memory usage
- **Health**: Connection status, last check time
- **Storage**: Database size, compression ratio (KuzuDB)

## 🔧 Migration Guide

### From Neo4j to KuzuDB

1. **Update Configuration**:
```yaml
# Before (Neo4j)
graph_db:
  provider: "neo4j"
  uri: "bolt://localhost:7687"

# After (KuzuDB)
graph_db:
  provider: "kuzu"
  kuzu_config:
    database_path: "./data/kuzu_db"
```

2. **Query Compatibility**: Most Cypher queries work with minimal changes
3. **Data Migration**: Export from Neo4j, import to KuzuDB using batch operations
4. **Performance Testing**: Benchmark your specific workload

### Migration Considerations

- **Schema Differences**: KuzuDB uses table-based schema vs. Neo4j's property graph
- **Transaction Handling**: KuzuDB auto-commits vs. Neo4j explicit transactions
- **Index Management**: Different indexing strategies between providers
- **Deployment**: Embedded vs. server architecture implications

## 🚨 Troubleshooting

### Common Issues

1. **KuzuDB Connection Fails**:
   - Check database path permissions
   - Verify disk space availability
   - Ensure no file locks

2. **Neo4j Connection Timeout**:
   - Verify server is running
   - Check network connectivity
   - Validate credentials

3. **Query Performance**:
   - Add appropriate indexes
   - Optimize query patterns
   - Monitor buffer pool usage

### Debug Mode

```go
// Enable detailed logging
config.Logging = true

// Monitor health status
status, err, lastCheck := db.GetHealth()
fmt.Printf("Status: %s, Last Check: %v", status, lastCheck)
```

## 📚 API Reference

### Core Types

- **GraphNode**: Represents a graph node with labels and properties
- **GraphRelation**: Represents a relationship between nodes
- **GraphPath**: Represents a path through the graph
- **GraphTraversalOptions**: Configuration for graph traversal
- **BatchOperation**: Batch operation specification

### Factory Functions

- `NewGraphDBFactory()`: Create new factory instance
- `CreateGraphDB()`: Create using global factory
- `GetOrCreateGraphDB()`: Get or create using global factory

### Provider Functions

- `NewKuzuGraphDB()`: Create KuzuDB instance
- `NewNeo4jGraphDB()`: Create Neo4j instance
- `DefaultKuzuDBGraphDBConfig()`: Default KuzuDB configuration
- `DefaultGraphDBConfig()`: Default Neo4j configuration

## 🤝 Contributing

Contributions welcome! Areas for improvement:

- **KuzuDB API Integration**: Improve Go API usage as documentation becomes available
- **Query Optimization**: Add query planning and optimization
- **Additional Providers**: Support for other graph databases
- **Performance**: Benchmarking and optimization
- **Testing**: Expand test coverage and integration tests

## 📄 License

This package is part of MemGOS and licensed under the MIT License.