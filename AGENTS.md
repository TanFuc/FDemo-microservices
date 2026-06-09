# AGENTS.md - FDemo Microservices

This file is the operating guide for AI agents working in this repository.

## Project Identity

- Repository: `https://github.com/TanFuc/FDemo-microservices`
- Local path: `C:\Users\nguye\OneDrive\Desktop\Project\FDemo-microservices`
- Product: TAFU e-commerce microservices backend
- Default branch: `main`
- Primary language: Go
- Secondary language: TypeScript/NestJS

Read `CLAW_PROJECT_GUIDE.md` before making architecture, infrastructure, or
cross-service changes.

## Source Of Truth

Use this priority when sources disagree:

1. Executable code and tests
2. Service `config.yaml`, `.env.example`, migrations, and proto files
3. Root compose files
4. `API_DOCUMENTATION.md` and `SERVICES_DOCUMENTATION.md`
5. Files named `*-master.md`, `*-instruction.md`, or `*PROMPT.md`

The prompt/mission documents describe intended designs and are not proof that
the implementation is complete.

## Repository Shape

- Go services: `analytic`, `api-gateway`, `auth`, `campaign`, `cart`,
  `catalog`, `inventory`, `logistic`, `media`, `notification`, `order`,
  `payment`, `profile`, `review`, `search`
- Go scaffold/reference: `template-service`
- NestJS alternatives: `auth-profile-nestjs/auth`,
  `auth-profile-nestjs/profile`
- Shared Go modules: `pkg/authorization`, `pkg/cache`, `pkg/customfields`,
  `pkg/logger`
- Infrastructure and schemas: `database`

There are duplicate Go and NestJS implementations of Auth and Profile.
Do not modify both automatically. Determine the intended runtime target from
the task, deployment config, or explicit owner decision.

## Development Rules

- Keep each service independently buildable and testable.
- Preserve the existing layered boundaries:
  `handler/router -> service/usecase -> repository/infrastructure -> domain`.
- Business logic belongs in service/usecase packages, not HTTP handlers.
- Keep HTTP DTOs, domain models, and persistence models separate where the
  service already follows that pattern.
- Use parameterized database operations.
- Keep credentials and provider keys out of source control.
- Preserve idempotency for payment webhooks, stock reservation, voucher
  claims, event consumers, and retryable jobs.
- Avoid breaking event subjects, proto contracts, Redis key formats, and
  public API paths without a migration plan.
- Add or update tests for behavioral changes.
- Do not edit generated `*.pb.go` files by hand. Update `.proto` and regenerate.
- Do not treat the root documentation as current without checking code.

## Change Workflow

1. Inspect the target service's `go.mod` or `package.json`, config, router,
   app wiring, migrations, and tests.
2. Identify affected HTTP, gRPC, event, database, and cache contracts.
3. Make the smallest coherent change inside the owning service.
4. Run formatting and focused tests.
5. For shared contracts, test every affected consumer and producer.
6. Update documentation when paths, events, config, schemas, or operations
   change.

## Verification Commands

For a Go service:

```powershell
cd <service>
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

For a NestJS service:

```powershell
cd auth-profile-nestjs\<auth-or-profile>
npm ci
npm run build
npm test -- --runInBand
```

Infrastructure validation:

```powershell
wsl.exe -d Ubuntu-22.04 --cd /mnt/c/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices --exec docker compose -f docker-compose.databases.yml config
wsl.exe -d Ubuntu-22.04 --cd /mnt/c/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices --exec docker compose -f docker-compose.prod.yml config
```

Run integration tests only with their required databases and brokers. Several
Go tests are guarded by `INTEGRATION_TEST=true`.

Run the development infrastructure in a separate visible terminal and keep
Compose attached:

```powershell
wsl.exe -d Ubuntu-22.04 --cd /mnt/c/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices --exec docker compose -f docker-compose.databases.yml up
```

Do not add `-d` unless the owner explicitly requests detached execution.

## Verified Baseline

- Go `1.26.4` is installed. All 22 Go modules pass `go test ./...`.
- Both NestJS projects build. Profile passes 17 unit tests; Auth currently has
  no test files and uses `--passWithNoTests`.
- All 12 Compose YAML files parse, and every relative bind-mount source exists.
- Docker Engine `29.3.1` and Compose `5.1.1` run in WSL2 Ubuntu 22.04.
- The database Compose stack has been started and smoke-tested in an attached
  WSL terminal. All 11 containers run; nine report healthy, while Kibana and
  Mongo Express have no Compose health check.
- PostgreSQL is published on `15432`, Redis on `16379`, MinIO API on `9002`,
  and ClickHouse native protocol retains `9000` to avoid host conflicts.
- Bootstrap verification found six PostgreSQL service databases, six MongoDB
  service databases, and 15 ClickHouse analytics tables.
- Auth production dependencies retain 8 high npm advisories. Fixing them
  requires a coordinated NestJS major-version migration; never run
  `npm audit fix --force` as an incidental change.
- Profile production dependencies retain 3 moderate npm advisories after the
  compatible audit fixes.
- The production compose uses prebuilt images and an `overlay` network; it is
  not a complete local source-build compose setup.
- Environment variable names are inconsistent between some compose entries,
  service config loaders, and `.env.example` files.
- RabbitMQ `3.12` reports an end-of-life warning and needs a planned upgrade.
- Some services have multiple entry points and older parallel implementations.
- `SERVICES_DOCUMENTATION.md` contains an outdated statement that the gateway
  lacks routes which are present in current code.

Do not hide these issues with local-only workarounds. Fix the contract or
document the intentional deployment requirement.
