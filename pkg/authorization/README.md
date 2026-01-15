# Authorization Package

A production-ready Casbin-based RBAC/ABAC authorization system for Go microservices.

## Features

- **RBAC Support**: Role-Based Access Control with role hierarchy
- **ABAC Support**: Attribute-Based Access Control with dynamic conditions
- **Hybrid Model**: Combined RBAC+ABAC for complex scenarios
- **Multi-tenancy**: Domain and tenant isolation
- **Caching**: Redis-based permission decision caching
- **Distributed**: Policy synchronization across instances via Redis pub/sub
- **Middleware**: HTTP (Fiber) and gRPC interceptors
- **Performance**: Sub-millisecond enforcement with caching

## Installation

```bash
go get microservices/pkg/authorization
```

## Quick Start

### Basic RBAC Setup

```go
package main

import (
    "context"
    "microservices/pkg/authorization"
    _ "microservices/pkg/authorization/casbin" // Register Casbin implementation
)

func main() {
    // Create configuration
    cfg := &authorization.Config{
        ModelType: authorization.ModelTypeRBAC,
        Database: &authorization.DatabaseConfig{
            Host:     "localhost",
            Port:     5432,
            User:     "postgres",
            Password: "password",
            Database: "authz",
        },
        Cache: &authorization.CacheConfig{
            Enabled: true,
            Type:    "redis",
            Redis: &authorization.RedisConfig{
                Addr: "localhost:6379",
            },
        },
    }

    // Create authorizer
    auth, err := authorization.New(cfg)
    if err != nil {
        panic(err)
    }
    defer auth.Close()

    ctx := context.Background()

    // Create a role
    adminRole := &authorization.Role{
        ID:   "admin",
        Name: "Administrator",
    }
    auth.AddRole(ctx, adminRole)

    // Add permission to role
    perm := &authorization.Permission{
        ID:       "orders-read",
        Resource: "orders",
        Action:   "read",
    }
    auth.AddPermission(ctx, perm)
    auth.AssignPermissionToRole(ctx, "admin", "orders-read")

    // Assign role to user
    auth.AssignRole(ctx, "user123", "admin")

    // Check permission
    result, _ := auth.Enforce(ctx, &authorization.AuthRequest{
        Subject: "user123",
        Object:  "orders",
        Action:  "read",
    })

    if result.Allowed {
        fmt.Println("Access granted!")
    }
}
```

### HTTP Middleware (Fiber)

```go
import (
    "github.com/gofiber/fiber/v2"
    "microservices/pkg/authorization"
)

func main() {
    app := fiber.New()

    // Create authorizer
    auth, _ := authorization.New(cfg)

    // Apply middleware to all routes
    app.Use(authorization.NewMiddleware(auth))

    // Or with custom configuration
    app.Use(authorization.NewMiddlewareWithConfig(authorization.MiddlewareConfig{
        Authorizer: auth,
        SubjectExtractor: func(c *fiber.Ctx) string {
            return c.Locals("user_id").(string)
        },
        Skipper: func(c *fiber.Ctx) bool {
            return c.Path() == "/health"
        },
    }))

    // Require specific role
    app.Get("/admin", authorization.RequireRole(auth, "admin"), adminHandler)

    // Require specific permission
    app.Delete("/orders/:id", authorization.RequirePermission(auth, "orders", "delete"), deleteOrderHandler)
}
```

### gRPC Interceptor

```go
import (
    "google.golang.org/grpc"
    "microservices/pkg/authorization"
)

func main() {
    auth, _ := authorization.New(cfg)

    server := grpc.NewServer(
        grpc.UnaryInterceptor(authorization.UnaryServerInterceptor(auth)),
        grpc.StreamInterceptor(authorization.StreamServerInterceptor(auth)),
    )
}
```

### ABAC with Attributes

```go
// Add ABAC policy
policy := &authorization.Policy{
    Type:    authorization.PolicyTypeABAC,
    Subject: "user",
    Object:  "order",
    Action:  "update",
    Effect:  authorization.EffectAllow,
    Conditions: []*authorization.Condition{
        {
            Field:    "user.department",
            Operator: authorization.OpEquals,
            Value:    "order.department",
        },
        {
            Field:    "order.status",
            Operator: authorization.OpIn,
            Value:    []string{"pending", "processing"},
        },
    },
}
auth.AddPolicy(ctx, policy)

// Check with attributes
result, _ := auth.CheckResourceAccess(ctx, "user123", "order", "update", map[string]interface{}{
    "user": map[string]interface{}{
        "department": "sales",
    },
    "order": map[string]interface{}{
        "department": "sales",
        "status":     "pending",
    },
})
```

## Configuration

```go
type Config struct {
    // Model type: "rbac", "abac", or "hybrid"
    ModelType ModelType

    // Database configuration
    Database *DatabaseConfig

    // Redis cache configuration
    Cache *CacheConfig

    // Distributed watcher configuration
    Watcher *WatcherConfig

    // Multi-tenancy defaults
    DefaultTenantID string
    DefaultDomain   string

    // Performance settings
    AutoLoadPolicy bool
    AutoSavePolicy bool
    BatchSize      int
    QueryTimeout   time.Duration
}
```

## Interfaces

### Authorizer

```go
type Authorizer interface {
    // Core enforcement
    Enforce(ctx context.Context, request *AuthRequest) (*AuthResult, error)
    EnforceBatch(ctx context.Context, requests []*AuthRequest) ([]*AuthResult, error)

    // Policy management
    AddPolicy(ctx context.Context, policy *Policy) error
    RemovePolicy(ctx context.Context, policy *Policy) error
    GetPolicies(ctx context.Context, filter *PolicyFilter) ([]*Policy, error)

    // Role management
    AddRole(ctx context.Context, role *Role) error
    AssignRole(ctx context.Context, userID, roleID string, domain ...string) error
    RevokeRole(ctx context.Context, userID, roleID string, domain ...string) error
    GetUserRoles(ctx context.Context, userID string, domain ...string) ([]*Role, error)

    // Permission management
    AddPermission(ctx context.Context, perm *Permission) error
    AssignPermissionToRole(ctx context.Context, roleID, permID string) error
    GetRolePermissions(ctx context.Context, roleID string) ([]*Permission, error)

    // Health & stats
    HealthCheck(ctx context.Context) error
    GetStats(ctx context.Context) (*AuthStats, error)
}
```

## Casbin Models

### RBAC Model
- Supports role hierarchy
- Domain/tenant isolation
- Allow/deny effects

### ABAC Model
- Attribute-based conditions
- Dynamic rule evaluation
- Custom functions

### Hybrid Model
- Combines RBAC and ABAC
- Role-based with attribute conditions

## Performance

- **Caching**: Permission decisions cached in Redis with configurable TTL
- **Batch operations**: Check multiple permissions in single call
- **Policy compilation**: Pre-compiled policies for fast evaluation
- **Connection pooling**: Optimized database connections

## Testing

```bash
go test ./pkg/authorization/...
```

## License

MIT
