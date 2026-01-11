.PHONY: build run test clean deps

# Build the server
build:
	go build -o server cmd/server/main.go

# Run the server
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./tests/...

# Run all tests with coverage
test-coverage:
	go test -v -cover ./...

# Download dependencies
deps:
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	rm -f server
	rm -f server.exe

# Generate protobuf files (requires protoc)
proto:
	protoc --go_out=. --go-grpc_out=. proto/order/order.proto

# Development run with hot reload (requires air)
dev:
	air

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o server-linux cmd/server/main.go

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -o server.exe cmd/server/main.go
