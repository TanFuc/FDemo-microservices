.PHONY: all build run test clean docker-build docker-up docker-down migrate lint

# Go parameters
BINARY_NAME=logistics-service
MAIN_PATH=./cmd/server
GO=go

# Docker parameters
DOCKER_COMPOSE=docker-compose

all: build

## Build the application
build:
	$(GO) build -o $(BINARY_NAME) $(MAIN_PATH)

## Run the application
run:
	$(GO) run $(MAIN_PATH)

## Run tests
test:
	$(GO) test -v ./...

## Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## Run integration tests
test-integration:
	$(GO) test -v -run Integration ./tests/...

## Clean build artifacts
clean:
	$(GO) clean
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

## Run linter
lint:
	golangci-lint run ./...

## Build Docker image
docker-build:
	docker build -t $(BINARY_NAME):latest .

## Start all services with Docker Compose
docker-up:
	$(DOCKER_COMPOSE) up -d

## Stop all services
docker-down:
	$(DOCKER_COMPOSE) down

## View logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## Start only infrastructure (postgres, redis, nats)
infra-up:
	$(DOCKER_COMPOSE) up -d postgres redis nats

## Stop infrastructure
infra-down:
	$(DOCKER_COMPOSE) down

## Run migrations (requires psql)
migrate:
	psql $(DATABASE_URL) -f migrations/001_create_shipping_orders.sql

## Generate mocks (requires mockgen)
mocks:
	mockgen -source=internal/core/ports/provider.go -destination=internal/core/ports/mocks/provider_mock.go
	mockgen -source=internal/core/ports/repository.go -destination=internal/core/ports/mocks/repository_mock.go
	mockgen -source=internal/core/ports/cache.go -destination=internal/core/ports/mocks/cache_mock.go
	mockgen -source=internal/core/ports/messaging.go -destination=internal/core/ports/mocks/messaging_mock.go

## Help
help:
	@echo "Available targets:"
	@echo "  build            - Build the application"
	@echo "  run              - Run the application"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  clean            - Clean build artifacts"
	@echo "  deps             - Install dependencies"
	@echo "  lint             - Run linter"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-up        - Start all services"
	@echo "  docker-down      - Stop all services"
	@echo "  docker-logs      - View logs"
	@echo "  infra-up         - Start infrastructure only"
	@echo "  infra-down       - Stop infrastructure"
	@echo "  migrate          - Run database migrations"
	@echo "  mocks            - Generate mocks"
