package http

import (
	"github.com/gofiber/fiber/v2"

	"microservices/logistic/internal/core/services"
	"microservices/pkg/response"
)

// HealthHandler handles health check HTTP requests
type HealthHandler struct {
	healthService *services.HealthService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthService *services.HealthService) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

// Health returns basic health status
// @Summary Health check
// @Description Basic health check endpoint
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /health [get]
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status":  "healthy",
		"service": "logistic-service",
	})
}

// HealthDetailed returns detailed health status with dependency checks
// @Summary Detailed health check
// @Description Detailed health check with dependency status
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 503 {object} response.Response
// @Router /health/detailed [get]
func (h *HealthHandler) HealthDetailed(c *fiber.Ctx) error {
	if h.healthService == nil {
		return response.Success(c, fiber.Map{
			"status":  "healthy",
			"service": "logistic-service",
			"message": "Health service not configured",
		})
	}

	status, err := h.healthService.CheckHealth(c.Context())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"status":  "unhealthy",
			"error":   err.Error(),
		})
	}

	return response.Success(c, status)
}

// Ready returns readiness status
// @Summary Readiness check
// @Description Readiness probe endpoint
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 503 {object} response.Response
// @Router /ready [get]
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	if h.healthService == nil {
		return response.Success(c, fiber.Map{
			"ready": true,
		})
	}

	status, err := h.healthService.CheckHealth(c.Context())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"ready":   false,
			"error":   err.Error(),
		})
	}

	return response.Success(c, fiber.Map{
		"ready":  status.Status == "healthy",
		"status": status,
	})
}

// DBStats returns database pool statistics
// @Summary Database stats
// @Description Get database connection pool statistics
// @Tags Metrics
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /metrics/db [get]
func (h *HealthHandler) DBStats(c *fiber.Ctx) error {
	if h.healthService == nil {
		return response.Success(c, fiber.Map{
			"message": "Health service not configured",
		})
	}

	stats := h.healthService.GetDBStats()
	return response.Success(c, stats)
}
