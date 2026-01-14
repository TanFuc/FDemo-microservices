# Cache Package Architecture

## Overview

This cache package provides a clean, well-organized, and production-ready caching solution for Go microservices. The architecture follows Go best practices with a flat package structure for core functionality and sub-packages for implementations.

## Package Structure

### Root Package (`cache`)

All core types, interfaces, and utilities are in the main `cache` package for easy importing:

```go
import "microservices/pkg/cache"
```

#### Core Files (8 files - consolidated & clean)

| File | Purpose | Key Exports | Lines |
|------|---------|-------------|-------|
| `cache.go` | Interface definitions | `Cache`, `BatchCache`, `PatternCache`, `ListCache`, `MetricsCache`, `CRUDCache` | ~200 |
| `factory.go` | Factory functions | `New()`, `NewRedis()`, `NewMemory()`, `NewMultilayer()`, `NewWithCRUD()` | ~200 |
| `types.go` | **All type definitions** | `Config`, `RedisConfig`, `MemoryConfig`, `CacheType`<br>`ErrCacheMiss`, `ErrInvalidValue`, etc.<br>`Option`, `WithLogger()`, `WithMetrics()` | ~500 |
| `helpers.go` | **All helper utilities** | `Serializer`, `JSONSerializer`<br>`GetTyped[T]()`, `SetTyped[T]()`, `Remember[T]()` | ~350 |
| `keybuilder.go` | Flexible key generation | `KeyBuilder`, `NewKeyBuilder()`, `ForItem()`, `ForList()` | ~250 |
| `crud.go` | CRUD with cache sync | `CRUDCache`, `NewCRUDCache()`, `CreateWithListSync()` | ~280 |
| `metrics.go` | Metrics collection | `Metrics`, `CacheStats`, `NewMetrics()` | ~150 |
| `cache_test.go` | Core tests | All test functions | ~350 |

**Total: 8 files (previously 15+ files)**

### Sub-Packages

#### `redis/` - Redis Implementation
```go
import "microservices/pkg/cache/redis"
```

- **redis.go**: Main Redis cache implementation
- **cluster.go**: Redis Cluster support
- **pipeline.go**: Pipeline operations for batch requests
- **list.go**: List/array cache operations

#### `memory/` - In-Memory Implementation
```go
import "microservices/pkg/cache/memory"
```

- **memory.go**: In-memory cache implementation
- **lru.go**: LRU (Least Recently Used) eviction policy
- **memory_test.go**: Comprehensive tests

#### `multilayer/` - Multi-Layer Cache
```go
import "microservices/pkg/cache/multilayer"
```

- **multilayer.go**: L1 (memory) + L2 (Redis) implementation

#### `examples/` - Usage Examples
```go
import "microservices/pkg/cache/examples"
```

- **basic_usage.go**: Simple examples
- **redis_example.go**: Redis-specific usage
- **crud_example.go**: CRUD operations demonstration

## Design Principles

### 1. Flat Root Structure
- **8 core files** at root (down from 15+)
- **Consolidated related code**: types.go (config+errors+options), helpers.go (serialization+generics)
- Easy to import: `cache.Cache`, `cache.Config`, `cache.KeyBuilder`
- No deep package nesting for commonly used types
- Follows Go convention: simple is better than complex

### 2. Consolidation Strategy
- **types.go**: All type definitions in one place
  - Configuration structs (Config, RedisConfig, MemoryConfig)
  - Error definitions and helpers
  - Functional options pattern
- **helpers.go**: All utility functions together
  - Serialization (JSON, MessagePack support)
  - Generic type-safe helpers (GetTyped, SetTyped, Remember)
  - Cache-aside pattern implementations

### 2. Interface Segregation
- Multiple small interfaces (`Cache`, `BatchCache`, `PatternCache`, etc.)
- Clients depend only on interfaces they use
- Easy to mock for testing
- Clear contracts between components

### 3. Sub-Packages for Implementations
- Each backend has its own package
- Implementation details are hidden
- Users interact through main `Cache` interface
- Easy to add new backends without changing core

### 4. Clean Naming Convention
- Files are named by their primary purpose
- `config.go` → configuration types
- `errors.go` → error definitions
- `crud.go` → CRUD operations
- Self-documenting file organization

## Import Patterns

### Most Common Import (Single Package)
```go
import "microservices/pkg/cache"

// Access everything through cache package
c, _ := cache.New(cache.Config{...})
kb := cache.NewKeyBuilder(":")
```

### Direct Backend Import
```go
import (
    "microservices/pkg/cache"
    "microservices/pkg/cache/redis"
)

// Create Redis cache directly
c, _ := redis.New(&cache.RedisConfig{...}, nil)
```

### Multi-layer Setup
```go
import (
    "microservices/pkg/cache/memory"
    "microservices/pkg/cache/redis"
    "microservices/pkg/cache/multilayer"
)

l1, _ := memory.New(...)
l2, _ := redis.New(...)
ml, _ := multilayer.New(l1, l2)
```

## Component Relationships

```
┌─────────────────────────────────────────────────────────────┐
│                    User Application                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ imports cache package
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              cache (Root Package)                            │
│                                                              │
│  Interfaces:                                                │
│    • Cache, BatchCache, PatternCache                        │
│    • ListCache, MetricsCache, CRUDCache                     │
│                                                              │
│  Factory:                                                   │
│    • New(), NewRedis(), NewMemory()                         │
│                                                              │
│  Utilities:                                                 │
│    • KeyBuilder, Serializer, Metrics                        │
│    • Generic helpers, CRUD helpers                          │
│                                                              │
│  Types:                                                     │
│    • Config, RedisConfig, MemoryConfig                      │
│    • Errors, Options                                        │
└───┬──────────────────┬──────────────────┬──────────────────┘
    │                  │                  │
    ▼                  ▼                  ▼
┌─────────┐      ┌──────────┐      ┌────────────┐
│ redis/  │      │ memory/  │      │multilayer/ │
│         │      │          │      │            │
│ • Redis │      │ • Memory │      │ • L1 + L2  │
│   impl  │      │   impl   │      │   cache    │
│ • Cluster│     │ • LRU    │      │            │
│ • Pipeline│    │          │      │            │
└─────────┘      └──────────┘      └────────────┘
```

## Extension Points

### Adding a New Backend

1. Create new sub-package: `pkg/cache/newbackend/`
2. Implement `cache.Cache` interface
3. Optionally implement `BatchCache`, `PatternCache`, `ListCache`
4. Add factory function in `factory.go`
5. Add configuration in `config.go`

### Adding a New Feature

1. Add interface method to appropriate interface in `cache.go`
2. Implement in all backends (`redis/`, `memory/`, `multilayer/`)
3. Add tests in `cache_test.go` or backend-specific tests
4. Update documentation and examples

## Testing Strategy

- **Unit tests**: Test interfaces and utilities (`cache_test.go`)
- **Integration tests**: Test with real backends (memory_test.go)
- **Example tests**: Runnable examples in `examples/`
- **Benchmark tests**: Performance validation

## Why This Structure?

### Advantages

✅ **Simple imports**: Single import for most use cases  
✅ **Clean organization**: Files grouped by purpose  
✅ **Easy navigation**: Flat structure, no deep nesting  
✅ **Extensible**: Easy to add new backends or features  
✅ **Testable**: Clear separation of concerns  
✅ **Go idiomatic**: Follows Go community standards  
✅ **Zero breaking changes**: Implementation details hidden  

### Comparison with Alternatives

| Approach | Pros | Cons |
|----------|------|------|
| **Current (Flat)** | ✅ Simple imports<br>✅ Easy to find code | ⚠️ Many files at root |
| **Deep nesting** | ✅ Very organized | ❌ Complex imports<br>❌ Hard to navigate |
| **Single file** | ✅ Everything in one place | ❌ Huge file<br>❌ Hard to maintain |

## Best Practices

### For Package Users

1. Import only `cache` package for most use cases
2. Use factory functions (`cache.New()`) instead of direct constructors
3. Use generic helpers (`GetTyped[T]`) for type safety
4. Use CRUD helper for automatic cache synchronization

### For Package Maintainers

1. Keep interfaces small and focused
2. Add new features as optional interfaces
3. Maintain backward compatibility
4. Keep root package clean (only essential files)
5. Document all exported types and functions
6. Write tests for all new features

## Migration Guide

If you have old code using the previous structure, migration is seamless:

```go
// Old import (still works)
import "microservices/pkg/cache"

// All code continues to work
c, _ := cache.New(...)
kb := cache.NewKeyBuilder(":")
```

No changes needed! The internal reorganization doesn't affect the public API.
