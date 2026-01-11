# MISSION: BUILD SECURE PAYMENT AGGREGATOR SERVICE (GO + POSTGRES + NATS)

**Role:** Principal Fintech Engineer.
**Goal:** Build a PCI-DSS compliant `payment-service` that aggregates multiple payment gateways (Stripe, MoMo, VNPay) and handles asynchronous Webhooks securely.

**Tech Stack:**
- **Language:** Go 1.22+.
- **Database:** PostgreSQL (Transaction Ledger).
- **Communication:** NATS JetStream (Publishing events).
- **Architecture:** Hexagonal Architecture (Ports & Adapters).

---

## PHASE 1: DATABASE SCHEMA (THE LEDGER)

We need a double-entry bookkeeping style or at least a strict state machine.

### 1. `PaymentTransaction`
- **Table:** `payment_transactions`
- **Fields:**
    - `ID` (UUID).
    - `OrderID` (UUID) - Reference to Order Service.
    - `UserID` (UUID).
    - `Amount` (Decimal) - Use `shopspring/decimal`.
    - `Currency` (USD, VND).
    - `Provider` (Enum: STRIPE, MOMO, COD).
    - `ProviderTxID` (String) - Transaction ID from the Gateway (e.g., Stripe PaymentIntent ID).
    - `Status` (Enum: PENDING, SUCCESS, FAILED, REFUNDED).
    - `Metadata` (JSONB) - Store raw request/response for audit.
    - `CreatedAt`, `UpdatedAt`.

### 2. `PaymentLog` (Audit Trail)
- Store every raw Webhook received from providers for debugging.
- `ID`, `Provider`, `RawPayload` (JSON), `IPAddress`, `CreatedAt`.

## PHASE 2: ADAPTER PATTERN (THE CORE)

Create an interface to standardize all gateways.

```go
type PaymentRequest struct {
    OrderID     string
    Amount      decimal.Decimal
    Description string
    CallbackURL string // Where Gateway calls us back
}

type PaymentResponse struct {
    PaymentURL   string // Redirect user here
    ProviderTxID string
    RawData      map[string]interface{}
}

// THE INTERFACE
type PaymentGateway interface {
    CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error)
    VerifyWebhook(r *http.Request) (bool, *WebhookData, error) // Check signature
}
Implementations (internal/adapter/...):
StripeAdapter: Uses Stripe SDK.

MoMoAdapter: Implements HMAC SHA256 signature generation.

CODAdapter: (Cash on Delivery) - Auto-returns SUCCESS or PENDING immediately.

PHASE 3: USECASE LOGIC
1. InitiatePayment(ctx, orderId, provider)
Step 1: Validate Order amount (Call Order Service or trust input payload signed by JWT).

Step 2: Create PaymentTransaction record (Status: PENDING).

Step 3: Call Gateway.CreatePayment.

Step 4: Update PaymentTransaction with ProviderTxID.

Step 5: Return Payment URL to Frontend.

2. HandleWebhook(ctx, provider, request) (CRITICAL)
Step 1: Signature Verification:

Use Gateway.VerifyWebhook.

IF fail -> Return 400 immediately (Stop hackers from faking payments).

Step 2: Idempotency Check:

Check DB: Is this ProviderTxID already marked SUCCESS?

IF yes -> Return 200 OK immediately (Do nothing).

Step 3: Update DB Status = SUCCESS.

Step 4: Publish Event (NATS):

Subject: payment.processed

Payload: { orderId: "...", status: "SUCCESS", transactionId: "..." }

Note: Order Service listens to this -> Updates Order to PAID -> Tells Inventory to Confirm Stock.

PHASE 4: CRON JOB (RECONCILIATION - "ĐỐI SOÁT")
Webhooks can be lost (Network error). We need a safety net.

Job: Runs every 10 minutes.

Logic:

Find transactions created_at < 10 mins ago AND status = PENDING.

Loop through them:

Call Gateway API (Query Status).

If Gateway says "PAID" -> Sync DB -> Publish NATS event.

If Gateway says "EXPIRED/FAILED" -> Mark DB FAILED -> Publish NATS event (Order Service will Cancel order).

PHASE 5: AUTO-VERIFICATION
Instructions for AI:

Init: Setup Go project.

Generate: Adapters structure.

TEST (Integration):

Create internal/usecase/payment_test.go.

Mock Gateway: Create a MockGateway that returns a fake URL.

Test Idempotency: Call HandleWebhook twice with same payload.

Expectation: NATS event published ONLY ONCE. DB updated ONLY ONCE.

Build: Run go build.

IMPORTANT RULES
Floating Point: NEVER use floats for money. Use shopspring/decimal.

Security: In Webhook handler, never trust the body content until Signature is verified.

Logging: Log everything (Raw Request/Response) to PaymentLog table because debugging payments is a nightmare without logs.