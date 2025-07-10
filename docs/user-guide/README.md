# MemGOS User Guide

## Table of Contents

1. [Getting Started](#getting-started)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Basic Usage](#basic-usage)
5. [Advanced Features](#advanced-features)
6. [Memory Management](#memory-management)
7. [Chat Functionality](#chat-functionality)
8. [API Usage](#api-usage)
9. [Troubleshooting](#troubleshooting)
10. [Best Practices](#best-practices)

## Getting Started

MemGOS (Memory Operating System in Go) is a powerful system for managing AI memory across different types: textual, activation, and parametric memory. This guide will help you get started with MemGOS and make the most of its capabilities.

### What is MemGOS?

MemGOS provides:
- **Unified Memory Management**: Store and retrieve different types of AI-related data
- **Semantic Search**: Find relevant information using natural language queries
- **Chat Integration**: AI-powered conversations with memory context
- **Multi-User Support**: Secure user management and access control
- **High Performance**: Optimized Go implementation for production use

### Key Concepts

- **Memory Cube**: A container that holds different types of memories
- **Textual Memory**: Text-based knowledge and information
- **Activation Memory**: Cached model activations and KV pairs
- **Parametric Memory**: Model parameters and adaptations (LoRA, etc.)
- **MOS Core**: The central system that orchestrates all operations

## Installation

### Prerequisites

- Go 1.24 or later
- (Optional) Docker for containerized deployment
- (Optional) External services:
  - Qdrant for vector search
  - Neo4j for graph operations
  - Redis for caching

### Install from Source

```bash
# Clone the repository
git clone https://github.com/memtensor/memgos.git
cd memgos

# Install dependencies
make deps

# Build the application
make build

# Install globally
make install
```

### Install via Go

```bash
go install github.com/memtensor/memgos/cmd/memgos@latest
```

### Docker Installation

```bash
# Pull the official image
docker pull memtensor/memgos:latest

# Run with configuration
docker run -v $(pwd)/config:/config memtensor/memgos:latest
```

### Verify Installation

```bash
memgos --version
```

## Configuration

### Basic Configuration

Create a configuration file `config.yaml`:

```yaml
# User and session configuration
user_id: "your-user-id"
session_id: "session-001"

# Enable memory types
enable_textual_memory: true
enable_activation_memory: false
enable_parametric_memory: false

# Chat model configuration
chat_model:
  backend: "openai"
  model: "gpt-3.5-turbo"
  api_key: "${OPENAI_API_KEY}"
  max_tokens: 1024
  temperature: 0.7

# Memory reader configuration
mem_reader:
  backend: "general"
  memory_filename: "textual_memory.json"
  top_k: 5
  
  # Embedder for semantic search
  embedder:
    backend: "openai"
    model: "text-embedding-ada-002"
    api_key: "${OPENAI_API_KEY}"

# API server configuration (optional)
api:
  enabled: false
  host: "localhost"
  port: 8000
  cors_enabled: true

# Logging and monitoring
log_level: "info"
metrics_enabled: true
health_check_enabled: true
```

### Environment Variables

Set required environment variables:

```bash
# Required for OpenAI integration
export OPENAI_API_KEY="your-openai-api-key"

# Optional for other services
export QDRANT_URL="http://localhost:6333"
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
```

### Configuration Validation

Validate your configuration:

```bash
memgos config validate --config config.yaml
```

## Basic Usage

### Command Line Interface

#### Initialize MemGOS

```bash
# Start with a configuration file
memgos --config config.yaml

# Start in interactive mode
memgos --interactive --config config.yaml
```

#### Register a Memory Cube

```bash
# Register from existing directory
memgos register /path/to/cube my-cube

# Create empty cube
memgos create-cube my-new-cube "My Knowledge Base"
```

#### Add Memories

```bash
# Add text content
memgos add "Go is a programming language developed by Google"

# Add from file
memgos add --file document.txt

# Add to specific cube
memgos add --cube my-cube "Specific knowledge for this cube"
```

#### Search Memories

```bash
# Basic search
memgos search "programming languages"

# Search with options
memgos search "programming" --top-k 10 --cube my-cube

# Semantic search
memgos search --semantic "concepts related to software development"
```

#### Chat with Memory Context

```bash
# Start chat session
memgos chat "What programming languages do I know?"

# Chat with specific cubes
memgos chat --cubes "tech-knowledge,personal-notes" "Tell me about my projects"
```

### Interactive Mode

Start interactive mode for exploratory usage:

```bash
memgos --interactive --config config.yaml
```

Available commands in interactive mode:

```
> help                          # Show available commands
> register /path/to/cube id     # Register memory cube
> add "content"                 # Add memory
> search "query"                # Search memories
> chat "message"                # Chat with AI
> list cubes                    # List all cubes
> list memories cube-id         # List memories in cube
> user info                     # Show user information
> config show                   # Show current configuration
> exit                          # Exit interactive mode
```

### API Server Mode

Start the API server for programmatic access:

```bash
# Start API server
memgos --api --config config.yaml

# Server will be available at http://localhost:8000
```

Test the API:

```bash
# Health check
curl http://localhost:8000/api/v1/health

# Search memories
curl -X GET "http://localhost:8000/api/v1/search?q=golang" \
  -H "X-User-ID: your-user-id"

# Add memory
curl -X POST "http://localhost:8000/api/v1/memories" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: your-user-id" \
  -d '{"memory_content": "New knowledge", "cube_id": "my-cube"}'
```

## Advanced Features

### Semantic Search with Embeddings

Enable semantic search for more intelligent memory retrieval:

```yaml
mem_reader:
  backend: "general"
  use_semantic_search: true
  embedder:
    backend: "openai"
    model: "text-embedding-ada-002"
    api_key: "${OPENAI_API_KEY}"
```

Usage:

```bash
# Semantic search finds conceptually related content
memgos search --semantic "machine learning concepts"
```

### Vector Database Integration

Configure Qdrant for high-performance vector search:

```yaml
vector_db:
  backend: "qdrant"
  host: "localhost"
  port: 6333
  collection: "memgos_vectors"
  vector_size: 1536
```

### Graph Database Integration

Use Neo4j for relationship-based memory organization:

```yaml
graph_db:
  backend: "neo4j"
  uri: "bolt://localhost:7687"
  username: "neo4j"
  password: "${NEO4J_PASSWORD}"
```

### Multi-User Setup

Configure multi-user support:

```yaml
users:
  enable_multi_user: true
  default_role: "user"
  admin_users: ["admin-user-id"]
  
access_control:
  enable: true
  default_permissions:
    - "read_own_cubes"
    - "write_own_cubes"
```

User management:

```bash
# Create user
memgos user create john_doe --role user --email john@example.com

# List users
memgos user list

# Grant cube access
memgos access grant john_doe cube-id read,write
```

### Memory Scheduling

Enable background memory optimization:

```yaml
scheduler:
  enable: true
  optimization_interval: "1h"
  cleanup_interval: "24h"
  backup_interval: "12h"
```

## Memory Management

### Memory Cube Organization

Organize your knowledge into logical cubes:

```bash
# Create domain-specific cubes
memgos create-cube programming "Programming Knowledge"
memgos create-cube research "Research Papers"
memgos create-cube personal "Personal Notes"

# Add content to appropriate cubes
memgos add --cube programming "Go channels enable goroutine communication"
memgos add --cube research "Neural networks require large datasets for training"
memgos add --cube personal "Remember to review code before merging"
```

### Memory Operations

#### Viewing Memories

```bash
# List all cubes
memgos list cubes

# List memories in a cube
memgos list memories programming

# Get specific memory
memgos get memory-id --cube programming
```

#### Updating Memories

```bash
# Update memory content
memgos update memory-id "Updated content" --cube programming

# Update metadata
memgos update memory-id --metadata '{"topic": "concurrency", "level": "advanced"}'
```

#### Deleting Memories

```bash
# Delete specific memory
memgos delete memory-id --cube programming

# Clear all memories in cube
memgos clear --cube programming --confirm
```

### Memory Persistence

#### Export and Import

```bash
# Export cube to directory
memgos export programming /backup/programming-cube/

# Import cube from directory
memgos import /backup/programming-cube/ programming-restored

# Export to JSON
memgos export programming --format json > programming.json

# Import from JSON
memgos import --format json programming.json programming-imported
```

#### Backup and Restore

```bash
# Create backup
memgos backup --all /backup/memgos-backup/

# Restore from backup
memgos restore /backup/memgos-backup/

# Scheduled backups (in config)
scheduler:
  backup_enabled: true
  backup_path: "/backup/memgos/"
  backup_interval: "12h"
```

## Chat Functionality

### Basic Chat

```bash
# Simple chat query
memgos chat "What do you know about Go programming?"

# Chat with specific context
memgos chat --cubes "programming,documentation" "How do I implement concurrency?"
```

### Advanced Chat Features

#### Context Control

```bash
# Limit context memories
memgos chat --max-context 3 "Explain channels"

# Use specific memory types
memgos chat --memory-types textual "What are the benefits of Go?"
```

#### Conversation History

```bash
# Start named conversation
memgos chat --conversation golang-learning "Start teaching me Go"

# Continue conversation
memgos chat --conversation golang-learning "Tell me more about interfaces"

# View conversation history
memgos history --conversation golang-learning
```

### Chat Configuration

```yaml
chat:
  model: "gpt-4"
  max_tokens: 2048
  temperature: 0.7
  context_window: 5
  save_conversations: true
  
  # System prompt customization
  system_prompt: |
    You are a helpful AI assistant with access to the user's knowledge base.
    Use the provided context to give accurate and relevant responses.
    Always cite specific memories when referencing information.
```

## API Usage

For detailed API documentation, see [API Documentation](../api/README.md).

### Quick Examples

#### Search API

```bash
curl -X GET "http://localhost:8000/api/v1/search?q=golang&top_k=5" \
  -H "X-User-ID: user-123"
```

#### Add Memory API

```bash
curl -X POST "http://localhost:8000/api/v1/memories" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "memory_content": "Go is fast and efficient",
    "cube_id": "programming",
    "metadata": {"topic": "performance"}
  }'
```

#### Chat API

```bash
curl -X POST "http://localhost:8000/api/v1/chat" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "message": "What makes Go efficient?",
    "context_cubes": ["programming"]
  }'
```

## Troubleshooting

### Common Issues

#### Configuration Problems

**Issue**: "Config file not found"
```bash
# Solution: Specify config path explicitly
memgos --config /full/path/to/config.yaml

# Or use environment variable
export MEMGOS_CONFIG=/path/to/config.yaml
```

**Issue**: "Invalid API key"
```bash
# Check environment variable
echo $OPENAI_API_KEY

# Verify in config
memgos config show | grep api_key
```

#### Memory Cube Issues

**Issue**: "Cube not found"
```bash
# List available cubes
memgos list cubes

# Check cube registration
memgos cube info cube-id
```

**Issue**: "Permission denied"
```bash
# Check user permissions
memgos user info

# Verify cube access
memgos access list --user your-user-id
```

#### Performance Issues

**Issue**: Slow search performance
```bash
# Enable metrics to diagnose
memgos --metrics --config config.yaml

# Check search times
memgos metrics search-performance
```

**Issue**: High memory usage
```bash
# Monitor memory usage
memgos metrics memory-usage

# Enable memory optimization
memgos optimize --all-cubes
```

### Debug Mode

Enable debug logging:

```yaml
log_level: "debug"
metrics_enabled: true
```

Or via command line:

```bash
memgos --debug --config config.yaml
```

### Log Analysis

View logs:

```bash
# Application logs
tail -f ~/.memgos/logs/memgos.log

# API access logs
tail -f ~/.memgos/logs/api.log

# Error logs
grep ERROR ~/.memgos/logs/memgos.log
```

## Best Practices

### Memory Organization

1. **Use Descriptive Cube Names**: Choose clear, domain-specific names
2. **Logical Separation**: Keep different types of knowledge in separate cubes
3. **Metadata Strategy**: Use consistent metadata schema across memories
4. **Regular Cleanup**: Remove outdated or duplicate memories

### Search Optimization

1. **Use Specific Queries**: More specific queries yield better results
2. **Leverage Metadata**: Use metadata filters for precise searches
3. **Semantic Search**: Enable for conceptual searches
4. **Result Ranking**: Review and improve search relevance

### Performance Optimization

1. **Memory Limits**: Set appropriate memory limits for large datasets
2. **Indexing**: Enable vector database indexing for large collections
3. **Caching**: Use Redis for frequently accessed data
4. **Batch Operations**: Use bulk operations for large data imports

### Security Best Practices

1. **API Key Management**: Use environment variables, never hardcode
2. **Access Control**: Implement proper user permissions
3. **Data Encryption**: Enable encryption for sensitive data
4. **Regular Backups**: Implement automated backup strategies

### Development Workflow

1. **Version Control**: Track configuration changes
2. **Environment Separation**: Use different configs for dev/prod
3. **Testing**: Test memory operations before production
4. **Monitoring**: Set up alerts for system health

### Integration Patterns

1. **API-First**: Use REST API for application integration
2. **Event-Driven**: Implement webhooks for real-time updates
3. **Batch Processing**: Use bulk operations for data migration
4. **Microservices**: Deploy as independent service

## Next Steps

1. **Explore Examples**: Check out [examples directory](../examples/) for practical use cases
2. **Architecture Deep Dive**: Read [Architecture Documentation](../architecture/README.md)
3. **API Integration**: Review [API Documentation](../api/README.md)
4. **Advanced Configuration**: Learn about all configuration options
5. **Community**: Join discussions and contribute to the project

## Getting Help

- **Documentation**: Comprehensive guides and API references
- **GitHub Issues**: Report bugs and request features
- **Discussions**: Community Q&A and best practices
- **Examples**: Practical implementation patterns

For more advanced topics, see our specialized guides:
- [Migration Guide](../MIGRATION_GUIDE.md)
- [Performance Tuning](performance-tuning.md)
- [Production Deployment](production-deployment.md)
- [Developer Guide](developer-guide.md)
