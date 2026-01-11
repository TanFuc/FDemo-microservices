# Cart Service

High-performance cart service using Go, Redis, and MongoDB with write-behind caching strategy.

## Architecture

- **Primary Store (Hot):** Redis - Source of truth for fast reads/writes
- **Persistence Store (Cold):** MongoDB - Async backup for durability
- **Strategy:** Write-Behind Caching - Immediate Redis writes, async MongoDB persistence

## Project Structure

```
cart-service/
├── cmd/
│   └── server/
│       └── main.go           # Entry point
├── internal/
│   ├── domain/
│   │   ├── cart.go           # Cart & CartItem models
│   │   ├── dto.go            # Request/Response DTOs
│   │   └── repository.go     # Repository interfaces
│   ├── infrastructure/
│   │   ├── redis/
│   │   │   ├── client.go     # Redis connection
│   │   │   └── cart_repository.go
│   │   └── mongo/
│   │       ├── client.go     # MongoDB connection
│   │       └── cart_repository.go
│   ├── usecase/
│   │   ├── cart_usecase.go   # Business logic
│   │   └── cart_test.go      # Integration tests
│   └── delivery/
│       └── http/
│           └── handler.go    # Fiber HTTP handlers
├── go.mod
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## Prerequisites

- Go 1.22+
- Redis 7+
- MongoDB 7+
- Docker & Docker Compose (optional)

## Quick Start

### Using Docker Compose (Recommended)

```bash
docker-compose up -d
```

This starts:
- Cart Service on port 8080
- Redis on port 6379
- MongoDB on port 27017

### Manual Setup

1. Install dependencies:
```bash
go mod tidy
```

2. Set environment variables (copy `.env.example` to `.env`):
```bash
PORT=8080
REDIS_HOST=localhost
REDIS_PORT=6379
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=cart_service
```

3. Run the service:
```bash
go run cmd/server/main.go
```

4. Build the binary:
```bash
go build -o server cmd/server/main.go
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/cart/:userId` | Get user's cart |
| POST | `/api/v1/cart/:userId/items` | Add item to cart |
| DELETE | `/api/v1/cart/:userId/items/:skuId` | Remove item |
| PUT | `/api/v1/cart/:userId/items/:skuId/quantity` | Update quantity |
| PUT | `/api/v1/cart/:userId/items/:skuId/selection` | Update selection |
| DELETE | `/api/v1/cart/:userId` | Clear cart |

## API Examples

### Add Item to Cart
```bash
curl -X POST http://localhost:8080/api/v1/cart/user123/items \
  -H "Content-Type: application/json" \
  -d '{
    "skuId": "SKU_001",
    "name": "Product Name",
    "price": 150000,
    "quantity": 2,
    "thumbnail": "http://example.com/image.jpg",
    "selected": true
  }'
```

### Get Cart
```bash
curl http://localhost:8080/api/v1/cart/user123
```

### Update Quantity
```bash
curl -X PUT http://localhost:8080/api/v1/cart/user123/items/SKU_001/quantity \
  -H "Content-Type: application/json" \
  -d '{"quantity": 5}'
```

### Remove Item
```bash
curl -X DELETE http://localhost:8080/api/v1/cart/user123/items/SKU_001
```

### Clear Cart
```bash
curl -X DELETE http://localhost:8080/api/v1/cart/user123
```

## Running Tests

Integration tests require running Redis and MongoDB:

```bash
INTEGRATION_TEST=true go test ./internal/usecase/... -v
```

## Important Notes

1. **Price Validity:** The price stored in cart is for display only. Final price must be re-validated with Catalog Service during Checkout/Order creation.

2. **Max Items:** Cart is limited to 100 unique SKUs to prevent Redis memory abuse.

3. **TTL:** Cart data expires after 30 days of inactivity (TTL refreshed on every interaction).

4. **Concurrency:** Async MongoDB sync uses `context.Background()` (detached context) so it doesn't get cancelled when HTTP request finishes.
