# MemGOS Architecture Documentation

## Overview

MemGOS (Memory Operating System in Go) is a comprehensive Go 1.24 implementation of the Python MemOS memory operating system for Large Language Models. It provides a modular, performant, and scalable architecture for managing different types of memory (textual, activation, and parametric) in AI systems.

## Architecture Principles

### 1. Interface-Driven Design
- All major components are defined as interfaces in `pkg/interfaces/`
- Enables easy testing, mocking, and component replacement
- Promotes loose coupling between components

### 2. Modular Package Structure
- Clear separation of concerns with dedicated packages
- Each package has a specific responsibility
- Dependencies flow in one direction (no circular dependencies)

### 3. Configuration-First Approach
- All components are configurable through structured configuration
- Support for JSON, YAML, and environment variables
- Runtime configuration validation

### 4. Error Handling
- Structured error types with context
- Comprehensive error codes for different scenarios
- Error wrapping and unwrapping support

### 5. Observability
- Built-in logging with structured fields
- Metrics collection for performance monitoring
- Health checks for system components

## Package Structure

```
memgos/
├── cmd/                    # Application entry points
│   └── memgos/            # Main CLI application
├── pkg/                   # Public packages
│   ├── types/             # Core types and data structures
│   ├── interfaces/        # Interface definitions
│   ├── config/            # Configuration management
│   ├── errors/            # Error handling
│   ├── core/              # MOS Core implementation
│   ├── memory/            # Memory implementations
│   ├── llm/               # LLM integrations
│   ├── embedders/         # Embedding providers
│   ├── vectordb/          # Vector database integrations
│   ├── graphdb/           # Graph database integrations
│   ├── parsers/           # Document parsers
│   ├── schedulers/        # Memory schedulers
│   ├── users/             # User management
│   ├── chat/              # Chat functionality
│   ├── logger/            # Logging implementations
│   └── metrics/           # Metrics collection
├── internal/              # Private packages
│   ├── models/            # Internal data models
│   ├── services/          # Internal services
│   └── utils/             # Internal utilities
├── api/                   # API definitions and handlers
├── tests/                 # Test files
├── examples/              # Usage examples
└── docs/                  # Documentation
```

## Core Components

### 1. Types (`pkg/types/`)
Defines all core data structures:
- **MessageDict**: Chat message structure
- **TextualMemoryItem**: Textual memory storage
- **ActivationMemoryItem**: KV cache and activation storage
- **ParametricMemoryItem**: Model parameter storage
- **MOSSearchResult**: Search result aggregation
- **User**: User management structure
- **MemCube**: Memory cube metadata

### 2. Interfaces (`pkg/interfaces/`)
Defines contracts for all major components:
- **Memory**: Base memory interface
- **TextualMemory**: Text-specific memory operations
- **ActivationMemory**: Activation memory operations
- **ParametricMemory**: Parametric memory operations
- **MemCube**: Memory cube container
- **LLM**: Language model interface
- **Embedder**: Embedding generation
- **VectorDB**: Vector database operations
- **GraphDB**: Graph database operations
- **MOSCore**: Main orchestration interface

### 3. Configuration (`pkg/config/`)
Hierarchical configuration system:
- **BaseConfig**: Common configuration functionality
- **MOSConfig**: Main system configuration
- **LLMConfig**: Language model configuration
- **MemoryConfig**: Memory backend configuration
- **MemCubeConfig**: Memory cube configuration
- **APIConfig**: API server configuration

### 4. Error Handling (`pkg/errors/`)
Structured error management:
- **MemGOSError**: Main error type with context
- **ErrorCode**: Specific error classifications
- **ErrorList**: Multiple error aggregation
- Comprehensive error constructors for different scenarios

### 5. Memory System (`pkg/memory/`)
Core memory implementations:
- **BaseMemory**: Common memory functionality
- **TextualMemory**: Text storage and search
- **ActivationMemory**: KV cache and activation storage
- **ParametricMemory**: Model parameter storage
- **GeneralMemCube**: Memory cube implementation
- **CubeRegistry**: Memory cube management

### 6. MOS Core (`pkg/core/`)
Main orchestration layer:
- **MOSCore**: Central coordinator
- Memory cube registration and management
- Cross-memory search and operations
- User management integration
- Chat functionality coordination

## Data Flow

### 1. Memory Storage Flow
```
Input → Parser → Chunker → Embedder → Memory Storage
                                   ↓
                              Vector Database
                                   ↓
                              Graph Database (optional)
```

### 2. Memory Retrieval Flow
```
Query → Embedder → Vector Search → Memory Retrieval → Ranking → Results
```

### 3. Chat Flow
```
User Input → Memory Search → Context Building → LLM Generation → Response
```

## Memory Types

### 1. Textual Memory
- **Purpose**: Store and retrieve text-based knowledge
- **Backends**: Naive, Tree, General
- **Features**: Semantic search, chunking, metadata
- **Storage**: JSON files, vector databases

### 2. Activation Memory
- **Purpose**: Cache model activations and KV pairs
- **Backends**: KV Cache, General
- **Features**: Fast retrieval, compression
- **Storage**: Binary files, in-memory caches

### 3. Parametric Memory
- **Purpose**: Store model adaptations (LoRA, adapters)
- **Backends**: LoRA, General
- **Features**: Parameter loading, adaptation
- **Storage**: Model files, parameter databases

## Integration Points

### 1. LLM Providers
- **OpenAI**: GPT models via API
- **Ollama**: Local model serving
- **HuggingFace**: Transformers integration

### 2. Vector Databases
- **Qdrant**: High-performance vector search
- **Future**: Support for other vector DBs

### 3. Graph Databases
- **Neo4j**: Relationship modeling
- **Future**: Support for other graph DBs

### 4. Document Parsers
- **PDF**: PDF document parsing
- **DOCX**: Word document parsing
- **Text**: Plain text processing
- **Future**: Additional format support

## Configuration Architecture

### Hierarchical Configuration
```yaml
# MOS Core Configuration
user_id: "user123"
session_id: "session456"
enable_textual_memory: true
enable_activation_memory: false
enable_parametric_memory: false

# LLM Configuration
chat_model:
  backend: "openai"
  model: "gpt-4"
  api_key: "${OPENAI_API_KEY}"
  max_tokens: 1024

# Memory Configuration
mem_reader:
  backend: "general"
  memory_filename: "textual_memory.json"
  top_k: 5
  
  # Embedder Configuration
  embedder:
    backend: "openai"
    model: "text-embedding-ada-002"
    
  # Vector DB Configuration
  vector_db:
    backend: "qdrant"
    host: "localhost"
    port: 6333
    collection: "memories"
```

### Environment Variables
- Automatic environment variable substitution
- Prefix-based variable organization
- Override capability for all configuration values

## Performance Characteristics

### 1. Memory Efficiency
- Lazy loading of memory components
- Streaming for large operations
- Memory pooling for frequent allocations

### 2. Concurrency
- Thread-safe memory operations
- Concurrent search across memory types
- Parallel processing for bulk operations

### 3. Scalability
- Horizontal scaling through multiple instances
- Vertical scaling through resource allocation
- Database sharding support

## Development Guidelines

### 1. Adding New Memory Backends
1. Implement the appropriate memory interface
2. Add configuration structure
3. Register with factory
4. Add tests and documentation

### 2. Adding New LLM Providers
1. Implement the LLM interface
2. Add provider-specific configuration
3. Handle provider-specific errors
4. Add integration tests

### 3. Adding New Features
1. Define interfaces first
2. Implement with proper error handling
3. Add comprehensive tests
4. Update documentation

## Testing Strategy

### 1. Unit Tests
- Test individual components in isolation
- Mock external dependencies
- Focus on business logic

### 2. Integration Tests
- Test component interactions
- Use test containers for databases
- Validate end-to-end flows

### 3. Performance Tests
- Benchmark critical paths
- Memory usage validation
- Concurrency testing

## Security Considerations

### 1. API Keys
- Secure storage and rotation
- Environment variable injection
- Audit logging

### 2. Data Protection
- Encryption at rest
- Secure transmission
- Access control

### 3. Input Validation
- Schema validation
- Sanitization
- Rate limiting

## Future Enhancements

### 1. Advanced Features
- Distributed memory systems
- Advanced caching strategies
- Machine learning optimizations

### 2. Additional Integrations
- More LLM providers
- Additional database backends
- Cloud storage integration

### 3. Operational Features
- Advanced monitoring
- Auto-scaling capabilities
- Disaster recovery

## Migration from Python MemOS

### 1. Configuration Compatibility
- YAML/JSON configuration migration
- Environment variable mapping
- Feature flag translation

### 2. Data Migration
- Memory format conversion
- Database schema migration
- Metadata preservation

### 3. API Compatibility
- REST API equivalence
- Client library migration
- Backwards compatibility layer