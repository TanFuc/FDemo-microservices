.PHONY: build run test test-integration lint clean docker-up docker-down deps

# Build the service
build:
	go build -o bin/analytics-service ./cmd/server

# Run the service locally
run:
	go run ./cmd/server

# Run unit tests
test:
	go test -short -v ./...

# Run integration tests (requires running ClickHouse and NATS)
test-integration:
	go test -v ./tests/...

# Run load test
test-load:
	go test -v -run TestBatchIngestion ./tests/...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./tests/...

# Lint the code
lint:
	golangci-lint run ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/

# Start infrastructure with Docker Compose
docker-up:
	docker-compose up -d clickhouse nats

# Start full stack with Docker Compose
docker-up-all:
	docker-compose up -d

# Stop Docker Compose
docker-down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Initialize ClickHouse schema manually
init-schema:
	docker exec -it analytics-clickhouse clickhouse-client --query "CREATE DATABASE IF NOT EXISTS analytics"
	docker exec -it analytics-clickhouse clickhouse-client --query "CREATE TABLE IF NOT EXISTS analytics.user_events (event_id UUID, user_id String, event_type String, metadata String, url String, ip_address String, user_agent String, created_at DateTime) ENGINE = MergeTree() PARTITION BY toYYYYMM(created_at) ORDER BY (event_type, created_at)"

# Query event count
count-events:
	docker exec -it analytics-clickhouse clickhouse-client --query "SELECT count(*) FROM analytics.user_events"
