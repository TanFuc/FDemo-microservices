# MISSION: BUILD HIGH-PERFORMANCE INVENTORY SERVICE (GO + REDIS + POSTGRES)

**Role:** Principal Backend Engineer (Golang & Distributed Systems Expert).
**Goal:** Build a production-ready `inventory-service` capable of handling massive concurrency (e.g., Flash Sales) without overselling, using a Two-Phase Reservation pattern.

**Tech Stack:**
- **Language:** Go 1.22+.
- **Framework:** `github.com/gofiber/fiber/v2`.
- **Database:** PostgreSQL (`gorm.io/gorm` or `pgx`).
- **Cache/Locking:** Redis (`github.com/redis/go-redis/v9`) with Lua Scripts.
- **Architecture:** Clean Architecture (`internal/domain`, `internal/usecase`, `internal/repository`).

---

## PHASE 1: DATABASE SCHEMA (POSTGRESQL)

### 1. `InventoryItem` (The Ledger)
- **Table:** `inventory_items`
- **Fields:**
    - `SkuID` (String, Primary Key or Unique Index).
    - `TotalStock` (Int) - Physical count in warehouse.
    - `ReservedStock` (Int) - Amount currently held by pending orders.
    - `UpdatedAt` (Timestamp).
- **Constraint:** `TotalStock >= ReservedStock` (Always).

### 2. `StockReservation` (The Transaction Log)
- **Table:** `stock_reservations`
- **Fields:**
    - `ID` (UUID).
    - `OrderID` (String, Indexed).
    - `SkuID` (String).
    - `Quantity` (Int).
    - `Status` (Enum: PENDING, CONFIRMED, CANCELLED).
    - `ExpiresAt` (Timestamp) - For cleanup jobs.

## PHASE 2: REDIS LUA SCRIPTING (THE GUARD)

Create a Lua script to handle atomic reservation checks in Redis.

- **Key:** `inventory:{skuId}` (Hash: `total`, `reserved`).
- **Logic (`reserve.lua`):**
    ```lua
    local key = KEYS[1]
    local qty = tonumber(ARGV[1])
    
    local total = tonumber(redis.call("HGET", key, "total") or 0)
    local reserved = tonumber(redis.call("HGET", key, "reserved") or 0)
    
    if (total - reserved) >= qty then
        redis.call("HINCRBY", key, "reserved", qty)
        return 1 -- Success
    else
        return 0 -- Failed (Out of Stock)
    end
    ```

## PHASE 3: USECASE LOGIC (CORE)

### 1. `ReserveStock(ctx, orderId, items)`
- **Step 1 (Redis Fast-Fail):**
    - Iterate through items. Execute Lua Script for each.
    - **Critical:** If ANY item fails, strictly **Rollback** (decrement reserved) for all previously successful items in this batch and return Error.
- **Step 2 (Postgres Persistence):**
    - Open Transaction (`tx`).
    - Create `StockReservation` records (Status: PENDING).
    - Update `InventoryItem`: `SET reserved_stock = reserved_stock + ?`.
    - Commit `tx`.
    - *Note:* If DB commit fails, rollback Redis.

### 2. `ConfirmStock(ctx, orderId)` (Called via gRPC/NATS after Payment)
- **Logic:**
    - Find Reservations by `orderId`.
    - **DB Transaction:**
        - Update `InventoryItem`:
            - `total_stock = total_stock - qty`
            - `reserved_stock = reserved_stock - qty`
        - Update Reservation Status = CONFIRMED.
    - **Redis Update:** Sync new `total` and `reserved` values to Redis to keep it consistent.

### 3. `ReleaseStock(ctx, orderId)` (Called on Cancel/Timeout)
- **Logic:**
    - Find Reservations by `orderId` (Status PENDING).
    - **DB Transaction:**
        - Update `InventoryItem`: `reserved_stock = reserved_stock - qty`.
        - Update Reservation Status = CANCELLED.
    - **Redis Update:** Decrement `reserved` in Redis.

## PHASE 4: AUTO-VERIFICATION LOOP

**Instructions for AI:**
1.  **Init:** `go mod init inventory-service`. Setup project structure.
2.  **Generate:**
    - `internal/infrastructure`: Redis & Postgres connection.
    - `internal/usecase`: The 3 core methods above.
    - `scripts/reserve.lua`: The Lua script file.
3.  **TEST (Concurrency Stress Test):**
    - Create `internal/usecase/inventory_stress_test.go`.
    - **Scenario:**
        - Init SKU "A" with Total=100, Reserved=0.
        - Launch **150 Goroutines** simultaneously, each trying to Reserve 1 item.
    - **Expectation:**
        - Exactly 100 requests succeed.
        - Exactly 50 requests fail.
        - Final Redis State: Total=100, Reserved=100.
        - Final DB State: ReservedStock=100.
4.  **Build:** Run `go build`.

---

## IMPORTANT RULES
- **Idempotency:** `ConfirmStock` might be called twice (network retry). Check `Status` before deducting. If already CONFIRMED, return Success immediately.
- **Syncing:** Add a helper `SyncRedisFromDB(skuId)` to restore Redis state from Postgres in case Redis crashes/restarts.