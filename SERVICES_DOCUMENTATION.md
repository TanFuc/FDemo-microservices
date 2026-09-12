# NexusCommerce — Microservices System Architecture Documentation

## System Architecture Overview

NexusCommerce is an enterprise distributed commerce and fulfillment platform built following the **Microservices Architecture Pattern**. The platform comprises **15 independent, decoupled microservices** communicating through synchronous protocols (HTTP/REST, gRPC) and asynchronous message broker topologies (Apache Kafka / NATS JetStream, RabbitMQ).

### High-Level Architectural Diagram

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
        │      (Go + PG)        │     │      (Go + MongoDB)         │     │  (Go + Redis)   │
        └───────────────────────┘     └─────────────────────────────┘     └─────────────────┘
                    │                               │                               │
                    │                    ┌──────────┴──────────┐                    │
                    │                    │                     │                    │
        ┌───────────▼───────────┐        │        ┌────────────▼────────────┐       │
        │       PROFILE         │        │        │         SEARCH          │       │
        │   (Go + MongoDB)      │        │        │   (Go + Elasticsearch)  │       │
        └───────────────────────┘        │        └─────────────────────────┘       │
                                         │                                          │
                               ┌─────────▼─────────┐                  ┌─────────────▼─────────────┐
                               │     INVENTORY     │◄─────────────────│           ORDER           │
                               │  (Go + PG + Redis)│   (gRPC/Kafka)   │     (Go + PG + Outbox)    │
                               └───────────────────┘                  └───────────────────────────┘
                                         │                                          │
                               ┌─────────┴─────────┐                  ┌─────────────┴─────────────┐
                               │                   │                  │                           │
                     ┌─────────▼─────────┐  ┌──────▼───────┐  ┌───────▼───────┐  ┌────────────────▼───────────────────┐
                     │      PAYMENT      │  │   LOGISTIC   │  │   CAMPAIGN    │  │          NOTIFICATION              │
                     │    (Go + PG)      │  │  (Go + PG)   │  │   (Go + PG)   │  │    (Go + RabbitMQ + MongoDB)       │
                     └───────────────────┘  └──────────────┘  └───────────────┘  └────────────────────────────────────┘
                               │                   │                  │                           │
                     ┌─────────┴───────────────────┴──────────────────┴───────────────────────────┘
                     │
        ┌────────────▼──────────┐     ┌─────────────────────────────┐     ┌─────────────────────────┐
        │        MEDIA          │     │         ANALYTIC            │     │         REVIEW          │
        │    (Go + MinIO)       │     │     (Go + ClickHouse)       │     │     (Go + MongoDB)      │
        └───────────────────────┘     └─────────────────────────────┘     └─────────────────────────┘
```

---

## Detailed Service Specifications

### 1. 🚪 API-GATEWAY
**Technology Stack:** Go 1.22+ | Fiber | Redis | JWT

**Purpose:** Unified Single Entry Point (Reverse Proxy) for all incoming client traffic.

**Key Responsibilities:**
- **Routing & Reverse Proxy:** Dynamic proxying and dispatching requests to downstream microservices.
- **Authentication & Claims Forwarding:** Verifying JWT access tokens and injecting identity headers (`X-User-ID`, `X-User-Roles`).
- **Rate Limiting:** Distributed token bucket rate limiting (100 req/60s per client IP) backed by Redis.
- **Load Balancing:** Distributing traffic across healthy backend replica instances.

**Routing Table:**
| Public Route Prefix | Downstream Target Service | Authentication |
|---|---|---|
| `/api/v1/identity/*` | `identity-service` (`auth`) | Public / Protected |
| `/api/v1/profiles/*` | `profile-service` | Protected (JWT) |
| `/api/v1/catalog/*` | `catalog-service` | Public (Read) / Seller (Write) |
| `/api/v1/cart/*` | `cart-service` | Protected (JWT) |
| `/api/v1/orders/*` | `order-service` | Protected (JWT) |
| `/api/v1/payments/*` | `payment-service` | Protected / Webhook |
| `/api/v1/logistics/*` | `logistics-service` | Protected / Webhook |
| `/api/v1/campaigns/*` | `campaign-service` | Public / Protected |
| `/api/v1/notifications/*` | `notification-service` | Protected (JWT) |
| `/api/v1/search/*` | `search-service` | Public |
| `/api/v1/media/*` | `media-service` | Protected (JWT) |
| `/api/v1/reviews/*` | `review-service` | Public (Read) / Protected (Write) |
| `/api/v1/analytics/*` | `analytics-service` | Protected (Admin/Seller) |

**Status:** ✅ Fully Implemented

---

### 2. 🔐 AUTH (Identity Service)
**Technology Stack:** Go 1.22+ | Fiber | PostgreSQL | Redis | JWT

**Purpose:** Identity provider, credential management, token lifecycle, and role-based access control (RBAC).

**Key Responsibilities:**
- **Authentication:** Registration, login, logout, password hashing using argon2id / bcrypt.
- **Token Lifecycle:** Short-lived Access Tokens (15m), Refresh Tokens (7d) with automatic Token Rotation.
- **Role-Based Access Control (RBAC):** Roles (`CUSTOMER`, `SELLER`, `ADMIN`) and granular permission checking (`product:create`, `order:read_own`, etc.).
- **Permission Caching:** Caching user permissions in Redis with a 1-hour TTL.
- **Token Blacklisting:** Revoking compromised access tokens using JTI (JWT ID) keys in Redis.

**Primary Domain Entities:**
- `User` - Authentication credentials, status, email, phone.
- `Role` - System roles (`CUSTOMER`, `SELLER`, `ADMIN`).
- `Permission` - Fine-grained permissions.
- `RolePermission` - Many-to-many role-to-permission mapping.
- `RefreshToken` - Tracked refresh tokens with device footprint and revocation status.

**Core Endpoints:**
| Method | Endpoint | Description |
|---|---|---|
| POST | `/auth/register` | Register a new user account |
| POST | `/auth/login` | Authenticate user and issue JWT token pair |
| POST | `/auth/refresh` | Rotate and issue a new access token |
| POST | `/auth/logout` | Revoke active refresh token and blacklist access token |
| GET | `/auth/profile` | Retrieve identity record for the authenticated user |

**Status:** ✅ Fully Implemented

---

### 3. 👤 PROFILE
**Technology Stack:** Go 1.22+ | Fiber | MongoDB

**Purpose:** User personal profiles, delivery address book management, and merchant/shop configurations.

**Key Responsibilities:**
- **User Profile Management:** Demographic information (`displayName`, `avatarUrl`, `bio`).
- **Address Book:** Multi-address management with single `isDefault` address validation rules.
- **Merchant / Shop Profile:** Seller onboarding, shop status, business registration metadata.

**Primary Collections:**
- `profiles` - User profile document + embedded `shopConfig` for sellers.
- `addresses` - Delivery address book with compound indexing on `userId`.

**Core Endpoints:**
| Method | Endpoint | Description |
|---|---|---|
| GET | `/profiles/me` | Fetch or lazily initialize user profile document |
| PATCH | `/profiles/me` | Update demographic user profile fields |
| POST | `/profiles/me/shop` | Register or update merchant/seller shop data |
| GET | `/profiles/me/addresses` | List all saved delivery addresses |
| POST | `/profiles/me/addresses` | Add a new delivery address |
| PATCH | `/profiles/me/addresses/:id/set-default` | Designate specific address as default shipping destination |

**Status:** ✅ Fully Implemented

---

### 4. 📦 CATALOG
**Technology Stack:** Go 1.22+ | Fiber | MongoDB | Redis | Event Bus (Kafka / NATS)

**Purpose:** Flexible category hierarchies, brand registries, product listings, and SKU variations.

**Key Responsibilities:**
- **Category Hierarchy:** Recursive tree structure with dynamic attribute schemas.
- **Brand Registry:** Official brand management with verification statuses.
- **Product & Variation Management:** Multi-variant SKUs (sizes, colors), technical specifications, rich media URLs.
- **Domain Event Publishing:** Emits events upon product mutations to trigger search indexing.

**Primary Domain Models:**
- `Category` - Category tree with `AttributeDefinitions`.
- `Brand` - Verified brand entity.
- `Product` - Parent product model containing variations, specifications, and media references.
- `Variation` - Individual SKU item with price, stock inventory tracking ID, and custom attributes.

**Published Events:**
- `catalog.product.created` → Triggers search engine indexing.
- `catalog.product.updated` → Triggers search engine re-indexing and campaign synchronization.
- `catalog.product.deleted` → Removes document from search index.

**Status:** ✅ Fully Implemented

---

### 5. 🛒 CART
**Technology Stack:** Go 1.22+ | Fiber | Redis | MongoDB

**Purpose:** High-throughput, low-latency shopping cart sessions utilizing a **Write-Behind Caching** strategy.

**Key Responsibilities:**
- **Atomic Operations:** Adding, updating, and removing items with Redis atomic operations.
- **Session Resilience:** Primary read/write in Redis with asynchronous MongoDB backup persistence.
- **Lazy Hydration:** Automatic cache hydrate from MongoDB upon Redis key expiration or cache misses.
- **Cart Checkout Clearing:** Atomically purging user cart upon successful order creation.

**Data Structures:**
- **Redis:** Hash `cart:{userId}` with 30-day sliding TTL.
- **MongoDB:** `carts` collection providing backup persistence.

**Core Endpoints:**
| Method | Endpoint | Description |
|---|---|---|
| GET | `/cart` | Retrieve current user's shopping cart |
| POST | `/cart/items` | Add item/SKU to shopping cart |
| PUT | `/cart/items/:skuId` | Update quantity of a specific item |
| DELETE | `/cart/items/:skuId` | Remove item from cart |
| DELETE | `/cart` | Clear entire shopping cart |

**Status:** ✅ Fully Implemented

---

### 6. 📋 ORDER
**Technology Stack:** Go 1.22+ | Fiber | PostgreSQL | Event Bus | gRPC

**Purpose:** Central transaction hub, order state machine, and Saga transaction coordinator.

**Key Responsibilities:**
- **Order Placement:** Capturing point-in-time pricing and item metadata snapshots.
- **Finite State Machine:** `PENDING` → `PAID` → `SHIPPED` → `COMPLETED` / `CANCELLED`.
- **Inventory Allocation:** Interfacing with Inventory Service via gRPC or Saga choreography.
- **Event-Driven Outbox:** Emitting order events through the Transactional Outbox pattern.

**Primary Entities:**
- `Order` - Aggregate root with lifecycle status, financial totals, and immutable shipping address (JSONB).
- `OrderItem` - SKU snapshot with purchased price, name, and thumbnail image.

**Inter-Service Interactions:**
| Target Service | Protocol | Purpose |
|---|---|---|
| Inventory Service | gRPC | Reserve, confirm, or release stock reservations |
| Payment Service | Event Bus / REST | Trigger and verify payment authorization |
| Notification Service | Event Bus | Dispatch transactional order confirmations |

**Published Events:**
- `order.created` → Triggers inventory reservation & payment capture.
- `order.cancelled` → Triggers stock release and refund compensations.
- `order.confirmed` / `order.paid` → Transitions order and triggers shipping dispatch.

**Status:** ✅ Fully Implemented

---

### 7. 📊 INVENTORY
**Technology Stack:** Go 1.22+ | Fiber | PostgreSQL | Redis (Lua Scripts) | gRPC

**Purpose:** Warehouse stock management with Two-Phase Reservation and atomic Lua script execution.

**Key Responsibilities:**
- **Stock Reservation (Phase 1):** Atomically holding stock during checkout with temporary expiration (30m TTL).
- **Stock Confirmation (Phase 2):** Finalizing stock deduction when payment succeeds.
- **Stock Release (Compensation):** Returning reserved stock to inventory on order cancellation or checkout timeout.
- **Redis-PostgreSQL Synchronization:** High-throughput reservation in Redis with transactional persistence in PostgreSQL.

**Primary Database Tables:**
- `inventory_items` - `TotalStock`, `ReservedStock`, SKU identifier, warehouse location.
- `stock_reservations` - Reservation audit ledger tracking order IDs, quantities, and expiration timestamps.

**Atomic Redis Lua Script (`reserve.lua`):**
```lua
-- Atomic check-and-reserve script
local total = tonumber(redis.call('HGET', KEYS[1], 'total')) or 0
local reserved = tonumber(redis.call('HGET', KEYS[1], 'reserved')) or 0
local requested = tonumber(ARGV[1])

if (total - reserved) >= requested then
    redis.call('HINCRBY', KEYS[1], 'reserved', requested)
    return 1 -- Success
else
    return 0 -- Insufficient stock
end
```

**gRPC Interface Methods:**
- `ReserveStock(skuId, qty, orderId)` → Returns `reservationId`
- `ConfirmStock(orderId)` → Permanently decrements available inventory
- `ReleaseStock(orderId)` → Reverts temporary hold

**Status:** ✅ Fully Implemented

---

### 8. 💳 PAYMENT
**Technology Stack:** Go 1.22+ | Fiber | PostgreSQL | Event Bus | Redis

**Purpose:** Resilient payment aggregator handling multi-gateway routing, idempotency, and webhook verification.

**Key Responsibilities:**
- **Multi-Gateway Support:** Pluggable payment drivers (Stripe, VNPay, MoMo, COD, Mock Provider).
- **Idempotent Webhook Processing:** Cryptographic signature validation and Redis deduplication.
- **Transaction Ledger:** Immutable audit trail recording state transitions and raw provider payloads.
- **Reconciliation Engine:** Periodic cron worker verifying pending transaction statuses against payment gateways.

**Primary Database Tables:**
- `payment_transactions` - `OrderID`, `Amount`, `Currency`, `Provider`, `Status`, `Metadata` (JSONB).
- `payment_logs` - Structured audit log capturing inbound webhook payloads and external API calls.

**Published Events:**
- `payment.processed` → Order service transitions order to `PAID`.
- `payment.failed` → Order service triggers cancellation and inventory release.
- `payment.refunded` → Financial records updated and customer notified.

**Status:** ✅ Fully Implemented

---

### 9. 🚚 LOGISTIC
**Technology Stack:** Go 1.22+ | Fiber | PostgreSQL | Redis | Event Bus

**Purpose:** Shipping aggregator, automated carrier dispatch, rate calculation, and real-time shipment tracking.

**Key Responsibilities:**
- **Multi-Carrier Abstraction:** Unified interface for shipping carriers (GHN, GHTK, ViettelPost, Mock Carrier).
- **Dynamic Fee Matrix:** Distance and weight-based rate calculation with 1-hour Redis caching.
- **Waybill Generation:** Automated dispatch creating tracking codes and shipping labels.
- **Webhook Ingestion:** Ingesting status updates from shipping providers.

**Status Mapping Matrix:**
| Carrier Status | Internal System Status | Action Triggered |
|---|---|---|
| `ready_to_pick` | `PENDING` | Dispatch label ready |
| `picking` / `delivering` | `SHIPPING` | Order marked as in-transit |
| `delivered` | `DELIVERED` | Order marked as completed; unlock review |
| `return` | `RETURNED` | Initiate return & refund workflow |

**Status:** ✅ Fully Implemented

---

### 10. 📧 NOTIFICATION
**Technology Stack:** Go 1.22+ | Event Bus | RabbitMQ | MongoDB | Gomail

**Purpose:** Multi-channel alert dispatcher utilizing a resilient **Dual-Broker Architecture**.

**Key Responsibilities:**
- **Event Bus Bridge:** Consumes domain events from the primary event bus and routes to task queues.
- **RabbitMQ Worker Pool:** High-reliability worker pool with Dead Letter Queue (DLQ) for retries.
- **Multi-Channel Dispatch:** Email (SMTP/SES), SMS, WebSocket push, Webhooks.
- **Audit Logging:** Persisting notification delivery status in MongoDB.

**Dual-Broker Architecture:**
```
Domain Events → Notification Bridge → RabbitMQ (Exchange) → Worker Pool → Email/SMS/Push
                                             ↓
                                    Dead Letter Queue (DLQ)
```

**Status:** ✅ Fully Implemented

---

### 11. 🖼️ MEDIA
**Technology Stack:** Go 1.22+ | Fiber | MinIO (S3 API) | Event Bus

**Purpose:** Asset storage utilizing the **Presigned URL Pattern** for direct client uploads.

**Key Responsibilities:**
- **Direct-to-S3 Uploads:** Generates secure, short-lived presigned PUT URLs for clients.
- **Asynchronous Image Processing:** Worker generating standardized thumbnails (200x200) and web formats (800x800).
- **MIME Type Validation:** Magic number validation preventing malicious executable uploads.

**Status:** ✅ Fully Implemented

---

### 12. 📈 ANALYTIC
**Technology Stack:** Go 1.22+ | ClickHouse | Event Bus

**Purpose:** Real-time business intelligence, user clickstream ingestion, and OLAP aggregations.

**Key Responsibilities:**
- **Event Ingestion:** High-volume user behavioral events (`view_item`, `add_to_cart`, `checkout_start`).
- **Batch Processing:** Memory buffer flushing every 1,000 events or 5 seconds into ClickHouse.
- **Executive Analytics:** High-performance queries generating revenue charts and funnel conversion rates.

**Status:** ✅ Fully Implemented

---

### 13. 🎯 CAMPAIGN
**Technology Stack:** Go 1.22+ | Fiber | PostgreSQL | Redis (Lua Scripts)

**Purpose:** Promotional voucher and flash-sale engine with high-concurrency atomic claiming.

**Key Responsibilities:**
- **Voucher Rules Engine:** Minimum spend, maximum discount, category restrictions, eligible SKU lists.
- **Atomic Quota Claiming:** Redis Lua script guaranteeing zero over-allocation during flash spikes.
- **Cart Discount Evaluation:** Deterministic cart recalculation applying active promotions.

**Status:** ✅ Fully Implemented

---

### 14. ⭐ REVIEW
**Technology Stack:** Go 1.22+ | Fiber | MongoDB | Redis | gRPC

**Purpose:** Verified buyer ratings and reviews with pre-calculated materialized rating views.

**Key Responsibilities:**
- **Verified Buyer Enforcement:** Validates purchase history with Order Service via gRPC before review creation.
- **Materialized Ratings:** Pre-calculated mean ratings and distribution histograms for instant retrieval.
- **Merchant Responses:** Allows seller replies to buyer reviews.

**Status:** ✅ Fully Implemented

---

### 15. 🔍 SEARCH
**Technology Stack:** Go 1.22+ | Fiber | Elasticsearch 8.11 | Redis | Event Bus

**Purpose:** CQRS read-model product search engine with fuzzy matching and faceted filtering.

**Key Responsibilities:**
- **Automated Indexing:** Ingests `catalog.product.*` events to update the Elasticsearch cluster.
- **Full-Text Querying:** Multi-match search with field boosting (`name^3`, `description`).
- **Faceted Filters:** Filter by brand, category, price range, dynamic specifications, and seller location.
- **Result Caching:** Redis caching for top searches with a 2-minute TTL.

**Status:** ✅ Fully Implemented

---

## Inter-Service Communication Topology

### Synchronous Communication (HTTP & gRPC)

| Calling Service | Target Service | Protocol | Business Purpose |
|---|---|---|---|
| **API Gateway** | All Microservices | HTTP Reverse Proxy | Inbound request routing & authentication |
| **Order Service** | **Inventory Service** | gRPC | Real-time stock reservation, confirmation, release |
| **Review Service** | **Order Service** | gRPC | Verify purchase authenticity before review submission |
| **Cart Service** | **Catalog Service** | HTTP | Real-time SKU and price validation |

### Asynchronous Communication (Domain Events)

| Publishing Service | Event Subject | Subscribing Services |
|---|---|---|
| **Order Service** | `order.created` | Payment Service, Notification Service |
| **Order Service** | `order.cancelled` | Inventory Service, Notification Service |
| **Order Service** | `order.confirmed` | Logistic Service, Notification Service |
| **Payment Service** | `payment.processed` | Order Service, Notification Service |
| **Payment Service** | `payment.failed` | Order Service, Notification Service |
| **Catalog Service** | `catalog.product.created` | Search Service |
| **Catalog Service** | `catalog.product.updated` | Search Service, Campaign Service |
| **Catalog Service** | `catalog.product.deleted` | Search Service |
| **Logistic Service** | `logistics.shipment.created` | Order Service, Notification Service |
| **Logistic Service** | `logistics.status.updated` | Order Service, Notification Service, Review Service |
| **Media Service** | `media.uploaded` | Media Worker (Image resizing) |
| **Analytic Service** | `analytics.events.raw` | Analytic Worker (ClickHouse batch insert) |

---

## Database & Infrastructure Topology

| Logical Database | Engine / Version | Assigned Service | Default Port |
|---|---|---|---|
| `identity_db` | PostgreSQL 15 | `auth` | 15432 (mapped) |
| `order_db` | PostgreSQL 15 | `order` | 15432 (mapped) |
| `inventory_db` | PostgreSQL 15 | `inventory` | 15432 (mapped) |
| `payment_db` | PostgreSQL 15 | `payment` | 15432 (mapped) |
| `logistics_db` | PostgreSQL 15 | `logistic` | 15432 (mapped) |
| `campaign_db` | PostgreSQL 15 | `campaign` | 15432 (mapped) |
| `profile_db` | MongoDB 6.0 | `profile` | 27017 |
| `catalog_db` | MongoDB 6.0 | `catalog` | 27017 |
| `cart_db` | MongoDB 6.0 | `cart` (backup) | 27017 |
| `review_db` | MongoDB 6.0 | `review` | 27017 |
| `notification_db` | MongoDB 6.0 | `notification` | 27017 |
| `media_db` | MongoDB 6.0 | `media` | 27017 |
| `analytics` | ClickHouse 23.8 | `analytic` | 8123 (HTTP) / 9000 (Native) |
| `products_index` | Elasticsearch 8.11 | `search` | 9200 |
| In-Memory Cache | Redis 7.0 | `cart`, `inventory`, `idempotency`, etc. | 16379 (mapped) |
| Object Store | MinIO | `media` | 9002 (API) / 9001 (Console) |

---

## Implementation Status Summary

| Service | Architecture | Core Capabilities | Integration Status |
|---|---|---|---|
| **API Gateway** | Go Fiber | Reverse proxy, rate limiting, JWT validation | ✅ Verified |
| **Auth** | Go Fiber | Argon2id, JWT rotation, RBAC, Redis permission cache | ✅ Verified |
| **Profile** | Go Fiber | User profile, address book with default logic, seller shop | ✅ Verified |
| **Catalog** | Go Fiber | Category trees, dynamic attributes, variation SKUs | ✅ Verified |
| **Cart** | Go Fiber | Redis write-behind, atomic operations, Mongo backup | ✅ Verified |
| **Order** | Go Fiber | Snapshotting, order state machine, Saga orchestrator | ✅ Verified |
| **Inventory** | Go Fiber | Optimistic locking, 2-phase reservation, Redis Lua script | ✅ Verified |
| **Payment** | Go Fiber | Multi-gateway provider, idempotent webhooks, ledger | ✅ Verified |
| **Logistic** | Go Fiber | Multi-carrier calculation, waybill creation, tracking codes | ✅ Verified |
| **Notification**| Worker | Event bridge, RabbitMQ worker, DLQ, template dispatch | ✅ Verified |
| **Media** | Go Fiber | Presigned S3 URLs, async thumbnail processing worker | ✅ Verified |
| **Analytic** | Go Fiber | High-throughput ingestion, buffer batching into ClickHouse | ✅ Verified |
| **Campaign** | Go Fiber | Voucher rule engine, atomic Redis quota claiming | ✅ Verified |
| **Review** | Go Fiber | Verified purchase verification via gRPC, rating aggregates | ✅ Verified |
| **Search** | Go Fiber | Elasticsearch 8 sync, faceted search, fuzzy matching | ✅ Verified |
