package handler

import (
	"github.com/gofiber/fiber/v2"

	"tafu-profile/internal/config"
	"tafu-profile/internal/domain/service"
	"tafu-profile/pkg/errors"
	"tafu-profile/pkg/response"
)

type InternalHandler struct {
	cfg            *config.Config
	profileService *service.ProfileService
}

func NewInternalHandler(cfg *config.Config, profileService *service.ProfileService) *InternalHandler {
	return &InternalHandler{
		cfg:            cfg,
		profileService: profileService,
	}
}

// GetUserInfo godoc
// @Summary Get user info (internal API)
// @Tags internal
// @Produce json
// @Param X-Internal-Service-Key header string true "Internal service key"
// @Param userId path string true "User ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /internal/users/{userId} [get]
func (h *InternalHandler) GetUserInfo(c *fiber.Ctx) error {
	// Verify internal service key
	serviceKey := c.Get("X-Internal-Service-Key")
	if serviceKey == "" || serviceKey != h.cfg.App.InternalServiceKey {
		return response.Error(c, errors.ErrUnauthorized)
	}

	userID := c.Params("userId")
	if userID == "" {
		return response.BadRequest(c, "User ID is required")
	}

	userInfo, err := h.profileService.GetUserInfo(c.Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, userInfo)
}

// Health godoc
// @Summary Internal health check
// @Tags internal
// @Produce json
// @Success 200 {object} map[string]string
// @Router /internal/health [get]
func (h *InternalHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}
