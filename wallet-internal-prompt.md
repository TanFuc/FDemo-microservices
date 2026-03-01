# Wallet Service — Implementation Prompt for Claude Code

## Overview

Create **`wallet/`** as a fully independent microservice.  
Every user **MUST** have a wallet — automatically created when the user registers via a NATS event.  
Users can top up their wallet balance via VNPay/ZaloPay/MoMo/Stripe and pay for orders using their wallet.  
**No data loss is acceptable** — enforce via Transactional Outbox Pattern + NATS JetStream Durable Consumers.

---

## Event Flow Architecture

```
[Auth Service] ──► NATS "user.registered" ────────────────────────────────┐
                                                                           │
                                                                [Wallet Service]
                                                                ┌──────────┴──────────┐
                                                                │   WalletDB           │
                                                                │   wallets            │
                                                                │   wallet_transactions│
                                                                │   outbox_events      │
                                                                └──────────┬──────────┘
                                                                           │
[User] ──► POST /api/v1/wallet/topup ──► Wallet Service                    │
           POST /api/v1/wallet/pay   ──► Wallet Service ──► NATS "wallet.>"
           GET  /api/v1/wallet       ──► Wallet Service
           GET  /api/v1/wallet/history ──► Wallet Service

[Payment Service] ──► NATS "payment.processed" ──► [Wallet Service]
                    (top-up webhook completed)        (credit balance)

NATS "wallet.payment.completed" ──► [Order Service]  (mark order PAID)
NATS "wallet.topped_up"         ──► (notification, analytics...)
NATS "wallet.refunded"          ──► (notification...)

[Order Service] ──► order cancelled/refunded ──► POST /api/v1/wallet/refund (internal)
```

---

## Service Structure

```
wallet/
├── cmd/
│   └── main.go
├── internal/
│   ├── app/
│   │   └── app.go                    # Bootstrap: DB, NATS, consumers, workers
│   ├── config/
│   │   └── config.go                 # Viper config
│   ├── domain/
│   │   ├── wallet.go                 # Wallet aggregate + business logic
│   │   ├── transaction.go            # WalletTransaction ledger entity
│   │   ├── events.go                 # WalletEvent structs + NATS subjects
│   │   └── errors.go                 # Domain errors
│   ├── port/
│   │   ├── repository.go             # WalletRepository interface
│   │   └── publisher.go              # EventPublisher interface
│   ├── usecase/
│   │   ├── provision_wallet.go       # ProvisionWallet (called on user.registered)
│   │   ├── get_wallet.go             # GetWallet + GetHistory
│   │   ├── topup.go                  # InitiateTopUp + CompleteTopUp
│   │   ├── pay.go                    # PayWithWallet
│   │   └── refund.go                 # RefundToWallet
│   ├── infrastructure/
│   │   ├── database/
│   │   │   └── postgres.go           # pgx/v5 pool connection
│   │   ├── repository/
│   │   │   └── wallet_repository.go  # PostgreSQL implementation
│   │   ├── messaging/
│   │   │   ├── nats_publisher.go     # Publish wallet.> events
│   │   │   ├── user_listener.go      # Subscribe user.registered
│   │   │   └── payment_listener.go   # Subscribe payment.processed
│   │   └── outbox/
│   │       └── outbox_worker.go      # Poll outbox_events → publish to NATS
│   ├── handler/
│   │   └── http/
│   │       ├── wallet_handler.go
│   │       └── health_handler.go
│   └── router/
│       └── router.go
├── migrations/
│   └── 001_init.sql
├── go.mod
├── go.sum
├── Dockerfile
├── config.yaml
└── .env.example
```

---

## Module & Dependencies

### `wallet/go.mod`
```
module microservices/wallet

go 1.23.0

require (
    github.com/gofiber/fiber/v2 v2.52.5
    github.com/google/uuid v1.6.0
    github.com/jackc/pgx/v5 v5.7.2
    github.com/nats-io/nats.go v1.38.0
    github.com/shopspring/decimal v1.4.0
    github.com/spf13/viper v1.18.2
    microservices/pkg/authclient v0.0.0
    microservices/pkg/logger v0.0.0
    microservices/pkg/response v0.0.0
)

replace (
    microservices/pkg/authclient => ../pkg/authclient
    microservices/pkg/logger     => ../pkg/logger
    microservices/pkg/response   => ../pkg/response
)
```

---

## Domain Layer

### `wallet/internal/domain/wallet.go`

```go
package domain

import (
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type WalletStatus string

const (
    WalletStatusActive    WalletStatus = "ACTIVE"
    WalletStatusSuspended WalletStatus = "SUSPENDED"
    WalletStatusFrozen    WalletStatus = "FROZEN"
)

type Wallet struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Balance   decimal.Decimal
    Currency  string          // "VND"
    Status    WalletStatus
    CreatedAt time.Time
    UpdatedAt time.Time
    Version   int             // used for optimistic locking
}

func NewWallet(userID uuid.UUID) *Wallet {
    now := time.Now()
    return &Wallet{
        ID: uuid.New(), UserID: userID,
        Balance: decimal.Zero, Currency: "VND",
        Status: WalletStatusActive, Version: 1,
        CreatedAt: now, UpdatedAt: now,
    }
}

func (w *Wallet) CanTransact() error {
    switch w.Status {
    case WalletStatusSuspended:
        return ErrWalletSuspended
    case WalletStatusFrozen:
        return ErrWalletFrozen
    }
    return nil
}

// Credit adds funds to wallet. Returns (balanceBefore, balanceAfter, error).
func (w *Wallet) Credit(amount decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
    if amount.LessThanOrEqual(decimal.Zero) {
        return decimal.Zero, decimal.Zero, ErrInvalidAmount
    }
    if err := w.CanTransact(); err != nil {
        return decimal.Zero, decimal.Zero, err
    }
    before := w.Balance
    w.Balance = w.Balance.Add(amount)
    w.UpdatedAt = time.Now()
    w.Version++
    return before, w.Balance, nil
}

// Debit subtracts funds from wallet. Returns (balanceBefore, balanceAfter, error).
func (w *Wallet) Debit(amount decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
    if amount.LessThanOrEqual(decimal.Zero) {
        return decimal.Zero, decimal.Zero, ErrInvalidAmount
    }
    if err := w.CanTransact(); err != nil {
        return decimal.Zero, decimal.Zero, err
    }
    if w.Balance.LessThan(amount) {
        return decimal.Zero, decimal.Zero, ErrInsufficientBalance
    }
    before := w.Balance
    w.Balance = w.Balance.Sub(amount)
    w.UpdatedAt = time.Now()
    w.Version++
    return before, w.Balance, nil
}
```

### `wallet/internal/domain/transaction.go`

```go
package domain

import (
    "encoding/json"
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type TxType   string
type TxStatus string

const (
    TxTypeTopUp      TxType = "TOP_UP"
    TxTypePayment    TxType = "PAYMENT"
    TxTypeRefund     TxType = "REFUND"
    TxTypeAdjustment TxType = "ADJUSTMENT"

    TxStatusPending   TxStatus = "PENDING"
    TxStatusCompleted TxStatus = "COMPLETED"
    TxStatusFailed    TxStatus = "FAILED"
    TxStatusReversed  TxStatus = "REVERSED"
)

// WalletTransaction is an immutable ledger entry. Never UPDATE amount or type.
type WalletTransaction struct {
    ID             uuid.UUID
    WalletID       uuid.UUID
    UserID         uuid.UUID
    Type           TxType
    Status         TxStatus
    Amount         decimal.Decimal  // always positive
    BalanceBefore  decimal.Decimal
    BalanceAfter   decimal.Decimal
    Currency       string
    ReferenceID    string           // order_id or payment_tx_id
    ReferenceType  string           // "order" or "payment"
    Description    string
    PaymentTxID    *uuid.UUID       // for TOP_UP: links back to payment service tx
    IdempotencyKey string           // UNIQUE — prevents duplicate processing
    Metadata       json.RawMessage
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### `wallet/internal/domain/events.go`

```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

// NATS subjects
const (
    SubjectWalletToppedUp         = "wallet.topped_up"
    SubjectWalletPaymentCompleted = "wallet.payment.completed"
    SubjectWalletPaymentFailed    = "wallet.payment.failed"
    SubjectWalletRefunded         = "wallet.refunded"
    WalletStreamName              = "WALLET_EVENTS"
)

// WalletEvent is published to NATS after every wallet operation.
type WalletEvent struct {
    EventID       string    `json:"event_id"`
    EventType     string    `json:"event_type"`
    WalletID      uuid.UUID `json:"wallet_id"`
    UserID        uuid.UUID `json:"user_id"`
    TxID          uuid.UUID `json:"tx_id"`
    Amount        string    `json:"amount"`
    BalanceBefore string    `json:"balance_before"`
    BalanceAfter  string    `json:"balance_after"`
    Currency      string    `json:"currency"`
    ReferenceID   string    `json:"reference_id,omitempty"`
    ReferenceType string    `json:"reference_type,omitempty"`
    Timestamp     time.Time `json:"timestamp"`
}

// OutboxEvent is persisted in DB for guaranteed delivery via polling worker.
type OutboxEvent struct {
    ID          uuid.UUID
    AggregateID string
    EventType   string
    Payload     []byte
    Status      string     // "PENDING", "PUBLISHED", "FAILED"
    RetryCount  int
    CreatedAt   time.Time
    PublishedAt *time.Time
}
```

### `wallet/internal/domain/errors.go`

```go
package domain

import "errors"

var (
    ErrWalletNotFound      = errors.New("wallet not found")
    ErrWalletAlreadyExists = errors.New("wallet already exists for this user")
    ErrWalletSuspended     = errors.New("wallet is suspended")
    ErrWalletFrozen        = errors.New("wallet is frozen")
    ErrInsufficientBalance = errors.New("insufficient wallet balance")
    ErrInvalidAmount       = errors.New("amount must be positive")
    ErrDuplicateTx         = errors.New("duplicate transaction (idempotency key already exists)")
    ErrVersionConflict     = errors.New("wallet version conflict — please retry")
    ErrPaymentTxNotFound   = errors.New("payment transaction not found")
)
```

---

## Port Interfaces

### `wallet/internal/port/repository.go`

```go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "microservices/wallet/internal/domain"
)

type WalletRepository interface {
    // Wallet
    CreateWallet(ctx context.Context, w *domain.Wallet) error
    GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error)
    GetByID(ctx context.Context, walletID uuid.UUID) (*domain.Wallet, error)
    ExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error)

    // Optimistic locking balance update.
    // Returns ErrVersionConflict if no rows updated (version mismatch).
    UpdateBalanceOptimistic(ctx context.Context, walletID uuid.UUID, newBalance decimal.Decimal, newVersion, expectedVersion int) error

    // Ledger
    CreateTransaction(ctx context.Context, tx *domain.WalletTransaction) error
    GetTransactionByID(ctx context.Context, txID uuid.UUID) (*domain.WalletTransaction, error)
    GetTransactionByIdempotencyKey(ctx context.Context, key string) (*domain.WalletTransaction, error)
    GetTransactionByPaymentTxID(ctx context.Context, paymentTxID uuid.UUID) (*domain.WalletTransaction, error)
    UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status domain.TxStatus) error
    ListTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*domain.WalletTransaction, int64, error)

    // Outbox
    CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error
    GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
    MarkOutboxPublished(ctx context.Context, eventID uuid.UUID) error
    MarkOutboxFailed(ctx context.Context, eventID uuid.UUID) error

    // RunInTransaction executes fn within a single PostgreSQL transaction.
    // balance update + ledger insert + outbox insert must ALL be in the SAME pg tx.
    RunInTransaction(ctx context.Context, fn func(tx TxContext) error) error
}

// TxContext represents the operations available inside an active DB transaction.
type TxContext interface {
    CreateTransaction(ctx context.Context, tx *domain.WalletTransaction) error
    UpdateBalanceOptimistic(ctx context.Context, walletID uuid.UUID, newBalance decimal.Decimal, newVersion, expectedVersion int) error
    CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error
}
```

### `wallet/internal/port/publisher.go`

```go
package port

import (
    "context"
    "microservices/wallet/internal/domain"
)

type EventPublisher interface {
    Publish(ctx context.Context, subject string, event *domain.WalletEvent) error
    PublishRaw(ctx context.Context, subject string, payload []byte) error
    Close() error
}
```

---

## Usecase Layer

### `wallet/internal/usecase/provision_wallet.go`

Called by `UserRegisteredListener` when `user.registered` event is received.

```go
// Execute is idempotent: if wallet already exists, return nil (no error, no duplicate).
func (uc *ProvisionWalletUseCase) Execute(ctx context.Context, userID uuid.UUID, email string) error {
    exists, err := uc.repo.ExistsByUserID(ctx, userID)
    if err != nil {
        return fmt.Errorf("check wallet exists: %w", err)
    }
    if exists {
        uc.logger.Info("wallet already provisioned (idempotent)", "user_id", userID)
        return nil
    }
    wallet := domain.NewWallet(userID)
    if err := uc.repo.CreateWallet(ctx, wallet); err != nil {
        if isDuplicateKeyError(err) {
            return nil // race condition: another instance just created it
        }
        return fmt.Errorf("create wallet: %w", err)
    }
    uc.logger.Info("wallet provisioned", "user_id", userID, "wallet_id", wallet.ID)
    return nil
}
```

### `wallet/internal/usecase/topup.go`

```go
// InitiateTopUp creates a PENDING wallet transaction, then calls the Payment Service
// HTTP API to generate a payment URL. Wallet service does NOT embed gateway adapters —
// it delegates all payment gateway logic to the Payment Service.
type InitiateTopUpRequest struct {
    UserID         uuid.UUID
    Amount         decimal.Decimal
    Provider       string          // "VNPAY", "ZALOPAY", "MOMO", "STRIPE"
    Description    string
    IdempotencyKey string          // from X-Idempotency-Key header
    ReturnURL      string
}

type InitiateTopUpResponse struct {
    WalletTxID  uuid.UUID `json:"wallet_tx_id"`
    PaymentURL  string    `json:"payment_url"`
    PaymentTxID string    `json:"payment_tx_id"`
    Amount      string    `json:"amount"`
}

// Flow:
// 1. Check idempotency key → if PENDING wallet tx exists → return its linked payment_url
// 2. GetByUserID(userID) → return ErrWalletNotFound if not found
// 3. Validate amount: min 10,000 VND, max 50,000,000 VND
// 4. Call Payment Service HTTP: POST /api/v1/payments
//    { order_id: walletTxID, user_id, amount, provider, is_wallet_topup: true, wallet_tx_id: walletTxID }
// 5. Create WalletTransaction { type=TOP_UP, status=PENDING, payment_tx_id, idempotency_key }
// 6. Return payment_url to the client

// CompleteTopUp is called by PaymentListener when payment.processed event is received.
// Flow:
// 1. GetTransactionByPaymentTxID(paymentTxID) → find the PENDING wallet tx
// 2. If not found → skip (not a wallet top-up payment)
// 3. If already COMPLETED → return nil (idempotent)
// 4. GetByID(walletTx.WalletID)
// 5. wallet.Credit(amount)
// 6. RunInTransaction:
//    a. UpdateBalanceOptimistic (optimistic lock, retry up to 3x on version conflict)
//    b. UpdateTransactionStatus(txID, COMPLETED)
//    c. CreateOutboxEvent { event_type="wallet.topped_up" }
// 7. Attempt direct NATS publish (best effort); outbox worker handles failures
```

### `wallet/internal/usecase/pay.go`

```go
// PayWithWallet debits the wallet for an order payment.
type WalletPayRequest struct {
    UserID         uuid.UUID
    OrderID        uuid.UUID
    Amount         decimal.Decimal
    Description    string
    IdempotencyKey string
}

type WalletPayResponse struct {
    WalletTxID    uuid.UUID       `json:"wallet_tx_id"`
    BalanceBefore decimal.Decimal `json:"balance_before"`
    BalanceAfter  decimal.Decimal `json:"balance_after"`
}

// Flow:
// 1. Build idempotencyKey = "pay:{userID}:{orderID}" (merge with client key if provided)
// 2. GetTransactionByIdempotencyKey(key) → if COMPLETED → return existing (idempotent, HTTP 200)
// 3. GetByUserID(userID) → ErrWalletNotFound (HTTP 404) if no wallet
// 4. wallet.Debit(amount) → ErrInsufficientBalance (HTTP 422)
// 5. RunInTransaction (ALL must succeed or rollback):
//    a. UpdateBalanceOptimistic → retry up to 3x on ErrVersionConflict
//    b. CreateTransaction { type=PAYMENT, status=COMPLETED, idempotency_key, reference_id=orderID }
//    c. CreateOutboxEvent { event_type="wallet.payment.completed", reference_id=orderID }
// 6. Attempt direct NATS publish (best effort)
// 7. Return WalletPayResponse
```

### `wallet/internal/usecase/refund.go`

```go
// RefundToWallet credits the wallet when an order is cancelled or refunded.
type WalletRefundRequest struct {
    UserID         uuid.UUID
    OrderID        uuid.UUID
    Amount         decimal.Decimal
    Reason         string
    IdempotencyKey string
}

// Flow:
// 1. Build idempotencyKey = "refund:{orderID}"
// 2. Check idempotency → if already COMPLETED → return nil
// 3. GetByUserID(userID)
// 4. wallet.Credit(amount)
// 5. RunInTransaction:
//    a. UpdateBalanceOptimistic
//    b. CreateTransaction { type=REFUND, status=COMPLETED }
//    c. CreateOutboxEvent { event_type="wallet.refunded" }
// 6. Attempt direct NATS publish (best effort)
```

---

## Database Migration

### `wallet/migrations/001_init.sql`

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE wallet_status    AS ENUM ('ACTIVE', 'SUSPENDED', 'FROZEN');
CREATE TYPE wallet_tx_type   AS ENUM ('TOP_UP', 'PAYMENT', 'REFUND', 'ADJUSTMENT');
CREATE TYPE wallet_tx_status AS ENUM ('PENDING', 'COMPLETED', 'FAILED', 'REVERSED');

-- =====================================================
-- wallets — one row per user (enforced by UNIQUE)
-- =====================================================
CREATE TABLE wallets (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL UNIQUE,
    balance     DECIMAL(19,4) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    currency    VARCHAR(3) NOT NULL DEFAULT 'VND',
    status      wallet_status NOT NULL DEFAULT 'ACTIVE',
    version     INT NOT NULL DEFAULT 1,           -- optimistic locking
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_wallets_user_id ON wallets(user_id);

-- =====================================================
-- wallet_transactions — immutable ledger
-- =====================================================
CREATE TABLE wallet_transactions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id       UUID NOT NULL REFERENCES wallets(id),
    user_id         UUID NOT NULL,
    type            wallet_tx_type NOT NULL,
    status          wallet_tx_status NOT NULL DEFAULT 'PENDING',
    amount          DECIMAL(19,4) NOT NULL CHECK (amount > 0),
    balance_before  DECIMAL(19,4) NOT NULL,
    balance_after   DECIMAL(19,4) NOT NULL,
    currency        VARCHAR(3) NOT NULL DEFAULT 'VND',
    reference_id    VARCHAR(255),
    reference_type  VARCHAR(50),
    description     TEXT,
    payment_tx_id   UUID,                   -- links to payment service transaction
    idempotency_key VARCHAR(512),
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_wallet_tx_wallet_id    ON wallet_transactions(wallet_id);
CREATE INDEX idx_wallet_tx_user_id      ON wallet_transactions(user_id);
CREATE INDEX idx_wallet_tx_status       ON wallet_transactions(status);
CREATE INDEX idx_wallet_tx_reference    ON wallet_transactions(reference_id, reference_type);
CREATE INDEX idx_wallet_tx_payment_tx   ON wallet_transactions(payment_tx_id) WHERE payment_tx_id IS NOT NULL;
CREATE INDEX idx_wallet_tx_created_at   ON wallet_transactions(created_at DESC);
CREATE UNIQUE INDEX idx_wallet_tx_idempotency ON wallet_transactions(idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- =====================================================
-- outbox_events — Transactional Outbox Pattern
-- Guarantees at-least-once NATS delivery even when NATS is temporarily down.
-- =====================================================
CREATE TABLE outbox_events (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id VARCHAR(255) NOT NULL,
    event_type   VARCHAR(100) NOT NULL,
    payload      JSONB NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'PENDING',   -- PENDING | PUBLISHED | FAILED
    retry_count  INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);
CREATE INDEX idx_outbox_pending   ON outbox_events(created_at) WHERE status = 'PENDING';
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_id);

-- =====================================================
-- Auto-update updated_at trigger
-- =====================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_wallets_updated_at
    BEFORE UPDATE ON wallets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_wallet_tx_updated_at
    BEFORE UPDATE ON wallet_transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE wallets IS
    'Internal user wallets — one wallet per user (enforced by UNIQUE on user_id)';
COMMENT ON TABLE wallet_transactions IS
    'Immutable double-entry ledger — never UPDATE amount or type fields';
COMMENT ON TABLE outbox_events IS
    'Transactional outbox — guarantees at-least-once NATS delivery';
COMMENT ON COLUMN wallets.version IS
    'Optimistic locking version — must match on UPDATE or reject with ErrVersionConflict';
COMMENT ON COLUMN wallet_transactions.idempotency_key IS
    'Idempotency key format: {type}:{userID}:{referenceID}';
```

---

## Messaging Layer

### `wallet/internal/infrastructure/messaging/user_listener.go`

Subscribes to `user.registered` from the `USERS` stream (already created by the auth service).

```go
package messaging

// UserRegisteredEvent mirrors auth/internal/queue/nats.go (UserRegisteredEvent)
type UserRegisteredEvent struct {
    UserID       string `json:"userId"`
    Email        string `json:"email"`
    FullName     string `json:"fullName"`
    RegisteredAt string `json:"registeredAt"`
}

type UserRegisteredListener struct {
    js          nats.JetStreamContext
    provisionUC ProvisionWalletUseCase
    logger      *slog.Logger
}

func (l *UserRegisteredListener) Start(ctx context.Context) error {
    // The "USERS" stream is already created by the auth service.
    // Wallet service only creates a durable consumer on the existing stream.
    _, err := l.js.Subscribe(
        "user.registered",
        l.handle,
        nats.Durable("wallet-user-provision-consumer"),
        nats.DeliverNew(),         // only new messages after consumer creation
        nats.ManualAck(),
        nats.AckWait(30*time.Second),
        nats.MaxDeliver(10),       // retry up to 10 times on failure
    )
    if err != nil {
        return fmt.Errorf("subscribe user.registered: %w", err)
    }
    <-ctx.Done()
    return nil
}

func (l *UserRegisteredListener) handle(msg *nats.Msg) {
    var event UserRegisteredEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        l.logger.Error("failed to unmarshal user.registered", "error", err)
        msg.Term() // malformed — do not retry
        return
    }
    userID, err := uuid.Parse(event.UserID)
    if err != nil {
        l.logger.Error("invalid userID in event", "user_id", event.UserID)
        msg.Term()
        return
    }
    if err := l.provisionUC.Execute(context.Background(), userID, event.Email); err != nil {
        l.logger.Error("failed to provision wallet", "user_id", event.UserID, "error", err)
        msg.Nak() // retry
        return
    }
    msg.Ack()
}
```

### `wallet/internal/infrastructure/messaging/payment_listener.go`

Subscribes to `payment.processed` from the `PAYMENTS` stream (already created by the payment service).

```go
package messaging

// PaymentProcessedEvent mirrors payment service's PaymentEvent
type PaymentProcessedEvent struct {
    OrderID       string `json:"order_id"`
    TransactionID string `json:"transaction_id"`
    Status        string `json:"status"`    // "SUCCESS" or "FAILED"
    Amount        string `json:"amount"`
    Currency      string `json:"currency"`
    IsWalletTopup bool   `json:"is_wallet_topup"`
    WalletTxID    string `json:"wallet_tx_id,omitempty"`
    Timestamp     string `json:"timestamp"`
}

type PaymentEventListener struct {
    js              nats.JetStreamContext
    completeTopUpUC CompleteTopUpUseCase
    logger          *slog.Logger
}

func (l *PaymentEventListener) Start(ctx context.Context) error {
    // The "PAYMENTS" stream is already created by the payment service.
    _, err := l.js.Subscribe(
        "payment.processed",
        l.handle,
        nats.Durable("wallet-topup-completion-consumer"),
        nats.DeliverNew(),
        nats.ManualAck(),
        nats.AckWait(30*time.Second),
        nats.MaxDeliver(5),
    )
    if err != nil {
        return fmt.Errorf("subscribe payment.processed: %w", err)
    }
    <-ctx.Done()
    return nil
}

func (l *PaymentEventListener) handle(msg *nats.Msg) {
    var event PaymentProcessedEvent
    json.Unmarshal(msg.Data, &event)

    // Only handle successful wallet top-up payments
    if event.Status != "SUCCESS" || !event.IsWalletTopup {
        msg.Ack()
        return
    }

    paymentTxID, _ := uuid.Parse(event.TransactionID)
    if err := l.completeTopUpUC.Execute(context.Background(), paymentTxID); err != nil {
        if errors.Is(err, domain.ErrPaymentTxNotFound) {
            msg.Ack() // not a wallet top-up, skip silently
            return
        }
        l.logger.Error("failed to complete top-up", "payment_tx_id", event.TransactionID, "error", err)
        msg.Nak() // retry
        return
    }
    msg.Ack()
}
```

### `wallet/internal/infrastructure/messaging/nats_publisher.go`

```go
package messaging

type NATSPublisher struct {
    nc *nats.Conn
    js nats.JetStreamContext
}

func NewNATSPublisher(url string) (*NATSPublisher, error) {
    nc, err := nats.Connect(url,
        nats.RetryOnFailedConnect(true),
        nats.MaxReconnects(10),
        nats.ReconnectWait(time.Second),
    )
    if err != nil {
        return nil, fmt.Errorf("connect to NATS: %w", err)
    }
    js, err := nc.JetStream()
    if err != nil {
        nc.Close()
        return nil, err
    }

    // Create WALLET_EVENTS stream (idempotent — ok if already exists)
    js.AddStream(&nats.StreamConfig{
        Name:      "WALLET_EVENTS",
        Subjects:  []string{"wallet.>"},
        Storage:   nats.FileStorage,
        Retention: nats.WorkQueuePolicy,
        MaxAge:    7 * 24 * time.Hour,
    })

    return &NATSPublisher{nc: nc, js: js}, nil
}

func (p *NATSPublisher) Publish(ctx context.Context, subject string, event *domain.WalletEvent) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    _, err = p.js.Publish(subject, data)
    return err
}

func (p *NATSPublisher) PublishRaw(ctx context.Context, subject string, payload []byte) error {
    _, err := p.js.Publish(subject, payload)
    return err
}

func (p *NATSPublisher) Close() error {
    p.nc.Drain()
    return nil
}
```

### `wallet/internal/infrastructure/outbox/outbox_worker.go`

Polls the `outbox_events` table every 500ms and publishes pending events to NATS.  
Mirrors the pattern used in `order/internal/infrastructure/outbox/outbox.go`.

```go
package outbox

type OutboxWorker struct {
    repo      port.WalletRepository
    publisher port.EventPublisher
    logger    *slog.Logger
    interval  time.Duration
    batchSize int
}

func NewOutboxWorker(repo port.WalletRepository, publisher port.EventPublisher, logger *slog.Logger) *OutboxWorker {
    return &OutboxWorker{
        repo: repo, publisher: publisher, logger: logger,
        interval:  500 * time.Millisecond,
        batchSize: 100,
    }
}

func (w *OutboxWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    w.logger.Info("outbox worker started", "interval", w.interval, "batch_size", w.batchSize)
    for {
        select {
        case <-ctx.Done():
            w.logger.Info("outbox worker stopped")
            return
        case <-ticker.C:
            w.processBatch(ctx)
        }
    }
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
    events, err := w.repo.GetPendingOutboxEvents(ctx, w.batchSize)
    if err != nil {
        w.logger.Error("failed to fetch pending outbox events", "error", err)
        return
    }
    for _, e := range events {
        if err := w.publisher.PublishRaw(ctx, e.EventType, e.Payload); err != nil {
            w.logger.Error("failed to publish outbox event",
                "event_id", e.ID, "event_type", e.EventType, "error", err)
            _ = w.repo.MarkOutboxFailed(ctx, e.ID)
            continue
        }
        _ = w.repo.MarkOutboxPublished(ctx, e.ID)
        w.logger.Debug("outbox event published", "event_id", e.ID, "event_type", e.EventType)
    }
}
```

---

## Repository Implementation

### `wallet/internal/infrastructure/repository/wallet_repository.go`

Implement `port.WalletRepository` using `pgx/v5`. Follow the same pattern as `payment/internal/adapter/repository/postgres.go`.

**Critical — `RunInTransaction` must wrap balance + ledger + outbox in a single `pgx` transaction:**

```go
func (r *PostgresWalletRepository) RunInTransaction(ctx context.Context, fn func(tx port.TxContext) error) error {
    pgTx, err := r.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer pgTx.Rollback(ctx) // no-op if committed

    txCtx := &pgTxContext{tx: pgTx}
    if err := fn(txCtx); err != nil {
        return err // rollback via defer
    }
    return pgTx.Commit(ctx)
}

// UpdateBalanceOptimistic inside transaction — optimistic lock:
func (t *pgTxContext) UpdateBalanceOptimistic(
    ctx context.Context,
    walletID uuid.UUID,
    newBalance decimal.Decimal,
    newVersion, expectedVersion int,
) error {
    tag, err := t.tx.Exec(ctx,
        `UPDATE wallets SET balance=$1, version=$2, updated_at=NOW()
         WHERE id=$3 AND version=$4`,
        newBalance, newVersion, walletID, expectedVersion,
    )
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return domain.ErrVersionConflict
    }
    return nil
}
```

---

## HTTP Handler

### `wallet/internal/handler/http/wallet_handler.go`

```go
type WalletHandler struct {
    provisionUC ProvisionWalletUseCase
    getUC       GetWalletUseCase
    topupUC     InitiateTopUpUseCase
    payUC       PayWithWalletUseCase
    refundUC    RefundToWalletUseCase
}

// GET /api/v1/wallet
// Auth: Required. Extract userID via c.Locals("user_id").(string)
func (h *WalletHandler) GetWallet(c *fiber.Ctx) error

// GET /api/v1/wallet/history?limit=20&offset=0
// Auth: Required.
func (h *WalletHandler) GetHistory(c *fiber.Ctx) error

// POST /api/v1/wallet/topup
// Body: { "amount":"200000", "provider":"VNPAY", "return_url":"https://..." }
// Header: X-Idempotency-Key (optional)
// Auth: Required.
func (h *WalletHandler) InitiateTopUp(c *fiber.Ctx) error

// POST /api/v1/wallet/pay
// Body: { "order_id":"uuid", "amount":"150000" }
// Header: X-Idempotency-Key (optional — defaults to "pay:{userID}:{orderID}")
// Auth: Required.
func (h *WalletHandler) PayWithWallet(c *fiber.Ctx) error

// POST /api/v1/wallet/refund   (internal endpoint — called by order service)
// Body: { "user_id":"uuid", "order_id":"uuid", "amount":"150000", "reason":"cancelled" }
// Auth: Internal service token OR admin role
func (h *WalletHandler) RefundToWallet(c *fiber.Ctx) error
```

---

## Config

### `wallet/internal/config/config.go`

```go
type Config struct {
    App     AppConfig
    DB      DatabaseConfig
    NATS    NATSConfig
    Payment PaymentServiceConfig // used by InitiateTopUp to call payment service
    Wallet  WalletLimitsConfig
}

type AppConfig struct {
    Port         string
    Host         string
    AuthGRPCAddr string // for Auth middleware JWT validation
}

type DatabaseConfig struct {
    URL string // postgres://user:pass@host:port/db?sslmode=disable
}

type NATSConfig struct {
    URL string
}

type PaymentServiceConfig struct {
    BaseURL string // e.g. "http://payment-service:8083"
}

type WalletLimitsConfig struct {
    MinTopUpVND   int64 // default: 10_000
    MaxTopUpVND   int64 // default: 50_000_000
    MaxBalanceVND int64 // default: 200_000_000
}
```

### `wallet/.env.example`

```
SERVER_PORT=8086
SERVER_HOST=0.0.0.0
DATABASE_URL=postgres://wallet:wallet_secret@localhost:5432/wallet_db?sslmode=disable
NATS_URL=nats://localhost:4222
AUTH_GRPC_ADDR=localhost:50051
PAYMENT_SERVICE_URL=http://localhost:8083
WALLET_MIN_TOPUP=10000
WALLET_MAX_TOPUP=50000000
WALLET_MAX_BALANCE=200000000
```

---

## Changes Required in Existing Services

### 1. Payment Service (`payment/`) — Add `is_wallet_topup` flag

When the wallet service calls `POST /api/v1/payments` for a top-up, it passes `is_wallet_topup: true` and `wallet_tx_id` in the request body. Payment service must store these and include them in the `payment.processed` event.

**`payment/internal/domain/payment.go`** — add to `PaymentTransaction`:
```go
IsWalletTopup bool    `json:"is_wallet_topup"`
WalletTxID   *string `json:"wallet_tx_id,omitempty"`
```

**`payment/migrations/004_wallet_flag.sql`**:
```sql
ALTER TABLE payment_transactions
    ADD COLUMN IF NOT EXISTS is_wallet_topup BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS wallet_tx_id    VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_payment_tx_wallet_topup
    ON payment_transactions(is_wallet_topup) WHERE is_wallet_topup = TRUE;
```

**`payment/internal/handler/http/payment_handler.go`** — add to `CreatePaymentRequest`:
```go
IsWalletTopup bool   `json:"is_wallet_topup"`
WalletTxID   string `json:"wallet_tx_id"`
```

The published `payment.processed` event must include `is_wallet_topup` and `wallet_tx_id` fields so the wallet service can filter relevant events.

### 2. Order Service (`order/`) — Listen to `wallet.payment.completed`

**`order/internal/infrastructure/messaging/wallet_listener.go`** (new file):

```go
package messaging

// WalletPaymentCompletedEvent mirrors wallet service's WalletEvent
type WalletPaymentCompletedEvent struct {
    EventID       string `json:"event_id"`
    EventType     string `json:"event_type"`
    UserID        string `json:"user_id"`
    TxID          string `json:"tx_id"`
    Amount        string `json:"amount"`
    ReferenceID   string `json:"reference_id"`   // = order_id
    ReferenceType string `json:"reference_type"` // = "order"
    Timestamp     string `json:"timestamp"`
}

type WalletEventListener struct {
    js        nats.JetStreamContext
    orderRepo domain.OrderRepository
    confirmer StockConfirmer
    publisher EventPublisher
    logger    *slog.Logger
}

func (l *WalletEventListener) Start(ctx context.Context) error {
    // The "WALLET_EVENTS" stream is created by the wallet service on startup.
    _, err := l.js.Subscribe(
        "wallet.payment.completed",
        l.handleWalletPayment,
        nats.Durable("order-wallet-payment-consumer"),
        nats.DeliverNew(),
        nats.ManualAck(),
        nats.AckWait(30*time.Second),
        nats.MaxDeliver(5),
    )
    if err != nil {
        return fmt.Errorf("subscribe wallet.payment.completed: %w", err)
    }
    <-ctx.Done()
    return nil
}

func (l *WalletEventListener) handleWalletPayment(msg *nats.Msg) {
    var event WalletPaymentCompletedEvent
    json.Unmarshal(msg.Data, &event)

    if event.ReferenceType != "order" {
        msg.Ack()
        return
    }

    orderID, _ := uuid.Parse(event.ReferenceID)
    order, err := l.orderRepo.GetByID(context.Background(), orderID)
    if err != nil {
        msg.Nak()
        return
    }

    // Idempotent: if already PAID, skip
    if order.PaymentStatus == domain.PaymentPaid {
        msg.Ack()
        return
    }

    order.MarkAsPaid()
    order.PaymentMethod = "WALLET"
    order.PaymentProvider = "WALLET"
    if err := l.orderRepo.Update(context.Background(), order); err != nil {
        msg.Nak()
        return
    }

    // Confirm stock reservation
    l.confirmer.ConfirmStock(context.Background(), order.ID.String())

    // Publish order.paid event
    l.publisher.PublishOrderPaid(context.Background(), &OrderPaidEvent{
        OrderID: order.ID,
        UserID:  order.UserID,
        PaidAt:  time.Now(),
    })

    msg.Ack()
}
```

**`order/internal/app/app.go`** — wire up the wallet listener:
```go
walletListener, err := messaging.NewWalletEventListener(&cfg.NATS, orderRepo, stockConfirmer, eventPublisher, slogger)
if err != nil {
    logger.Warn().Err(err).Msg("Failed to create wallet event listener")
} else {
    go func() {
        if err := walletListener.Start(ctx); err != nil {
            logger.Error().Err(err).Msg("Wallet event listener error")
        }
    }()
    logger.Info().Msg("Wallet event listener started")
}
```

---

## Data Integrity Guarantees

### Mechanism 1: Transactional Outbox
```
BEGIN DB Transaction (single pgx tx):
  UPDATE wallets SET balance=X, version=N+1   ← balance changes here
  INSERT INTO wallet_transactions (...)        ← immutable ledger entry
  INSERT INTO outbox_events (PENDING, ...)     ← event queued for NATS
COMMIT

OutboxWorker (every 500ms):
  SELECT * FROM outbox_events WHERE status='PENDING' LIMIT 100
  → Publish to NATS JetStream
  → UPDATE status='PUBLISHED'
```

### Mechanism 2: NATS JetStream Durable Consumers
```
"wallet-user-provision-consumer"      ← creates wallet on user.registered
"wallet-topup-completion-consumer"    ← completes top-up on payment.processed
"order-wallet-payment-consumer"       ← marks order PAID on wallet.payment.completed

MaxDeliver: 5–10, AckWait: 30s
→ Messages are never lost even if service restarts
```

### Mechanism 3: Optimistic Locking
```sql
UPDATE wallets SET balance=$1, version=$2
WHERE id=$3 AND version=$4  -- must match current version
-- RowsAffected == 0 → ErrVersionConflict → retry up to 3x with backoff
```

### Mechanism 4: Idempotency Keys (UNIQUE constraint)
```sql
CREATE UNIQUE INDEX idx_wallet_tx_idempotency
    ON wallet_transactions(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Key formats:
-- Top-up:  idempotency key from X-Idempotency-Key header
-- Payment: "pay:{userID}:{orderID}"
-- Refund:  "refund:{orderID}"
-- Duplicate request → return existing result (HTTP 200, not 409)
```

### Mechanism 5: Retry with Exponential Backoff
```go
for attempt := 0; attempt < 3; attempt++ {
    err := repo.RunInTransaction(ctx, fn)
    if err == nil { break }
    if !errors.Is(err, domain.ErrVersionConflict) { return err }
    time.Sleep(time.Duration(attempt*10) * time.Millisecond) // 0ms, 10ms, 20ms
}
```

---

## NATS Streams Summary

| Stream | Subjects | Created by | Consumers |
|--------|----------|------------|-----------|
| `USERS` | `user.*` | Auth Service | Wallet Service (provision) |
| `PAYMENTS` | `payment.*` | Payment Service | Wallet Service (complete top-up), Order Service |
| `WALLET_EVENTS` | `wallet.*` | Wallet Service | Order Service (mark order paid), Notification... |
| `ORDERS` | `order.*` | Order Service | Inventory, Analytics... |

---

## Implementation Checklist

### Wallet Service (create from scratch)
- [ ] `wallet/go.mod`
- [ ] `wallet/cmd/main.go`
- [ ] `wallet/internal/config/config.go`
- [ ] `wallet/internal/domain/wallet.go`
- [ ] `wallet/internal/domain/transaction.go`
- [ ] `wallet/internal/domain/events.go`
- [ ] `wallet/internal/domain/errors.go`
- [ ] `wallet/internal/port/repository.go`
- [ ] `wallet/internal/port/publisher.go`
- [ ] `wallet/internal/infrastructure/database/postgres.go`
- [ ] `wallet/internal/infrastructure/repository/wallet_repository.go`
- [ ] `wallet/internal/infrastructure/messaging/nats_publisher.go`
- [ ] `wallet/internal/infrastructure/messaging/user_listener.go`
- [ ] `wallet/internal/infrastructure/messaging/payment_listener.go`
- [ ] `wallet/internal/infrastructure/outbox/outbox_worker.go`
- [ ] `wallet/internal/usecase/provision_wallet.go`
- [ ] `wallet/internal/usecase/get_wallet.go`
- [ ] `wallet/internal/usecase/topup.go`
- [ ] `wallet/internal/usecase/pay.go`
- [ ] `wallet/internal/usecase/refund.go`
- [ ] `wallet/internal/handler/http/wallet_handler.go`
- [ ] `wallet/internal/handler/http/health_handler.go`
- [ ] `wallet/internal/router/router.go`
- [ ] `wallet/internal/app/app.go`
- [ ] `wallet/migrations/001_init.sql`
- [ ] `wallet/Dockerfile`
- [ ] `wallet/.env.example`

### Payment Service (minor changes)
- [ ] `payment/internal/domain/payment.go` — add `IsWalletTopup`, `WalletTxID` fields
- [ ] `payment/internal/handler/http/payment_handler.go` — accept `is_wallet_topup` in request body
- [ ] `payment/migrations/004_wallet_flag.sql` — add columns to `payment_transactions`

### Order Service (minor changes)
- [ ] `order/internal/infrastructure/messaging/wallet_listener.go` — new file
- [ ] `order/internal/app/app.go` — start `WalletEventListener`

---

## Critical Implementation Notes

1. **Wallet service does NOT embed payment gateway code.** It calls the Payment Service REST API (`POST /api/v1/payments`) to generate a payment URL. Gateway logic remains exclusively in the payment service.

2. **HTTP port for wallet service: `8086`** (payment=8083, order=8084 — adjust if different).

3. **Separate database: `wallet_db`** — do not share the payment service database.

4. **`RunInTransaction` is non-negotiable** — balance update, ledger insert, and outbox insert MUST be atomic within a single `pgx` transaction.

5. **User listener uses `DeliverNew()`** — only receives users who register AFTER the wallet service is deployed. Run a one-time backfill migration for existing users:
   ```sql
   -- One-time script: provision wallets for all existing users
   INSERT INTO wallet_db.wallets (id, user_id, balance, currency, status, version, created_at, updated_at)
   SELECT uuid_generate_v4(), id, 0, 'VND', 'ACTIVE', 1, NOW(), NOW()
   FROM auth_db.users
   ON CONFLICT (user_id) DO NOTHING;
   ```

6. **`payment_tx_id` in `wallet_transactions`** is the link that allows the wallet service to locate the correct pending top-up transaction when it receives a `payment.processed` event.

7. **Optimistic lock retry pattern:**
   ```go
   for attempt := 0; attempt < 3; attempt++ {
       if err := repo.RunInTransaction(ctx, fn); err == nil {
           break
       }
       if !errors.Is(err, domain.ErrVersionConflict) {
           return err
       }
       time.Sleep(time.Duration(attempt*10) * time.Millisecond)
   }
   ```

8. **Dead-letter handling:** If `outbox_events.retry_count > 5`, alert the on-call team. NATS `MaxDeliver` exceeded messages move to `wallet.dlq.>` subject for investigation.
