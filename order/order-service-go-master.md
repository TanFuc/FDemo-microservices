# MISSION: BUILD ORDER SERVICE (GOLANG + POSTGRES + NATS + GRPC)

**Role:** Principal Backend Architect.
**Goal:** Build the `order-service`, the central transaction hub. It handles Order Placement, State Machine management, and Data Snapshotting.

**Tech Stack:**
- **Language:** Go 1.22+.
- **Framework:** `github.com/gofiber/fiber/v2`.
- **Database:** PostgreSQL (`gorm.io/gorm` or `pgx`).
- **Communication:**
    - NATS JetStream (Async Events).
    - gRPC (Sync calls to Inventory Service).
- **Architecture:** Clean Architecture.

---

## PHASE 1: DOMAIN MODELS (`internal/domain`)

### 1. `Order` (The Aggregate Root)
```go
type OrderStatus string
const (
    StatusPending   OrderStatus = "PENDING"   // Created, waiting for payment
    StatusPaid      OrderStatus = "PAID"      // Payment success
    StatusCancelled OrderStatus = "CANCELLED" // Payment failed / User cancelled
    StatusShipped   OrderStatus = "SHIPPED"
    StatusCompleted OrderStatus = "COMPLETED"
)

type Order struct {
    ID              uuid.UUID       `gorm:"type:uuid;primary_key"`
    UserID          uuid.UUID       `gorm:"index"` // From Identity
    
    // --- Money (Use strict decimal) ---
    TotalAmount     decimal.Decimal `gorm:"type:numeric(19,4)"`
    ShippingFee     decimal.Decimal `gorm:"type:numeric(19,4)"`
    DiscountAmount  decimal.Decimal `gorm:"type:numeric(19,4)"`
    FinalAmount     decimal.Decimal `gorm:"type:numeric(19,4)"`

    Status          OrderStatus     `gorm:"index"`
    PaymentMethod   string          // MOMO, COD, STRIPE
    
    // --- Snapshot Data (JSONB) ---
    // Store full address here. If Profile updates address later, this order remains unchanged.
    ShippingAddress json.RawMessage `gorm:"type:jsonb"` 
    
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
2. OrderItem (The Snapshot)
Go

type OrderItem struct {
    ID           uuid.UUID       `gorm:"type:uuid;primary_key"`
    OrderID      uuid.UUID       `gorm:"index"`
    
    // --- Product Snapshot ---
    ProductID    string          // MongoID string
    SkuID        string          // Inventory SkuID
    ProductName  string          // Snapshot name at time of purchase
    SkuCode      string          // Snapshot code
    Thumbnail    string
    
    Quantity     int
    UnitPrice    decimal.Decimal `gorm:"type:numeric(19,4)"` // Price at time of purchase
    SubTotal     decimal.Decimal `gorm:"type:numeric(19,4)"`
}
PHASE 2: INFRASTRUCTURE & GRPC CLIENT
1. Inventory gRPC Client
Implement a client to call InventoryService.ReserveStock.

Logic: Request {skuId, quantity}. Response {success, reservationId}.

Failover: If Inventory is down, Order creation MUST fail immediately.

2. PostgreSQL Repository
Use Transactions (tx) strictly. Creating Order and OrderItems must be atomic.

PHASE 3: USECASE LOGIC (Create Order - The Critical Path)
CreateOrder(ctx, userId, dto)
Validation: Check params.

Price Calculation: (In a real app, we verify prices with Catalog, for now assume DTO sends correct unit prices or verify with a dummy check).

Stock Reservation (gRPC):

Call Inventory Service to reserve stock.

If fail (Out of stock) -> Return Error 409 immediately.

DB Transaction:

Create Order.

Create OrderItems.

Save ShippingAddress snapshot.

Event Publishing (NATS):

Publish order.created.

Payload: { orderId, finalAmount, userId, paymentMethod }.

This triggers Payment Service (future) and Notification Service.

CancelOrder(ctx, orderId)
Check Status: Can only cancel if PENDING.

Update DB: Set Status = CANCELLED.

Release Stock (gRPC/NATS):

Call Inventory Service to ReleaseStock.

Publish: order.cancelled.

PHASE 4: AUTO-VERIFICATION LOOP
Instructions for AI:

Init: Setup Go project.

Generate: Clean Architecture structure.

TEST (Critical): Create internal/usecase/create_order_test.go.

Mock Inventory Client:

Scenario A: Returns "Success". -> Expect Order created in DB with Status PENDING.

Scenario B: Returns "OutOfStock". -> Expect Error, No DB record created.

Build: Run go build.

IMPORTANT RULES
Snapshotting: NEVER reference Product ID lookup for Price/Name in the GetOrder API. Always read from OrderItem table.

Decimal: Use github.com/shopspring/decimal for all money calculations. NEVER use float64.

Idempotency: Optionally implement an IdempotencyKey check (using Redis) to prevent double-submit on the Create Order endpoint.