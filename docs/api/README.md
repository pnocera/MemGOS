# MemGOS API Documentation

## Overview

The MemGOS API provides a comprehensive RESTful interface for managing memory operations, user authentication, and chat functionality. The API is designed following REST principles and provides JSON responses for all endpoints.

## Base URL

```
http://localhost:8000/api/v1
```

## Authentication

Currently, MemGOS uses simple user ID-based authentication. In production deployments, this should be replaced with proper authentication mechanisms like JWT tokens or OAuth.

### Headers

```http
Content-Type: application/json
X-User-ID: your-user-id
```

## API Endpoints

### Core Memory Operations

#### Search Memories

**GET** `/search`

Search across all accessible memory cubes for relevant content.

**Query Parameters:**
- `q` (required): Search query string
- `top_k` (optional): Maximum number of results to return (default: 5)
- `cube_ids` (optional): Comma-separated list of cube IDs to search
- `memory_types` (optional): Comma-separated list of memory types (textual, activation, parametric)

**Example Request:**
```bash
curl -X GET "http://localhost:8000/api/v1/search?q=golang&top_k=10" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "textual_memories": [
      {
        "id": "mem-001",
        "memory": "Go is a programming language...",
        "metadata": {
          "topic": "programming",
          "source": "manual"
        },
        "cube_id": "tech-knowledge",
        "score": 0.95,
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ],
    "activation_memories": [],
    "parametric_memories": [],
    "total_results": 1,
    "search_time_ms": 15
  }
}
```

#### Add Memory

**POST** `/memories`

Add new memory content to a specific memory cube.

**Request Body:**
```json
{
  "memory_content": "Your memory content here",
  "cube_id": "target-cube-id",
  "memory_type": "textual",
  "metadata": {
    "category": "programming",
    "tags": ["go", "tutorial"]
  }
}
```

**Example Request:**
```bash
curl -X POST "http://localhost:8000/api/v1/memories" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "memory_content": "Go channels provide a way for goroutines to communicate",
    "cube_id": "golang-knowledge",
    "memory_type": "textual",
    "metadata": {
      "topic": "concurrency",
      "difficulty": "intermediate"
    }
  }'
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "memory_id": "mem-12345",
    "cube_id": "golang-knowledge",
    "created_at": "2024-01-15T10:30:00Z"
  },
  "message": "Memory added successfully"
}
```

#### Get Memory

**GET** `/memories/{memory_id}`

Retrieve a specific memory by ID.

**Path Parameters:**
- `memory_id`: The unique identifier of the memory

**Query Parameters:**
- `cube_id`: The cube ID containing the memory

**Example Request:**
```bash
curl -X GET "http://localhost:8000/api/v1/memories/mem-12345?cube_id=golang-knowledge" \
  -H "X-User-ID: user-123"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "id": "mem-12345",
    "memory": "Go channels provide a way for goroutines to communicate",
    "metadata": {
      "topic": "concurrency",
      "difficulty": "intermediate"
    },
    "cube_id": "golang-knowledge",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

#### Update Memory

**PUT** `/memories/{memory_id}`

Update an existing memory.

**Request Body:**
```json
{
  "memory_content": "Updated memory content",
  "cube_id": "target-cube-id",
  "metadata": {
    "updated": true
  }
}
```

#### Delete Memory

**DELETE** `/memories/{memory_id}`

Delete a specific memory.

**Query Parameters:**
- `cube_id`: The cube ID containing the memory

**Example Request:**
```bash
curl -X DELETE "http://localhost:8000/api/v1/memories/mem-12345?cube_id=golang-knowledge" \
  -H "X-User-ID: user-123"
```

### Memory Cube Management

#### List Memory Cubes

**GET** `/cubes`

List all memory cubes accessible to the user.

**Example Response:**
```json
{
  "success": true,
  "data": {
    "cubes": [
      {
        "id": "golang-knowledge",
        "name": "Go Programming Knowledge",
        "description": "Collection of Go programming concepts",
        "memory_types": ["textual"],
        "memory_count": {
          "textual": 150,
          "activation": 0,
          "parametric": 0
        },
        "created_at": "2024-01-10T09:00:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total_count": 1
  }
}
```

#### Register Memory Cube

**POST** `/cubes`

Register a new memory cube from a directory or create an empty one.

**Request Body:**
```json
{
  "cube_id": "new-cube-id",
  "cube_name": "New Memory Cube",
  "cube_path": "/path/to/cube/directory",
  "create_empty": false
}
```

#### Get Memory Cube Info

**GET** `/cubes/{cube_id}`

Get detailed information about a specific memory cube.

#### Unregister Memory Cube

**DELETE** `/cubes/{cube_id}`

Unregister a memory cube from the system.

### Chat Operations

#### Send Chat Message

**POST** `/chat`

Send a chat message and receive an AI-generated response with memory context.

**Request Body:**
```json
{
  "message": "What can you tell me about Go channels?",
  "context_cubes": ["golang-knowledge"],
  "max_context_memories": 5,
  "conversation_id": "conv-123"
}
```

**Example Request:**
```bash
curl -X POST "http://localhost:8000/api/v1/chat" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "message": "How do Go channels work?",
    "context_cubes": ["golang-knowledge"],
    "max_context_memories": 3
  }'
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "response": "Go channels are a powerful feature for goroutine communication...",
    "conversation_id": "conv-123",
    "context_memories": [
      {
        "memory_id": "mem-12345",
        "memory_content": "Go channels provide a way for goroutines to communicate",
        "relevance_score": 0.95
      }
    ],
    "response_time_ms": 1200,
    "timestamp": "2024-01-15T10:35:00Z"
  }
}
```

#### Get Chat History

**GET** `/chat/history`

Retrieve chat conversation history.

**Query Parameters:**
- `conversation_id` (optional): Specific conversation ID
- `limit` (optional): Maximum number of messages to return
- `offset` (optional): Number of messages to skip

### User Management

#### Create User

**POST** `/users`

Create a new user account.

**Request Body:**
```json
{
  "username": "john_doe",
  "email": "john@example.com",
  "role": "user",
  "metadata": {
    "department": "engineering"
  }
}
```

#### Get User Info

**GET** `/users/me`

Get current user information.

**Example Response:**
```json
{
  "success": true,
  "data": {
    "user_id": "user-123",
    "username": "john_doe",
    "email": "john@example.com",
    "role": "user",
    "accessible_cubes": ["golang-knowledge", "personal-notes"],
    "created_at": "2024-01-01T00:00:00Z",
    "last_active": "2024-01-15T10:30:00Z"
  }
}
```

### System Operations

#### Health Check

**GET** `/health`

Check system health and component status.

**Example Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "uptime_seconds": 86400,
    "components": {
      "memory_system": "healthy",
      "database": "healthy",
      "llm_service": "healthy",
      "vector_db": "healthy"
    },
    "metrics": {
      "total_memories": 1500,
      "active_users": 25,
      "requests_per_minute": 45
    }
  }
}
```

#### System Metrics

**GET** `/metrics`

Get detailed system metrics.

**Example Response:**
```json
{
  "success": true,
  "data": {
    "memory_usage": {
      "total_memories": 1500,
      "textual_memories": 1200,
      "activation_memories": 200,
      "parametric_memories": 100
    },
    "performance": {
      "avg_search_time_ms": 25,
      "avg_add_time_ms": 15,
      "cache_hit_rate": 0.85
    },
    "usage": {
      "requests_24h": 5000,
      "active_users_24h": 100,
      "popular_operations": [
        {"operation": "search", "count": 3000},
        {"operation": "add", "count": 1500},
        {"operation": "chat", "count": 500}
      ]
    }
  }
}
```

## Error Handling

### Standard Error Response

All error responses follow this format:

```json
{
  "success": false,
  "error": {
    "code": "MEMORY_NOT_FOUND",
    "message": "The specified memory could not be found",
    "details": {
      "memory_id": "mem-12345",
      "cube_id": "golang-knowledge"
    },
    "request_id": "req-abcd1234"
  }
}
```

### Error Codes

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `INVALID_REQUEST` | Request validation failed | 400 |
| `UNAUTHORIZED` | Authentication required | 401 |
| `FORBIDDEN` | Insufficient permissions | 403 |
| `MEMORY_NOT_FOUND` | Memory not found | 404 |
| `CUBE_NOT_FOUND` | Memory cube not found | 404 |
| `USER_NOT_FOUND` | User not found | 404 |
| `CONFLICT` | Resource already exists | 409 |
| `VALIDATION_ERROR` | Data validation failed | 422 |
| `INTERNAL_ERROR` | Internal server error | 500 |
| `SERVICE_UNAVAILABLE` | External service unavailable | 503 |

## Rate Limiting

API endpoints are rate-limited to ensure system stability:

- **Search operations**: 100 requests per minute per user
- **Memory operations**: 200 requests per minute per user
- **Chat operations**: 30 requests per minute per user
- **Administrative operations**: 10 requests per minute per user

When rate limits are exceeded, the API returns HTTP 429 with:

```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded",
    "details": {
      "limit": 100,
      "window_seconds": 60,
      "retry_after_seconds": 30
    }
  }
}
```

## SDK and Client Libraries

### Go Client

```go
import "github.com/memtensor/memgos/pkg/client"

client := client.NewMemGOSClient("http://localhost:8000", "user-123")

// Search memories
results, err := client.Search(ctx, "golang channels", 5)

// Add memory
err = client.AddMemory(ctx, "Memory content", "cube-id", metadata)

// Chat
response, err := client.Chat(ctx, "How do channels work?", []string{"cube-id"})
```

### Python Client

```python
from memgos_client import MemGOSClient

client = MemGOSClient("http://localhost:8000", user_id="user-123")

# Search memories
results = client.search("golang channels", top_k=5)

# Add memory
client.add_memory("Memory content", cube_id="cube-id", metadata={})

# Chat
response = client.chat("How do channels work?", context_cubes=["cube-id"])
```

### JavaScript Client

```javascript
import { MemGOSClient } from '@memgos/client';

const client = new MemGOSClient('http://localhost:8000', 'user-123');

// Search memories
const results = await client.search('golang channels', { topK: 5 });

// Add memory
await client.addMemory('Memory content', { cubeId: 'cube-id', metadata: {} });

// Chat
const response = await client.chat('How do channels work?', { contextCubes: ['cube-id'] });
```

## OpenAPI Specification

The complete OpenAPI 3.0 specification is available at:

```
http://localhost:8000/api/v1/openapi.json
```

You can view the interactive documentation at:

```
http://localhost:8000/docs
```

## Examples

See the [examples directory](../examples/) for complete usage examples including:

- Basic API operations
- Advanced search patterns
- Chat integration
- Multi-cube operations
- Error handling
- Authentication flows

## Migration from Python MemOS API

See the [Migration Guide](../MIGRATION_GUIDE.md) for detailed information about migrating from the Python MemOS API to MemGOS.
