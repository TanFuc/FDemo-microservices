package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"microservices/logistic/internal/api/dto"
	"microservices/logistic/internal/core/domain"
	"microservices/logistic/internal/core/services"
	"microservices/pkg/response"
)

// ShippingHandler handles shipping-related HTTP requests
type ShippingHandler struct {
	shippingService *services.ShippingService
}

// NewShippingHandler creates a new shipping handler
func NewShippingHandler(shippingService *services.ShippingService) *ShippingHandler {
	return &ShippingHandler{
		shippingService: shippingService,
	}
}

// CalculateFee handles fee calculation requests
// @Summary Calculate shipping fee
// @Description Calculate shipping fee for a provider
// @Tags Shipping
// @Accept json
// @Produce json
// @Param request body dto.CalculateFeeRequest true "Calculate fee request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /shipping/calculate-fee [post]
func (h *ShippingHandler) CalculateFee(c *fiber.Ctx) error {
	var req dto.CalculateFeeRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate provider
	providerName := domain.ProviderName(req.Provider)
	if !providerName.IsValid() {
		return response.BadRequest(c, "Invalid provider name")
	}

	// Calculate fee
	result, err := h.shippingService.CalculateFee(c.Context(), &services.CalculateFeeRequest{
		Provider:       providerName,
		FromDistrictID: req.FromDistrictID,
		ToDistrictID:   req.ToDistrictID,
		WeightGram:     req.WeightGram,
		InsuranceValue: req.InsuranceValue,
	})
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, &dto.CalculateFeeResponse{
		Provider:  string(result.Provider),
		Fee:       result.Fee,
		FromCache: result.FromCache,
	})
}

// CreateShipment handles shipment creation requests
// @Summary Create shipment
// @Description Create a new shipment with a shipping provider
// @Tags Shipping
// @Accept json
// @Produce json
// @Param request body dto.CreateShipmentRequest true "Create shipment request"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /shipping/create [post]
func (h *ShippingHandler) CreateShipment(c *fiber.Ctx) error {
	var req dto.CreateShipmentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Parse internal order ID
	internalOrderID, err := req.ParseInternalOrderID()
	if err != nil {
		return response.BadRequest(c, "Invalid internal order ID")
	}

	// Validate provider
	providerName := domain.ProviderName(req.Provider)
	if !providerName.IsValid() {
		return response.BadRequest(c, "Invalid provider name")
	}

	// Create shipment
	result, err := h.shippingService.CreateShipment(c.Context(), &services.CreateShipmentRequest{
		InternalOrderID: internalOrderID,
		Provider:        providerName,
		Sender:          req.Sender.ToDomain(),
		Receiver:        req.Receiver.ToDomain(),
		Parcels:         dto.ToDomainParcels(req.Parcels),
		IsCOD:           req.IsCOD,
		CODAmount:       req.CODAmount,
		Note:            req.Note,
	})
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Created(c, &dto.CreateShipmentResponse{
		ID:           result.ID,
		TrackingCode: result.TrackingCode,
		LabelURL:     result.LabelURL,
		ShippingFee:  result.ShippingFee,
		Provider:     string(result.Provider),
	})
}

// GetShipment handles getting a shipment by ID
// @Summary Get shipment
// @Description Get a shipment by ID
// @Tags Shipping
// @Accept json
// @Produce json
// @Param id path string true "Shipment ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /shipping/{id} [get]
func (h *ShippingHandler) GetShipment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "Invalid shipment ID")
	}

	order, err := h.shippingService.GetShipment(c.Context(), id)
	if err != nil {
		return response.NotFound(c, "Shipment not found")
	}

	return response.Success(c, dto.FromShippingOrder(order))
}

// GetShipmentByTracking handles getting a shipment by tracking code
// @Summary Get shipment by tracking code
// @Description Get a shipment by its tracking code
// @Tags Shipping
// @Accept json
// @Produce json
// @Param tracking_code path string true "Tracking code"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /shipping/track/{tracking_code} [get]
func (h *ShippingHandler) GetShipmentByTracking(c *fiber.Ctx) error {
	trackingCode := c.Params("tracking_code")
	if trackingCode == "" {
		return response.BadRequest(c, "Tracking code is required")
	}

	order, err := h.shippingService.GetShipmentByTrackingCode(c.Context(), trackingCode)
	if err != nil {
		return response.NotFound(c, "Shipment not found")
	}

	return response.Success(c, dto.FromShippingOrder(order))
}

// ListProviders handles listing available providers
// @Summary List providers
// @Description List all available shipping providers
// @Tags Shipping
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /shipping/providers [get]
func (h *ShippingHandler) ListProviders(c *fiber.Ctx) error {
	providers := h.shippingService.ListProviders()

	providerNames := make([]string, len(providers))
	for i, p := range providers {
		providerNames[i] = string(p)
	}

	return response.Success(c, &dto.ProvidersResponse{
		Providers: providerNames,
	})
}
