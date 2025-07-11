# MemGOS Docker Setup

This document explains how to run MemGOS using Docker containers.

## 🐳 Docker Images

MemGOS provides two Docker images:

1. **Main Application** (`memgos`): API server with full functionality
2. **MCP Server** (`memgos-mcp`): Model Context Protocol server for AI tool integration

## 🚀 Quick Start

### Using Docker Compose (Recommended)

1. **Create environment file**:
   ```bash
   echo "MEMGOS_API_TOKEN=your-api-token-here" > .env
   ```

2. **Start all services**:
   ```bash
   docker-compose up -d
   ```

3. **View logs**:
   ```bash
   docker-compose logs -f
   ```

4. **Stop services**:
   ```bash
   docker-compose down
   ```

### Using Docker Commands

#### 1. Build Images

```bash
# Build main application
docker build -t memgos:latest .

# Build MCP server
docker build -t memgos-mcp:latest -f Dockerfile.mcp .
```

#### 2. Run Main Application

```bash
# Run API server
docker run -d \
  --name memgos-api \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  memgos:latest

# Check health
curl http://localhost:8080/health
```

#### 3. Run MCP Server

```bash
# Run MCP server (requires API server to be running)
docker run -d \
  --name memgos-mcp \
  --link memgos-api:memgos \
  -e MEMGOS_API_TOKEN=your-api-token \
  -e MEMGOS_API_URL=http://memgos:8080 \
  memgos-mcp:latest
```

## 📋 Configuration

### Environment Variables

#### Main Application (`memgos`)
- `MEMGOS_LOG_LEVEL`: Log level (debug, info, warn, error)
- `MEMGOS_API_PORT`: API server port (default: 8080)
- `MEMGOS_USER_ID`: Default user ID
- `MEMGOS_SESSION_ID`: Default session ID

#### MCP Server (`memgos-mcp`)
- `MEMGOS_API_TOKEN`: **Required** - API token for authentication
- `MEMGOS_API_URL`: MemGOS API base URL (default: http://localhost:8080)
- `MEMGOS_USER_ID`: User ID for MCP session (default: default-user)
- `MEMGOS_SESSION_ID`: Session ID for MCP server (default: mcp-session)

### Volume Mounts

- `/app/data`: Persistent data storage
- `/app/logs`: Application logs
- `/app/config`: Configuration files

## 🔧 Advanced Configuration

### Custom Configuration File

```bash
# Mount custom config
docker run -d \
  --name memgos-api \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config/config.yaml \
  memgos:latest ./memgos --api --config config/config.yaml
```

### Production Deployment

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  memgos:
    image: memgos:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - memgos_data:/app/data
      - memgos_logs:/app/logs
    environment:
      - MEMGOS_LOG_LEVEL=warn
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  memgos-mcp:
    image: memgos-mcp:latest
    restart: unless-stopped
    depends_on:
      memgos:
        condition: service_healthy
    environment:
      - MEMGOS_API_TOKEN=${MEMGOS_API_TOKEN}
      - MEMGOS_API_URL=http://memgos:8080

volumes:
  memgos_data:
  memgos_logs:
```

## 🔗 Service Integration

### With Redis (Caching)

```yaml
services:
  memgos:
    # ... main config
    environment:
      - REDIS_URL=redis://redis:6379

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
```

### With Qdrant (Vector Database)

```yaml
services:
  memgos:
    # ... main config
    environment:
      - QDRANT_URL=http://qdrant:6333

  qdrant:
    image: qdrant/qdrant:v1.7.4
    ports:
      - "6333:6333"
    volumes:
      - qdrant_data:/qdrant/storage
```

## 🧪 Development

### Development with Live Reload

```bash
# Mount source code for development
docker run -it \
  --name memgos-dev \
  -p 8080:8080 \
  -v $(pwd):/app \
  -w /app \
  golang:1.24-alpine \
  go run cmd/memgos/main.go --api --config examples/config/api_config.yaml
```

### Building for Multiple Architectures

```bash
# Enable buildx
docker buildx create --name multiarch --driver docker-container --use

# Build for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t memgos:latest \
  --push .
```

## 📊 Monitoring

### Health Checks

```bash
# Check API health
curl http://localhost:8080/health

# Check container health
docker ps --filter "name=memgos"
```

### Logs

```bash
# View logs
docker logs memgos-api
docker logs memgos-mcp

# Follow logs
docker logs -f memgos-api
```

### Metrics

```bash
# If metrics are enabled
curl http://localhost:8080/metrics
```

## 🛠️ Troubleshooting

### Common Issues

1. **MCP Server can't connect to API**:
   ```bash
   # Check if API is running
   docker ps
   curl http://localhost:8080/health
   
   # Check MCP server logs
   docker logs memgos-mcp
   ```

2. **Permission denied errors**:
   ```bash
   # Fix volume permissions
   sudo chown -R 1000:1000 ./data ./logs
   ```

3. **Port conflicts**:
   ```bash
   # Use different ports
   docker run -p 8081:8080 memgos:latest
   ```

### Debug Mode

```bash
# Run with debug logging
docker run -it \
  --name memgos-debug \
  -p 8080:8080 \
  -e MEMGOS_LOG_LEVEL=debug \
  memgos:latest
```

## 🔐 Security

### Best Practices

1. **Use secrets management**:
   ```bash
   # Docker secrets
   echo "your-api-token" | docker secret create memgos-api-token -
   ```

2. **Run as non-root user** (already configured in Dockerfiles)

3. **Use specific image tags** in production:
   ```yaml
   image: memgos:v1.0.0  # instead of :latest
   ```

4. **Limit container resources**:
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '0.50'
         memory: 512M
   ```

## 📚 Examples

### Complete Stack with AI Services

```yaml
# docker-compose.ai.yml
version: '3.8'

services:
  memgos:
    image: memgos:latest
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    depends_on:
      - redis
      - qdrant

  memgos-mcp:
    image: memgos-mcp:latest
    depends_on:
      memgos:
        condition: service_healthy
    environment:
      - MEMGOS_API_TOKEN=${MEMGOS_API_TOKEN}
      - MEMGOS_API_URL=http://memgos:8080

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  qdrant:
    image: qdrant/qdrant:v1.7.4
    volumes:
      - qdrant_data:/qdrant/storage

volumes:
  redis_data:
  qdrant_data:
```

### Load Balancing with Multiple Instances

```yaml
# docker-compose.scale.yml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - memgos

  memgos:
    image: memgos:latest
    deploy:
      replicas: 3
    environment:
      - MEMGOS_LOG_LEVEL=info
```

## 🚀 Next Steps

1. Check the [API documentation](../api/README.md) for available endpoints
2. See [MCP integration guide](../mcp/README.md) for AI tool setup
3. Review [configuration options](../user-guide/README.md) for customization

---

For more information, visit the [MemGOS documentation](../README.md) or check the [troubleshooting guide](../user-guide/troubleshooting.md).