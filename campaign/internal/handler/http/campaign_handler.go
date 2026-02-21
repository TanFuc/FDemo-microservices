package http

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"microservices/campaign/internal/model"
	"microservices/campaign/internal/service"
	"microservices/campaign/internal/service/impl"
	"microservices/campaign/pkg/response"
)

// CampaignHandler handles HTTP requests for campaign operations.
type CampaignHandler struct {
	campaignService service.CampaignService
	validate        *validator.Validate
}

// NewCampaignHandler creates a new campaign handler.
func NewCampaignHandler(campaignService service.CampaignService) *CampaignHandler {
	return &CampaignHandler{
		campaignService: campaignService,
		validate:        validator.New(),
	}
}

// CreateCampaign creates a new campaign.
// POST /api/v1/campaigns
func (h *CampaignHandler) CreateCampaign(c *fiber.Ctx) error {
	var req model.CreateCampaignRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	campaign, err := h.campaignService.CreateCampaign(c.Context(), &req)
	if err != nil {
		return response.InternalError(c, "Failed to create campaign")
	}

	return response.Created(c, campaign)
}

// GetCampaign retrieves a campaign by ID.
// GET /api/v1/campaigns/:id
func (h *CampaignHandler) GetCampaign(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "Campaign ID is required")
	}

	campaign, err := h.campaignService.GetCampaign(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrCampaignNotFound) {
			return response.NotFound(c, "Campaign not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid campaign ID")
		}
		return response.InternalError(c, "Failed to get campaign")
	}

	return response.Success(c, campaign)
}

// ListCampaigns lists campaigns with pagination.
// GET /api/v1/campaigns
func (h *CampaignHandler) ListCampaigns(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	campaigns, total, err := h.campaignService.ListCampaigns(c.Context(), page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to list campaigns")
	}

	return response.Success(c, fiber.Map{
		"campaigns": campaigns,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// ListActiveCampaigns lists all active campaigns.
// GET /api/v1/campaigns/active
func (h *CampaignHandler) ListActiveCampaigns(c *fiber.Ctx) error {
	campaigns, err := h.campaignService.ListActiveCampaigns(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to list active campaigns")
	}

	return response.Success(c, campaigns)
}

// HealthCheck returns the health status of the service.
// GET /health
func (h *CampaignHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "campaign-service",
	})
}

func formatValidationErrors(err error) []string {
	var errors []string
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, formatFieldError(e))
		}
	}
	return errors
}

func formatFieldError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "gt":
		return e.Field() + " must be greater than " + e.Param()
	case "min":
		return e.Field() + " must be at least " + e.Param()
	case "max":
		return e.Field() + " must be at most " + e.Param()
	case "uuid":
		return e.Field() + " must be a valid UUID"
	case "oneof":
		return e.Field() + " must be one of: " + e.Param()
	case "gtfield":
		return e.Field() + " must be greater than " + e.Param()
	default:
		return e.Field() + " is invalid"
	}
}
