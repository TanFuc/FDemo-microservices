# Mandatory Operating Protocol & Engineering Rules

> **DOCUMENT CLASSIFICATION**: GOVERNING SPECIFICATION  
> **APPLIES TO**: AI AGENTS, HUMAN DEVELOPERS, CODE COMPLETIONS, AUTOMATED SCRIPTS  
> **ENFORCEMENT**: STRICT & UNCONDITIONAL  
> **LANGUAGE REQUIREMENT**: 100% EXCLUSIVE TECHNICAL ENGLISH

---

## 1. Documentation Language Governance (STRICT ENGLISH ONLY)

### 1.1 Scope of Enforcement
- **Universal English Requirement**: Every documentation file, markdown specification, architectural decision record (ADR), run log, execution outcome, status summary, script comment, git commit message, and inline code comment MUST be authored exclusively in English.
- **Zero-Tolerance for Non-English**: No Vietnamese or any other non-English language is permitted within this repository under any circumstances.
- **Remediation**: If an agent or contributor identifies non-English text in existing operational documents or code comments within their change scope, they are required to translate and normalize it to English as part of the commit.

---

## 2. Mandatory Pre-Execution Verification Protocol

Prior to invoking **ANY** command capable of inspecting or altering state (`run_command`), or executing **ANY** file edit (`replace_file_content`, `write_to_file`), the AI Agent **MUST** execute this tri-fold inspection step:

```text
+-------------------------------------------------------------------------------+
|                        PRE-EXECUTION MANDATORY INSPECTION                      |
+-------------------------------------------------------------------------------+
| 1. Review run-context/rule.md:                                                |
|    - Confirm language rules, safety barriers, and execution environment.      |
| 2. Review run-context/context.md:                                             |
|    - Verify target service architecture, port mappings, and datastores.      |
| 3. Review run-context/results/latest.md:                                       |
|    - Audit previous execution status, open regressions, and pending tasks.    |
+-------------------------------------------------------------------------------+
```

### 2.1 Explicit Steps Before Action
1. **Target Identification**: Determine precisely which microservice, shared package (`pkg/*`), or infrastructure manifest owns the requested change.
2. **Contract Surface Audit**: Determine if the change alters an HTTP endpoint, gRPC protobuf contract, NATS subject, RabbitMQ queue, Redis key format, or SQL migration schema.
3. **Environment Segregation Check**: Identify whether the planned command targets the host Windows environment (PowerShell) or the containerized infrastructure running in WSL2 Ubuntu-22.04.

---

## 3. Engineering & Code Modification Rules

### 3.1 Strict Layered Architecture
Every Go microservice follows clean architectural boundaries:
`cmd/ -> internal/handler (or router) -> internal/usecase (or service) -> internal/repository (or infrastructure) -> internal/domain`

1. **Handler / Transport Layer**:
   - Sole responsibility: Decode incoming HTTP/gRPC requests, validate input DTOs, invoke the appropriate domain use case, and serialize responses.
   - **PROHIBITED**: Writing database queries, orchestration logic, payment calculations, or business rules directly inside HTTP handlers.
2. **Service / Use Case Layer**:
   - Owns core business logic, domain transaction coordination, authorization enforcement, and event dispatch.
   - Must remain decoupled from transport protocols (HTTP frameworks, gRPC status codes).
3. **Repository / Infrastructure Layer**:
   - Implements domain repository interfaces using SQL (pgx/sqlx), MongoDB Go driver, ClickHouse native client, or Redis client.
   - Every database query must use parameterized placeholders (`$1, $2` in Postgres; `?` in MySQL/ClickHouse; BSON filters in Mongo). Raw string concatenation for SQL or NoSQL queries is strictly prohibited.
4. **Domain Layer**:
   - Pure Go models, domain errors, domain events, and repository interfaces. Must not import infrastructure drivers or web frameworks.

### 3.2 Protobuf & gRPC Code Generation
- **NEVER Edit Generated Files**: Files matching `*.pb.go` or `*_grpc.pb.go` are strictly machine-generated. Hand-modifying them is grounds for immediate rejection.
- **Workflow for Proto Changes**:
  1. Modify the `.proto` definition in the target service proto directory.
  2. Regenerate Go code using the standard `protoc` compiler flags or service Makefile (`make proto`).
  3. Verify that both client callers and server implementations compile cleanly.

### 3.3 Concurrency, Distributed Transactions & Idempotency
- **Distributed Saga Transactions**:
  - The `order` service coordinates Saga transactions across `inventory`, `payment`, and `logistic`.
  - Every participating microservice MUST support compensating transactions (rollbacks).
  - Every step must record execution state to guarantee forward recovery or safe unwinding upon failure.
- **Mandatory Idempotency**:
  - All payment webhook handlers, stock reservation endpoints, voucher redemption routines, and asynchronous event consumers MUST enforce idempotency.
  - Use `pkg/idempotency` or unique transactional idempotency keys stored in Redis or PostgreSQL with strict expiration windows.
- **Redis Atomic Operations**:
  - High-throughput counters, stock pre-reservations, and voucher claims MUST use Redis Lua scripts or atomic primitives (`DECRBY`, `HINCRBY`, `SET NX EX`). Avoid non-atomic read-then-write sequences.

### 3.4 Data Loss Prevention & Database Safety
- **Forbidden Operations**:
  - Running unconstrained SQL commands: `DROP DATABASE`, `DROP TABLE`, `TRUNCATE`, or `DELETE` without explicit, parameterized `WHERE` clauses.
  - Running blanket filesystem deletion commands (`rm -rf *`, `Remove-Item -Recurse -Force *`) at root or system levels.
- **Database Schema Evolution**:
  - Schema changes must be written as sequential, reversible migrations inside the service's `migrations/` directory.
  - Never alter existing migration files that have already been executed against local or staging databases; create a new versioned migration step.

---

## 4. Environment & Command Execution Protocol

### 4.1 Host Windows (PowerShell) vs WSL2 Linux Boundary
The host development environment has distinct execution boundaries that must never be mixed:

```text
[Host Windows: PowerShell 7 / 5.1]
  ├── Go Compilation (go build, go test, go vet, gofmt)
  ├── Git Operations (git status, git diff, git commit)
  ├── Script orchestration (.\scripts\fdemo.ps1)
  └── Node.js/TypeScript frontend commands (npm, npx)

[WSL2 Ubuntu-22.04 Subsystem]
  ├── Docker Engine 29.3.1 daemon
  ├── Docker Compose 5.1.1 stack execution
  ├── Database initialization scripts
  └── Multi-container infrastructure network
```

### 4.2 Docker Compose Execution Directives
- **WSL2 Invocation Format**:
  All Docker Compose operations MUST be invoked from Windows via WSL2 using the explicit Ubuntu-22.04 distribution:
  ```powershell
  wsl.exe -d Ubuntu-22.04 --cd /mnt/c/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices --exec docker compose -f <compose-file> <command>
  ```
- **Attached Mode Enforcement**:
  - When starting infrastructure (`docker compose up`), **DO NOT append `-d`** unless specifically requested by the human operator.
  - Keeping Compose attached ensures initialization logs, bootstrap scripts, and container failures remain visible in real time.
- **Preservation of External Containers**:
  - Do NOT stop or restart independent developer containers (e.g., `n8n`, personal databases) that are running in the user's WSL instance. Only manage containers defined in `FDemo-microservices` compose files.

### 4.3 Go Service Execution Directives
- Always navigate into the target service directory before executing Go toolchain commands:
  ```powershell
  cd <service-directory>
  gofmt -w .
  go vet ./...
  go test -race -v ./...
  ```
- Do not launch multiple services concurrently using their default configuration if their HTTP/gRPC ports collide on the host. Check `config.yaml` or `.env` before running.

---

## 5. Gateway & API Route Synchronization Rules

1. **Path Appending Behavior**:
   - The `api-gateway` routes incoming requests by appending the client URL (`c.OriginalURL()`) to the downstream microservice base URL.
   - When introducing or altering an endpoint, ensure the path registered in `api-gateway/internal/routes/` perfectly matches the downstream service router path (e.g., `catalog/internal/router/`).
2. **Contract Triangulation**:
   - Whenever an endpoint is updated, verify four surfaces:
     1. Downstream Service Router & Handler.
     2. Gateway Route Proxy Definition.
     3. API Reference Documentation (`API_DOCUMENTATION.md`).
     4. Frontend/Consumer Client DTOs.

---

## 6. Verification & Post-Execution Protocol

Every code change or batch of terminal commands must conclude with a formal verification and recording pass:

### 6.1 Automated Verification Pass
1. **Formatting**: Run `gofmt -w <file>` on all edited Go files.
2. **Linting / Static Analysis**: Run `go vet ./...` in the modified service module.
3. **Automated Testing**: Run unit and integration tests. If tests require databases, ensure database containers are healthy before running.

### 6.2 Mandatory Post-Execution Documentation
Upon concluding an execution run:
1. **Update `run-context/results/latest.md`**:
   - Document the objective, executed commands, files modified, test results, and any unresolved issues or blockers.
2. **Archive Snapshot in `run-context/results/history/`**:
   - Create an immutable file named `YYYY-MM-DD_HH-mm_<short-run-title>.md` following `run-context/results/TEMPLATE.md`.
3. **Language Check**:
   - Verify that all newly created documentation is written 100% in English.
