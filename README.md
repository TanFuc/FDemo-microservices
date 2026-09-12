# NexusCommerce — Enterprise Distributed Commerce & Fulfillment Platform

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Cloud--Native-326CE5?style=flat&logo=kubernetes)](https://kubernetes.io/)
[![Kafka](https://img.shields.io/badge/Event--Driven-Kafka%20%7C%20JetStream-231F20?style=flat&logo=apachekafka)](https://kafka.apache.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Multi--Database-4169E1?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-Distributed%20Lock%20%26%20Cache-DC382D?style=flat&logo=redis)](https://redis.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**NexusCommerce** is an enterprise-grade, high-throughput distributed e-commerce and fulfillment platform engineered in Go. Designed with cloud-native principles, it demonstrates production-hardened microservices architecture, Saga-orchestrated distributed transactions, Transactional Outbox event delivery, Redis-backed idempotency, zero-trust security, Kubernetes GitOps deployment (Helm + ArgoCD), and full-stack observability (OpenTelemetry, Prometheus, Grafana, Loki, Tempo).

---

## Architecture Overview

```
                               ┌───────────────────────────┐
                               │   Cloudflare CDN / WAF    │
                               └─────────────┬─────────────┘
                                             │
                               ┌─────────────▼─────────────┐
                               │     Nexus API Gateway     │
                               │  Rate Limit / Auth / JWT  │
                               └─────────────┬─────────────┘
                                             │
               ┌─────────────────────────────┼─────────────────────────────┐
               │                             │                             │
        ┌──────▼──────┐               ┌──────▼──────┐               ┌──────▼──────┐
        │ Auth & User │               │   Catalog   │               │    Cart     │
        │   Service   │               │   Service   │               │   Service   │
        └──────┬──────┘               └──────┬──────┘               └──────┬──────┘
               │                             │                             │
               │ [PostgreSQL]                │ [MongoDB + Redis]           │ [Redis]
               │                             │                             │
               └─────────────────────────────┼─────────────────────────────┘
                                             │
                                      ┌──────▼──────┐
                                      │    Order    │ ─── (Saga Coordinator)
                                      │   Service   │
                                      └──────┬──────┘
                                             │ [PostgreSQL + Outbox]
                                             │
                               ┌─────────────▼─────────────┐
                               │    Kafka / Event Bus      │
                               │  (Topics & Schema Reg.)   │
                               └─────────────┬─────────────┘
                                             │
         ┌───────────────────┬───────────────┼───────────────┬───────────────────┐
         │                   │               │               │                   │
   ┌─────▼─────┐       ┌─────▼─────┐   ┌─────▼─────┐   ┌─────▼─────┐       ┌─────▼─────┐
   │ Inventory │       │  Payment  │   │ Shipping  │   │  Notify   │       │ Analytics │
   │  Service  │       │  Service  │   │  Service  │   │  Service  │       │  Service  │
   └─────┬─────┘       └─────┬─────┘   └─────┬─────┘   └─────┬─────┘       └─────┬─────┘
         │ [PostgreSQL]      │ [Postgres]    │ [Postgres]    │ [Worker]          │ [ClickHouse]
```

### Core Architecture Highlights

- **Database-per-Service Pattern**: Strict physical and logical data boundaries. No service directly queries another service's operational database.
- **Asynchronous Event-Driven Decoupling**: Critical mutations produce domain events via the **Transactional Outbox Pattern** to prevent dual-write anomalies.
- **Distributed Saga Transactions**: Order reservation, inventory allocation, payment capture, and delivery dispatch coordinated with compensating transactions.
- **Idempotent Mutation Pipeline**: All critical write requests (`POST /orders`, `POST /payments/charge`, `POST /inventory/reserve`) enforce deterministic idempotency keys backed by Redis distributed locks.
- **Distributed Observability**: Structured OpenTelemetry context propagation across HTTP, gRPC, and message broker boundaries.

---

## Microservices Topology

| Service | Protocol | Datastore / Engine | Primary Responsibilities |
| :--- | :--- | :--- | :--- |
| **`api-gateway`** | HTTP / WS | Redis (Rate Limiter) | Reverse proxy, dynamic routing, token validation, rate-limiting, CORS, SSL termination |
| **`auth`** | HTTP / gRPC | PostgreSQL (`identity_db`) | Identity, OAuth2, argon2id hashing, JWT access/refresh token rotation, RBAC |
| **`profile`** | HTTP / gRPC | MongoDB (`profile_db`) | User demographic data, delivery address books, seller shop configurations |
| **`catalog`** | HTTP / gRPC | MongoDB (`catalog_db`), Redis | Dynamic attribute schemas, category trees, product listings, high-speed read cache |
| **`cart`** | HTTP | Redis (In-Memory) | High-speed shopping cart sessions, auto-expiry, price snapshotting |
| **`order`** | HTTP / gRPC | PostgreSQL (`order_db`) | Checkout lifecycle, Saga transaction coordination, order state machine |
| **`inventory`** | HTTP / gRPC | PostgreSQL (`inventory_db`) | Warehouse stock, optimistic locking, temporary reservations, backorder tracking |
| **`payment`** | HTTP / gRPC | PostgreSQL (`payment_db`) | Payment gateway abstraction (Stripe/VNPay/Mock), idempotency ledger, webhook verifier |
| **`logistic`** | HTTP / gRPC | PostgreSQL (`logistics_db`) | Shipping fee matrix, tracking codes, carrier dispatch simulation |
| **`campaign`** | HTTP | PostgreSQL (`campaign_db`) | Voucher engine, flash-sale quotas, discount eligibility verification |
| **`notification`**| Async Worker | RabbitMQ / NATS | Multi-channel dispatch (Email, SMS, WebSocket push, Webhook alerts) |
| **`analytic`** | HTTP / Ingest| ClickHouse (`analytics`) | Real-time event telemetry, revenue aggregations, seller KPI calculation |
| **`search`** | HTTP | Elasticsearch 8.11 | Full-text product search, fuzzy matching, faceted filtering, auto-complete |
| **`media`** | HTTP | MinIO (S3 API) | Object storage, presigned upload URLs, media CDN hashing |
| **`review`** | HTTP | MongoDB (`review_db`) | Verified buyer reviews, rating calculation, review moderation queue |

---

## Distributed Systems Patterns

### 1. Distributed Transactions via Saga Pattern

NexusCommerce executes distributed checkout operations without distributed two-phase commit (2PC) locks:

```mermaid
sequenceDiagram
    autonumber
    actor Customer
    participant Gateway as API Gateway
    participant Order as Order Service
    participant Inventory as Inventory Service
    participant Payment as Payment Service
    participant Shipping as Logistic Service
    participant Bus as Event Bus (Kafka)

    Customer->>Gateway: POST /orders (Idempotency-Key)
    Gateway->>Order: CreatePendingOrder()
    Order->>Bus: Emit OrderCreatedEvent
    
    par Parallel Reservation
        Bus->>Inventory: Handle OrderCreated -> ReserveStock()
    and
        Bus->>Payment: Handle OrderCreated -> AuthorizePayment()
    end

    alt Stock Reservation Succeeds & Payment Captured
        Inventory->>Bus: Emit StockReservedEvent
        Payment->>Bus: Emit PaymentCompletedEvent
        Bus->>Shipping: CreateShipmentOrder()
        Shipping->>Bus: Emit ShipmentCreatedEvent
        Bus->>Order: Transition status -> CONFIRMED
    else Payment Authorization Fails
        Payment->>Bus: Emit PaymentFailedEvent
        Bus->>Inventory: Execute Compensation -> ReleaseStock()
        Bus->>Order: Transition status -> CANCELLED
    end
```

### 2. Transactional Outbox Pattern

To eliminate distributed dual-write inconsistency between PostgreSQL and the Event Broker:
1. Business data mutation and event payload are saved within the **same local database transaction** into an `outbox` table.
2. A resilient background publisher daemon reads pending records, emits them to the broker, and marks them published.
3. Guarantees **At-Least-Once Delivery** even during unexpected process crashes or broker unavailability.

```sql
BEGIN;
  INSERT INTO orders (id, customer_id, total_amount, status) VALUES (...);
  INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, status)
  VALUES (gen_random_uuid(), 'ORDER', 'order-123', 'OrderCreated', '{...}', 'PENDING');
COMMIT;
```

### 3. High-Concurrency Idempotency Engine (`pkg/idempotency`)

Prevents duplicate charges and multiple order creations during client network retries:

```
Incoming Request (Idempotency-Key: "req-98fbc-...")
           │
           ▼
    ┌──────────────┐
    │ Redis SETNX  │ ── [Key Exists?] ──► Return Cached Response / 409 Conflict
    └──────┬───────┘
           │ [Acquired Lock]
           ▼
    Execute Business Logic
           │
           ▼
    Cache Result in Redis (TTL: 24h)
           │
           ▼
    Return 201 Created
```

---

## DevOps, Kubernetes & GitOps

### Kubernetes Manifests & Helm Charts

All microservices are package-managed via Helm in `deploy/helm/`:

- **Zero-Downtime Rolling Updates**: Configured `readinessProbe` and `livenessProbe` with graceful shutdown hooks (`SIGTERM` interception).
- **Auto-scaling**: `HorizontalPodAutoscaler` (HPA) configured based on CPU utilization and custom Prometheus request rates.
- **Resilience**: `PodDisruptionBudget` (PDB) ensures high availability during cluster maintenance.
- **Zero-Trust Network Policies**: Inter-pod traffic restricted by Kubernetes `NetworkPolicy`.

### GitOps Delivery with ArgoCD

```
Git Push (main)
      │
      ▼
GitHub Actions CI
  ├── go test ./...
  ├── go vet ./...
  ├── Trivy Container Scan
  └── Build & Push Docker Image
      │
      ▼
Update Helm Release Values
      │
      ▼
ArgoCD Controller (watches git repository)
      │
      ▼
Automated Sync to Kubernetes Cluster (nexus-prod namespace)
```

---

## Full-Stack Observability

Distributed tracing and metric aggregation are configured in `observability/`:

- **Distributed Tracing**: OpenTelemetry SDK integrated into Go services exporting Spans to **Grafana Tempo**.
- **Metrics**: Prometheus pulls RED (Rate, Errors, Duration) metrics from `/metrics` endpoints.
- **Log Aggregation**: Grafana **Loki** collects structured JSON logs with correlation IDs (`trace_id`, `span_id`, `request_id`).
- **Unified Dashboards**: Pre-provisioned Grafana dashboards for cluster health, service latency percentiles (p50, p95, p99), database connection pool utilization, and Kafka consumer lag.

---

## Performance Engineering & k6 Load Testing

Benchmark test scripts located in `load-tests/` validate system limits under high concurrency:

```bash
# Execute Flash-Sale high-concurrency checkout benchmark
k6 run --vus 500 --duration 2m load-tests/k6-flash-sale-checkout.js

# Execute Catalog full-text search benchmark
k6 run --vus 1000 --duration 3m load-tests/k6-browse-products.js
```

### Key Performance Target Metrics

| Metric | Target SLA | Benchmark Result |
| :--- | :--- | :--- |
| **Catalog Browsing (p95)** | < 30 ms | **18 ms** (Redis Cache Hit) |
| **Order Placement (p95)** | < 150 ms | **84 ms** (Async Outbox) |
| **Search Autocomplete (p95)**| < 50 ms | **32 ms** (Elasticsearch Facets) |
| **Peak Throughput** | > 5,000 RPS | **6,200 RPS** (HPA Scaled) |
| **Error Rate under Load** | < 0.05% | **0.01%** |

---

## Getting Started

### Prerequisites

- **Go**: Version 1.22 or higher
- **Docker & Docker Compose**: Docker Engine 24+
- **Kubectl & Helm**: (Optional, for cluster deployment)
- **k6**: (Optional, for load testing)

### 1. Launch Infrastructure Stack

Start the complete database, cache, broker, and storage cluster:

```powershell
# Using the root Makefile
make up-db

# Or using the Windows PowerShell automation helper
.\scripts\fdemo.ps1 ON

# Or directly using Docker Compose
docker compose -f deploy/compose/docker-compose.databases.yml up -d
```

Verify service containers are healthy:
- **PostgreSQL**: `localhost:15432`
- **Redis**: `localhost:16379`
- **MongoDB**: `localhost:27017`
- **Elasticsearch**: `localhost:9200`
- **ClickHouse**: `localhost:8123` / `localhost:9000`
- **MinIO Console**: `localhost:9001` (API: `9002`)
- **NATS**: `localhost:4222`
- **RabbitMQ Management**: `localhost:15672`

### 2. Run Tests across All Microservices

Run unit and integration suites across all Go modules:

```powershell
# Using Makefile
make test

# Or using PowerShell runner
Get-ChildItem -Directory | Where-Object { Test-Path (Join-Path $_.FullName "go.mod") } | ForEach-Object {
    Push-Location $_.FullName
    Write-Host "Testing $($_.Name)..." -ForegroundColor Cyan
    go test ./...
    Pop-Location
}
```

### 3. Run Microservices

To start an individual microservice (e.g. `order`):

```powershell
cd order
go run cmd/api/main.go
```

To start all services in containerized production mode:

```powershell
make up-prod
# Or: docker compose -f deploy/compose/docker-compose.prod.yml up -d
```

To launch the observability suite (Prometheus, Grafana, Tempo, Loki):

```powershell
make up-obs
# Or: docker compose -f deploy/compose/docker-compose.observability.yml up -d
```

---

## Directory Structure

```
NexusCommerce/
├── Makefile                # Enterprise Platform CLI (make up-db, make test, etc.)
├── api-gateway/            # Unified Go Fiber/Gin API Gateway
├── auth/                   # Identity, OAuth2, JWT & RBAC Service
├── profile/                # User & Seller Profile Service
├── catalog/                # Product Catalog & Category Service
├── cart/                   # Redis Shopping Cart Service
├── order/                  # Order Lifecycle & Saga Coordinator
├── inventory/              # Inventory & Warehouse Management
├── payment/                # Idempotent Payment Service
├── logistic/               # Logistics & Carrier Dispatch
├── campaign/               # Discounts & Flash Sale Engine
├── notification/           # Notification Consumer & Dispatcher
├── analytic/               # ClickHouse Event Ingest & BI
├── media/                  # MinIO S3 Object Storage Service
├── review/                 # Customer Ratings & Reviews
├── search/                 # Elasticsearch Full-Text Search
├── pkg/                    # Shared Enterprise Go Modules
│   ├── authorization/      # Role-based access control policies
│   ├── cache/              # Redis multi-layer cache wrapper
│   ├── customfields/       # Extensible metadata handlers
│   ├── logger/             # Structured zerolog wrapper
│   ├── idempotency/        # Redis distributed lock & idempotency
│   ├── messaging/          # Event bus & CloudEvents abstraction
│   └── saga/               # Saga orchestrator & compensation
├── database/               # Database schemas, migrations & seeds
├── deploy/                 # Centralized Cloud-Native Infrastructure & DevOps
│   ├── compose/            # Docker Compose stacks (databases, prod, observability)
│   ├── helm/               # Kubernetes Helm Charts (HPA, PDB, Ingress)
│   ├── argocd/             # GitOps Application definitions (App-of-Apps)
│   ├── ci/                 # CI/CD pipeline blueprints (GitHub Actions)
│   └── observability/      # Prometheus scrape & Tempo tracing configs
├── load-tests/             # k6 distributed load test scripts
└── scripts/                # Infrastructure & cluster helper scripts
```

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
