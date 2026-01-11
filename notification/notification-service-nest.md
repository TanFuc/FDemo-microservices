# MISSION: BUILD NOTIFICATION SERVICE (GO + RABBITMQ + NATS BRIDGE)

**Role:** Principal Backend Engineer (Golang & DevOps Expert).
**Goal:** Build a robust `notification-service` that acts as a Dual-Consumer:
1.  **Bridge:** Listens to NATS events (e.g., from Order Service) and forwards them to RabbitMQ.
2.  **Worker:** Consumes RabbitMQ messages to send Email/Push with robust Retry & DLQ mechanisms.

**Tech Stack:**
- **Language:** Go 1.22+.
- **Message Broker 1 (Internal Task Queue):** RabbitMQ (`github.com/rabbitmq/amqp091-go`).
- **Message Broker 2 (External Events):** NATS JetStream (`github.com/nats-io/nats.go`).
- **Database:** MongoDB (`go.mongodb.org/mongo-driver`) for Audit Logs.
- **Templating:** `github.com/aymerick/raymond` (Handlebars for Go).
- **Mailer:** `gopkg.in/gomail.v2`.
- **Architecture:** Clean Architecture with Bridge Pattern.

---

## PHASE 1: INFRASTRUCTURE SETUP

### 1. RabbitMQ Topology (Reliability Layer)
On startup, declare:
- **Exchanges:**
    - `notification.exchange` (Topic).
    - `notification.dlx` (Fanout) - Dead Letter Exchange.
- **Queues:**
    - `queue.email`: Bind to `notification.exchange` (Key: `notification.email.*`).
        - Args: `x-dead-letter-exchange`: `notification.dlx`.
    - `queue.dead_letter`: Bind to `notification.dlx`.

### 2. MongoDB Schema
- **Collection:** `notification_logs`.
- **Fields:** `_id`, `userId`, `type` (EMAIL/PUSH), `status` (PENDING/SENT/FAILED), `payload` (JSON), `error`, `createdAt`.

### 3. NATS Connection
- Connect to NATS JetStream to listen for domain events (`order.created`, etc.).

## PHASE 2: PROVIDER STRATEGY (`internal/provider`)

### 1. Email Provider
- Interface: `Send(ctx, recipient, subject, body)`.
- Implementation: Use `gomail` with SMTP config (Host, Port, User, Pass).
- **Template Engine:**
    - Load `.hbs` files from `templates/`.
    - Function: `RenderTemplate(templateName string, data interface{}) (string, error)` using `raymond`.

## PHASE 3: THE BRIDGE (NATS -> RABBITMQ) (`internal/bridge`)

**Goal:** Decouple domain events from notification logic.
**Component:** `OrderEventListener`

### Logic:
1.  **Subscribe:** Listen to NATS subject `order.created`.
2.  **Transform:**
    - Receive `OrderCreatedEvent` { OrderID, UserID, Amount, ... }.
    - **Map to Notification Job:**
        ```json
        {
          "type": "EMAIL",
          "recipient": "user-email-from-identity-or-payload",
          "template": "order_confirmation",
          "data": { "order_id": "...", "total": "..." }
        }
        ```
3.  **Forward:** Publish this JSON to RabbitMQ Exchange `notification.exchange` with Key `notification.email.order`.

## PHASE 4: THE WORKER (RABBITMQ -> EMAIL) (`internal/worker`)

**Goal:** Reliable delivery with Retry.
**Component:** `EmailConsumer`

### Logic:
1.  **Consume:** Read from `queue.email`.
2.  **Log Start:** Insert MongoDB Log (Status: PENDING).
3.  **Process:**
    - Parse Payload.
    - Render Template (`order_confirmation.hbs`).
    - Call `EmailProvider.Send()`.
4.  **Success Path:**
    - Update Mongo Log (Status: SENT).
    - **Ack** message.
5.  **Failure Path:**
    - Update Mongo Log (Status: FAILED, Error: ...).
    - **Nack** (requeue=false).
    - *Result:* Message moves to DLQ automatically.

## PHASE 5: AUTO-VERIFICATION LOOP

**Instructions for AI:**
1.  **Init:** `go mod init notification-service`. Get all dependencies.
2.  **Generate:**
    - `internal/infrastructure`: RabbitMQ & NATS setup.
    - `internal/bridge`: The NATS listener.
    - `internal/worker`: The RabbitMQ consumer.
    - `templates/order_confirmation.hbs`: A simple HTML template.
3.  **TEST (Integration Simulation):**
    - Create `tests/integration_test.go`.
    - **Step 1:** Simulate Order Service by publishing a message to NATS `order.created`.
    - **Step 2:** Wait 1 second.
    - **Step 3:** Check MongoDB `notification_logs`. Expect a record with `status: "SENT"` (Implies Bridge worked AND Worker worked).
4.  **Build:** Run `go build -o server cmd/main.go`.

---

## IMPORTANT RULES
- **Environment:** Load all secrets (SMTP, NATS_URL, AMQP_URL) from `.env`.
- **Graceful Shutdown:** Close NATS connection and RabbitMQ channel properly on exit.
- **Bridge Reliability:** If RabbitMQ is down when NATS message arrives, the Bridge should return error so NATS redelivers later (Do not Ack NATS msg if RabbitMQ publish fails).