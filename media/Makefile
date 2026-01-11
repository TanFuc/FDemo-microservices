.PHONY: all build run test clean docker-up docker-down integration-test

# Build the application
build:
	go build -o bin/media-service ./cmd/server

# Run the application locally
run:
	go run ./cmd/server

# Run unit tests
test:
	go test -v ./...

# Run integration tests (requires MinIO and NATS running)
integration-test:
	INTEGRATION_TEST=true go test -v ./tests/...

# Clean build artifacts
clean:
	rm -rf bin/

# Start infrastructure with Docker Compose
docker-up:
	docker-compose up -d minio nats

# Stop infrastructure
docker-down:
	docker-compose down

# Build and run with Docker Compose
docker-all:
	docker-compose up --build

# Download dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate mocks (if needed)
mocks:
	go generate ./...
