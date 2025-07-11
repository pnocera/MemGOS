# MemGOS MCP Server Documentation

The MemGOS Model Context Protocol (MCP) server provides a standardized interface for AI tools and applications to interact with MemGOS memory systems. This comprehensive server implementation includes 16 tools covering all aspects of memory management, chat functionality, and system operations.

## 🚀 Quick Start

### Installation

```bash
# Build the MCP server
go build -o memgos-mcp ./cmd/memgos-mcp

# Install system-wide (optional)
sudo cp memgos-mcp /usr/local/bin/
```

### Basic Usage

```bash
# Run with API token authentication
./memgos-mcp --token your-api-token --api-url http://localhost:8080

# Run with environment variable
export MEMGOS_API_TOKEN="your-api-token"
./memgos-mcp --api-url http://localhost:8080

# Show version and help
./memgos-mcp --version
./memgos-mcp --help
```

### Claude Desktop Integration

Add to your Claude Desktop configuration (`~/.claude/config.json`):

```json
{
  "mcpServers": {
    "memgos": {
      "command": "/path/to/memgos-mcp",
      "args": ["--token", "your-api-token", "--api-url", "http://localhost:8080"],
      "env": {
        "MEMGOS_API_TOKEN": "your-api-token"
      }
    }
  }
}
```

## 🔧 Configuration

### Command Line Options

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--token` | API token for authentication | - | Yes (if no env var) |
| `--api-url` | MemGOS API base URL | `http://localhost:8080` | No |
| `--user` | User ID for the session | `default-user` | No |
| `--session` | Session ID for the MCP server | `mcp-session` | No |
| `--config` | Path to configuration file | - | No |
| `--verbose` | Enable verbose logging | `false` | No |
| `--version` | Show version information | - | No |

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `MEMGOS_API_TOKEN` | API authentication token | `memgos_abc123...` |

### Configuration File

You can optionally provide a YAML configuration file:

```yaml
# mcp-config.yaml
user_id: "alice"
session_id: "mcp-session-001" 
api_token: "your-api-token"
api_url: "https://memgos.example.com"
log_level: "info"
```

## 🛠️ Available Tools

The MCP server provides 16 comprehensive tools organized into categories:

### Memory Operations

#### 1. search_memories
Search across all memory types with advanced filtering and ranking.

**Parameters:**
- `query` (string, required): Search query text
- `top_k` (number, optional): Number of results to return (default: 5)
- `cube_ids` (array, optional): Specific cubes to search
- `user_id` (string, optional): User context for search

**Example:**
```json
{
  "query": "machine learning algorithms",
  "top_k": 10,
  "cube_ids": ["ml-research", "ai-papers"]
}
```

#### 2. add_memory
Add new memory content to a specified cube.

**Parameters:**
- `memory_content` (string, required): Content to store
- `mem_cube_id` (string, required): Target memory cube ID
- `user_id` (string, optional): User context

**Example:**
```json
{
  "memory_content": "Neural networks are inspired by biological neurons",
  "mem_cube_id": "ai-concepts",
  "user_id": "researcher"
}
```

#### 3. get_memory
Retrieve a specific memory by ID.

**Parameters:**
- `memory_id` (string, required): Unique memory identifier
- `cube_id` (string, required): Memory cube ID
- `user_id` (string, optional): User context

#### 4. update_memory
Update existing memory content.

**Parameters:**
- `memory_id` (string, required): Memory to update
- `cube_id` (string, required): Memory cube ID
- `updated_content` (string, required): New content
- `user_id` (string, optional): User context

#### 5. delete_memory
Delete a specific memory from a cube.

**Parameters:**
- `memory_id` (string, required): Memory to delete
- `cube_id` (string, required): Memory cube ID
- `user_id` (string, optional): User context

### Cube Management

#### 6. register_cube
Register a new memory cube in the system.

**Parameters:**
- `cube_id` (string, required): Unique cube identifier
- `cube_path` (string, optional): File system path
- `user_id` (string, optional): Owner user ID

#### 7. list_cubes
List all available memory cubes for a user.

**Parameters:**
- `user_id` (string, optional): User context

#### 8. cube_info
Get detailed information about a specific cube.

**Parameters:**
- `cube_id` (string, required): Cube to inspect
- `user_id` (string, optional): User context

### Chat Integration

#### 9. chat
Send a message and get an AI response with memory context.

**Parameters:**
- `query` (string, required): Chat message
- `user_id` (string, optional): User context
- `max_tokens` (number, optional): Response length limit
- `top_k` (number, optional): Memory context size

**Example:**
```json
{
  "query": "What did I learn about neural networks?",
  "max_tokens": 500,
  "top_k": 3
}
```

#### 10. chat_history
Retrieve conversation history for a session.

**Parameters:**
- `user_id` (string, optional): User context
- `session_id` (string, optional): Session identifier
- `limit` (number, optional): Number of messages

#### 11. context_search
Search for relevant context based on a query.

**Parameters:**
- `query` (string, required): Context search query
- `user_id` (string, optional): User context
- `top_k` (number, optional): Context size

### User Management

#### 12. get_user_info
Get information about a user account.

**Parameters:**
- `user_id` (string, optional): User to query (defaults to current user)

#### 13. session_info
Get details about the current session.

**Parameters:**
- `session_id` (string, optional): Session to query

#### 14. validate_access
Check if a user has access to specific resources.

**Parameters:**
- `user_id` (string, required): User to validate
- `resource_type` (string, required): Type of resource
- `resource_id` (string, required): Specific resource

### System Tools

#### 15. health_check
Check the health and status of the MemGOS system.

**Parameters:** None

#### 16. list_tools
Get a list of all available MCP tools (self-discovery).

**Parameters:** None

## 🔐 Authentication

The MCP server uses API token authentication for secure communication with the MemGOS backend.

### API Token Format

API tokens follow the format: `memgos_<random-string>`

### Authentication Flow

1. **Token Validation**: Each request includes `Authorization: Bearer <token>`
2. **Health Check**: Server validates API connectivity on startup
3. **Request Processing**: All tool calls are authenticated via the API token

### Getting an API Token

```bash
# First, authenticate with the MemGOS API to get a JWT
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'

# Create an API token using the JWT
curl -X POST http://localhost:8080/api/v1/tokens \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt-token>" \
  -d '{
    "name": "MCP Server Token",
    "description": "Token for MCP server integration",
    "scopes": ["read", "write"]
  }'
```

## 🏗️ Architecture

### Server Components

```
MCP Server
├── Authentication Layer (API Token)
├── Tool Registry (16 Tools)
├── API Client (HTTP Backend)
├── Request/Response Handling
└── Stdio Transport
```

### Communication Flow

1. **MCP Client** → **Stdio Protocol** → **MCP Server**
2. **MCP Server** → **HTTP API** → **MemGOS Backend**
3. **MemGOS Backend** → **HTTP Response** → **MCP Server**
4. **MCP Server** → **Stdio Protocol** → **MCP Client**

### Dual Architecture Support

The server supports two operational modes:

#### API Mode (Recommended)
- Communicates with MemGOS via HTTP API
- Requires API token authentication
- Supports distributed deployments
- Full feature compatibility

#### Direct Mode
- Directly interfaces with MOSCore
- For embedded scenarios
- Requires local database access
- Legacy compatibility

## 🧪 Testing

### Manual Testing

```bash
# Test server connectivity
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}' | ./memgos-mcp --token test-token

# Test tool listing
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./memgos-mcp --token test-token
```

### Integration Testing

```bash
# Test with actual MemGOS backend
./memgos-mcp --token your-real-token --api-url http://localhost:8080 --verbose
```

## 🐛 Troubleshooting

### Common Issues

#### Connection Refused
```
Error: failed to make health check request: dial tcp: connect: connection refused
```
**Solution**: Ensure MemGOS API server is running on the specified URL.

#### Invalid Token
```
Error: health check failed with status 401: Unauthorized
```
**Solution**: Verify your API token is valid and has appropriate scopes.

#### MCP Client Not Recognizing Server
**Solution**: Check your MCP client configuration and ensure the server binary path is correct.

### Debug Mode

Run with verbose logging for detailed information:

```bash
./memgos-mcp --token your-token --verbose
```

### Logs Analysis

The server logs include:
- API connection status
- Tool execution details  
- Error messages with context
- Performance metrics

## 📚 Examples

### Basic Memory Operations

```python
# Using the MCP tools via a Python client
import json

# Search for memories
search_request = {
    "query": "artificial intelligence",
    "top_k": 5
}

# Add new memory
add_request = {
    "memory_content": "AI is transforming healthcare through diagnostic imaging",
    "mem_cube_id": "healthcare-ai"
}

# Chat with memory context
chat_request = {
    "query": "How is AI being used in healthcare?",
    "max_tokens": 300
}
```

### Advanced Usage

```bash
# Set up a complete workflow
export MEMGOS_API_TOKEN="your-token"

# 1. Register a new cube
./memgos-mcp --api-url http://localhost:8080 << EOF
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"register_cube","arguments":{"cube_id":"research-notes","cube_path":"./research"}}}
EOF

# 2. Add memories
./memgos-mcp --api-url http://localhost:8080 << EOF  
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"add_memory","arguments":{"memory_content":"Transformers architecture revolutionized NLP","mem_cube_id":"research-notes"}}}
EOF

# 3. Search and chat
./memgos-mcp --api-url http://localhost:8080 << EOF
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"chat","arguments":{"query":"Tell me about transformer architecture"}}}
EOF
```

## 🔗 Integration Guides

### Claude Desktop
See [Claude Desktop Integration Guide](claude-desktop.md)

### VS Code Extension
See [VS Code MCP Integration Guide](vscode.md)

### Custom MCP Clients
See [Building MCP Clients Guide](custom-clients.md)

## 📊 Performance

### Benchmarks

- **Tool Execution**: < 100ms average response time
- **Memory Search**: < 50ms for 10K memories
- **Chat Response**: < 2s with context retrieval
- **Concurrent Connections**: Supports 100+ simultaneous MCP clients

### Optimization Tips

1. **Use specific cube_ids** in searches to reduce scope
2. **Limit top_k** parameter for faster responses
3. **Cache API tokens** to avoid repeated authentication
4. **Use session persistence** for multiple operations

## 🤝 Contributing

### Adding New Tools

1. Define tool schema in `setupTools()`
2. Implement handler function
3. Add to tool registry
4. Update documentation

### Testing New Features

```bash
# Run tests
go test ./mcp ./cmd/memgos-mcp

# Lint code
go vet ./mcp ./cmd/memgos-mcp

# Format code
go fmt ./mcp ./cmd/memgos-mcp
```

## 📄 License

The MCP server is part of MemGOS and is licensed under the MIT License.