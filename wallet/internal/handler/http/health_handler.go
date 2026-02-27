package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	pool *pgxpool.Pool
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

// Health handles GET /health
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "wallet",
	})
}

// Ready handles GET /ready
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	// Check database connection
	if h.pool != nil {
		if err := h.pool.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":  "unhealthy",
				"service": "wallet",
				"error":   "database connection failed",
			})
		}
	}

	return c.JSON(fiber.Map{
		"status":  "ready",
		"service": "wallet",
	})
}
