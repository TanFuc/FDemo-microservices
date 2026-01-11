package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"tafu-logistic/logistics-service/internal/api/dto"
	"tafu-logistic/logistics-service/internal/core/domain"
	"tafu-logistic/logistics-service/internal/core/services"
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
// POST /api/v1/shipping/calculate-fee
func (h *ShippingHandler) CalculateFee(c *gin.Context) {
	var req dto.CalculateFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	// Validate provider
	providerName := domain.ProviderName(req.Provider)
	if !providerName.IsValid() {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_PROVIDER", "Invalid provider name"))
		return
	}

	// Calculate fee
	result, err := h.shippingService.CalculateFee(c.Request.Context(), &services.CalculateFeeRequest{
		Provider:       providerName,
		FromDistrictID: req.FromDistrictID,
		ToDistrictID:   req.ToDistrictID,
		WeightGram:     req.WeightGram,
		InsuranceValue: req.InsuranceValue,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("CALCULATION_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.CalculateFeeResponse{
		Provider:  string(result.Provider),
		Fee:       result.Fee,
		FromCache: result.FromCache,
	}))
}

// CreateShipment handles shipment creation requests
// POST /api/v1/shipping/create
func (h *ShippingHandler) CreateShipment(c *gin.Context) {
	var req dto.CreateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	// Parse internal order ID
	internalOrderID, err := req.ParseInternalOrderID()
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_ORDER_ID", "Invalid internal order ID"))
		return
	}

	// Validate provider
	providerName := domain.ProviderName(req.Provider)
	if !providerName.IsValid() {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_PROVIDER", "Invalid provider name"))
		return
	}

	// Create shipment
	result, err := h.shippingService.CreateShipment(c.Request.Context(), &services.CreateShipmentRequest{
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
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("CREATION_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, dto.NewSuccessResponse(&dto.CreateShipmentResponse{
		ID:           result.ID,
		TrackingCode: result.TrackingCode,
		LabelURL:     result.LabelURL,
		ShippingFee:  result.ShippingFee,
		Provider:     string(result.Provider),
	}))
}

// GetShipment handles getting a shipment by ID
// GET /api/v1/shipping/:id
func (h *ShippingHandler) GetShipment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_ID", "Invalid shipment ID"))
		return
	}

	order, err := h.shippingService.GetShipment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.NewErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(dto.FromShippingOrder(order)))
}

// GetShipmentByTracking handles getting a shipment by tracking code
// GET /api/v1/shipping/track/:tracking_code
func (h *ShippingHandler) GetShipmentByTracking(c *gin.Context) {
	trackingCode := c.Param("tracking_code")
	if trackingCode == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_TRACKING", "Tracking code is required"))
		return
	}

	order, err := h.shippingService.GetShipmentByTrackingCode(c.Request.Context(), trackingCode)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.NewErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(dto.FromShippingOrder(order)))
}

// ListProviders handles listing available providers
// GET /api/v1/shipping/providers
func (h *ShippingHandler) ListProviders(c *gin.Context) {
	providers := h.shippingService.ListProviders()

	providerNames := make([]string, len(providers))
	for i, p := range providers {
		providerNames[i] = string(p)
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.ProvidersResponse{
		Providers: providerNames,
	}))
}
