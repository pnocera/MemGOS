# MemGOS Troubleshooting Guide

## Table of Contents

1. [Common Issues](#common-issues)
2. [Installation Problems](#installation-problems)
3. [Configuration Issues](#configuration-issues)
4. [Memory Operations Issues](#memory-operations-issues)
5. [API Issues](#api-issues)
6. [Performance Problems](#performance-problems)
7. [Database Connection Issues](#database-connection-issues)
8. [Authentication and Authorization](#authentication-and-authorization)
9. [Logging and Debugging](#logging-and-debugging)
10. [Error Codes Reference](#error-codes-reference)

## Common Issues

### Issue: "MemGOS binary not found"

**Symptoms**:
```
command not found: memgos
```

**Solutions**:

1. **Check installation**:
   ```bash
   which memgos
   echo $PATH
   ```

2. **Install MemGOS**:
   ```bash
   # Via Go
   go install github.com/memtensor/memgos/cmd/memgos@latest
   
   # Via source
   git clone https://github.com/memtensor/memgos.git
   cd memgos
   make install
   ```

3. **Check GOPATH/GOBIN**:
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   # Add to your shell profile (.bashrc, .zshrc, etc.)
   ```

### Issue: "Config file not found"

**Symptoms**:
```
Error: configuration file not found: config.yaml
```

**Solutions**:

1. **Specify config path explicitly**:
   ```bash
   memgos --config /full/path/to/config.yaml
   ```

2. **Use environment variable**:
   ```bash
   export MEMGOS_CONFIG=/path/to/config.yaml
   memgos
   ```

3. **Create default config**:
   ```bash
   memgos config init --output config.yaml
   ```

### Issue: "Invalid API key"

**Symptoms**:
```
Error: OpenAI API request failed: invalid API key
```

**Solutions**:

1. **Check API key**:
   ```bash
   echo $OPENAI_API_KEY
   # Should show your API key (masked)
   ```

2. **Set API key**:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   # Add to shell profile for persistence
   ```

3. **Verify API key in config**:
   ```yaml
   chat_model:
     api_key: "${OPENAI_API_KEY}"  # Correct
     # NOT: api_key: "$OPENAI_API_KEY"  # Wrong
   ```

4. **Test API key**:
   ```bash
   curl -H "Authorization: Bearer $OPENAI_API_KEY" \
        "https://api.openai.com/v1/models"
   ```

## Installation Problems

### Issue: Go version incompatibility

**Symptoms**:
```
go: module requires Go 1.24 or later
```

**Solutions**:

1. **Check Go version**:
   ```bash
   go version
   ```

2. **Update Go**:
   ```bash
   # Download from https://golang.org/dl/
   # Or use your package manager
   brew install go  # macOS
   sudo apt install golang-go  # Ubuntu/Debian
   ```

3. **Use Docker if Go update isn't possible**:
   ```bash
   docker run -v $(pwd):/workspace memtensor/memgos:latest
   ```

### Issue: Build failures

**Symptoms**:
```
build failed: module not found
```

**Solutions**:

1. **Clean module cache**:
   ```bash
   go clean -modcache
   go mod download
   ```

2. **Verify network connectivity**:
   ```bash
   go env GOPROXY
   # Should show proxy URLs
   ```

3. **Use private module settings if needed**:
   ```bash
   export GOPRIVATE=github.com/your-org/*
   export GONOPROXY=github.com/your-org/*
   ```

### Issue: Permission denied during installation

**Symptoms**:
```
permission denied: /usr/local/bin/memgos
```

**Solutions**:

1. **Install to user directory**:
   ```bash
   go install github.com/memtensor/memgos/cmd/memgos@latest
   # Installs to $(go env GOPATH)/bin
   ```

2. **Use sudo for system installation**:
   ```bash
   sudo make install
   ```

3. **Install to custom directory**:
   ```bash
   make install PREFIX=$HOME/.local
   export PATH=$PATH:$HOME/.local/bin
   ```

## Configuration Issues

### Issue: YAML syntax errors

**Symptoms**:
```
Error: yaml: line 15: found character that cannot start any token
```

**Solutions**:

1. **Validate YAML syntax**:
   ```bash
   # Using Python
   python -c "import yaml; yaml.safe_load(open('config.yaml'))"
   
   # Using yq
   yq eval . config.yaml
   
   # Using MemGOS
   memgos config validate --config config.yaml
   ```

2. **Common YAML issues**:
   ```yaml
   # Wrong - missing quotes for special characters
   password: my@password
   
   # Correct
   password: "my@password"
   
   # Wrong - inconsistent indentation
   chat_model:
   backend: openai
     model: gpt-3.5-turbo
   
   # Correct
   chat_model:
     backend: openai
     model: gpt-3.5-turbo
   ```

3. **Use validation tools**:
   ```bash
   # Install yamllint
   pip install yamllint
   yamllint config.yaml
   ```

### Issue: Environment variable substitution

**Symptoms**:
```
api_key: "${OPENAI_API_KEY}" appears as literal string
```

**Solutions**:

1. **Check environment variable**:
   ```bash
   echo $OPENAI_API_KEY
   env | grep OPENAI
   ```

2. **Verify substitution syntax**:
   ```yaml
   # Correct formats
   api_key: "${OPENAI_API_KEY}"
   api_key: "${OPENAI_API_KEY:-default-value}"
   
   # Wrong formats
   api_key: "$OPENAI_API_KEY"  # Shell-style, not supported
   api_key: "{OPENAI_API_KEY}"  # Missing $
   ```

3. **Test variable expansion**:
   ```bash
   memgos config show | grep api_key
   # Should show actual key value, not placeholder
   ```

### Issue: Missing required configuration

**Symptoms**:
```
Error: required field 'user_id' is not set
```

**Solutions**:

1. **Check required fields**:
   ```yaml
   # Minimum required configuration
   user_id: "your-user-id"          # Required
   session_id: "your-session-id"    # Required
   enable_textual_memory: true      # At least one memory type required
   ```

2. **Use config template**:
   ```bash
   memgos config template > config.yaml
   # Edit the generated template
   ```

3. **Validate configuration**:
   ```bash
   memgos config validate --config config.yaml --verbose
   ```

## Memory Operations Issues

### Issue: "Cube not found"

**Symptoms**:
```
Error: memory cube 'my-cube' not found
```

**Solutions**:

1. **List available cubes**:
   ```bash
   memgos list cubes
   ```

2. **Register the cube**:
   ```bash
   memgos register /path/to/cube my-cube
   ```

3. **Check cube directory structure**:
   ```
   cube-directory/
   ├── config.json          # Required
   ├── textual_memory.json  # For textual memories
   ├── activation_memory/   # For activation memories
   └── parametric_memory/   # For parametric memories
   ```

4. **Create cube if missing**:
   ```bash
   memgos create-cube my-cube "My Knowledge Base"
   ```

### Issue: Memory persistence problems

**Symptoms**:
- Memories don't persist after restart
- "Failed to save memory" errors

**Solutions**:

1. **Check file permissions**:
   ```bash
   ls -la /path/to/cube/
   # Ensure write permissions
   chmod -R 755 /path/to/cube/
   ```

2. **Check disk space**:
   ```bash
   df -h /path/to/cube/
   ```

3. **Verify JSON format**:
   ```bash
   jq . /path/to/cube/textual_memory.json
   # Should parse without errors
   ```

4. **Enable debug logging**:
   ```yaml
   log_level: "debug"
   ```

### Issue: Search returns no results

**Symptoms**:
- Search queries return empty results
- Known content not found

**Solutions**:

1. **Check cube contents**:
   ```bash
   memgos list memories my-cube
   ```

2. **Try broader search terms**:
   ```bash
   # Instead of very specific terms
   memgos search "exact phrase match"
   
   # Try broader terms
   memgos search "key concepts"
   ```

3. **Check memory cube registration**:
   ```bash
   memgos cube info my-cube
   ```

4. **Enable semantic search**:
   ```yaml
   mem_reader:
     use_semantic_search: true
     embedder:
       backend: "openai"
       model: "text-embedding-ada-002"
   ```

5. **Test with simple query**:
   ```bash
   memgos search "*"  # Find all memories
   ```

### Issue: Memory addition failures

**Symptoms**:
```
Error: failed to add memory: validation failed
```

**Solutions**:

1. **Check memory content**:
   ```bash
   # Ensure content is not empty
   memgos add "Valid content here"
   
   # Not: memgos add ""
   ```

2. **Verify cube permissions**:
   ```bash
   memgos access list --user your-user-id
   ```

3. **Check memory format**:
   ```json
   {
     "memory_content": "Your content",
     "cube_id": "valid-cube-id",
     "metadata": {
       "key": "value"
     }
   }
   ```

## API Issues

### Issue: API server won't start

**Symptoms**:
```
Error: failed to start API server: address already in use
```

**Solutions**:

1. **Check port availability**:
   ```bash
   netstat -tulpn | grep :8000
   lsof -i :8000
   ```

2. **Use different port**:
   ```yaml
   api:
     port: 8080  # Change from default 8000
   ```

3. **Kill existing process**:
   ```bash
   pkill memgos
   # Or find and kill specific PID
   kill $(lsof -t -i:8000)
   ```

### Issue: API authentication failures

**Symptoms**:
```
HTTP 401: Unauthorized
```

**Solutions**:

1. **Check authentication method**:
   ```bash
   # With user ID header
   curl -H "X-User-ID: your-user-id" \
        http://localhost:8000/api/v1/search?q=test
   ```

2. **Verify user exists**:
   ```bash
   memgos user info --user-id your-user-id
   ```

3. **Check API configuration**:
   ```yaml
   api:
     auth:
       enabled: true  # Check if auth is required
       jwt_secret: "your-secret"  # For JWT auth
   ```

### Issue: CORS errors in browser

**Symptoms**:
```
Access to fetch at 'http://localhost:8000' has been blocked by CORS policy
```

**Solutions**:

1. **Enable CORS in configuration**:
   ```yaml
   api:
     cors_enabled: true
     cors_origins:
       - "http://localhost:3000"
       - "https://your-domain.com"
   ```

2. **Check CORS headers**:
   ```bash
   curl -H "Origin: http://localhost:3000" \
        -H "Access-Control-Request-Method: POST" \
        -X OPTIONS \
        http://localhost:8000/api/v1/search
   ```

### Issue: Rate limiting problems

**Symptoms**:
```
HTTP 429: Rate limit exceeded
```

**Solutions**:

1. **Check rate limit configuration**:
   ```yaml
   api:
     rate_limit:
       enabled: true
       requests_per_minute: 100  # Increase if needed
       burst_size: 20
   ```

2. **Monitor rate limit headers**:
   ```bash
   curl -I http://localhost:8000/api/v1/search?q=test
   # Look for X-RateLimit-* headers
   ```

3. **Implement exponential backoff**:
   ```javascript
   async function retryWithBackoff(fn, maxRetries = 3) {
     for (let i = 0; i < maxRetries; i++) {
       try {
         return await fn();
       } catch (error) {
         if (error.status === 429) {
           await new Promise(resolve => 
             setTimeout(resolve, Math.pow(2, i) * 1000)
           );
         } else {
           throw error;
         }
       }
     }
   }
   ```

## Performance Problems

### Issue: Slow search performance

**Symptoms**:
- Search takes more than 5 seconds
- High CPU usage during search

**Solutions**:

1. **Enable performance metrics**:
   ```yaml
   metrics_enabled: true
   ```
   
   ```bash
   memgos metrics search-performance
   ```

2. **Optimize search parameters**:
   ```bash
   # Reduce top_k for faster results
   memgos search "query" --top-k 5  # instead of 50
   ```

3. **Enable vector database**:
   ```yaml
   vector_db:
     backend: "qdrant"
     host: "localhost"
     port: 6333
   ```

4. **Use semantic search caching**:
   ```yaml
   cache:
     backend: "redis"
     host: "localhost"
     port: 6379
     ttl: 3600  # Cache for 1 hour
   ```

5. **Optimize memory cube size**:
   ```bash
   # Split large cubes into smaller ones
   memgos cube split large-cube --size 1000
   ```

### Issue: High memory usage

**Symptoms**:
- MemGOS uses excessive RAM
- Out of memory errors

**Solutions**:

1. **Monitor memory usage**:
   ```bash
   memgos metrics memory-usage
   ps aux | grep memgos
   ```

2. **Configure memory limits**:
   ```yaml
   performance:
     memory_limit: "1GB"
     gc_percentage: 100  # More aggressive GC
   ```

3. **Enable memory pooling**:
   ```yaml
   memory_pool:
     enabled: true
     initial_size: "100MB"
     max_size: "500MB"
   ```

4. **Optimize cube loading**:
   ```yaml
   mem_reader:
     lazy_loading: true  # Load cubes on demand
   ```

### Issue: Slow startup time

**Symptoms**:
- MemGOS takes long time to start
- Timeout errors during initialization

**Solutions**:

1. **Enable lazy loading**:
   ```yaml
   lazy_initialization: true
   cube_loading:
     parallel: true
     batch_size: 10
   ```

2. **Optimize cube discovery**:
   ```bash
   # Pre-register cubes instead of auto-discovery
   memgos register /path/to/cube1 cube1
   memgos register /path/to/cube2 cube2
   ```

3. **Use cube caching**:
   ```yaml
   cube_cache:
     enabled: true
     cache_dir: "/tmp/memgos-cache"
   ```

## Database Connection Issues

### Issue: Vector database connection failures

**Symptoms**:
```
Error: failed to connect to Qdrant: connection refused
```

**Solutions**:

1. **Check Qdrant service**:
   ```bash
   # Check if Qdrant is running
   curl http://localhost:6333/health
   
   # Start Qdrant with Docker
   docker run -p 6333:6333 qdrant/qdrant:latest
   ```

2. **Verify configuration**:
   ```yaml
   vector_db:
     backend: "qdrant"
     host: "localhost"  # Check hostname
     port: 6333         # Check port
     timeout: 30        # Increase timeout
   ```

3. **Test connection manually**:
   ```bash
   telnet localhost 6333
   # Should connect successfully
   ```

4. **Check firewall/networking**:
   ```bash
   # On server hosting Qdrant
   sudo ufw status
   sudo iptables -L
   ```

### Issue: Graph database connection problems

**Symptoms**:
```
Error: Neo4j connection failed: authentication failed
```

**Solutions**:

1. **Check Neo4j credentials**:
   ```yaml
   graph_db:
     backend: "neo4j"
     uri: "bolt://localhost:7687"
     username: "neo4j"
     password: "${NEO4J_PASSWORD}"  # Check env var
   ```

2. **Test Neo4j connection**:
   ```bash
   # Using cypher-shell
   cypher-shell -a bolt://localhost:7687 -u neo4j -p password
   
   # Or check via HTTP
   curl -u neo4j:password http://localhost:7474/db/data/
   ```

3. **Check Neo4j service**:
   ```bash
   # Status
   sudo systemctl status neo4j
   
   # Logs
   sudo journalctl -u neo4j -f
   ```

4. **Verify Neo4j configuration**:
   ```bash
   # Check neo4j.conf
   grep "dbms.connector.bolt.listen_address" /etc/neo4j/neo4j.conf
   ```

### Issue: Redis connection failures

**Symptoms**:
```
Error: Redis connection failed: no route to host
```

**Solutions**:

1. **Check Redis service**:
   ```bash
   redis-cli ping
   # Should return PONG
   
   # Start Redis
   sudo systemctl start redis
   # Or with Docker
   docker run -p 6379:6379 redis:latest
   ```

2. **Verify Redis configuration**:
   ```yaml
   cache:
     backend: "redis"
     host: "localhost"
     port: 6379
     password: "${REDIS_PASSWORD}"  # If auth enabled
     database: 0
   ```

3. **Test Redis connectivity**:
   ```bash
   telnet localhost 6379
   # Or
   nc -zv localhost 6379
   ```

## Authentication and Authorization

### Issue: User not found

**Symptoms**:
```
Error: user 'user123' not found
```

**Solutions**:

1. **Create user**:
   ```bash
   memgos user create user123 --role user --email user@example.com
   ```

2. **List existing users**:
   ```bash
   memgos user list
   ```

3. **Check user configuration**:
   ```yaml
   users:
     enable_multi_user: true
     auto_create_users: true  # Auto-create missing users
   ```

### Issue: Permission denied

**Symptoms**:
```
Error: user 'user123' does not have permission to access cube 'private-cube'
```

**Solutions**:

1. **Check user permissions**:
   ```bash
   memgos access list --user user123
   ```

2. **Grant cube access**:
   ```bash
   memgos access grant user123 private-cube read,write
   ```

3. **Check access control configuration**:
   ```yaml
   access_control:
     enable: true
     default_permissions:
       - "read_own_cubes"
       - "write_own_cubes"
   ```

### Issue: JWT token problems

**Symptoms**:
```
Error: invalid JWT token
```

**Solutions**:

1. **Check JWT configuration**:
   ```yaml
   api:
     auth:
       enabled: true
       jwt_secret: "${JWT_SECRET}"  # Must be set
       jwt_expiry: "24h"
   ```

2. **Verify token format**:
   ```bash
   # JWT should have 3 parts separated by dots
   echo "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiOjE2MzQ2NDE2MzAsImV4cCI6MTY2NjE3NzYzMCwiYXVkIjoid3d3LmV4YW1wbGUuY29tIiwic3ViIjoianJvY2tldEBleGFtcGxlLmNvbSIsIkdpdmVuTmFtZSI6IkpvaG5ueSIsIlN1cm5hbWUiOiJSb2NrZXQiLCJFbWFpbCI6Impyb2NrZXRAZXhhbXBsZS5jb20iLCJSb2xlIjpbIk1hbmFnZXIiLCJQcm9qZWN0IEFkbWluaXN0cmF0b3IiXX0.lUt6CRMwBthp0S45kW7s2aUMawwGHwZwc0Zm4j4HE9M" | cut -d. -f1 | base64 -d
   ```

3. **Check token expiry**:
   ```bash
   # Decode JWT payload to check exp claim
   echo "token_payload" | base64 -d | jq .exp
   ```

## Logging and Debugging

### Enable Debug Logging

```yaml
# config.yaml
log_level: "debug"
metrics_enabled: true

# Detailed logging
logging:
  level: "debug"
  format: "json"  # or "text"
  output: "file"  # or "stdout"
  file: "/var/log/memgos/memgos.log"
  max_size: "100MB"
  max_backups: 10
  max_age: 30  # days
```

### Debug Command Line Options

```bash
# Debug mode
memgos --debug --config config.yaml

# Verbose output
memgos --verbose search "query"

# Trace mode (very detailed)
memgos --trace --config config.yaml

# Profile mode
memgos --profile --config config.yaml
```

### Log Analysis

```bash
# Follow logs in real-time
tail -f /var/log/memgos/memgos.log

# Search for errors
grep ERROR /var/log/memgos/memgos.log

# Filter by component
grep "component=search" /var/log/memgos/memgos.log

# JSON log parsing
jq '.level == "error"' /var/log/memgos/memgos.log
```

### Performance Profiling

```bash
# CPU profiling
memgos --profile-cpu --config config.yaml

# Memory profiling
memgos --profile-memory --config config.yaml

# Generate profile reports
go tool pprof cpu.prof
go tool pprof mem.prof
```

## Error Codes Reference

### Configuration Errors (1xxx)

| Code | Description | Solution |
|------|-------------|----------|
| 1001 | Config file not found | Specify correct config path |
| 1002 | Invalid YAML syntax | Fix YAML formatting |
| 1003 | Missing required field | Add required configuration |
| 1004 | Invalid environment variable | Check env var exists |

### Memory Operation Errors (2xxx)

| Code | Description | Solution |
|------|-------------|----------|
| 2001 | Cube not found | Register cube or check name |
| 2002 | Memory not found | Verify memory ID and cube |
| 2003 | Invalid memory format | Check memory content format |
| 2004 | Persistence failure | Check file permissions |

### API Errors (3xxx)

| Code | Description | Solution |
|------|-------------|----------|
| 3001 | Authentication failed | Check user ID or token |
| 3002 | Authorization failed | Verify user permissions |
| 3003 | Rate limit exceeded | Reduce request frequency |
| 3004 | Invalid request format | Check API request format |

### Database Errors (4xxx)

| Code | Description | Solution |
|------|-------------|----------|
| 4001 | Vector DB connection failed | Check Qdrant service |
| 4002 | Graph DB connection failed | Check Neo4j service |
| 4003 | Cache connection failed | Check Redis service |
| 4004 | Database query failed | Check query syntax |

### System Errors (5xxx)

| Code | Description | Solution |
|------|-------------|----------|
| 5001 | Out of memory | Increase memory limit |
| 5002 | Disk space full | Free up disk space |
| 5003 | Network timeout | Check network connectivity |
| 5004 | Internal error | Check logs and file issue |

### Getting More Help

1. **Check documentation**: [MemGOS Docs](https://memgos.dev)
2. **Search existing issues**: [GitHub Issues](https://github.com/memtensor/memgos/issues)
3. **Ask the community**: [Discussions](https://github.com/memtensor/memgos/discussions)
4. **Report bugs**: [File an Issue](https://github.com/memtensor/memgos/issues/new)

### Diagnostic Information to Include

When reporting issues, include:

```bash
# System information
memgos version
go version
uname -a

# Configuration (sanitized)
memgos config show --sanitize

# Logs (relevant excerpts)
tail -100 /var/log/memgos/memgos.log

# System resources
free -h
df -h
ps aux | grep memgos
```

This troubleshooting guide should help resolve most common issues with MemGOS. For complex problems, don't hesitate to reach out to the community or file detailed bug reports.
