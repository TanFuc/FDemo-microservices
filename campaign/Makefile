.PHONY: build run test clean tidy migrate

# Build the application
build:
	go build -o bin/campaign-service ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run all tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race -v ./...

# Run tests with coverage
test-cover:
	go test -cover -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Tidy dependencies
tidy:
	go mod tidy

# Download dependencies
deps:
	go mod download

# Run database migrations (requires psql)
migrate:
	psql -h localhost -U postgres -d campaign_service -f migrations/001_init.sql

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run
