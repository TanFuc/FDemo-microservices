.PHONY: build run test clean docker-up docker-down migrate

# Build the application
build:
	go build -o bin/order-service ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Start dependencies with Docker
docker-up:
	docker-compose up -d

# Stop dependencies
docker-down:
	docker-compose down

# Download dependencies
deps:
	go mod download
	go mod tidy

# Generate protobuf files (requires protoc and plugins)
proto:
	protoc --go_out=. --go-grpc_out=. proto/inventory/inventory.proto

# Lint code
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
