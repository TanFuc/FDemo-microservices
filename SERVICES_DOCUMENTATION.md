# TÀI LIỆU HỆ THỐNG MICROSERVICES E-COMMERCE

## Tổng Quan Hệ Thống

Hệ thống E-commerce được xây dựng theo kiến trúc **Microservices**, bao gồm **15 services** độc lập nhưng được liên kết chặt chẽ thông qua các cơ chế giao tiếp đồng bộ (gRPC, HTTP) và bất đồng bộ (NATS JetStream, RabbitMQ).

### Sơ Đồ Kiến Trúc Tổng Quan

```
                                    ┌─────────────────────────────────┐
                                    │         API GATEWAY             │
                                    │     (Go + Fiber + Proxy)        │
                                    └─────────────────────────────────┘
                                                    │
                    ┌───────────────────────────────┼───────────────────────────────┐
                    │                               │                               │
        ┌───────────▼───────────┐     ┌─────────────▼───────────────┐     ┌────────▼────────┐
        │        AUTH           │     │         CATALOG             │     │      CART       │
        │      (Go + PG)        │     │      (Go + MongoDB)         │     │ (Go + Redis)    │
        └───────────────────────┘     └─────────────────────────────┘     └─────────────────┘
                    │                               │                               │
                    │                    ┌──────────┴──────────┐                    │
                    │                    │                     │                    │
        ┌───────────▼───────────┐       │         ┌────────────▼────────────┐      │
        │       PROFILE         │       │         │         SEARCH          │      │
        │   (Go + MongoDB)      │       │         │   (Go + Elasticsearch)  │      │
        └───────────────────────┘       │         └─────────────────────────┘      │
                                        │                                           │
                              ┌─────────▼─────────┐                  ┌──────────────▼──────────────┐
                              │     INVENTORY     │◄─────────────────│           ORDER             │
                              │  (Go + PG + Redis)│   (gRPC/NATS)    │     (Go + PG + NATS)        │
                              └───────────────────┘                  └─────────────────────────────┘
                                        │                                           │
                              ┌─────────┴─────────┐                  ┌──────────────┴──────────────┐
                              │                   │                  │                              │
                    ┌─────────▼─────────┐  ┌──────▼───────┐  ┌───────▼───────┐  ┌──────────────────▼───────────────────┐
                    │      PAYMENT      │  │   LOGISTIC   │  │   CAMPAIGN    │  │          NOTIFICATION                │
                    │    (Go + PG)      │  │  (Go + PG)   │  │ (Go + PG)     │  │    (Go + RabbitMQ + MongoDB)         │
                    └───────────────────┘  └──────────────┘  └───────────────┘  └──────────────────────────────────────┘
                              │                   │                  │                              │
                    ┌─────────┴───────────────────┴──────────────────┴──────────────────────────────┘
                    │
        ┌───────────▼───────────┐     ┌─────────────────────────────┐     ┌─────────────────────────┐
        │        MEDIA          │     │         ANALYTIC            │     │         REVIEW          │
        │    (Go + MinIO)       │     │     (Go + ClickHouse)       │     │   (Go + MongoDB)        │
        └───────────────────────┘     └─────────────────────────────┘     └─────────────────────────┘
```

---

## Chi Tiết Từng Service

### 1. 🚪 API-GATEWAY
**Công nghệ:** Go 1.22+ | Fiber | Redis | JWT

**Mục đích:** Cổng vào duy nhất (Single Entry Point) cho toàn bộ hệ thống E-commerce.

**Chức năng chính:**
- **Routing & Reverse Proxy:** Định tuyến requests đến các downstream services
- **Authentication:** Xác thực JWT token cho các protected routes
- **Rate Limiting:** Giới hạn số lượng request (100 req/60s) sử dụng Redis
- **Load Balancing:** Phân phối tải đến các service instances

**Endpoints đang route:**
| Route | Target Service | Auth Required |
|-------|---------------|---------------|
| `/api/v1/identity/*` | identity-service | Partial |
| `/api/v1/catalog/*` | catalog-service | Partial |
| `/api/v1/cart/*` | cart-service | ✅ Yes |
| `/api/v1/orders/*` | order-service | ✅ Yes |

**Trạng thái:** ✅ Đã implement cơ bản

---

### 2. 🔐 AUTH (Identity Service)
**Công nghệ:** Go 1.22+ | Fiber | PostgreSQL | Redis | JWT

**Mục đích:** Quản lý xác thực và phân quyền người dùng.

**Chức năng chính:**
- **Authentication:** Register, Login, Logout
- **Token Management:** Access Token (15m), Refresh Token (7d) với Token Rotation
- **RBAC:** Role-Based Access Control (CUSTOMER, SELLER, ADMIN)
- **Permission Caching:** Cache permissions vào Redis với TTL 1 giờ
- **Token Blacklist:** Quản lý revoked tokens

**Entities chính:**
- `User` - Thông tin người dùng
- `Role` - Vai trò (CUSTOMER, SELLER, ADMIN)
- `Permission` - Quyền hạn (product:create, order:read_own, etc.)
- `RolePermission` - Mapping role-permission
- `RefreshToken` - Lưu refresh tokens với device tracking

**API Endpoints:**
| Method | Endpoint | Mô tả |
|--------|----------|-------|
| POST | `/auth/register` | Đăng ký tài khoản mới |
| POST | `/auth/login` | Đăng nhập, nhận tokens |
| POST | `/auth/refresh` | Làm mới access token |
| POST | `/auth/logout` | Đăng xuất, revoke token |
| GET | `/auth/profile` | Lấy thông tin profile |

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 3. 👤 PROFILE
**Công nghệ:** Go 1.22+ | Fiber | MongoDB

**Mục đích:** Quản lý thông tin cá nhân, địa chỉ giao hàng và Shop profiles.

**Chức năng chính:**
- **User Profile:** displayName, avatarUrl, bio
- **Address Book:** Quản lý nhiều địa chỉ ship với logic "Default Address"
- **Shop Management:** Đăng ký làm Seller, quản lý thông tin Shop

**Collections chính:**
- `profiles` - Thông tin user + shopConfig (nếu là Seller)
- `addresses` - Sổ địa chỉ với compound index trên userId

**API Endpoints:**
| Method | Endpoint | Mô tả |
|--------|----------|-------|
| GET | `/profiles/me` | Lấy/Tạo profile (Lazy Creation) |
| PATCH | `/profiles/me` | Cập nhật profile |
| POST | `/profiles/me/shop` | Đăng ký/Cập nhật Shop |
| GET | `/profiles/me/addresses` | Danh sách địa chỉ |
| POST | `/profiles/me/addresses` | Thêm địa chỉ mới |
| PATCH | `/profiles/me/addresses/:id/set-default` | Đặt làm mặc định |

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 4. 📦 CATALOG
**Công nghệ:** Go 1.22+ | Fiber | MongoDB | Redis | NATS JetStream

**Mục đích:** Quản lý danh mục sản phẩm, thương hiệu và metadata linh hoạt.

**Chức năng chính:**
- **Category Management:** Quản lý danh mục có attribute definitions
- **Brand Management:** Quản lý thương hiệu với status
- **Product Management:** Sản phẩm với Variations (SKUs) và Metadata động
- **Event Publishing:** Publish events khi CRUD sản phẩm (cho Search Service)

**Domain Models chính:**
- `Category` - Danh mục với AttributeDefinitions
- `Brand` - Thương hiệu
- `Product` - Sản phẩm với Variations, Specs, Media, Metadata
- `Variation` - SKU với giá, stock ảo, attributes

**NATS Events published:**
- `catalog.product.created` → Trigger Search indexing
- `catalog.product.updated` → Trigger Search re-indexing
- `catalog.product.deleted` → Trigger Search removal

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 5. 🛒 CART
**Công nghệ:** Go 1.22+ | Fiber | Redis | MongoDB

**Mục đích:** Giỏ hàng hiệu suất cao với "Write-Behind Caching" strategy.

**Chức năng chính:**
- **Add to Cart:** Thêm sản phẩm với Redis Atomic operations
- **Get Cart:** Redis-first với Lazy Loading từ MongoDB
- **Update/Remove:** Cập nhật số lượng, xóa items
- **Clear Cart:** Xóa giỏ hàng sau khi đặt order

**Data Structure:**
- **Redis:** Hash `cart:{userId}` với TTL 30 ngày
- **MongoDB:** Collection `carts` làm backup persistence

**Logic đặc biệt:**
- Write-Behind Strategy: Ghi Redis trước, async sync xuống MongoDB
- Lazy Loading: Cache miss → Load từ MongoDB → Hydrate Redis

**API Endpoints:**
| Method | Endpoint | Mô tả |
|--------|----------|-------|
| GET | `/cart` | Lấy giỏ hàng |
| POST | `/cart/items` | Thêm sản phẩm |
| PUT | `/cart/items/:skuId` | Cập nhật số lượng |
| DELETE | `/cart/items/:skuId` | Xóa sản phẩm |
| DELETE | `/cart` | Xóa giỏ hàng |

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 6. 📋 ORDER
**Công nghệ:** Go 1.22+ | Fiber | PostgreSQL | NATS JetStream | gRPC

**Mục đích:** Trung tâm giao dịch - Quản lý vòng đời đơn hàng.

**Chức năng chính:**
- **Order Creation:** Tạo đơn với Data Snapshotting
- **State Machine:** PENDING → PAID → SHIPPED → COMPLETED / CANCELLED
- **Stock Reservation:** Gọi Inventory Service qua gRPC
- **Event Publishing:** Publish order events cho các services khác

**Domain Models:**
- `Order` - Aggregate Root với status, amounts, shipping address (JSONB)
- `OrderItem` - Snapshot data (price, name, thumbnail tại thời điểm mua)

**Service Interactions:**
| Target Service | Protocol | Purpose |
|----------------|----------|---------|
| Inventory Service | gRPC | Reserve/Release/Confirm Stock |
| Payment Service | NATS | Trigger payment flow |
| Notification Service | NATS | Trigger email notifications |

**NATS Events published:**
- `order.created` → Trigger Payment + Notification
- `order.cancelled` → Release stock, notify user
- `order.paid` → Confirm stock, update status

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 7. 📊 INVENTORY
**Công nghệ:** Go 1.22+ | Fiber | PostgreSQL | Redis (Lua Scripts)

**Mục đích:** Quản lý tồn kho với Two-Phase Reservation pattern.

**Chức năng chính:**
- **Reserve Stock:** Giữ hàng khi tạo order (TTL 30 phút)
- **Confirm Stock:** Xác nhận trừ kho khi payment success
- **Release Stock:** Hoàn lại kho khi order cancelled/timeout
- **Sync Redis-DB:** Đồng bộ state giữa Redis và PostgreSQL

**Database Tables:**
- `inventory_items` - TotalStock, ReservedStock
- `stock_reservations` - Transaction log với status, expiresAt

**Redis Lua Script (`reserve.lua`):**
```lua
-- Atomic check-and-reserve
if (total - reserved) >= qty then
    HINCRBY reserved qty
    return 1 -- Success
else
    return 0 -- Out of Stock
end
```

**gRPC Methods:**
- `ReserveStock(skuId, qty, orderId)` → reservationId
- `ConfirmStock(orderId)`
- `ReleaseStock(orderId)`

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 8. 💳 PAYMENT
**Công nghệ:** Go 1.22+ | Fiber | PostgreSQL | NATS JetStream

**Mục đích:** Payment Aggregator - Tích hợp nhiều cổng thanh toán.

**Chức năng chính:**
- **Multi-Gateway Support:** Stripe, MoMo, VNPay, COD
- **Webhook Handling:** Xác thực signature, idempotency
- **Transaction Ledger:** Lưu trữ audit trail
- **Reconciliation:** Cron job đối soát mỗi 10 phút

**Database Tables:**
- `payment_transactions` - OrderID, Amount, Provider, Status, Metadata (JSONB)
- `payment_logs` - Audit trail cho debugging

**Payment Gateway Interface:**
```go
type PaymentGateway interface {
    CreatePayment(ctx, req) (*PaymentResponse, error)
    VerifyWebhook(r *http.Request) (bool, *WebhookData, error)
}
```

**NATS Events published:**
- `payment.processed` → Order Service updates to PAID
- `payment.failed` → Trigger order cancellation

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 9. 🚚 LOGISTIC
**Công nghệ:** Go 1.22+ | Fiber | PostgreSQL | Redis | NATS

**Mục đích:** Logistics Aggregator - Tích hợp các đơn vị vận chuyển.

**Chức năng chính:**
- **Multi-Carrier Support:** GHN, GHTK, ViettelPost, Mock
- **Fee Calculation:** Tính phí ship với caching
- **Shipment Creation:** Tạo vận đơn tự động
- **Webhook Processing:** Nhận cập nhật trạng thái từ carriers

**Database Tables:**
- `shipping_orders` - TrackingCode, CarrierStatus, SystemStatus, LabelURL

**Provider Interface:**
```go
type Provider interface {
    GetName() string
    CalculateFee(ctx, req *RateRequest) (float64, error)
    CreateOrder(ctx, req *ShipRequest) (trackingCode, labelUrl, error)
    ParseWebhook(r *http.Request) (internalOrderID, newStatus, error)
}
```

**Status Mapping:**
| Carrier Status | System Status |
|----------------|---------------|
| ready_to_pick | PENDING |
| delivering | SHIPPING |
| delivered | DELIVERED |
| return | RETURNED |

**NATS Events published:**
- `logistics.shipment.created`
- `logistics.status.updated` → Order completes

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 10. 📧 NOTIFICATION
**Công nghệ:** Go 1.22+ | NATS | RabbitMQ | MongoDB | Gomail

**Mục đích:** Notification Hub với Dual-Consumer architecture.

**Chức năng chính:**
- **NATS Bridge:** Lắng nghe events từ các services
- **RabbitMQ Worker:** Xử lý gửi Email/Push với DLQ
- **Template Engine:** Handlebars templates
- **Audit Logging:** Lưu log vào MongoDB

**Architecture Pattern:**
```
NATS Events → Bridge → RabbitMQ → Worker → Email/Push
                           ↓
                     Dead Letter Queue (DLQ)
```

**RabbitMQ Topology:**
- Exchange: `notification.exchange` (Topic)
- Queue: `queue.email` với DLX binding
- DLX: `notification.dlx` → `queue.dead_letter`

**NATS Subscriptions:**
- `order.created` → Email xác nhận đơn hàng
- `payment.processed` → Email biên lai thanh toán
- `logistics.status.updated` → Email cập nhật vận chuyển

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 11. 🖼️ MEDIA
**Công nghệ:** Go 1.22+ | Fiber | MinIO (S3) | NATS

**Mục đích:** Media Service với Presigned URL pattern.

**Chức năng chính:**
- **Presigned Upload:** Client upload trực tiếp lên MinIO
- **Image Processing:** Async resize (Thumbnail 200x200, Medium 800x800)
- **File Validation:** Validate MIME type, chặn file nguy hiểm

**Upload Flow:**
1. Client gọi `GetUploadURL(fileType, purpose)`
2. Server trả về `uploadUrl`, `fileKey`, `publicUrl`
3. Client PUT file trực tiếp lên MinIO
4. Client gọi `ConfirmUpload(fileKey)`
5. Worker xử lý resize async

**NATS Events published:**
- `media.uploaded` → Trigger image processing worker

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 12. 📈 ANALYTIC
**Công nghệ:** Go 1.22+ | ClickHouse | NATS JetStream

**Mục đích:** Real-time Analytics với Batch Ingestion.

**Chức năng chính:**
- **Event Ingestion:** Nhận user behavior events (Views, Clicks, Add to Cart)
- **Batch Processing:** Buffer 1000 events hoặc 5s → Batch insert
- **Dashboard APIs:** Aggregation queries cho Admin

**ClickHouse Schema:**
```sql
CREATE TABLE user_events (
    event_id UUID,
    user_id String,
    event_type String,  -- view_item, add_to_cart, checkout_start
    metadata String,    -- JSON
    url String,
    created_at DateTime
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (event_type, created_at);
```

**Pipeline Architecture:**
```
POST /collect → NATS → Batch Worker → ClickHouse
     (202 Accepted)     (Buffer)      (Bulk Insert)
```

**Dashboard APIs:**
- `GetProductViews(skuId, timeRange)` - Views theo giờ
- `GetConversionRate()` - Funnel analysis

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 13. 🎯 CAMPAIGN
**Công nghệ:** Go 1.22+ | Fiber | PostgreSQL | Redis (Lua Scripts)

**Mục đích:** Quản lý Vouchers/Coupons với Rule Engine.

**Chức năng chính:**
- **Voucher Management:** CRUD vouchers với conditions (JSONB)
- **Atomic Claiming:** Claim voucher với Redis Lua script
- **Cart Calculation:** Tính giảm giá theo rules
- **User Wallet:** Quản lý vouchers của user

**Database Tables:**
- `campaigns` - Campaign metadata
- `vouchers` - Code, Type, Value, Conditions (JSONB)
- `user_vouchers` - User's claimed vouchers

**Conditions Schema (JSONB):**
```json
{
  "min_order_value": 500000,
  "max_discount": 50000,
  "allowed_categories": ["electronics"],
  "excluded_products": ["sku_iphone_15"]
}
```

**Core Methods:**
- `ClaimVoucher(userId, code)` → Atomic với Lua script
- `CalculateCart(items, voucherCode)` → Rule engine validation

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 14. ⭐ REVIEW
**Công nghệ:** Go 1.22+ | Fiber | MongoDB | Redis | gRPC

**Mục đích:** Review & Rating với Materialized View pattern.

**Chức năng chính:**
- **Verified Reviews:** Xác thực mua hàng qua Order Service (gRPC)
- **Rating Aggregation:** Pre-calculated stats
- **Quick Summary:** Redis-cached rating summaries
- **Seller Reply:** Shop trả lời reviews

**MongoDB Collections:**
- `reviews` - Raw review data với images
- `product_ratings` - Pre-calculated averages (Materialized View)

**Service Interactions:**
| Target Service | Protocol | Purpose |
|----------------|----------|---------|
| Order Service | gRPC | Verify purchase before review |
| Media Service | HTTP | Validate review images |

**Rating Calculation:**
```
NewAvg = ((OldAvg * OldTotal) + NewRating) / (OldTotal + 1)
```

**Trạng thái:** ✅ Đã implement đầy đủ

---

### 15. 🔍 SEARCH
**Công nghệ:** Go 1.22+ | Fiber | Elasticsearch | Redis | NATS

**Mục đích:** Search Service với CQRS Read-Model pattern.

**Chức năng chính:**
- **Event Consumer:** Lắng nghe `catalog.product.*` events
- **Elasticsearch Indexing:** Auto-sync products
- **Full-Text Search:** Multi-match với boosting
- **Faceted Search:** Filter by category, brand, price, specs, metadata
- **Redis Caching:** Short TTL (2 phút) cho search results

**Elasticsearch Mapping:**
```json
{
  "mappings": {
    "properties": {
      "name": { "type": "text", "analyzer": "standard" },
      "specs": { "type": "object", "dynamic": true },
      "metadata": { "type": "object", "dynamic": true }
    }
  }
}
```

**Search Query Building:**
- `Must`: multi_match on [name^3, description]
- `Filter`: term (category, brand, status), range (price)
- `Dynamic`: specs.*, metadata.isFlashSale, etc.

**NATS Subscriptions:**
- `catalog.product.created` → Index document
- `catalog.product.updated` → Re-index document
- `catalog.product.deleted` → Delete document

**Trạng thái:** ✅ Đã implement đầy đủ

---

## Ma Trận Đồng Bộ Service

### Giao Tiếp Đồng Bộ (Sync)

| From Service | To Service | Protocol | Purpose |
|--------------|------------|----------|---------|
| API Gateway | All Services | HTTP Proxy | Route requests |
| Order | Inventory | gRPC | Reserve/Confirm/Release Stock |
| Review | Order | gRPC | Verify purchase |
| Cart | Catalog | HTTP | Validate prices (optional) |

### Giao Tiếp Bất Đồng Bộ (Async via NATS)

| Publisher | Subject | Subscribers |
|-----------|---------|-------------|
| Order | `order.created` | Payment, Notification |
| Order | `order.cancelled` | Inventory, Notification |
| Payment | `payment.processed` | Order, Notification |
| Payment | `payment.failed` | Order, Notification |
| Catalog | `catalog.product.created` | Search |
| Catalog | `catalog.product.updated` | Search, Campaign |
| Catalog | `catalog.product.deleted` | Search |
| Logistic | `logistics.shipment.created` | Order, Notification |
| Logistic | `logistics.status.updated` | Order, Notification |
| Media | `media.uploaded` | Media Worker (self) |
| Analytic | `analytics.events.raw` | Analytic Worker (self) |

### Giao Tiếp qua RabbitMQ

| Publisher | Queue | Consumer |
|-----------|-------|----------|
| Notification Bridge | `queue.email` | Notification Worker |
| Notification Bridge | `queue.dead_letter` | Manual Retry |

---

## Các Vấn Đề Đồng Bộ Cần Khắc Phục

### ⚠️ Chưa Đồng Bộ

1. **API Gateway thiếu routes:**
   - Chưa có route cho: `/api/v1/payment/*`, `/api/v1/logistics/*`, `/api/v1/media/*`, `/api/v1/reviews/*`, `/api/v1/search/*`, `/api/v1/campaigns/*`, `/api/v1/analytics/*`, `/api/v1/profile/*`

2. **Profile Service chưa liên kết với Auth:**
   - Cần sync email từ Auth → Profile khi user register

3. **Cart chưa validate price với Catalog:**
   - Comment trong code: "Price must be re-validated during checkout"
   - Chưa implement gọi Catalog để verify giá

4. **Review chưa fetch user info từ Profile:**
   - Cần gọi Profile Service để lấy displayName, avatarUrl

5. **Campaign chưa notify sản phẩm tagged:**
   - Cần subscribe `catalog.product.updated` để track metadata.campaignId

### ⚠️ Event Flow Chưa Hoàn Chỉnh

```
Current:
Order.created → Payment, Notification

Missing:
1. Order.paid → Inventory.ConfirmStock (Currently via gRPC, should also have NATS backup)
2. Payment.processed → Logistic.CreateShipment (Auto-create shipping after payment)
3. Logistic.delivered → Review.EnableReview (Allow review after delivery)
4. Profile.updated → Order, Cart (Update cached user info)
```

---

## Tech Stack Tổng Hợp

| Layer | Technologies |
|-------|--------------|
| **Languages** | Go 1.22+, TypeScript (NestJS) |
| **Web Frameworks** | Fiber (Go), NestJS (Node) |
| **SQL Database** | PostgreSQL |
| **NoSQL Database** | MongoDB |
| **OLAP Database** | ClickHouse |
| **Search Engine** | Elasticsearch |
| **Caching** | Redis |
| **Object Storage** | MinIO (S3 Compatible) |
| **Message Broker (Async)** | NATS JetStream |
| **Message Broker (Task Queue)** | RabbitMQ |
| **RPC** | gRPC |
| **Auth** | JWT (Access + Refresh Tokens) |
| **Containerization** | Docker, Docker Compose |

---

## Database Infrastructure

### Tổng Quan Databases

| Database | Engine | Service(s) | Port |
|----------|--------|------------|------|
| identity_db | PostgreSQL 15 | auth | 5432 |
| order_db | PostgreSQL 15 | order | 5432 |
| inventory_db | PostgreSQL 15 | inventory | 5432 |
| payment_db | PostgreSQL 15 | payment | 5432 |
| logistics_db | PostgreSQL 15 | logistic | 5432 |
| campaign_db | PostgreSQL 15 | campaign | 5432 |
| profile_db | MongoDB 6 | profile | 27017 |
| catalog_db | MongoDB 6 | catalog | 27017 |
| cart_db | MongoDB 6 | cart | 27017 |
| review_db | MongoDB 6 | review | 27017 |
| notification_db | MongoDB 6 | notification | 27017 |
| media_db | MongoDB 6 | media | 27017 |
| analytics_db | ClickHouse 23 | analytic | 8123/9000 |
| products_index | Elasticsearch 8 | search | 9200 |

### Caching Layer (Redis)

| Purpose | Key Pattern | TTL |
|---------|-------------|-----|
| User Permissions | `identity:user:{id}:permissions` | 1 hour |
| Token Blacklist | `identity:blacklist:{jti}` | Token TTL |
| Shopping Cart | `cart:{userId}` | 30 days |
| Stock Cache | `inventory:{skuId}` | Permanent |
| Voucher Stock | `voucher:{code}:stock` | Campaign TTL |
| Search Cache | `search:{hash}` | 2 minutes |
| Shipping Fee | `fee:{provider}:{from}:{to}:{weight}` | 1 hour |
| Product Rating | `rating:{productId}` | 30 minutes |

### Message Brokers

| Broker | Purpose | Port |
|--------|---------|------|
| NATS JetStream | Async events (order.created, payment.processed, etc.) | 4222 |
| RabbitMQ | Notification task queue với DLQ | 5672 |

### Object Storage

| Storage | Purpose | Port |
|---------|---------|------|
| MinIO (S3) | Product images, User avatars, Review images | 9000/9001 |

### Files Database Đã Tạo

```
microservices/
├── database/
│   ├── COMPLETE_DATABASE_SCHEMA.sql      # Schema tổng hợp tất cả services
│   ├── postgres/
│   │   └── init-multiple-databases.sh    # Script khởi tạo multi-DB
│   ├── mongodb/
│   │   └── 001_init_mongodb.js           # Script khởi tạo collections
│   └── redis/
│       └── REDIS_KEY_PATTERNS.md         # Documentation Redis keys
├── auth/migrations/
│   └── 001_init_schema.sql               # Auth service schema
├── order/migrations/
│   └── 001_init_schema.sql               # Order service schema
├── inventory/migrations/
│   └── 001_init_schema.sql               # Inventory service schema
│   └── scripts/
│       ├── reserve_stock.lua             # Atomic stock reservation
│       └── release_stock.lua             # Release reserved stock
├── campaign/
│   ├── migrations/001_init.sql           # Campaign schema
│   └── scripts/claim_voucher.lua         # Atomic voucher claiming
├── payment/migrations/
│   └── 001_init.sql                      # Payment schema
├── logistic/migrations/
│   ├── 001_create_shipping_orders.sql    # Shipping orders
│   └── 002_create_webhook_logs.sql       # Webhook logs
├── analytic/migrations/
│   └── 001_init_schema.sql               # ClickHouse analytics schema
├── search/elasticsearch/
│   └── products_index_mapping.json       # Elasticsearch index mapping
└── docker-compose.databases.yml          # All databases Docker Compose
```

---

## Tóm Tắt Trạng Thái Implementation

| Service | Status | Core Features | Integration |
|---------|--------|---------------|-------------|
| API Gateway | ✅ | Routing, Auth, Rate Limit | Partial |
| Auth | ✅ | Register, Login, RBAC | Complete |
| Profile | ✅ | User Info, Addresses, Shop | Needs sync |
| Catalog | ✅ | Products, Categories, Brands | Complete |
| Cart | ✅ | Redis+MongoDB hybrid | Needs validation |
| Order | ✅ | State machine, Snapshotting | Complete |
| Inventory | ✅ | Two-phase reservation | Complete |
| Payment | ✅ | Multi-gateway, Webhook | Complete |
| Logistic | ✅ | Multi-carrier, Webhook | Complete |
| Notification | ✅ | Bridge pattern, DLQ | Complete |
| Media | ✅ | Presigned URL, Processing | Complete |
| Analytic | ✅ | Batch ingestion | Complete |
| Campaign | ✅ | Rule engine, Atomic claim | Complete |
| Review | ✅ | Verified review, Aggregation | Needs integration |
| Search | ✅ | CQRS, Faceted search | Complete |

---

*Tài liệu được tạo tự động ngày: 2026-01-07*
