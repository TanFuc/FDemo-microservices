package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"microservices/profile/internal/repository/mongodb"
	"microservices/profile/pkg/response"
)

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

type HealthHandler struct {
	mongodb *mongodb.MongoDB
}

func NewHealthHandler(mongodb *mongodb.MongoDB) *HealthHandler {
	return &HealthHandler{
		mongodb: mongodb,
	}
}

// Health godoc
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} response.Response
// @Failure 503 {object} response.ErrorResponse
// @Router /health [get]
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	services := make(map[string]string)
	overallStatus := "healthy"

	// Check MongoDB
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	if err := h.mongodb.HealthCheck(ctx); err != nil {
		services["mongodb"] = "unhealthy: " + err.Error()
		overallStatus = "unhealthy"
	} else {
		services["mongodb"] = "healthy"
	}

	status := &HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
	}

	if overallStatus == "unhealthy" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"code":    "SERVICE_UNAVAILABLE",
			"message": "Service is unhealthy",
			"data":    status,
		})
	}

	return response.Success(c, status)
}

// Liveness godoc
// @Summary Liveness probe
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health/live [get]
func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

// Readiness godoc
// @Summary Readiness probe
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /health/ready [get]
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	if err := h.mongodb.HealthCheck(ctx); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "not ready",
			"reason": "database connection failed",
		})
	}

	return c.JSON(fiber.Map{
		"status": "ready",
	})
}
