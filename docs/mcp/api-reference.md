# MemGOS MCP Server API Reference

This document provides detailed API reference for all 16 tools available in the MemGOS Model Context Protocol server.

## 🔧 Tool Categories

- [Memory Operations](#memory-operations) (5 tools)
- [Cube Management](#cube-management) (3 tools)  
- [Chat Integration](#chat-integration) (3 tools)
- [User Management](#user-management) (3 tools)
- [System Tools](#system-tools) (2 tools)

---

## Memory Operations

### 1. search_memories

Search across all memory types with advanced filtering and ranking capabilities.

#### Schema
```json
{
  "name": "search_memories",
  "description": "Search memories across all cubes with semantic similarity and filtering",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Search query text for semantic matching"
      },
      "top_k": {
        "type": "number",
        "description": "Maximum number of results to return",
        "default": 5,
        "minimum": 1,
        "maximum": 100
      },
      "cube_ids": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Specific cube IDs to search within (optional)"
      },
      "user_id": {
        "type": "string",
        "description": "User context for access control (optional)"
      }
    },
    "required": ["query"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "results": {
    "text_mem": [
      {
        "cube_id": "research-notes",
        "memories": [
          {
            "id": "mem_001",
            "content": "Neural networks are computational models...",
            "metadata": {
              "score": 0.92,
              "created_at": "2024-01-15T10:30:00Z"
            }
          }
        ]
      }
    ],
    "act_mem": [],
    "para_mem": []
  },
  "total_results": 5,
  "search_time_ms": 45
}
```

#### Example Usage
```json
{
  "query": "machine learning algorithms",
  "top_k": 10,
  "cube_ids": ["ai-research", "ml-papers"]
}
```

---

### 2. add_memory

Add new memory content to a specified memory cube.

#### Schema
```json
{
  "name": "add_memory",
  "description": "Add new textual memory to a memory cube",
  "inputSchema": {
    "type": "object",
    "properties": {
      "memory_content": {
        "type": "string",
        "description": "The content to store as memory",
        "minLength": 1,
        "maxLength": 50000
      },
      "mem_cube_id": {
        "type": "string",
        "description": "Target memory cube identifier"
      },
      "user_id": {
        "type": "string",
        "description": "User context for ownership (optional)"
      },
      "metadata": {
        "type": "object",
        "description": "Additional metadata to store with memory (optional)",
        "additionalProperties": true
      }
    },
    "required": ["memory_content", "mem_cube_id"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "memory_id": "mem_12345",
  "cube_id": "ai-research",
  "created_at": "2024-01-15T10:30:00Z",
  "message": "Memory added successfully"
}
```

#### Example Usage
```json
{
  "memory_content": "Transformers revolutionized NLP by introducing self-attention mechanisms that allow models to weigh the importance of different parts of the input sequence.",
  "mem_cube_id": "ai-research",
  "metadata": {
    "source": "research_paper",
    "category": "nlp",
    "importance": "high"
  }
}
```

---

### 3. get_memory

Retrieve a specific memory by its unique identifier.

#### Schema
```json
{
  "name": "get_memory",
  "description": "Retrieve a specific memory by ID from a cube",
  "inputSchema": {
    "type": "object",
    "properties": {
      "memory_id": {
        "type": "string",
        "description": "Unique memory identifier"
      },
      "cube_id": {
        "type": "string",
        "description": "Memory cube identifier"
      },
      "user_id": {
        "type": "string",
        "description": "User context for access control (optional)"
      }
    },
    "required": ["memory_id", "cube_id"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "memory": {
    "id": "mem_12345",
    "content": "Transformers revolutionized NLP...",
    "metadata": {
      "source": "research_paper",
      "category": "nlp",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    },
    "cube_id": "ai-research"
  }
}
```

---

### 4. update_memory

Update the content of an existing memory.

#### Schema
```json
{
  "name": "update_memory",
  "description": "Update existing memory content",
  "inputSchema": {
    "type": "object",
    "properties": {
      "memory_id": {
        "type": "string",
        "description": "Memory identifier to update"
      },
      "cube_id": {
        "type": "string",
        "description": "Memory cube identifier"
      },
      "updated_content": {
        "type": "string",
        "description": "New memory content",
        "minLength": 1,
        "maxLength": 50000
      },
      "user_id": {
        "type": "string",
        "description": "User context for access control (optional)"
      },
      "metadata": {
        "type": "object",
        "description": "Updated metadata (optional)",
        "additionalProperties": true
      }
    },
    "required": ["memory_id", "cube_id", "updated_content"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "memory_id": "mem_12345",
  "updated_at": "2024-01-15T11:45:00Z",
  "message": "Memory updated successfully"
}
```

---

### 5. delete_memory

Remove a specific memory from a cube.

#### Schema
```json
{
  "name": "delete_memory",
  "description": "Delete a specific memory from a cube",
  "inputSchema": {
    "type": "object",
    "properties": {
      "memory_id": {
        "type": "string",
        "description": "Memory identifier to delete"
      },
      "cube_id": {
        "type": "string", 
        "description": "Memory cube identifier"
      },
      "user_id": {
        "type": "string",
        "description": "User context for access control (optional)"
      }
    },
    "required": ["memory_id", "cube_id"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "memory_id": "mem_12345",
  "deleted_at": "2024-01-15T12:00:00Z",
  "message": "Memory deleted successfully"
}
```

---

## Cube Management

### 6. register_cube

Register a new memory cube in the system.

#### Schema
```json
{
  "name": "register_cube",
  "description": "Register a new memory cube for storing memories",
  "inputSchema": {
    "type": "object",
    "properties": {
      "cube_id": {
        "type": "string",
        "description": "Unique identifier for the new cube",
        "pattern": "^[a-zA-Z0-9_-]+$",
        "minLength": 1,
        "maxLength": 100
      },
      "cube_path": {
        "type": "string",
        "description": "File system path for the cube (optional)"
      },
      "user_id": {
        "type": "string",
        "description": "Owner user ID (optional)"
      },
      "name": {
        "type": "string",
        "description": "Human-readable cube name (optional)"
      },
      "description": {
        "type": "string",
        "description": "Cube description (optional)"
      }
    },
    "required": ["cube_id"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "cube_id": "ai-research",
  "created_at": "2024-01-15T10:00:00Z",
  "message": "Memory cube registered successfully"
}
```

---

### 7. list_cubes

List all available memory cubes for a user.

#### Schema
```json
{
  "name": "list_cubes",
  "description": "List all available memory cubes for the user",
  "inputSchema": {
    "type": "object",
    "properties": {
      "user_id": {
        "type": "string",
        "description": "User context for filtering cubes (optional)"
      },
      "include_stats": {
        "type": "boolean",
        "description": "Include memory count statistics",
        "default": false
      }
    }
  }
}
```

#### Response Format
```json
{
  "success": true,
  "cubes": [
    {
      "id": "ai-research",
      "name": "AI Research Notes",
      "owner_id": "user123",
      "created_at": "2024-01-15T10:00:00Z",
      "is_loaded": true,
      "memory_count": 245
    },
    {
      "id": "personal-notes",
      "name": "Personal Knowledge",
      "owner_id": "user123", 
      "created_at": "2024-01-10T08:30:00Z",
      "is_loaded": true,
      "memory_count": 89
    }
  ],
  "total_cubes": 2
}
```

---

### 8. cube_info

Get detailed information about a specific memory cube.

#### Schema
```json
{
  "name": "cube_info",
  "description": "Get detailed information about a specific memory cube",
  "inputSchema": {
    "type": "object",
    "properties": {
      "cube_id": {
        "type": "string",
        "description": "Memory cube identifier to inspect"
      },
      "user_id": {
        "type": "string",
        "description": "User context for access control (optional)"
      }
    },
    "required": ["cube_id"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "cube": {
    "id": "ai-research",
    "name": "AI Research Notes",
    "path": "/data/cubes/ai-research",
    "owner_id": "user123",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T11:30:00Z",
    "is_loaded": true,
    "statistics": {
      "total_memories": 245,
      "text_memories": 230,
      "activation_memories": 10,
      "parametric_memories": 5,
      "total_size_bytes": 2048576,
      "last_accessed": "2024-01-15T11:30:00Z"
    }
  }
}
```

---

## Chat Integration

### 9. chat

Send a message and receive an AI response with memory context.

#### Schema
```json
{
  "name": "chat",
  "description": "Send a chat message with memory-augmented response",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "The chat message or question",
        "minLength": 1,
        "maxLength": 10000
      },
      "user_id": {
        "type": "string",
        "description": "User context for personalized responses (optional)"
      },
      "max_tokens": {
        "type": "number",
        "description": "Maximum tokens in response",
        "default": 500,
        "minimum": 50,
        "maximum": 4000
      },
      "top_k": {
        "type": "number",
        "description": "Number of memory contexts to include",
        "default": 3,
        "minimum": 1,
        "maximum": 10
      },
      "cube_ids": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Specific cubes to search for context (optional)"
      }
    },
    "required": ["query"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "response": "Based on your research notes, transformers revolutionized NLP through self-attention mechanisms...",
  "session_id": "chat_session_001",
  "user_id": "user123",
  "timestamp": "2024-01-15T12:00:00Z",
  "context_used": [
    {
      "memory_id": "mem_12345",
      "cube_id": "ai-research", 
      "relevance_score": 0.92
    }
  ],
  "token_count": 145
}
```

---

### 10. chat_history

Retrieve conversation history for a session.

#### Schema
```json
{
  "name": "chat_history",
  "description": "Retrieve chat conversation history for a session",
  "inputSchema": {
    "type": "object",
    "properties": {
      "user_id": {
        "type": "string",
        "description": "User context for history filtering (optional)"
      },
      "session_id": {
        "type": "string",
        "description": "Specific session to retrieve (optional)"
      },
      "limit": {
        "type": "number",
        "description": "Maximum number of messages to return",
        "default": 50,
        "minimum": 1,
        "maximum": 1000
      },
      "offset": {
        "type": "number",
        "description": "Number of messages to skip",
        "default": 0,
        "minimum": 0
      }
    }
  }
}
```

#### Response Format
```json
{
  "success": true,
  "chat_history": {
    "session_id": "chat_session_001",
    "user_id": "user123",
    "created_at": "2024-01-15T10:00:00Z",
    "total_messages": 12,
    "messages": [
      {
        "role": "user",
        "content": "What are transformers in machine learning?",
        "timestamp": "2024-01-15T10:00:00Z"
      },
      {
        "role": "assistant", 
        "content": "Transformers are a type of neural network architecture...",
        "timestamp": "2024-01-15T10:00:15Z"
      }
    ]
  }
}
```

---

### 11. context_search

Search for relevant context based on a query for enhanced chat responses.

#### Schema
```json
{
  "name": "context_search",
  "description": "Search for relevant memory context for chat enhancement",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Query to find relevant context for"
      },
      "user_id": {
        "type": "string",
        "description": "User context for search (optional)"
      },
      "top_k": {
        "type": "number",
        "description": "Number of context items to return",
        "default": 5,
        "minimum": 1,
        "maximum": 20
      },
      "cube_ids": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Specific cubes to search (optional)"
      }
    },
    "required": ["query"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "context": [
    {
      "memory_id": "mem_12345",
      "cube_id": "ai-research",
      "content": "Transformers use self-attention mechanisms...",
      "relevance_score": 0.95,
      "metadata": {
        "source": "research_paper",
        "category": "nlp"
      }
    }
  ],
  "total_context_items": 5,
  "search_time_ms": 23
}
```

---

## User Management

### 12. get_user_info

Get information about a user account and permissions.

#### Schema
```json
{
  "name": "get_user_info",
  "description": "Get user account information and permissions",
  "inputSchema": {
    "type": "object",
    "properties": {
      "user_id": {
        "type": "string",
        "description": "User ID to query (defaults to current user)"
      }
    }
  }
}
```

#### Response Format
```json
{
  "success": true,
  "user": {
    "user_id": "user123",
    "user_name": "alice_researcher",
    "role": "user",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z",
    "is_active": true,
    "permissions": {
      "can_read": true,
      "can_write": true,
      "can_admin": false
    },
    "cube_count": 5,
    "memory_count": 1234
  }
}
```

---

### 13. session_info

Get details about the current session.

#### Schema
```json
{
  "name": "session_info",
  "description": "Get information about the current session",
  "inputSchema": {
    "type": "object",
    "properties": {
      "session_id": {
        "type": "string",
        "description": "Specific session to query (optional)"
      }
    }
  }
}
```

#### Response Format
```json
{
  "success": true,
  "session": {
    "session_id": "mcp-session-001",
    "user_id": "user123",
    "created_at": "2024-01-15T10:00:00Z",
    "last_activity": "2024-01-15T12:00:00Z",
    "is_active": true,
    "api_mode": true,
    "api_url": "http://localhost:8080",
    "tool_calls_count": 25,
    "active_cubes": ["ai-research", "personal-notes"]
  }
}
```

---

### 14. validate_access

Check if a user has access to specific resources.

#### Schema
```json
{
  "name": "validate_access",
  "description": "Validate user access permissions for resources",
  "inputSchema": {
    "type": "object",
    "properties": {
      "user_id": {
        "type": "string",
        "description": "User to validate access for"
      },
      "resource_type": {
        "type": "string",
        "description": "Type of resource (cube, memory, etc.)",
        "enum": ["cube", "memory", "chat", "admin"]
      },
      "resource_id": {
        "type": "string",
        "description": "Specific resource identifier"
      },
      "action": {
        "type": "string",
        "description": "Action to validate",
        "enum": ["read", "write", "delete", "admin"]
      }
    },
    "required": ["user_id", "resource_type", "resource_id", "action"]
  }
}
```

#### Response Format
```json
{
  "success": true,
  "access": {
    "user_id": "user123",
    "resource_type": "cube",
    "resource_id": "ai-research",
    "action": "write",
    "has_access": true,
    "reason": "User is owner of the cube",
    "checked_at": "2024-01-15T12:00:00Z"
  }
}
```

---

## System Tools

### 15. health_check

Check the health and status of the MemGOS system.

#### Schema
```json
{
  "name": "health_check",
  "description": "Check system health and connectivity status",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

#### Response Format
```json
{
  "success": true,
  "health": {
    "status": "healthy",
    "timestamp": "2024-01-15T12:00:00Z",
    "api_connection": {
      "status": "connected",
      "url": "http://localhost:8080",
      "response_time_ms": 15
    },
    "services": {
      "memory_system": "operational",
      "chat_system": "operational",
      "user_system": "operational"
    },
    "version": "1.0.0",
    "uptime_seconds": 3600
  }
}
```

---

### 16. list_tools

Get a list of all available MCP tools with their schemas.

#### Schema
```json
{
  "name": "list_tools",
  "description": "List all available MCP tools with their schemas",
  "inputSchema": {
    "type": "object",
    "properties": {
      "category": {
        "type": "string",
        "description": "Filter by tool category (optional)",
        "enum": ["memory", "cube", "chat", "user", "system"]
      }
    }
  }
}
```

#### Response Format
```json
{
  "success": true,
  "tools": [
    {
      "name": "search_memories",
      "category": "memory",
      "description": "Search memories across all cubes with semantic similarity",
      "input_schema": {
        "type": "object",
        "properties": {...}
      }
    }
  ],
  "total_tools": 16,
  "categories": {
    "memory": 5,
    "cube": 3,
    "chat": 3,
    "user": 3,
    "system": 2
  }
}
```

---

## Error Handling

All tools return consistent error responses:

### Error Response Format
```json
{
  "success": false,
  "error": {
    "type": "validation_error",
    "message": "Required parameter 'query' is missing",
    "code": "MISSING_PARAMETER",
    "details": {
      "parameter": "query",
      "expected_type": "string"
    },
    "timestamp": "2024-01-15T12:00:00Z"
  }
}
```

### Common Error Types
- `validation_error`: Invalid input parameters
- `authentication_error`: Authentication token issues
- `authorization_error`: Insufficient permissions
- `not_found_error`: Resource not found
- `internal_error`: Server-side errors
- `network_error`: Connectivity issues

## Rate Limiting

The MCP server implements rate limiting for tool calls:
- **Default limit**: 100 calls per minute per user
- **Burst limit**: 20 calls in 10 seconds
- **Headers included**: `X-RateLimit-Remaining`, `X-RateLimit-Reset`

## Versioning

The API follows semantic versioning:
- **Current version**: 1.0.0
- **Breaking changes**: Will increment major version
- **New features**: Will increment minor version
- **Bug fixes**: Will increment patch version