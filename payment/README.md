# Payment Aggregator Service

A PCI-DSS compliant payment aggregator service built in Go that supports multiple payment gateways (Stripe, MoMo, COD).

## Architecture

This service uses **Hexagonal Architecture** (Ports & Adapters) for clean separation of concerns:

```
internal/
├── domain/          # Core business entities
├── port/            # Interface definitions (ports)
├── adapter/         # Implementation of ports (adapters)
│   ├── gateway/     # Payment gateway adapters
│   ├── repository/  # Database adapters
│   └── publisher/   # Event publishing adapters
├── usecase/         # Business logic
├── handler/         # HTTP handlers
└── job/             # Background jobs
```

## Features

- Multi-gateway support: Stripe, MoMo, Cash on Delivery
- Webhook handling with signature verification
- Idempotent payment processing
- Reconciliation cron job for pending payments
- NATS JetStream event publishing
- Audit logging for all transactions

## Prerequisites

- Go 1.22+
- Docker and Docker Compose
- PostgreSQL 16+
- NATS 2.10+

## Quick Start

1. **Clone and setup environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

2. **Start infrastructure:**
   ```bash
   docker-compose up -d
   ```

3. **Install dependencies:**
   ```bash
   go mod tidy
   ```

4. **Run the service:**
   ```bash
   go run cmd/server/main.go
   ```

5. **Run tests:**
   ```bash
   go test ./...
   ```

## API Endpoints

### Create Payment
```http
POST /api/v1/payments
Content-Type: application/json

{
  "order_id": "uuid",
  "user_id": "uuid",
  "amount": "100.50",
  "currency": "USD",
  "provider": "STRIPE",
  "description": "Order payment",
  "callback_url": "https://your-domain.com/api/v1/webhooks/stripe",
  "return_url": "https://your-domain.com/payment/complete"
}
```

Response:
```json
{
  "transaction_id": "uuid",
  "payment_url": "https://checkout.stripe.com/...",
  "provider_tx_id": "cs_..."
}
```

### Get Payment
```http
GET /api/v1/payments/{id}
```

### Get Payments by Order
```http
GET /api/v1/payments/order/{orderId}
```

### Webhook Endpoints
```http
POST /api/v1/webhooks/stripe
POST /api/v1/webhooks/momo
POST /api/v1/webhooks/cod
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | 8080 |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `NATS_URL` | NATS server URL | nats://localhost:4222 |
| `STRIPE_SECRET_KEY` | Stripe secret key | - |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook secret | - |
| `MOMO_PARTNER_CODE` | MoMo partner code | - |
| `MOMO_ACCESS_KEY` | MoMo access key | - |
| `MOMO_SECRET_KEY` | MoMo secret key | - |

## Event Publishing

The service publishes events to NATS JetStream:

| Subject | Description |
|---------|-------------|
| `payment.processed` | Payment completed successfully |
| `payment.failed` | Payment failed |
| `payment.refunded` | Payment was refunded |

Event payload:
```json
{
  "order_id": "uuid",
  "transaction_id": "uuid",
  "status": "SUCCESS",
  "provider": "STRIPE",
  "amount": "100.50",
  "currency": "USD",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Security

- All webhooks are verified using provider-specific signature verification
- Never trust webhook payload until signature is verified
- All monetary amounts use `shopspring/decimal` (no floating point)
- Raw requests/responses logged for audit trail

## Reconciliation

A background job runs every 10 minutes to:
1. Find pending transactions older than 10 minutes
2. Query payment gateway for actual status
3. Update database and publish events

This ensures no payments are lost due to missed webhooks.
