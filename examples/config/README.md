# MemGOS Configuration Examples

This directory contains example configurations for different MemGOS deployment scenarios.

## Configuration Files

### 1. `simple_config.yaml`
- **Purpose**: Basic MemGOS setup for development and testing
- **Features**: 
  - Textual memory only
  - Basic OpenAI integration
  - Simple logging to stdout
  - Scheduler disabled
- **Use Case**: Getting started, local development

### 2. `api_config.yaml`
- **Purpose**: Full-featured API server configuration
- **Features**:
  - All memory types enabled
  - NATS KV for scheduler (replaces Redis)
  - Advanced monitoring and metrics
  - API server with CORS and rate limiting
  - Vector database integration
- **Use Case**: Production API deployment

### 3. `nats_kv_config.yaml`
- **Purpose**: Demonstrates NATS.io Key-Value store integration
- **Features**:
  - Complete NATS KV setup
  - JetStream configuration
  - Clustered NATS support
  - Production configuration examples
  - Performance-optimized settings
- **Use Case**: NATS-based deployments, replacing Redis

## Key Configuration Changes

### Redis → NATS KV Migration

The scheduler configuration has been updated to use NATS.io Key-Value store instead of Redis:

**Old Redis Configuration:**
```yaml
mem_scheduler:
  enabled: true
  redis_host: "localhost"
  redis_port: 6379
  redis_password: "${REDIS_PASSWORD}"
  redis_db: 0
  worker_count: 4
  queue_size: 1000
```

**New NATS KV Configuration:**
```yaml
mem_scheduler:
  enabled: true
  
  # NATS Configuration
  nats_urls: ["nats://localhost:4222"]
  nats_password: "${NATS_PASSWORD}"
  nats_max_reconnect: 10
  
  # NATS KV Configuration
  use_nats_kv: true
  nats_kv_bucket_name: "memgos-scheduler"
  nats_kv_max_value_size: 1048576  # 1MB
  nats_kv_history: 10
  nats_kv_storage: "File"
  nats_kv_replicas: 1
  
  # Scheduler Configuration
  thread_pool_max_workers: 4
  enable_parallel_dispatch: false
  consume_interval_seconds: "5s"
```

### Benefits of NATS KV

1. **Simplified Architecture**: One service (NATS) handles both messaging and key-value storage
2. **Better Performance**: Native Go implementation, optimized for concurrent access
3. **Clustering Support**: Built-in clustering and replication
4. **Persistence**: Configurable storage backends (File/Memory)
5. **Versioning**: Automatic revision tracking for all key-value operations

## Environment Variables

The configurations use environment variables for sensitive data:

- `OPENAI_API_KEY`: OpenAI API key for LLM access
- `NATS_PASSWORD`: NATS server password (if authentication enabled)
- `QDRANT_API_KEY`: Qdrant vector database API key
- `NEO4J_PASSWORD`: Neo4j graph database password

## Configuration Validation

MemGOS validates configurations on startup using struct tags. Common validation rules:

- Required fields must be present
- URLs must be valid format
- Ports must be positive integers
- File paths must be accessible
- Backend types must be supported

## Docker Integration

These configurations work seamlessly with the Docker setup:

```bash
# Use with Docker Compose
docker-compose up -d

# Or mount custom config
docker run -v $(pwd)/examples/config/nats_kv_config.yaml:/app/config.yaml memgos:latest
```

## Performance Tuning

### NATS KV Performance Settings

For high-throughput scenarios:

```yaml
mem_scheduler:
  # Increase worker pool
  thread_pool_max_workers: 16
  enable_parallel_dispatch: true
  consume_interval_seconds: "1s"
  
  # Optimize KV settings
  nats_kv_max_value_size: 10485760    # 10MB
  nats_kv_max_bytes: 107374182400     # 100GB
  nats_kv_storage: "File"             # Persistent storage
  nats_kv_replicas: 3                 # High availability
  
  # JetStream tuning
  max_ack_pending: 100
  ack_wait: "60s"
```

### Memory Settings

For memory-intensive workloads:

```yaml
# Increase search results
top_k: 20

# Larger chunks for better context
mem_reader:
  chunk_size: 2000
  chunk_overlap: 400
```

## Monitoring and Observability

All configurations include monitoring endpoints:

- **Health Check**: `http://localhost:8080/health`
- **Metrics**: `http://localhost:9090/metrics`
- **API Status**: `http://localhost:8000/api/v1/status`

## Migration Guide

To migrate from Redis to NATS KV:

1. Update configuration file with NATS KV settings
2. Start NATS server with JetStream enabled
3. Update environment variables
4. Restart MemGOS service

No data migration is required as the scheduler operates on transient data.

## Support and Documentation

- [Main Documentation](../../docs/README.md)
- [Docker Setup](../../docs/docker/README.md)
- [API Reference](../../docs/api/README.md)
- [Troubleshooting](../../docs/user-guide/troubleshooting.md)