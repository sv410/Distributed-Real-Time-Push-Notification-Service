.PHONY: help build clean test run-api run-consumer docker-build docker-up docker-down deps up down logs

# Default target
help:
	@echo "Available commands:"
	@echo "  build         Build all binaries (api-gateway and consumer)"
	@echo "  run-api       Run the API Gateway service"
	@echo "  run-consumer  Run the Consumer service"
	@echo "  test          Run all tests"
	@echo "  clean         Clean build artifacts"
	@echo "  deps          Start dependencies (Kafka, Redis) with Docker Compose"
	@echo "  up            Start all services including dependencies"
	@echo "  down          Stop all services"
	@echo "  logs          Show service logs"
	@echo "  docker-build  Build Docker images"
	@echo "  urls          Show service URLs"

# Build all binaries
build:
	@echo "Building notification service..."
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api-gateway ./cmd/api-gateway
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/consumer ./cmd/consumer
	@echo "Build completed successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean
	@echo "Clean completed!"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run API Gateway
run-api: build
	@echo "Starting API Gateway..."
	@./bin/api-gateway

# Run Consumer
run-consumer: build
	@echo "Starting Consumer..."
	@./bin/consumer

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	@docker build -f Dockerfile.api-gateway -t notification-api-gateway .
	@docker build -f Dockerfile.consumer -t notification-consumer .

# Start dependencies only (Kafka, Redis, UIs)
deps:
	@echo "Starting dependencies..."
	@docker-compose up -d zookeeper kafka redis kafka-ui redis-commander
	@echo "Waiting for services to be ready..."
	@sleep 30
	@echo "Dependencies started. Check health with 'make logs'"

# Start all services with Docker Compose
up:
	@echo "Starting all services..."
	@docker-compose up -d

# Stop all services
down:
	@echo "Stopping all services..."
	@docker-compose down

# Show logs
logs:
	@docker-compose logs -f --tail=50

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Run linter (if golangci-lint is installed)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running linter..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

# Full check (format, vet, test)
check: fmt vet test
	@echo "All checks passed"

# Show service URLs
urls:
	@echo "Service URLs:"
	@echo "  API Gateway:          http://localhost:8080"
	@echo "  Health Check:         http://localhost:8080/health"
	@echo "  Kafka UI:             http://localhost:8081"
	@echo "  Redis Commander:      http://localhost:8082"

# Development workflow - start deps and run services
dev: deps
	@echo "Development setup complete. You can now run:"
	@echo "  make run-api     (in one terminal)"
	@echo "  make run-consumer (in another terminal)"

# Check if dependencies are running
check-deps:
	@echo "Checking dependency health..."
	@docker-compose ps
