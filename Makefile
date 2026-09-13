# ==============================================================================
# NexusCommerce Platform Makefile
# Enterprise Distributed Commerce & Fulfillment Platform
# ==============================================================================

SHELL := /bin/bash
.DEFAULT_GOAL := help

COMPOSE_DIR := deploy/compose
DATABASES_COMPOSE := $(COMPOSE_DIR)/docker-compose.databases.yml
PROD_COMPOSE := $(COMPOSE_DIR)/docker-compose.prod.yml
OBS_COMPOSE := $(COMPOSE_DIR)/docker-compose.observability.yml

.PHONY: help
help: ## Display available commands
	@echo "====================================================================="
	@echo "  NexusCommerce Enterprise Infrastructure & Development CLI"
	@echo "====================================================================="
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ------------------------------------------------------------------------------
# Database & Core Infrastructure
# ------------------------------------------------------------------------------
.PHONY: up-db
up-db: ## Launch all database & message broker containers (Postgres, Mongo, Redis, NATS, etc.)
	docker compose -f $(DATABASES_COMPOSE) up -d

.PHONY: down-db
down-db: ## Stop and remove all database containers (preserves volume data)
	docker compose -f $(DATABASES_COMPOSE) down

.PHONY: ps-db
ps-db: ## Show status of running infrastructure containers
	docker compose -f $(DATABASES_COMPOSE) ps

.PHONY: logs-db
logs-db: ## Stream logs from infrastructure containers
	docker compose -f $(DATABASES_COMPOSE) logs -f

# ------------------------------------------------------------------------------
# Observability Stack (Prometheus, Grafana, Tempo, Loki)
# ------------------------------------------------------------------------------
.PHONY: up-obs
up-obs: ## Start the observability stack (Prometheus :9090, Grafana :3001, Tempo, Loki)
	docker compose -f $(OBS_COMPOSE) up -d

.PHONY: down-obs
down-obs: ## Stop the observability stack
	docker compose -f $(OBS_COMPOSE) down

# ------------------------------------------------------------------------------
# Production Microservices Stack
# ------------------------------------------------------------------------------
.PHONY: up-prod
up-prod: ## Launch all 15 microservices + API Gateway
	docker compose -f $(PROD_COMPOSE) up -d

.PHONY: down-prod
down-prod: ## Stop all microservices
	docker compose -f $(PROD_COMPOSE) down

# ------------------------------------------------------------------------------
# Quality Assurance & Testing
# ------------------------------------------------------------------------------
.PHONY: test
test: ## Run unit tests across all Go microservices and shared libraries
	@echo "Running tests across all Go modules..."
	@for dir in pkg/idempotency pkg/messaging pkg/saga pkg/logger pkg/cache pkg/authorization pkg/customfields api-gateway auth profile catalog cart order inventory payment logistic campaign notification review media analytic; do \
		if [ -f "$$dir/go.mod" ]; then \
			echo "Testing $$dir..."; \
			(cd $$dir && go test -v -race ./...) || exit 1; \
		fi \
	done
	@echo "All tests passed successfully."

.PHONY: vet
vet: ## Run static analysis (go vet) across all modules
	@for dir in pkg/idempotency pkg/messaging pkg/saga pkg/logger pkg/cache pkg/authorization pkg/customfields api-gateway auth profile catalog cart order inventory payment logistic campaign notification review media analytic; do \
		if [ -f "$$dir/go.mod" ]; then \
			echo "Vetting $$dir..."; \
			(cd $$dir && go vet ./...) || exit 1; \
		fi \
	done

.PHONY: compose-validate
compose-validate: ## Validate syntax and integrity of all Docker Compose manifests
	docker compose -f $(DATABASES_COMPOSE) config --quiet
	docker compose -f $(PROD_COMPOSE) config --quiet
	docker compose -f $(OBS_COMPOSE) config --quiet
	@echo "All Docker Compose configurations are valid."

# ------------------------------------------------------------------------------
# Protocol Buffers & gRPC Code Generation
# ------------------------------------------------------------------------------
.PHONY: proto
proto: ## Compile all Protocol Buffer (.proto) definitions across all services
	@bash scripts/generate-protos.sh
