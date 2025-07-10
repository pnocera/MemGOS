# MemGOS Developer Guide

## Table of Contents

1. [Development Setup](#development-setup)
2. [Project Structure](#project-structure)
3. [Contributing Guidelines](#contributing-guidelines)
4. [Code Standards](#code-standards)
5. [Testing](#testing)
6. [Building and Packaging](#building-and-packaging)
7. [Adding New Features](#adding-new-features)
8. [Performance Optimization](#performance-optimization)
9. [Debugging and Profiling](#debugging-and-profiling)
10. [Release Process](#release-process)

## Development Setup

### Prerequisites

- **Go 1.24+**: Latest Go version for best performance
- **Git**: For version control
- **Make**: For build automation
- **Docker**: For containerized development (optional)
- **IDE**: VS Code, GoLand, or Vim with Go plugin

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/memtensor/memgos.git
cd memgos

# Install development dependencies
make deps

# Install development tools
make install-tools

# Setup pre-commit hooks
make setup-hooks

# Verify setup
make verify
```

### Development Environment

```bash
# Create development configuration
cp examples/config/development.yaml config/dev.yaml

# Set environment variables
export MEMGOS_CONFIG=config/dev.yaml
export OPENAI_API_KEY="your-dev-key"
export LOG_LEVEL="debug"

# Start development mode
make dev
```

### IDE Configuration

#### VS Code Setup

`.vscode/settings.json`:
```json
{
  "go.toolsManagement.checkForUpdates": "local",
  "go.useLanguageServer": true,
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.testFlags": ["-v", "-race"],
  "go.buildFlags": ["-race"],
  "go.vetFlags": ["-all"],
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.organizeImports": true
  }
}
```

`.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch MemGOS",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/memgos",
      "args": ["--config", "config/dev.yaml", "--debug"],
      "env": {
        "OPENAI_API_KEY": "your-key"
      }
    },
    {
      "name": "Launch API Server",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/memgos",
      "args": ["--api", "--config", "config/dev.yaml"]
    }
  ]
}
```

## Project Structure

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
│   ├── unit/              # Unit tests
│   ├── integration/       # Integration tests
│   └── benchmarks/        # Performance benchmarks
├── examples/              # Usage examples
├── docs/                  # Documentation
├── scripts/               # Build and utility scripts
├── .github/               # GitHub workflows
├── Makefile               # Build automation
├── go.mod                 # Go module definition
└── README.md              # Project README
```

### Package Organization

#### Core Packages (`pkg/`)

- **types/**: All data structures and type definitions
- **interfaces/**: Interface contracts for all components
- **config/**: Configuration parsing and validation
- **errors/**: Custom error types and handling
- **core/**: Main MOS orchestration logic

#### Feature Packages

- **memory/**: Memory backend implementations
- **llm/**: Language model integrations
- **embedders/**: Text embedding providers
- **vectordb/**: Vector database clients
- **graphdb/**: Graph database clients

#### Support Packages

- **logger/**: Structured logging
- **metrics/**: Performance monitoring
- **users/**: User management
- **chat/**: Chat functionality

## Contributing Guidelines

### Code of Conduct

We follow the [Go Community Code of Conduct](https://golang.org/conduct). Be respectful and inclusive.

### Contribution Workflow

1. **Fork the repository**
2. **Create feature branch**: `git checkout -b feature/amazing-feature`
3. **Make changes** following our code standards
4. **Add tests** for new functionality
5. **Run tests**: `make test`
6. **Update documentation** if needed
7. **Commit changes**: `git commit -m 'Add amazing feature'`
8. **Push to branch**: `git push origin feature/amazing-feature`
9. **Create Pull Request**

### Pull Request Guidelines

#### PR Title Format
```
type(scope): description

# Examples:
feat(memory): add parametric memory support
fix(api): resolve search timeout issue
docs(readme): update installation instructions
test(core): add integration tests for MOSCore
```

#### PR Description Template
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed
- [ ] Performance impact assessed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added for new functionality
- [ ] No breaking changes (or marked as such)
```

### Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

[optional body]

[optional footer(s)]
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Scopes**:
- `core`: MOSCore functionality
- `memory`: Memory operations
- `api`: REST API
- `config`: Configuration
- `llm`: LLM integrations
- `db`: Database operations
- `cli`: Command line interface

## Code Standards

### Go Style Guide

We follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and [Effective Go](https://golang.org/doc/effective_go.html).

### Code Quality Tools

```bash
# Format code
make fmt

# Lint code
make lint

# Vet code
make vet

# Run all checks
make check

# Fix common issues
make fix
```

### Naming Conventions

#### Package Names
```go
// Good: short, descriptive, lowercase
package memory
package config
package vectordb

// Bad: mixed case, underscores, too long
package memoryManagement
package vector_db
package configurationManagement
```

#### Interface Names
```go
// Good: descriptive with -er suffix
type Memory interface {}
type Embedder interface {}
type VectorDB interface {}

// Good: descriptive without -er when not applicable
type MOSCore interface {}
type MemCube interface {}
```

#### Variable Names
```go
// Good: clear, concise
var userID string
var searchResults []types.MemoryItem
var mosCore interfaces.MOSCore

// Bad: abbreviations, unclear
var uid string
var res []types.MemoryItem
var mc interfaces.MOSCore
```

### Error Handling

```go
// Good: wrap errors with context
func (m *Memory) Add(ctx context.Context, item types.MemoryItem) error {
    if err := m.validate(item); err != nil {
        return fmt.Errorf("failed to validate memory item: %w", err)
    }
    
    if err := m.store(ctx, item); err != nil {
        return fmt.Errorf("failed to store memory item %s: %w", item.ID, err)
    }
    
    return nil
}

// Use custom error types for domain errors
var ErrMemoryNotFound = errors.New("memory not found")

func (m *Memory) Get(ctx context.Context, id string) (types.MemoryItem, error) {
    item, exists := m.items[id]
    if !exists {
        return nil, fmt.Errorf("memory %s: %w", id, ErrMemoryNotFound)
    }
    return item, nil
}
```

### Logging

```go
// Use structured logging
logger.Info("memory added",
    "memory_id", item.ID,
    "cube_id", cubeID,
    "user_id", userID,
    "size", len(item.Memory),
)

// Log errors with context
logger.Error("failed to add memory",
    "error", err,
    "memory_id", item.ID,
    "cube_id", cubeID,
)
```

### Context Usage

```go
// Always accept context as first parameter
func (m *Memory) Search(ctx context.Context, query string) ([]types.MemoryItem, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Pass context to downstream calls
    return m.searcher.Search(ctx, query)
}

// Use context for timeouts
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

## Testing

### Test Structure

```go
func TestMemoryAdd(t *testing.T) {
    tests := []struct {
        name    string
        memory  types.MemoryItem
        wantErr bool
    }{
        {
            name: "valid memory",
            memory: types.MemoryItem{
                ID:     "test-1",
                Memory: "test content",
            },
            wantErr: false,
        },
        {
            name: "empty memory",
            memory: types.MemoryItem{
                ID:     "test-2",
                Memory: "",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            m := NewMemory()
            err := m.Add(context.Background(), tt.memory)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Categories

#### Unit Tests
```bash
# Run unit tests
make test-unit

# Run with coverage
make test-coverage

# Run with race detection
make test-race
```

#### Integration Tests
```bash
# Run integration tests (requires external services)
make test-integration

# Run with Docker containers
make test-integration-docker
```

#### Benchmarks
```bash
# Run benchmarks
make bench

# Run specific benchmark
go test -bench=BenchmarkSearch ./pkg/memory/

# Profile benchmarks
go test -bench=BenchmarkSearch -cpuprofile=cpu.prof ./pkg/memory/
```

### Test Utilities

```go
// Test helpers
package testutil

import (
    "context"
    "testing"
    "time"
)

// CreateTestMemory creates a memory item for testing
func CreateTestMemory(id, content string) types.MemoryItem {
    return types.MemoryItem{
        ID:        id,
        Memory:    content,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error) {
    t.Helper()
    if err == nil {
        t.Fatal("expected error, got nil")
    }
}
```

### Mocking

```go
// Use interfaces for easy mocking
type MockEmbedder struct {
    EmbedFunc func(ctx context.Context, text string) ([]float64, error)
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
    if m.EmbedFunc != nil {
        return m.EmbedFunc(ctx, text)
    }
    return []float64{0.1, 0.2, 0.3}, nil
}

// Use in tests
func TestWithMockEmbedder(t *testing.T) {
    embedder := &MockEmbedder{
        EmbedFunc: func(ctx context.Context, text string) ([]float64, error) {
            return []float64{1.0, 2.0, 3.0}, nil
        },
    }
    
    // Test code using embedder
}
```

## Building and Packaging

### Build Commands

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build with race detection
make build-race

# Build optimized for production
make build-prod
```

### Cross-Compilation

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 make build

# Build for Windows
GOOS=windows GOARCH=amd64 make build

# Build for macOS
GOOS=darwin GOARCH=amd64 make build
```

### Docker Build

```bash
# Build Docker image
make docker-build

# Build and push
make docker-push

# Build multi-arch image
make docker-buildx
```

### Release Build

```bash
# Create release build
make release

# Create release with version
VERSION=v1.2.3 make release

# Sign release
make release-sign
```

## Adding New Features

### Adding a New Memory Backend

1. **Define interface** (if not exists):
   ```go
   // pkg/interfaces/memory.go
   type TextualMemory interface {
       Add(ctx context.Context, memories []types.MemoryItem) error
       Search(ctx context.Context, query string, topK int) ([]types.MemoryItem, error)
       // ... other methods
   }
   ```

2. **Implement backend**:
   ```go
   // pkg/memory/newbackend.go
   package memory
   
   type NewBackend struct {
       config *config.MemoryConfig
       logger interfaces.Logger
   }
   
   func NewNewBackend(cfg *config.MemoryConfig, logger interfaces.Logger) *NewBackend {
       return &NewBackend{
           config: cfg,
           logger: logger,
       }
   }
   
   func (nb *NewBackend) Add(ctx context.Context, memories []types.MemoryItem) error {
       // Implementation
   }
   ```

3. **Add configuration**:
   ```go
   // pkg/config/memory.go
   type MemoryConfig struct {
       Backend string `yaml:"backend"`
       
       // Add new backend config
       NewBackend *NewBackendConfig `yaml:"new_backend,omitempty"`
   }
   
   type NewBackendConfig struct {
       Setting1 string `yaml:"setting1"`
       Setting2 int    `yaml:"setting2"`
   }
   ```

4. **Register in factory**:
   ```go
   // pkg/memory/factory.go
   func (f *Factory) createTextualMemory(cfg *config.MemoryConfig) (interfaces.TextualMemory, error) {
       switch cfg.Backend {
       case "new-backend":
           return NewNewBackend(cfg, f.logger), nil
       // ... other backends
       default:
           return nil, fmt.Errorf("unknown backend: %s", cfg.Backend)
       }
   }
   ```

5. **Add tests**:
   ```go
   // pkg/memory/newbackend_test.go
   func TestNewBackend(t *testing.T) {
       // Test implementation
   }
   ```

6. **Update documentation**:
   - Add to configuration examples
   - Update README
   - Add usage examples

### Adding a New LLM Provider

Follow similar pattern for LLM providers:

1. **Implement LLM interface**:
   ```go
   // pkg/llm/newprovider.go
   type NewProvider struct {
       config *config.LLMConfig
       client *http.Client
   }
   
   func (np *NewProvider) Generate(ctx context.Context, messages []types.MessageDict) (string, error) {
       // Implementation
   }
   ```

2. **Add configuration support**
3. **Register in factory**
4. **Add comprehensive tests**
5. **Update documentation**

### Adding API Endpoints

1. **Define handler**:
   ```go
   // api/handlers/new_endpoint.go
   func (h *Handler) HandleNewEndpoint(w http.ResponseWriter, r *http.Request) {
       // Implementation
   }
   ```

2. **Add to router**:
   ```go
   // api/router.go
   router.HandleFunc("/api/v1/new-endpoint", h.HandleNewEndpoint).Methods("POST")
   ```

3. **Add request/response types**:
   ```go
   // pkg/types/api.go
   type NewEndpointRequest struct {
       Field1 string `json:"field1"`
       Field2 int    `json:"field2"`
   }
   
   type NewEndpointResponse struct {
       Result string `json:"result"`
   }
   ```

4. **Add API tests**
5. **Update OpenAPI specification**
6. **Update API documentation**

## Performance Optimization

### Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Trace profiling
go test -trace=trace.out -bench=.
go tool trace trace.out
```

### Benchmarking

```go
func BenchmarkMemorySearch(b *testing.B) {
    memory := setupTestMemory(b)
    query := "test query"
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := memory.Search(context.Background(), query, 10)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Optimization Guidelines

1. **Minimize allocations**:
   ```go
   // Good: reuse slices
   var results []types.MemoryItem
   for _, item := range items {
       if matches(item, query) {
           results = append(results, item)
       }
   }
   
   // Bad: create new slice each time
   for _, item := range items {
       if matches(item, query) {
           results := []types.MemoryItem{item}
           // ...
       }
   }
   ```

2. **Use sync.Pool for frequent allocations**:
   ```go
   var bufferPool = sync.Pool{
       New: func() interface{} {
           return make([]byte, 0, 1024)
       },
   }
   
   func processData(data []byte) {
       buf := bufferPool.Get().([]byte)
       defer bufferPool.Put(buf[:0])
       
       // Use buf
   }
   ```

3. **Optimize hot paths**:
   ```go
   // Cache frequently used calculations
   type Cache struct {
       mu    sync.RWMutex
       items map[string]interface{}
   }
   
   func (c *Cache) Get(key string) (interface{}, bool) {
       c.mu.RLock()
       defer c.mu.RUnlock()
       
       value, exists := c.items[key]
       return value, exists
   }
   ```

## Debugging and Profiling

### Debug Mode

```bash
# Run with debug logging
memgos --debug --config config.yaml

# Enable pprof endpoint
memgos --profile --config config.yaml
# Access at http://localhost:6060/debug/pprof/
```

### Using Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug with delve
dlv debug cmd/memgos/main.go -- --config config.yaml

# Commands in delve:
# (dlv) b main.main     # Set breakpoint
# (dlv) c               # Continue
# (dlv) n               # Next line
# (dlv) s               # Step into
# (dlv) p variable      # Print variable
```

### Memory Debugging

```bash
# Check for memory leaks
GOMEMLIMIT=1GiB go test -memprofile=mem.prof -bench=.

# Analyze memory usage
go tool pprof mem.prof
(pprof) top
(pprof) list functionName
(pprof) web
```

### Race Condition Detection

```bash
# Build with race detection
go build -race cmd/memgos/main.go

# Run tests with race detection
go test -race ./...

# Continuous race detection in CI
make test-race
```

## Release Process

### Version Bumping

1. **Update version**:
   ```bash
   # Update version in:
   # - cmd/memgos/version.go
   # - README.md
   # - docs/
   ```

2. **Create changelog**:
   ```bash
   # Generate changelog
make changelog
   
   # Or manually update CHANGELOG.md
   ```

3. **Tag release**:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

### Release Checklist

- [ ] All tests pass
- [ ] Documentation updated
- [ ] Changelog updated
- [ ] Version bumped
- [ ] Performance benchmarks run
- [ ] Security scan completed
- [ ] Binary builds tested
- [ ] Docker images built
- [ ] Release notes prepared

### Automated Release

GitHub Actions automatically:
- Builds binaries for multiple platforms
- Creates Docker images
- Runs security scans
- Publishes release artifacts
- Updates package repositories

## Development Workflow

### Daily Development

```bash
# Start development session
git pull origin main
make deps
make dev

# Make changes...

# Before committing
make check
make test

# Commit and push
git add .
git commit -m "feat(memory): add new feature"
git push origin feature-branch
```

### Code Review Process

1. **Self-review**:
   - Run all checks
   - Test manually
   - Review diff
   - Update documentation

2. **Peer review**:
   - Focus on design and logic
   - Check for edge cases
   - Verify tests
   - Ensure documentation

3. **Maintainer review**:
   - Overall architecture
   - Performance impact
   - API consistency
   - Security considerations

### Best Practices

1. **Small, focused PRs**: Keep changes small and focused
2. **Descriptive commits**: Write clear commit messages
3. **Test thoroughly**: Add comprehensive tests
4. **Document changes**: Update relevant documentation
5. **Performance aware**: Consider performance impact
6. **Security minded**: Review for security issues

## Getting Help

### Development Support

- **Documentation**: [Developer Docs](https://memgos.dev/dev)
- **API Reference**: [API Docs](https://memgos.dev/api)
- **GitHub Discussions**: [Community Help](https://github.com/memtensor/memgos/discussions)
- **Discord**: [Developer Chat](https://discord.gg/memgos)

### Contributing Questions

- **Feature Requests**: [GitHub Issues](https://github.com/memtensor/memgos/issues)
- **Bug Reports**: [Bug Template](https://github.com/memtensor/memgos/issues/new?template=bug_report.md)
- **Architecture Discussions**: [GitHub Discussions](https://github.com/memtensor/memgos/discussions)

Welcome to the MemGOS development community! We appreciate your contributions.
