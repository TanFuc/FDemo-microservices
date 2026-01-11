# MISSION: BUILD LOGISTICS AGGREGATOR SERVICE (GO + POSTGRES + NATS)

**Role:** Senior Backend Engineer (Logistics Domain Expert).
**Goal:** Build a `logistics-service` that aggregates multiple shipping providers (GHN, GHTK, ViettelPost) using the Adapter Pattern. It handles Fee Calculation, Shipping Order Creation, and Webhook Status Updates.

**Tech Stack:**
- **Language:** Go 1.22+.
- **Database:** PostgreSQL (To store Shipping Orders & Webhook Logs).
- **Communication:** NATS JetStream (To notify Order Service of status changes).
- **Cache:** Redis (To cache Shipping Fees and Province/District Mappings).
- **Architecture:** Hexagonal Architecture (Ports & Adapters).

---

## PHASE 1: DATABASE SCHEMA

### 1. `ShippingOrder` (The Record)
- **Table:** `shipping_orders`
- **Fields:**
    - `ID` (UUID).
    - `InternalOrderID` (UUID) - Reference to `tafu-order`.
    - `Provider` (String) - Enum: "GHN", "GHTK", "MOCK".
    - `TrackingCode` (String) - The Waybill code (Mã vận đơn) from provider.
    - `CarrierStatus` (String) - Raw status from provider (e.g., "ready_to_pick", "delivering").
    - `SystemStatus` (String) - Normalized status (PENDING, SHIPPING, DELIVERED, RETURNED).
    - `ShippingFee` (Decimal) - Actual cost we pay to carrier.
    - `CODAmount` (Decimal) - Cash to collect.
    - `LabelURL` (String) - Link to PDF print.
    - `Metadata` (JSONB) - Store raw response from provider.

### 2. `LocationMapping` (Optional but Recommended)
- Since GHN uses DistrictID=123 but GHTK uses DistrictID=ABC, we might need a mapping table or rely on Frontend sending correct Provider-specific IDs. For MVP, assume Frontend sends the Provider's DistrictID directly.

## PHASE 2: STRATEGY PATTERN (THE PROVIDERS)

Create a unified interface so the core logic doesn't care which carrier is used.

```go
type RateRequest struct {
    FromDistrictID int
    ToDistrictID   int
    WeightGram     int
    InsuranceValue int // Values of goods
}

type ShipRequest struct {
    InternalOrderID string
    Sender          ContactInfo
    Receiver        ContactInfo
    Parcels         []Parcel
    IsCOD           bool
}

type Provider interface {
    GetName() string
    CalculateFee(ctx context.Context, req *RateRequest) (float64, error)
    CreateOrder(ctx context.Context, req *ShipRequest) (trackingCode string, labelUrl string, err error)
    ParseWebhook(r *http.Request) (internalOrderID string, newStatus string, err error)
}
Implementation (internal/adapter/...)
MockProvider: Returns random fees and a fake tracking code MOCK-12345. (Use this for Development).

GHNProvider: Implements real API calls to GiaoHangNhanh.

GHTKProvider: Implements real API calls to GHTK.

PHASE 3: USECASE LOGIC
1. CalculateShippingFee(ctx, req)
Input: Items (Weight), Address (ToDistrict), ChosenProvider.

Logic:

Check Redis Cache for fee (Key: fee:{provider}:{from}:{to}:{weight}).

If Miss: Call Provider.CalculateFee.

Set Cache (TTL 1 hour).

Return Fee.

2. CreateShipment(ctx, internalOrderId)
Trigger: Can be called via API (by Warehouse Staff scanning QR code) OR listening to NATS order.packed.

Logic:

Load Order Details (from Order Service via gRPC or local copy if carried in event).

Call Provider.CreateOrder.

Save to DB ShippingOrder.

Publish NATS: logistics.shipment.created (Payload: TrackingCode).

3. HandleWebhook(ctx, providerName)
Trigger: Endpoint POST /logistics/webhook/:provider.

Logic:

Factory loads correct Provider Adapter.

Call Provider.ParseWebhook(req).

Update ShippingOrder status in DB.

Crucial: Map Provider Status to System Status.

GHN "delivered" -> System "DELIVERED".

GHN "return" -> System "RETURNED".

Publish NATS: logistics.status.updated (Order Service listens to this to mark Order as Completed).

PHASE 4: AUTO-VERIFICATION LOOP
Instructions for AI:

Init: Setup Go project.

Generate:

internal/core/ports: The Provider Interface.

internal/adapter/mock: The Mock implementation.

internal/adapter/ghn: Skeleton for GHN (Setup HTTP Client).

TEST (Integration):

Create tests/shipping_test.go.

Scenario:

Call CalculateFee with Mock Provider -> Expect valid number.

Call CreateShipment -> Expect Tracking Code "MOCK-...".

Call HandleWebhook with fake payload -> Expect DB update & NATS message.

Build: Run go build.

IMPORTANT RULES
Resilience: External APIs (GHN/GHTK) often fail. Implement Retries with Exponential Backoff for CreateOrder.

Secrets: API Tokens for GHN/GHTK must come from ENV.

Address format: Assume the input address includes specific IDs required by the provider (e.g., ProvinceID, DistrictID, WardCode).