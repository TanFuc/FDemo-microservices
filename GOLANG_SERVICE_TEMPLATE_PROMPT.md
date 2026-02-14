# PROMPT: The "Standard & Simple" Golang Architecture (with gRPC)

> **How to use**: 
> 1. Run command: `claude -p GOLANG_SERVICE_TEMPLATE_PROMPT.md`
> 2. This prompt refactors your code into a **Standard, Scalable** structure supporting both HTTP and gRPC.

**Context**: You are a Senior Go Developer. You build services that need to speak both REST (for frontend) and gRPC (for reliable microservice-to-microservice communication). You prefer **Makefiles** for automation and **Standard Folder Structures**.

**Objective**: Refactor the code into strict layers (Model, Repository, Service, Handler) and configure gRPC properly.

## 1. The Directory Structure

```text
service-name/
├── cmd/
│   └── main.go               # Entry point. Starts BOTH HTTP and gRPC servers concurrently.
├── api/
│   └── proto/
│       └── service.proto     # Protocol Buffer definition (No "v1" needed)
├── docs/                     # Swagger generated docs
├── internal/
│   ├── model/                # DATA STRUCTURES (Structs).
│   │   ├── user.go
│   │   └── dto.go            # Request/Response structs
│   │
│   ├── repository/           # DATABASE LAYER.
│   │   ├── interfaces.go     # Interfaces (e.g. UserRepository)
│   │   └── postgres/         # Implementation (e.g. userRepo)
│   │       └── user_repo.go
│   │
│   ├── service/              # BUSINESS LOGIC.
│   │   ├── interfaces.go     # Interfaces (e.g. AuthService)
│   │   └── impl/             # Implementation (e.g. authService)
│   │       └── auth_service.go
│   │
│   ├── handler/              # CONTROLLERS.
│   │   ├── http/             # HTTP Handlers (REST)
│   │   │   └── auth_handler.go
│   │   └── grpc/             # gRPC Handlers (RPC)
│   │       └── auth_handler.go
│   │
│   ├── router/               # HTTP ROUTING.
│   │   └── router.go         # /api/v1/...
│   │
│   ├── config/               # CONFIG.
│   │   └── config.go
│   │
│   └── app/                  # WIRING (DI).
│       └── app.go            # Inits Repos, Services, Handlers, and Servers.
│
├── pkg/
│   └── pb/                   # GENERATED gRPC CODE.
└── Makefile                  # SCRIPTS (protoc, run, build)
```

## 2. Refactoring & Implementation Rules

### A. The gRPC & Proto Workflow
1.  **Proto File**: Created in `api/proto/`. Define messages and services here.
2.  **Generation**: Code MUST be generated into `pkg/pb/`.
3.  **Scripts**: You MUST create a `Makefile` with the following commands:
    *   `proto`: Generates Go code from `.proto` files.
    *   `run`: Runs the application.
    *   `build`: Builds the binary.
    *   `test`: Runs tests.
    *   *Example Proto Command*:
        ```makefile
        gen-proto:
        	protoc --go_out=. --go_opt=module=microservices/your-service \
        	       --go-grpc_out=. --go-grpc_opt=module=microservices/your-service \
        	       api/proto/*.proto
        ```

### B. Handler Split (HTTP vs gRPC)
*   **Legacy Code**: Move existing HTTP logic to `internal/handler/http`.
*   **New Code**: implementation of the gRPC server interface goes in `internal/handler/grpc`.
*   **Logic Sharing**: BOTH handlers must inject the SAME `Service` interface. **Never duplicate business logic.**
    *   `HTTP Handler` -> calls `Service` -> calls `Repo`
    *   `gRPC Handler` -> calls `Service` -> calls `Repo`

### C. The Application Wiring (`app/app.go`)
*   The `Run()` method must start **two** listeners:
    1.  Fiber/Gin/Echo for HTTP (on port 3000/8080).
    2.  gRPC Server (on port 50051).
*   Use a `go routine` for the gRPC server so it doesn't block the HTTP server.

## 3. Execution Plan for Claude Code

1.  **Analyze**: Map current files to the new `http` and `grpc` handler folders.
2.  **Move & Split**:
    *   Move Models -> `internal/model`.
    *   Move Repos -> `internal/repository` (split interface/impl).
    *   Move Services -> `internal/service` (split interface/impl).
    *   Move HTTP Handlers -> `internal/handler/http`.
3.  **Proto Setup**:
    *   Create `api/proto/service.proto` (if not exists, infer from service logic).
    *   Create `Makefile` with `gen-proto` command.
    *   Run generation (if tools available) or provide instruction.
4.  **gRPC Implementation**:
    *   Create `internal/handler/grpc/handler.go`.
    *   Implement the generated interface methods by calling the `Service` layer.
5.  **Wiring**:
    *   Update `internal/app/app.go` to initialize both servers.
    *   Update `cmd/main.go`.

**Constraint**: Keep terminology simple (`Model`, `Repo`, `Service`). Ensure the `Makefile` is practical and working.
