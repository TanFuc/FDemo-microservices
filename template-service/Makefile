.PHONY: proto run build test clean swagger

# Variables
SERVICE_NAME := template-service
BINARY_NAME := $(SERVICE_NAME)
PROTO_DIR := api/proto
PB_DIR := pkg/pb

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=module=microservices/$(SERVICE_NAME) \
		--go-grpc_out=. --go-grpc_opt=module=microservices/$(SERVICE_NAME) \
		$(PROTO_DIR)/*.proto
	@echo "Done!"

# Run the application
run:
	@go run cmd/main.go

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) cmd/main.go
	@echo "Done!"

# Run tests
test:
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Generate swagger docs
swagger:
	@swag init -g cmd/main.go -o docs

# Tidy dependencies
tidy:
	@go mod tidy

# Download dependencies
deps:
	@go mod download

# Lint code
lint:
	@golangci-lint run ./...

# Docker build
docker-build:
	@docker build -t $(SERVICE_NAME):latest .

# Docker run
docker-run:
	@docker run -p 8080:8080 -p 50051:50051 $(SERVICE_NAME):latest

# All-in-one: generate, build, run
all: proto build run
