#!/bin/bash

# Build script for the notification service

set -e

echo "Building Notification Service..."

# Create bin directory
mkdir -p bin

# Build API Gateway
echo "Building API Gateway..."
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api-gateway ./cmd/api-gateway

# Build Consumer
echo "Building Consumer..."
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/consumer ./cmd/consumer

echo "Build completed successfully!"
echo "Binaries available in ./bin/"
echo "  - api-gateway"
echo "  - consumer"