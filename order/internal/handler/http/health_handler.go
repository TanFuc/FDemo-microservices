package http

import (
	"github.com/gofiber/fiber/v2"
	"microservices/order/internal/infrastructure/database"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db *database.Database
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *database.Database) *HealthHandler {
	return &HealthHandler{db: db}
}


// Health handles GET /health - liveness probe
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

// Ready handles GET /ready - readiness probe
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	if err := h.db.Health(c.Context()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "error",
			"message": "Database connection failed",
		})
	}

	return c.JSON(fiber.Map{
		"status": "ready",
	})
}
