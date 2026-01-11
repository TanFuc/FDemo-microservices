# MISSION: BUILD HIGH-PERFORMANCE CART SERVICE (GO + REDIS + MONGO)

**Role:** Principal Backend Engineer (Golang & High-Performance Systems Expert).
**Goal:** Build a `cart-service` that handles massive "Add to Cart" throughput using Redis as the primary data store, with asynchronous persistence to MongoDB.

**Tech Stack:**
- **Language:** Go 1.22+.
- **Framework:** `github.com/gofiber/fiber/v2` (Chosen for speed).
- **Primary Store (Hot):** Redis (`github.com/redis/go-redis/v9`).
- **Persistence Store (Cold):** MongoDB (`go.mongodb.org/mongo-driver`).
- **Architecture:** Clean Architecture with "Write-Behind" caching strategy.

---

## PHASE 1: DATA STRUCTURES

### 1. Redis Structure (The Source of Truth)
- **Key Pattern:** `cart:{userId}` (Type: Hash).
- **Structure:**
    - **Field:** `{skuId}` (e.g., "SKU_123").
    - **Value:** JSON String.
      ```json
      {
        "skuId": "SKU_123",
        "name": "Áo Thun Coolmate...",
        "price": 150000,
        "quantity": 2,
        "thumbnail": "http://...",
        "selected": true,
        "addedAt": 1709223300
      }
      ```
- **TTL:** 30 Days (Refresh TTL on every interaction).

### 2. MongoDB Schema (The Backup)
- **Collection:** `carts`.
- **Struct:**
    ```go
    type CartItem struct {
        SkuID     string  `bson:"skuId" json:"skuId"`
        Name      string  `bson:"name" json:"name"`
        Price     float64 `bson:"price" json:"price"`
        Quantity  int     `bson:"quantity" json:"quantity"`
        Thumbnail string  `bson:"thumbnail" json:"thumbnail"`
        Selected  bool    `bson:"selected" json:"selected"`
        AddedAt   int64   `bson:"addedAt" json:"addedAt"`
    }

    type Cart struct {
        UserID    string     `bson:"_id" json:"userId"` // UserID is PK
        Items     []CartItem `bson:"items" json:"items"`
        UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
    }
    ```

## PHASE 2: CORE LOGIC (WRITE-BEHIND STRATEGY)

### 1. `AddToCart(ctx, userId, itemDto)`
- **Step 1 (Redis Atomic):**
    - Script: Check if SKU exists in Hash.
    - If exists: Increment Quantity.
    - If new: Set Field.
    - Set Key TTL to 30 days.
- **Step 2 (Async Persistence):**
    - Trigger a Goroutine (fire-and-forget) to sync this user's cart to MongoDB.
    - *Optimization:* Use a debouncer or buffered channel if traffic is insane, but for now, direct async update is fine.

### 2. `GetCart(ctx, userId)`
- **Step 1 (Redis Hit):** `HGETALL cart:{userId}`.
    - If data exists: Parse JSONs -> Return.
- **Step 2 (Redis Miss - Lazy Loading):**
    - If Redis is empty, query MongoDB `carts` collection.
    - If Mongo has data:
        - Pipeline it back to Redis (HMSET).
        - Return data.
    - If Mongo empty: Return empty list.

### 3. `RemoveItem(ctx, userId, skuId)`
- **Redis:** `HDEL cart:{userId} {skuId}`.
- **Mongo:** Async update `$pull` from items array.

### 4. `UpdateQuantity(ctx, userId, skuId, qty)`
- **Redis:** Update the JSON value in Hash.
- **Mongo:** Async update.

### 5. `ClearCart(ctx, userId)` (Called after Order Placed)
- **Redis:** `DEL cart:{userId}`.
- **Mongo:** `DELETE` document.

## PHASE 3: IMPLEMENTATION DETAILS

### 1. Repository Layer
- **RedisRepository:** Handles `HSet`, `HGetAll`, `HDel`.
- **MongoRepository:** Handles `Upsert` (Update or Insert) logic.

### 2. Usecase Layer (The Orchestrator)
- Injects both repositories.
- Handles the "Async Sync" logic using `go func() { ... }` with a recovery mechanism (so panic doesn't kill server).

## PHASE 4: AUTO-VERIFICATION LOOP

**Instructions for AI:**
1.  **Init:** Initialize Go module and folder structure.
2.  **Generate:**
    - `internal/infrastructure`: Redis & Mongo connections.
    - `internal/usecase`: The logic above.
    - `internal/delivery/http`: API Endpoints.
3.  **TEST (Integration Test):**
    - Create `internal/usecase/cart_test.go`.
    - **Test 1 (Add & Retrieve):**
        - Call `AddToCart` (User A, SKU 1, Qty 1).
        - Call `GetCart` (User A). -> Expect SKU 1, Qty 1 (From Redis).
    - **Test 2 (Persistence):**
        - Call `AddToCart` (User B, SKU 2).
        - Sleep 100ms (Wait for async sync).
        - Flush Redis (`FLUSHDB` mock).
        - Call `GetCart` (User B). -> Expect SKU 2 (Loaded from Mongo).
4.  **Build:** Run `go build -o server cmd/server/main.go`.

---

## IMPORTANT RULES
- **Price Validity:** Store price in Cart for display, BUT explicitly comment that: *"Final price must be re-validated with Catalog Service during Checkout/Order creation."*
- **Max Items:** Limit cart to 100 SKUs to prevent Redis memory abuse.
- **Concurrency:** The Async Sync must use `context.Background()` (detached context) so it doesn't get cancelled when the HTTP request finishes.