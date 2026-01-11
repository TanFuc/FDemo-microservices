# MISSION: BUILD HIGH-PERFORMANCE API GATEWAY (GOLANG + FIBER + PROXY)

**Role:** Principal Software Architect.
**Goal:** Build a centralized `api-gateway` that acts as the single entry point for the entire E-commerce ecosystem. It handles Routing, Authentication, Rate Limiting, and Response Aggregation.

**Tech Stack:**
- **Language:** Go 1.22+.
- **Framework:** `github.com/gofiber/fiber/v2`.
- **Proxy:** Fiber Proxy Middleware.
- **Rate Limit:** Redis (`github.com/gofiber/storage/redis/v3`).
- **Auth:** JWT Validation.

---

## PHASE 1: ROUTING & REVERSE PROXY (`internal/routes`)

Define the routing map. The Gateway forwards requests to downstream services.

**Configuration (`config.yaml`):**
```yaml
services:
  catalog_url: "http://catalog-service:3000"
  cart_url: "http://cart-service:3000"
  order_url: "http://order-service:3000"
  identity_url: "http://identity-service:3000"