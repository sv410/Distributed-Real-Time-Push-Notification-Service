.PHONY: build clean test run-api run-consumer docker-build docker-up docker-down

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

# Start all services with Docker Compose
docker-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d

# Stop all services
docker-down:
	@echo "Stopping services..."
	@docker-compose down

# Show logs
logs:
	@docker-compose logs -f

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@golangci-lint run