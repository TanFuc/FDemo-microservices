# Claw Project Guide - FDemo Microservices

Last repository scan: 2026-06-09

## 1. Purpose

This repository is a backend prototype for a TAFU e-commerce platform built
as independently deployable microservices. It covers identity, profiles,
catalog, carts, orders, stock, payment, shipping, notifications, media,
search, promotions, reviews, and analytics.

The repository is implementation-heavy but not yet an operationally unified
platform. It contains working code, scaffolds, design prompts, duplicate
implementations, and deployment files at different maturity levels.

## 2. Baseline

- Branch: `main`
- Scanned commit: `b5b942186405df8fff849a0698f14ad0b82d68a5`
- Commit subject: `Merge pull request #5 from TanFuc/develop`
- Files: about 760
- Go files: about 496
- TypeScript files: about 84
- Markdown files: about 26
- SQL migrations/schema files: about 16
- Proto files: about 10
- Go modules: 22 service/shared modules
- Node projects: 2 NestJS projects

Remote branches show active/refactor work for Auth, Profile, CI/CD, and service
structure. Do not assume `main` has resolved every architectural choice.

## 3. Architecture Map

| Area | Directory | Main storage/infrastructure | Main interfaces |
|---|---|---|---|
| Edge routing | `api-gateway` | Redis | HTTP reverse proxy, JWT |
| Identity | `auth` | PostgreSQL, Redis | HTTP, gRPC, NATS |
| Identity alternative | `auth-profile-nestjs/auth` | PostgreSQL, Redis | HTTP, gRPC |
| Profile | `profile` | MongoDB | HTTP, NATS |
| Profile alternative | `auth-profile-nestjs/profile` | MongoDB/Prisma artifacts | HTTP, gRPC |
| Catalog | `catalog` | MongoDB, Redis | HTTP, gRPC, NATS |
| Cart | `cart` | Redis, MongoDB | HTTP, gRPC client |
| Order | `order` | PostgreSQL | HTTP, gRPC client, NATS |
| Inventory | `inventory` | PostgreSQL, Redis/Lua | HTTP, gRPC |
| Payment | `payment` | PostgreSQL | HTTP/webhooks, NATS |
| Logistics | `logistic` | PostgreSQL, Redis | HTTP/webhooks, NATS |
| Notification | `notification` | MongoDB, RabbitMQ | HTTP, WebSocket, NATS |
| Media | `media` | MinIO | HTTP, NATS, worker |
| Search | `search` | Elasticsearch, Redis | HTTP, NATS consumer |
| Campaign | `campaign` | PostgreSQL, Redis/Lua | HTTP, gRPC |
| Review | `review` | MongoDB, Redis | HTTP, Order gRPC client |
| Analytics | `analytic` | ClickHouse | HTTP, NATS consumer |
| Shared packages | `pkg/*` | N/A | Go modules |
| Infrastructure | `database` | SQL, Mongo init, Redis docs | Bootstrap assets |

`template-service` is a reference/scaffold, not one of the documented product
services.

## 4. Main Business Flows

### Product discovery

1. Catalog owns product, category, brand, variation, and metadata records.
2. Catalog publishes product change events through NATS.
3. Search consumes product events and updates Elasticsearch.
4. API Gateway exposes public catalog and search routes.

### Checkout

1. Cart stores active state primarily in Redis and persists a backup in
   MongoDB.
2. Order snapshots item and shipping data.
3. Order reserves stock through Inventory gRPC.
4. Order publishes events for Payment and Notification.
5. Payment webhooks update payment state and emit downstream events.
6. Inventory confirms or releases reservations.
7. Logistics creates and tracks shipment state.

This flow crosses HTTP, gRPC, NATS, PostgreSQL, Redis, and provider webhooks.
Changes require contract-level testing, not only unit tests in one service.

### Media

1. Media returns a MinIO presigned upload URL.
2. Client uploads directly to object storage.
3. Client confirms upload.
4. Media publishes/consumes processing work and creates transformed variants.

### Promotions

Campaign uses PostgreSQL for campaign/voucher state and Redis Lua for atomic
voucher claims. Preserve atomicity and duplicate-claim protection.

## 5. Service Conventions

Most Go services use some form of:

```text
cmd/                 process entry points
internal/app/        dependency wiring and lifecycle
internal/config/     YAML/environment loading
internal/domain/     entities and domain contracts
internal/service/    business services
internal/usecase/    application use cases
internal/repository/ persistence implementations
internal/handler/    HTTP/gRPC adapters
internal/router/     route registration
pkg/                 service-local helpers or generated protobuf
```

The exact structure differs by service because the repository contains
several generations of architecture. Follow the target service's current
shape instead of forcing a repository-wide refactor during a feature change.

## 6. API And Contract Guidance

- Gateway routes are defined in `api-gateway/internal/routes/routes.go`.
- Service routes are authoritative in each `internal/router` or handler.
- `API_DOCUMENTATION.md` is broad reference documentation, but route prefixes
  do not always match current service routers.
- Generated gRPC code lives beside service proto contracts.
- Redis naming guidance is in `database/redis/REDIS_KEY_PATTERNS.md`.
- The consolidated SQL file is useful for review, but service migrations own
  service-specific schema evolution.

When an endpoint changes, check all four surfaces:

1. Service router and handler
2. API Gateway path rewriting/proxy behavior
3. API documentation
4. Calling services or frontend clients

The gateway currently appends `c.OriginalURL()` to the downstream base URL.
This means gateway prefixes and downstream route prefixes must be verified
together; a route existing in both places does not guarantee correct proxying.

## 7. Local Development

### Required tools

- Go version compatible with each module's `go.mod`
- Node.js and npm for the NestJS alternatives
- Docker Engine with Compose in WSL2 for infrastructure
- `protoc` and Go plugins when changing protobuf contracts

Verified toolchain on 2026-06-09: Go `1.26.4`, Node.js `v24.14.0`, npm
`11.9.0`, protobuf generators, Docker Engine `29.3.1`, and Docker Compose
`5.1.1` in WSL2 Ubuntu 22.04.

### Infrastructure

The database compose intends to provide:

- PostgreSQL: `15432` on the host, `5432` inside Compose
- MongoDB: `27017`
- Redis: `16379` on the host, `6379` inside Compose
- Elasticsearch: `9200`, `9300`
- Kibana: `5601`
- ClickHouse: `8123`, `9000`
- MinIO: `9002` (API), `9001` (console)
- NATS: `4222`, `6222`, `8222`
- RabbitMQ: `5672`, `15672`
- Redis Commander: `8081`
- Mongo Express: `8082`

Inside the Compose network, Media still connects to MinIO at `minio:9000`.
Only the host-side API port is remapped to `9002`.

Start the infrastructure in a separate visible PowerShell terminal. Compose
must remain attached so logs and failures are visible:

```powershell
wsl.exe -d Ubuntu-22.04 --cd /mnt/c/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices --exec docker compose -f docker-compose.databases.yml up
```

Do not use `-d` for the normal development workflow. Closing that terminal or
pressing `Ctrl+C` stops the stack.

### Claw operation keywords

The owner can use these Telegram keywords:

- `FDemo ON`: start the infrastructure in one Windows Terminal window with
  `FDemo Infrastructure` and `FDemo Shell` tabs.
- `FDemo OFF`: stop only the FDemo Compose services while preserving volumes.
- `FDemo STATUS`: report FDemo containers, WSL memory, and unrelated
  containers that remain active.

The local implementation is:

```powershell
.\scripts\fdemo.ps1 ON
.\scripts\fdemo.ps1 OFF
.\scripts\fdemo.ps1 STATUS
```

Do not stop unrelated containers such as `n8n` unless the owner explicitly
requests it.

### Service execution

Most Go services can be approached with:

```powershell
cd <service>
Copy-Item .env.example .env
go mod download
go run ./cmd
```

Some services use `./cmd/server` or another entry point. Inspect the service
before choosing the command. Do not launch all services using their checked-in
default ports because multiple defaults collide.

NestJS alternatives:

```powershell
cd auth-profile-nestjs\auth
npm ci
npm run start:dev
```

```powershell
cd auth-profile-nestjs\profile
npm ci
npm run start:dev
```

## 8. Testing Strategy

### Focused service change

- Run the target module's unit tests.
- Run `go vet ./...` or Nest build/lint.
- Test router behavior for path or middleware changes.
- Test repository behavior for query/schema changes.

### Shared contract change

- Proto: regenerate code and test server plus every client.
- Event: test publisher payload, subject, consumer, retries, and idempotency.
- Database: provide forward migration and account for existing data.
- Redis: preserve key format/TTL or provide explicit migration/invalidation.
- Gateway: test public path, auth, rate limit, and downstream path.

### Integration tests

Some Go test suites skip unless `INTEGRATION_TEST=true`. Start only the
required dependencies and use isolated test databases. Never point destructive
integration tests at shared development or production data.

## 9. Verified Repository Risks

### Operational gaps

1. Production compose declares only remote/prebuilt `tafu/*` images, with no
   source build definitions.
2. Production compose uses an `overlay` network, which normally implies Docker
   Swarm rather than ordinary local Compose.
3. The local database stack is verified, but the production compose still
   requires real secrets, prebuilt images, and a deployment-mode decision.

### Configuration drift

- Service defaults use many different HTTP/gRPC ports.
- Production compose tries to normalize services to port `3000`, but not every
  config loader clearly consumes the same variable names.
- Examples include `DB_USER` versus `DB_USERNAME`, `REDIS_URL` versus
  `REDIS_ADDR`/host+port, and `MONGODB_URI` versus `MONGO_URI`.
- Health checks assume routes and ports that must be confirmed against each
  selected implementation.

Before deployment, create a generated configuration matrix and validate every
container's effective environment.

### Architecture drift

- Auth and Profile each have Go and NestJS implementations.
- Several services expose both `cmd/main.go` and `cmd/server/main.go`.
- Some directories contain both newer layered routers and older handlers.
- Root documentation dated 2026-01-07 says gateway routes are missing, while
  current gateway code includes those routes.
- Mission/prompt documents may describe desired code rather than completed
  behavior.

### Delivery maturity

- Only some service directories have Dockerfiles.
- No repository-level Go workspace is committed; each Go module is separate.
- A root source-build compose and unified developer bootstrap are absent.
- The scan found limited automated tests relative to the number of services.
- Auth NestJS still has no test files. Its production dependency audit reports
  8 high advisories that require a coordinated NestJS major upgrade.
- Profile NestJS passes 17 tests and has 3 remaining moderate production
  dependency advisories.
- RabbitMQ `3.12` emits an end-of-life warning and should be upgraded as a
  planned compatibility change.

## 10. Stabilization Verification (2026-06-09)

- `go test ./...`: pass in all 22 Go modules.
- Auth NestJS: build pass; test command pass with no test files.
- Profile NestJS: build pass; 17/17 tests pass.
- Compose: 12/12 YAML files parse; all relative bind mounts exist.
- WSL2 database Compose: all 11 containers run; PostgreSQL, MongoDB, Redis,
  Elasticsearch, ClickHouse, MinIO, NATS, RabbitMQ, and Redis Commander report
  healthy. Kibana and Mongo Express respond successfully but have no health
  check in Compose.
- Endpoint smoke tests pass for Redis, Elasticsearch, ClickHouse, MinIO, NATS,
  RabbitMQ, Kibana, Redis Commander, and Mongo Express.
- Bootstrap data checks pass: six PostgreSQL service databases, six MongoDB
  service databases, and 15 ClickHouse analytics tables.
- `git diff --check`: pass.
- Generated Catalog and Inventory protobuf code is present and compiled.
- Inventory gRPC is registered, and Order uses the Inventory protobuf contract.
- Legacy duplicate implementations are excluded from default builds with the
  `legacy` build tag.
- Compatible `npm audit fix` updates were applied without forcing major
  dependency changes.

## 11. Recommended Stabilization Order

1. Decide canonical Auth and Profile implementations.
2. Add service-level integration and end-to-end tests against the verified
   WSL2 infrastructure stack.
3. Plan and test the Auth NestJS major dependency/security migration.
4. Build a local source-based compose with one authoritative environment
   contract per service.
5. Add a root verification script that tests all Go modules and both Node
   projects.
6. Add CI for build, unit test, config validation, and generated-code drift.
7. Reconcile gateway paths with downstream routers using automated smoke tests.
8. Refresh API and service documentation from verified code.
9. Add end-to-end tests for checkout, payment webhook, stock reservation,
   catalog indexing, and notification delivery.

## 12. Agent Checklist

Before implementation:

- Confirm the canonical service implementation.
- Read the router, app wiring, config loader, domain, repository, and tests.
- Search for consumers of any changed path, event, proto, schema, or key.

Before completion:

- Format changed files.
- Run focused tests and static checks.
- State exactly which checks could not run and why.
- Update API/config/operations documentation when contracts change.
- Keep unrelated refactors out of the patch.
