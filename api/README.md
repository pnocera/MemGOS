# MemGOS REST API

A production-ready REST API server for MemGOS (Memory-Augmented Generation Operating System) implemented in Go.

## Features

### Core Functionality
- **Memory Management**: Add, retrieve, update, and delete memories
- **Search Operations**: Semantic search across memory cubes
- **Chat Interface**: Conversational AI with memory-augmented responses
- **User Management**: Multi-user support with role-based access
- **Memory Cube Operations**: Register, unregister, and share memory cubes

### Security & Authentication
- **JWT Authentication**: Secure token-based authentication
- **Session Management**: Automatic session lifecycle management
- **Rate Limiting**: Protection against abuse (configurable)
- **CORS Support**: Cross-origin resource sharing
- **Request ID Tracking**: Unique request identification for debugging

### Monitoring & Observability
- **Health Checks**: Comprehensive health monitoring
- **Metrics Collection**: Performance and usage metrics
- **Structured Logging**: Detailed request/response logging
- **OpenAPI Documentation**: Complete API specification

### Production Features
- **Graceful Shutdown**: Safe server termination
- **Error Handling**: Comprehensive error responses
- **Middleware Stack**: Modular request processing
- **Configuration Management**: Environment-based configuration

## Quick Start

### Starting the API Server

```bash
# Start with default configuration
./memgos --api

# Start with custom configuration
./memgos --api --config config.yaml --log-level debug

# Start on specific port
PORT=9090 ./memgos --api
```

### Environment Variables

```bash
# Server Configuration
PORT=8080                    # API server port
GIN_MODE=release            # Gin mode (debug/release)
API_SECRET=your-jwt-secret  # JWT signing secret

# MemOS Configuration
MOS_USER_ID=default_user    # Default user ID
MOS_SESSION_ID=default_session # Default session ID
MOS_TOP_K=5                 # Default search results

# LLM Configuration
OPENAI_API_KEY=your-key     # OpenAI API key
OPENAI_API_BASE=https://api.openai.com/v1
MOS_CHAT_MODEL=gpt-3.5-turbo
MOS_CHAT_TEMPERATURE=0.7
```

## API Endpoints

### Authentication

```http
POST /auth/login     # Login and get JWT token
POST /auth/logout    # Logout and invalidate session
POST /auth/refresh   # Refresh JWT token
```

### Health & Monitoring

```http
GET  /health         # Health check
GET  /metrics        # Performance metrics
GET  /openapi.json   # OpenAPI specification
GET  /docs           # API documentation
```

### Configuration

```http
POST /configure      # Update MemOS configuration
```

### User Management

```http
GET  /users          # List all users
POST /users          # Create new user
GET  /users/me       # Get current user info
```

### Memory Cubes

```http
POST   /mem_cubes                    # Register memory cube
DELETE /mem_cubes/{cube_id}          # Unregister memory cube
POST   /mem_cubes/{cube_id}/share    # Share cube with user
```

### Memory Operations

```http
GET    /memories                              # Get all memories
POST   /memories                              # Add new memories
GET    /memories/{cube_id}/{memory_id}        # Get specific memory
PUT    /memories/{cube_id}/{memory_id}        # Update memory
DELETE /memories/{cube_id}/{memory_id}        # Delete memory
DELETE /memories/{cube_id}                    # Delete all memories in cube
```

### Search & Chat

```http
POST /search         # Search memories
POST /chat           # Chat with MemOS
```

## Request/Response Examples

### Authentication

**Login Request:**
```json
POST /auth/login
{
  "user_id": "john_doe",
  "password": "optional"
}
```

**Login Response:**
```json
{
  "code": 200,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "session_id": "abc123def456",
    "user_id": "john_doe",
    "expires_at": "2024-01-01T12:00:00Z"
  }
}
```

### Adding Memories

**Add Memory Request:**
```json
POST /memories
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
{
  "memory_content": "Important project meeting notes from Q4 planning",
  "mem_cube_id": "work_notes",
  "user_id": "john_doe"
}
```

**Add Memory Response:**
```json
{
  "code": 200,
  "message": "Memories added successfully",
  "data": null
}
```

### Search Memories

**Search Request:**
```json
POST /search
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
{
  "query": "Q4 planning meetings",
  "top_k": 5,
  "install_cube_ids": ["work_notes", "personal_notes"]
}
```

**Search Response:**
```json
{
  "code": 200,
  "message": "Search completed successfully",
  "data": {
    "text_memories": [
      {
        "id": "mem_001",
        "content": "Important project meeting notes from Q4 planning",
        "cube_id": "work_notes",
        "score": 0.95,
        "metadata": {...}
      }
    ],
    "activation_memories": [],
    "parametric_memories": [],
    "total_results": 1
  }
}
```

### Chat Interface

**Chat Request:**
```json
POST /chat
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
{
  "query": "What were the key decisions from our Q4 planning?",
  "max_tokens": 1000,
  "top_k": 5
}
```

**Chat Response:**
```json
{
  "code": 200,
  "message": "Chat response generated",
  "data": "Based on your Q4 planning meeting notes, the key decisions were: 1) Increase marketing budget by 20%, 2) Launch new product line in January, 3) Hire additional developers for the mobile team..."
}
```

## Error Handling

All errors follow a consistent format:

```json
{
  "code": 400,
  "message": "Invalid request format",
  "error": "missing required field: query",
  "details": "Request ID: req_123456789"
}
```

### Common HTTP Status Codes

- `200` - Success
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (invalid/missing token)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (resource doesn't exist)
- `500` - Internal Server Error

## Configuration

### YAML Configuration Example

```yaml
# config.yaml
user_id: "api_user"
session_id: "api_session"
enable_textual_memory: true
enable_activation_memory: false
top_k: 10
api_port: 8080
environment: "production"
metrics_enabled: true
metrics_port: 9090

chat_model:
  backend: "openai"
  config:
    model_name_or_path: "gpt-4"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    api_base: "https://api.openai.com/v1"

mem_reader:
  backend: "default"
  config:
    embedding_model: "sentence-transformers/all-MiniLM-L6-v2"
```

## Development

### Running Tests

```bash
# Run all tests
go test ./api/...

# Run tests with coverage
go test -cover ./api/...

# Run benchmarks
go test -bench=. ./api/...
```

### Building

```bash
# Build the API server
go build -o memgos-api ./cmd/memgos

# Build with version info
go build -ldflags "-X main.Version=v1.0.0 -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" -o memgos-api ./cmd/memgos
```

### Docker

```dockerfile
# Dockerfile example
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o memgos-api ./cmd/memgos

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/memgos-api .
EXPOSE 8080
CMD ["./memgos-api", "--api"]
```

## Architecture

### Server Components

```
┌─────────────────┐
│   Gin Router    │
├─────────────────┤
│   Middleware    │
│ • CORS          │
│ • Auth (JWT)    │
│ • Logging       │
│ • Metrics       │
│ • Rate Limit    │
├─────────────────┤
│   Handlers      │
│ • Memory Ops    │
│ • Search/Chat   │
│ • User Mgmt     │
│ • Health Check  │
├─────────────────┤
│   MOS Core      │
│ • Memory Mgmt   │
│ • LLM Interface │
│ • Vector DB     │
│ • Graph DB      │
└─────────────────┘
```

### File Structure

```
api/
├── server.go              # Main server implementation
├── server_enhanced.go     # Enhanced server with auth
├── handlers.go            # HTTP request handlers
├── middleware.go          # HTTP middleware
├── auth.go               # Authentication & sessions
├── models.go             # Request/response models
├── openapi.go            # OpenAPI specification
├── server_test.go        # Test suite
└── README.md             # This documentation
```

## Security Considerations

### Production Deployment

1. **JWT Secret**: Use a strong, randomly generated JWT secret
2. **HTTPS**: Always use HTTPS in production
3. **Rate Limiting**: Configure appropriate rate limits
4. **CORS**: Restrict allowed origins
5. **Input Validation**: All inputs are validated
6. **Error Messages**: Avoid exposing sensitive information

### Authentication Flow

1. Client sends login request with credentials
2. Server validates credentials and creates session
3. Server returns JWT token with expiration
4. Client includes token in Authorization header
5. Server validates token and extracts user context
6. Token can be refreshed before expiration

## Monitoring & Observability

### Health Checks

The `/health` endpoint provides comprehensive health information:

```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0.0",
  "uptime": "2h30m45s",
  "checks": {
    "core": "ok",
    "database": "ok",
    "memory": "ok",
    "auth": "ok",
    "sessions": "5 active"
  }
}
```

### Metrics

The `/metrics` endpoint provides performance metrics:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "uptime": "2h30m45s",
  "metrics": {
    "requests_total": 1250,
    "requests_success": 1200,
    "requests_error": 50,
    "memory_usage": "256MB",
    "active_sessions": 25,
    "avg_response_time": "120ms"
  }
}
```

### Logging

Structured logging with request correlation:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "info",
  "message": "HTTP Request",
  "method": "POST",
  "path": "/search",
  "status_code": 200,
  "latency": "125ms",
  "client_ip": "192.168.1.100",
  "user_agent": "MemGOS-Client/1.0",
  "request_id": "req_123456789",
  "user_id": "john_doe"
}
```

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Support

For issues and questions:
- GitHub Issues: https://github.com/memtensor/memgos/issues
- Documentation: https://memgos.dev/docs
- Community: https://discord.gg/memgos