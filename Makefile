# MemGOS Makefile

.PHONY: all build test clean install deps fmt lint vet check coverage docker help

# Build variables
BINARY_NAME=memgos
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Directories
BUILD_DIR=build
DIST_DIR=dist
COVERAGE_DIR=coverage

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Default target
all: deps fmt vet lint test build

# Build the binary
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	${GOBUILD} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/memgos

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@mkdir -p ${DIST_DIR}
	GOOS=linux GOARCH=amd64 ${GOBUILD} ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-linux-amd64 ./cmd/memgos

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p ${DIST_DIR}
	GOOS=darwin GOARCH=amd64 ${GOBUILD} ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-darwin-amd64 ./cmd/memgos
	GOOS=darwin GOARCH=arm64 ${GOBUILD} ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-darwin-arm64 ./cmd/memgos

build-windows:
	@echo "Building for Windows..."
	@mkdir -p ${DIST_DIR}
	GOOS=windows GOARCH=amd64 ${GOBUILD} ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-windows-amd64.exe ./cmd/memgos

# Install dependencies
deps:
	@echo "Installing dependencies..."
	${GOMOD} download
	${GOMOD} tidy

# Run tests
test:
	@echo "Running tests..."
	${GOTEST} -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p ${COVERAGE_DIR}
	${GOTEST} -v -coverprofile=${COVERAGE_DIR}/coverage.out ./...
	${GOCMD} tool cover -html=${COVERAGE_DIR}/coverage.out -o ${COVERAGE_DIR}/coverage.html
	@echo "Coverage report generated: ${COVERAGE_DIR}/coverage.html"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	${GOTEST} -v -race ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	${GOTEST} -v -bench=. -benchmem ./...

# Format code
fmt:
	@echo "Formatting code..."
	${GOFMT} -s -w .

# Lint code
lint:
	@echo "Linting code..."
	${GOLINT} run

# Vet code
vet:
	@echo "Vetting code..."
	${GOCMD} vet ./...

# Check everything
check: fmt vet lint test

# Clean build artifacts
clean:
	@echo "Cleaning..."
	${GOCLEAN}
	rm -rf ${BUILD_DIR}
	rm -rf ${DIST_DIR}
	rm -rf ${COVERAGE_DIR}

# Install the binary
install: build
	@echo "Installing ${BINARY_NAME}..."
	${GOCMD} install ${LDFLAGS} ./cmd/memgos

# Run the application
run: build
	./${BUILD_DIR}/${BINARY_NAME}

# Run with example configuration
run-example: build
	./${BUILD_DIR}/${BINARY_NAME} --config examples/config/simple_config.yaml

# Run API server mode
run-api: build
	./${BUILD_DIR}/${BINARY_NAME} --api --config examples/config/api_config.yaml

# Run interactive mode
run-interactive: build
	./${BUILD_DIR}/${BINARY_NAME} --interactive

# Generate mocks for testing
generate-mocks:
	@echo "Generating mocks..."
	${GOCMD} generate ./...

# Initialize a new MemGOS project
init-project:
	@echo "Initializing MemGOS project..."
	mkdir -p config
	mkdir -p data/cubes
	mkdir -p logs
	cp examples/config/simple_config.yaml config/memgos.yaml
	@echo "Project initialized. Edit config/memgos.yaml to get started."

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t memgos:${VERSION} .

docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it memgos:${VERSION}

# Development targets
dev-setup: deps generate-mocks
	@echo "Development environment setup complete"

dev-test: fmt vet test-race test-coverage
	@echo "Development testing complete"

# Release targets
release: clean deps check test build-all
	@echo "Release build complete"

# Tools installation
install-tools:
	@echo "Installing development tools..."
	${GOGET} github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	${GOGET} github.com/golang/mock/mockgen@latest

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	${GOMOD} get -u ./...
	${GOMOD} tidy

# Security audit
security:
	@echo "Running security audit..."
	${GOCMD} list -json -m all | nancy sleuth

# Performance profiling
profile-cpu:
	@echo "Running CPU profiling..."
	${GOTEST} -cpuprofile=cpu.prof -bench=. ./...

profile-mem:
	@echo "Running memory profiling..."
	${GOTEST} -memprofile=mem.prof -bench=. ./...

# Documentation generation
docs:
	@echo "Generating documentation..."
	${GOCMD} doc -all ./... > docs/api/godoc.txt

# Help target
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Build for all platforms"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  test-race      - Run tests with race detection"
	@echo "  bench          - Run benchmarks"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  vet            - Vet code"
	@echo "  check          - Run all checks"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install the binary"
	@echo "  run            - Run the application"
	@echo "  run-api        - Run in API server mode"
	@echo "  run-interactive - Run in interactive mode"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  dev-setup      - Setup development environment"
	@echo "  release        - Build release version"
	@echo "  install-tools  - Install development tools"
	@echo "  security       - Run security audit"
	@echo "  docs           - Generate documentation"
	@echo "  help           - Show this help"