.PHONY: help build run test clean deps up down logs docker-build

# Default target
help:
	@echo "Available commands:"
	@echo "  build        Build the notification service binary"
	@echo "  run          Run the service locally"
	@echo "  test         Run all tests"
	@echo "  clean        Clean build artifacts"
	@echo "  deps         Start dependencies (Kafka, Redis) with Docker Compose"
	@echo "  up           Start all services including dependencies"
	@echo "  down         Stop all services"
	@echo "  logs         Show service logs"
	@echo "  docker-build Build Docker image"
	@echo "  test-demo    Run the test demonstration script"

# Build the service binary
build:
	@echo "Building notification service..."
	@go build -o bin/notification-service ./cmd
	@echo "Build complete: bin/notification-service"

# Run the service locally
run: build
	@echo "Starting notification service..."
	@./bin/notification-service

# Run all tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Start dependencies only (Kafka, Redis, UIs)
deps:
	@echo "Starting dependencies..."
	@docker-compose up -d zookeeper kafka redis kafka-ui redis-commander
	@echo "Waiting for services to be ready..."
	@sleep 30
	@echo "Dependencies started. Check health with 'make logs'"

# Start all services
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

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t notification-service:latest .
	@echo "Docker image built: notification-service:latest"

# Run demonstration test
test-demo: 
	@echo "Running test demonstration..."
	@./test.sh

# Development workflow - start deps and run service
dev: deps
	@echo "Development setup complete. Starting service..."
	@sleep 5
	@$(MAKE) run

# Check if dependencies are running
check-deps:
	@echo "Checking dependency health..."
	@docker-compose ps
	@echo "Kafka topics:"
	@docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list || true
	@echo "Redis ping:"
	@docker-compose exec redis redis-cli ping || true

# Production build
prod-build:
	@echo "Building for production..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/notification-service ./cmd
	@echo "Production build complete"

# Install development dependencies
install-deps:
	@echo "Installing development dependencies..."
	@go mod tidy
	@go mod download
	@echo "Dependencies installed"

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
	@echo "  Notification Service: http://localhost:8080"
	@echo "  Health Check:         http://localhost:8080/health"
	@echo "  Metrics:              http://localhost:8080/metrics"
	@echo "  Kafka UI:             http://localhost:9080"
	@echo "  Redis Commander:      http://localhost:8081"