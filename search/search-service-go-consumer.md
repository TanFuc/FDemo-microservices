# MISSION: BUILD HIGH-PERFORMANCE SEARCH SERVICE (GO + ELASTIC + REDIS + NATS)

**Role:** Principal Search Engineer & Golang Expert.
**Goal:** Build a `search-service` that acts as a CQRS Read-Model. It consumes NATS events to index data into Elasticsearch and provides a high-speed Search API with Redis Caching.

**Tech Stack (Strict):**
- **Language:** Go 1.22+.
- **Framework:** `github.com/gofiber/fiber/v2`.
- **Search Engine:** `github.com/elastic/go-elasticsearch/v8` (Typed API).
- **Cache:** `github.com/redis/go-redis/v9`.
- **Broker:** NATS JetStream (`github.com/nats-io/nats.go`).
- **Architecture:** Clean Architecture (`internal/domain`, `internal/usecase`, `internal/infrastructure`).

---

## PHASE 1: ELASTICSEARCH INFRASTRUCTURE (`internal/infrastructure/elastic`)

### 1. Index Management
- **Index Name:** `products_index`
- **Mapping Strategy (Crucial):**
  On startup, check if index exists. If not, create with this mapping:
  ```json
  {
    "mappings": {
      "properties": {
        "id": { "type": "keyword" },
        "name": { "type": "text", "analyzer": "standard" },
        "slug": { "type": "keyword" },
        "categoryId": { "type": "keyword" },
        "brandId": { "type": "keyword" },
        "price": { "type": "double" },
        "thumbnail": { "type": "keyword" },
        "status": { "type": "keyword" },
        "createdAt": { "type": "date" },
        
        // Dynamic Fields for filtering
        "specs": { "type": "object", "dynamic": true }, 
        "metadata": { "type": "object", "dynamic": true }
      }
    }
  }
Note: Setting dynamic: true allows us to filter by specs.ram or metadata.isFlashSale automatically without manual mapping updates.

2. Client Wrapper
Implement IndexProduct(ctx, product): Uses Client.Index with Document ID = Product ID (Idempotency).

Implement DeleteProduct(ctx, id).

Implement Search(ctx, query): Returns raw Hits.

PHASE 2: NATS CONSUMER WORKER (internal/worker)
1. Worker Pool
Implement a concurrent worker pool (e.g., 10 goroutines) to process high-volume events during bulk imports.

Subscription: Durable Consumer on Stream CATALOG, Subjects catalog.product.>.

2. Event Handling Logic
Event: catalog.product.created OR updated

Unmarshal Payload.

Transform/Map data if necessary (ensure types match Elastic mapping).

Call Elastic.IndexProduct.

ACK: Acknowledge message ONLY after Elastic success.

Event: catalog.product.deleted

Call Elastic.DeleteProduct.

ACK.

PHASE 3: SEARCH USECASE WITH REDIS (internal/usecase)
SearchProducts(ctx, params)
Input: Keyword, CategoryID, BrandID, PriceMin, PriceMax, Specs (Map), Page, Limit.

Cache Strategy (Read-Through):

Key: search:hash(JSON.stringify(params))

TTL: 2 minutes (Short TTL because prices/stock change often).

Check Redis -> If Hit, return JSON.

Build Elastic Query (Bool Query):

Must: multi_match on fields ["name^3", "description"] (Boost name relevance).

Filter:

term: categoryId, brandId, status="PUBLISHED".

range: price.

Dynamic Specs: Loop through params.Specs. Add term filter for each key (e.g., specs.color.keyword: "Red").

Metadata Flags: If params include isFlashSale=true, add filter metadata.isFlashSale: true.

Sort: By _score (relevance) or price/createdAt.

Execute & Cache:

Run Query.

Map Elastic hits to simplified Product Response structs.

Set to Redis.

Return Result.

PHASE 4: API & AUTO-VERIFICATION
Instructions for AI:

Init: Setup Go module, project structure, and dependencies.

Generate:

cmd/worker/main.go: Entry point for NATS Consumer.

cmd/api/main.go: Entry point for Search API.

Internal packages.

TEST (Integration Test Simulation):

Create internal/usecase/search_test.go.

Test Sync: Mock NATS msg -> Run Consumer Handler -> Verify Elastic Index is called.

Test Search:

Mock Elastic Response with 2 hits.

Call SearchProducts.

Verify Redis Set is called.

Build: Run go build -o consumer cmd/worker/main.go AND go build -o api cmd/api/main.go.

IMPORTANT RULES
Resilience: If Elastic is down, Nack the NATS message (or let it timeout) so it retries. Never drop data.

Logging: Use structured logging (slog) with Trace IDs to track a product from Catalog -> NATS -> Search.