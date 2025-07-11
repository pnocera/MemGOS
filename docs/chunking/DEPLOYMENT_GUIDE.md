# MemGOS Chunking System - Production Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying the MemGOS Enhanced Chunking System in production environments. The system is designed for high availability, scalability, and operational excellence.

## Prerequisites

### System Requirements

- **Go**: Version 1.21 or higher
- **Memory**: Minimum 2GB RAM, recommended 4GB+
- **CPU**: Minimum 2 cores, recommended 4+ cores
- **Storage**: 10GB available space for logs and cache
- **Network**: Stable internet connection for external dependencies

### Dependencies

- **Docker**: For containerized deployment
- **Kubernetes**: For orchestrated deployment (optional)
- **Prometheus**: For metrics collection (optional)
- **Grafana**: For monitoring dashboards (optional)

## Installation Methods

### 1. Direct Go Installation

```bash
# Clone the repository
git clone https://github.com/memtensor/memgos.git
cd memgos

# Build the chunking service
go build -o chunking-service ./cmd/chunking

# Run with default configuration
./chunking-service
```

### 2. Docker Deployment

#### Build Docker Image

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chunking-service ./cmd/chunking

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

COPY --from=builder /app/chunking-service .
COPY config/ ./config/

EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./chunking-service"]
```

#### Build and Run

```bash
# Build the image
docker build -t memgos/chunking-service:latest .

# Run the container
docker run -d \
    --name chunking-service \
    -p 8080:8080 \
    -e MEMGOS_CHUNKING_MAX_CONCURRENCY=8 \
    -e MEMGOS_CACHE_ENABLED=true \
    -v $(pwd)/config:/root/config \
    memgos/chunking-service:latest
```

### 3. Docker Compose Deployment

```yaml
# docker-compose.yml
version: '3.8'

services:
  chunking-service:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MEMGOS_CHUNKING_MAX_CONCURRENCY=8
      - MEMGOS_CACHE_ENABLED=true
      - MEMGOS_MONITORING_ENABLED=true
    volumes:
      - ./config:/root/config
      - ./logs:/root/logs
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - ./monitoring/grafana/dashboards:/var/lib/grafana/dashboards
      - ./monitoring/grafana/provisioning:/etc/grafana/provisioning
```

### 4. Kubernetes Deployment

#### Namespace and ConfigMap

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: memgos-chunking

---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: chunking-config
  namespace: memgos-chunking
data:
  pipeline.yaml: |
    pipeline:
      max_concurrency: 8
      timeout: "30s"
      enable_parallel: true
      enable_fallback: true
      fallback_threshold: 0.5
      quality_threshold: 0.7
  
  production.yaml: |
    production:
      enable_caching: true
      cache_size: 10000
      cache_ttl: "1h"
      max_memory_usage: 2147483648
      max_concurrent_requests: 100
      rate_limit_rps: 100
  
  monitoring.yaml: |
    monitoring:
      sample_interval: "10s"
      enable_alerts: true
      alert_thresholds:
        memory_usage: 0.9
        error_rate: 0.05
        response_time: 5.0
```

#### Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chunking-service
  namespace: memgos-chunking
  labels:
    app: chunking-service
    version: v4.0.0
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chunking-service
  template:
    metadata:
      labels:
        app: chunking-service
        version: v4.0.0
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: chunking-service
        image: memgos/chunking-service:v4.0.0
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: MEMGOS_CHUNKING_MAX_CONCURRENCY
          value: "8"
        - name: MEMGOS_CACHE_ENABLED
          value: "true"
        - name: MEMGOS_MONITORING_ENABLED
          value: "true"
        - name: CONFIG_PATH
          value: "/config"
        volumeMounts:
        - name: config-volume
          mountPath: /config
        resources:
          limits:
            memory: "2Gi"
            cpu: "1000m"
          requests:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
      volumes:
      - name: config-volume
        configMap:
          name: chunking-config
```

#### Service and Ingress

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: chunking-service
  namespace: memgos-chunking
  labels:
    app: chunking-service
spec:
  selector:
    app: chunking-service
  ports:
  - port: 80
    targetPort: 8080
    name: http
  type: ClusterIP

---
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: chunking-ingress
  namespace: memgos-chunking
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-burst: "200"
spec:
  rules:
  - host: chunking.memgos.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: chunking-service
            port:
              number: 80
```

#### Horizontal Pod Autoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: chunking-hpa
  namespace: memgos-chunking
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: chunking-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Configuration

### Environment Variables

```bash
# Core configuration
MEMGOS_CHUNKING_MAX_CONCURRENCY=8
MEMGOS_CHUNKING_TIMEOUT=30s
MEMGOS_CHUNKING_ENABLE_PARALLEL=true
MEMGOS_CHUNKING_ENABLE_FALLBACK=true

# Pipeline configuration
MEMGOS_PIPELINE_FALLBACK_THRESHOLD=0.5
MEMGOS_PIPELINE_QUALITY_THRESHOLD=0.7
MEMGOS_PIPELINE_BATCH_SIZE=10

# Cache configuration
MEMGOS_CACHE_ENABLED=true
MEMGOS_CACHE_SIZE=10000
MEMGOS_CACHE_TTL=1h
MEMGOS_CACHE_EVICTION_POLICY=lru

# Production optimization
MEMGOS_PRODUCTION_MAX_MEMORY=2147483648
MEMGOS_PRODUCTION_MAX_REQUESTS=100
MEMGOS_PRODUCTION_RATE_LIMIT_RPS=100
MEMGOS_PRODUCTION_RATE_LIMIT_BURST=200

# Circuit breaker
MEMGOS_CIRCUIT_FAILURE_THRESHOLD=5
MEMGOS_CIRCUIT_RECOVERY_TIMEOUT=30s

# Monitoring
MEMGOS_MONITORING_ENABLED=true
MEMGOS_METRICS_INTERVAL=10s
MEMGOS_ALERTS_ENABLED=true
MEMGOS_HEALTH_CHECK_INTERVAL=30s

# Logging
MEMGOS_LOG_LEVEL=info
MEMGOS_LOG_FORMAT=json
MEMGOS_LOG_FILE=/var/log/memgos/chunking.log
```

### Configuration Files

#### pipeline.yaml

```yaml
pipeline:
  # Core settings
  max_concurrency: 8
  timeout: "30s"
  retry_attempts: 3
  retry_delay: "1s"
  
  # Processing modes
  enable_parallel: true
  enable_fallback: true
  fallback_threshold: 0.5
  quality_threshold: 0.7
  
  # Batch processing
  batch_size: 10
  buffer_size: 100
  
  # Metrics
  enable_metrics: true
  metrics_interval: "10s"
  health_check_interval: "30s"
```

#### production.yaml

```yaml
production:
  # Caching
  enable_caching: true
  cache_size: 10000
  cache_ttl: "1h"
  cache_eviction_policy: "lru"
  compression_enabled: true
  
  # Resource management
  max_memory_usage: 2147483648  # 2GB
  max_concurrent_requests: 100
  gc_threshold: 0.8
  
  # Circuit breaker
  failure_threshold: 5
  recovery_timeout: "30s"
  
  # Rate limiting
  requests_per_second: 100
  burst_size: 200
  
  # Performance tuning
  enable_parallelization: true
  worker_pool_size: 8
```

#### monitoring.yaml

```yaml
monitoring:
  # Sampling
  sample_interval: "10s"
  alert_buffer: 100
  enable_alerts: true
  enable_profiling: false
  
  # Alert thresholds
  alert_thresholds:
    memory_usage: 0.9
    error_rate: 0.05
    response_time: 5.0
    cpu_usage: 0.8
    goroutine_count: 1000
  
  # Storage
  enable_persistence: true
  retention_period: "24h"
  cleanup_interval: "1h"
```

## Monitoring and Observability

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "chunking_rules.yml"

scrape_configs:
  - job_name: 'chunking-service'
    static_configs:
      - targets: ['chunking-service:8080']
    metrics_path: /metrics
    scrape_interval: 10s

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

### Alert Rules

```yaml
# chunking_rules.yml
groups:
  - name: chunking_alerts
    rules:
      - alert: HighMemoryUsage
        expr: memgos_memory_usage_bytes / memgos_memory_limit_bytes > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is above 90% for more than 5 minutes"
      
      - alert: HighErrorRate
        expr: rate(memgos_requests_failed_total[5m]) / rate(memgos_requests_total[5m]) > 0.05
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is above 5% for more than 2 minutes"
      
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(memgos_request_duration_seconds_bucket[5m])) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "95th percentile latency is above 100ms for more than 5 minutes"
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "MemGOS Chunking Service",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(memgos_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(memgos_requests_failed_total[5m]) / rate(memgos_requests_total[5m])",
            "legendFormat": "Error Rate"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(memgos_request_duration_seconds_bucket[5m]))",
            "legendFormat": "p50"
          },
          {
            "expr": "histogram_quantile(0.95, rate(memgos_request_duration_seconds_bucket[5m]))",
            "legendFormat": "p95"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "memgos_memory_usage_bytes",
            "legendFormat": "Memory Usage"
          }
        ]
      }
    ]
  }
}
```

## Security Configuration

### HTTPS/TLS Setup

```yaml
# tls-config.yaml
tls:
  enabled: true
  cert_file: "/etc/ssl/certs/chunking-service.crt"
  key_file: "/etc/ssl/private/chunking-service.key"
  min_version: "1.2"
  cipher_suites:
    - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"
    - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
```

### Authentication

```yaml
# auth-config.yaml
authentication:
  enabled: true
  type: "jwt"
  jwt:
    secret_key: "${JWT_SECRET_KEY}"
    expiration: "1h"
    issuer: "memgos-chunking"
    
  # Alternative: API Key authentication
  api_key:
    enabled: false
    header_name: "X-API-Key"
    keys:
      - name: "service-a"
        key: "${API_KEY_SERVICE_A}"
      - name: "service-b"
        key: "${API_KEY_SERVICE_B}"
```

### Network Policies (Kubernetes)

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: chunking-network-policy
  namespace: memgos-chunking
spec:
  podSelector:
    matchLabels:
      app: chunking-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          role: api-gateway
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 53  # DNS
    - protocol: UDP
      port: 53  # DNS
  - to:
    - podSelector:
        matchLabels:
          app: prometheus
    ports:
    - protocol: TCP
      port: 9090
```

## Performance Tuning

### Resource Allocation

```yaml
# Production resource recommendations
resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "1000m"

# For high-throughput workloads
resources:
  requests:
    memory: "2Gi"
    cpu: "1000m"
  limits:
    memory: "4Gi"
    cpu: "2000m"
```

### JVM Tuning (if using Go with CGO)

```bash
# Go runtime tuning
export GOGC=100              # GC percentage
export GOMAXPROCS=8          # Max OS threads
export GOMEMLIMIT=2GiB       # Memory limit
export GODEBUG=gctrace=1     # GC tracing
```

### OS-Level Tuning

```bash
# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Tune network settings
echo "net.core.somaxconn = 65535" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65535" >> /etc/sysctl.conf
echo "net.core.netdev_max_backlog = 65535" >> /etc/sysctl.conf

# Apply changes
sysctl -p
```

## Backup and Recovery

### Configuration Backup

```bash
#!/bin/bash
# backup-config.sh

BACKUP_DIR="/backups/memgos-chunking"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p "${BACKUP_DIR}/${DATE}"

# Backup configuration files
kubectl get configmap chunking-config -n memgos-chunking -o yaml > "${BACKUP_DIR}/${DATE}/configmap.yaml"
kubectl get deployment chunking-service -n memgos-chunking -o yaml > "${BACKUP_DIR}/${DATE}/deployment.yaml"
kubectl get service chunking-service -n memgos-chunking -o yaml > "${BACKUP_DIR}/${DATE}/service.yaml"

# Backup secrets
kubectl get secrets -n memgos-chunking -o yaml > "${BACKUP_DIR}/${DATE}/secrets.yaml"

echo "Backup completed: ${BACKUP_DIR}/${DATE}"
```

### Disaster Recovery

```bash
#!/bin/bash
# restore-config.sh

BACKUP_DATE=$1
BACKUP_DIR="/backups/memgos-chunking"

if [ -z "$BACKUP_DATE" ]; then
    echo "Usage: $0 <backup_date>"
    exit 1
fi

# Restore configuration
kubectl apply -f "${BACKUP_DIR}/${BACKUP_DATE}/configmap.yaml"
kubectl apply -f "${BACKUP_DIR}/${BACKUP_DATE}/deployment.yaml"
kubectl apply -f "${BACKUP_DIR}/${BACKUP_DATE}/service.yaml"
kubectl apply -f "${BACKUP_DIR}/${BACKUP_DATE}/secrets.yaml"

echo "Restore completed from: ${BACKUP_DIR}/${BACKUP_DATE}"
```

## Troubleshooting

### Common Issues

#### High Memory Usage

```bash
# Check memory usage
kubectl top pods -n memgos-chunking

# Check for memory leaks
kubectl exec -it chunking-service-xxx -n memgos-chunking -- curl http://localhost:8080/debug/pprof/heap

# Adjust memory limits
kubectl patch deployment chunking-service -n memgos-chunking -p '{"spec":{"template":{"spec":{"containers":[{"name":"chunking-service","resources":{"limits":{"memory":"4Gi"}}}]}}}}'
```

#### High Latency

```bash
# Check CPU usage
kubectl top pods -n memgos-chunking

# Increase CPU limits
kubectl patch deployment chunking-service -n memgos-chunking -p '{"spec":{"template":{"spec":{"containers":[{"name":"chunking-service","resources":{"limits":{"cpu":"2000m"}}}]}}}}'

# Scale horizontally
kubectl scale deployment chunking-service --replicas=5 -n memgos-chunking
```

#### Connection Timeouts

```bash
# Check service endpoints
kubectl get endpoints chunking-service -n memgos-chunking

# Check pod readiness
kubectl get pods -n memgos-chunking -o wide

# Restart unhealthy pods
kubectl delete pod chunking-service-xxx -n memgos-chunking
```

### Log Analysis

```bash
# View service logs
kubectl logs -f deployment/chunking-service -n memgos-chunking

# Search for errors
kubectl logs deployment/chunking-service -n memgos-chunking | grep ERROR

# Check specific time range
kubectl logs --since=1h deployment/chunking-service -n memgos-chunking
```

## Maintenance

### Regular Maintenance Tasks

```bash
#!/bin/bash
# maintenance.sh

# Update configurations
kubectl apply -f config/

# Rolling update
kubectl rollout restart deployment/chunking-service -n memgos-chunking

# Check rollout status
kubectl rollout status deployment/chunking-service -n memgos-chunking

# Clean up old pods
kubectl delete pods --field-selector=status.phase=Succeeded -n memgos-chunking
```

### Health Checks

```bash
#!/bin/bash
# health-check.sh

# Check service health
curl -f http://chunking-service:8080/health

# Check metrics endpoint
curl http://chunking-service:8080/metrics

# Check readiness
curl -f http://chunking-service:8080/ready
```

## Support and Documentation

### Health Endpoints

- `GET /health` - Overall service health
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics
- `GET /debug/pprof/` - Go profiling endpoints

### API Documentation

- `GET /docs` - Swagger UI
- `GET /api/v1/openapi.json` - OpenAPI specification

### Support Channels

- **GitHub Issues**: https://github.com/memtensor/memgos/issues
- **Documentation**: https://memgos.io/docs/chunking
- **Community**: https://discord.gg/memgos

---

This deployment guide provides comprehensive instructions for production deployment of the MemGOS Enhanced Chunking System. For additional support or questions, please refer to the official documentation or contact the support team.