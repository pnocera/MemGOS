# Multi-stage build for MemGOS main application
FROM golang:1.24-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags '-w -s -extldflags "-static"' \
    -o memgos ./cmd/memgos

# Final stage
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 memgos && \
    adduser -D -u 1000 -G memgos memgos

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/memgos .

# Copy example configuration files
COPY --from=builder /app/examples/config ./examples/config

# Create necessary directories
RUN mkdir -p /app/data/cubes /app/logs /app/config && \
    chown -R memgos:memgos /app

# Switch to non-root user
USER memgos

# Expose default API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command (API server mode)
CMD ["./memgos", "--api", "--config", "examples/config/api_config.yaml"]