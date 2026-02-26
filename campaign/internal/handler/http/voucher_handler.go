package http

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"microservices/campaign/internal/middleware"
	"microservices/campaign/internal/model"
	"microservices/campaign/internal/service"
	"microservices/campaign/internal/service/impl"
	"microservices/campaign/pkg/response"
)

// VoucherHandler handles HTTP requests for voucher operations.
type VoucherHandler struct {
	voucherService service.VoucherService
	validate       *validator.Validate
}

// NewVoucherHandler creates a new voucher handler.
func NewVoucherHandler(voucherService service.VoucherService) *VoucherHandler {
	return &VoucherHandler{
		voucherService: voucherService,
		validate:       validator.New(),
	}
}

// CreateVoucher creates a new voucher.
// POST /api/v1/vouchers
func (h *VoucherHandler) CreateVoucher(c *fiber.Ctx) error {
	var req model.CreateVoucherRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	voucher, err := h.voucherService.CreateVoucher(c.Context(), &req)
	if err != nil {
		return response.InternalError(c, "Failed to create voucher")
	}

	return response.Created(c, voucher)
}

// GetVoucher retrieves a voucher by ID.
// GET /api/v1/vouchers/:id
func (h *VoucherHandler) GetVoucher(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "Voucher ID is required")
	}

	voucher, err := h.voucherService.GetVoucher(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrVoucherNotFound) {
			return response.NotFound(c, "Voucher not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid voucher ID")
		}
		return response.InternalError(c, "Failed to get voucher")
	}

	return response.Success(c, voucher)
}

// GetVoucherByCode retrieves a voucher by code.
// GET /api/v1/vouchers/code/:code
func (h *VoucherHandler) GetVoucherByCode(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return response.BadRequest(c, "Voucher code is required")
	}

	voucher, err := h.voucherService.GetVoucherByCode(c.Context(), code)
	if err != nil {
		if errors.Is(err, impl.ErrVoucherNotFound) {
			return response.NotFound(c, "Voucher not found")
		}
		return response.InternalError(c, "Failed to get voucher")
	}

	return response.Success(c, voucher)
}

// ClaimVoucher claims a voucher for the authenticated user.
// POST /api/v1/vouchers/claim
func (h *VoucherHandler) ClaimVoucher(c *fiber.Ctx) error {
	// Get user ID from auth context
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return response.Unauthorized(c, "User not authenticated")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	var req struct {
		Code string `json:"code" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Code == "" {
		return response.BadRequest(c, "Voucher code is required")
	}

	if err := h.voucherService.ClaimVoucher(c.Context(), userID, req.Code); err != nil {
		if errors.Is(err, model.ErrVoucherNotFound) {
			return response.NotFound(c, "Voucher not found")
		}
		if errors.Is(err, model.ErrVoucherInactive) {
			return response.BadRequest(c, "Voucher is inactive")
		}
		if errors.Is(err, model.ErrVoucherAlreadyClaimed) {
			return response.BadRequest(c, "Voucher already claimed")
		}
		if errors.Is(err, model.ErrVoucherOutOfStock) {
			return response.BadRequest(c, "Voucher out of stock")
		}
		return response.InternalError(c, "Failed to claim voucher")
	}

	return response.SuccessWithMessage(c, nil, "Voucher claimed successfully")
}

// CalculateCart calculates cart discount with voucher.
// POST /api/v1/vouchers/calculate
func (h *VoucherHandler) CalculateCart(c *fiber.Ctx) error {
	var req model.CalculateCartRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	// Convert request items to model
	items := make([]model.CartItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = item.ToModel()
	}

	result, err := h.voucherService.CalculateCart(c.Context(), items, req.VoucherCode, req.UserID)
	if err != nil {
		// Return the result with error message for validation errors
		if result != nil && result.ErrorMessage != "" {
			return response.Success(c, result)
		}
		return response.InternalError(c, "Failed to calculate cart")
	}

	return response.Success(c, result)
}

// ValidateVoucher validates a voucher for Cart Service (internal endpoint).
// POST /api/v1/vouchers/validate
func (h *VoucherHandler) ValidateVoucher(c *fiber.Ctx) error {
	var req model.ValidateVoucherRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	result, err := h.voucherService.ValidateVoucherForCart(c.Context(), &req)
	if err != nil {
		// For validation results, return the result even on error
		if result != nil {
			return response.Success(c, result)
		}
		return response.InternalError(c, "Failed to validate voucher")
	}

	return response.Success(c, result)
}

// GetUserVouchers retrieves all vouchers for the authenticated user.
// GET /api/v1/vouchers/user
func (h *VoucherHandler) GetUserVouchers(c *fiber.Ctx) error {
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		return response.Unauthorized(c, "User not authenticated")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	vouchers, err := h.voucherService.GetUserVouchers(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to get user vouchers")
	}

	return response.Success(c, vouchers)
}

// InitializeVoucherStock initializes voucher stock in Redis.
// POST /api/v1/vouchers/:code/initialize
func (h *VoucherHandler) InitializeVoucherStock(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return response.BadRequest(c, "Voucher code is required")
	}

	if err := h.voucherService.InitializeVoucherStock(c.Context(), code); err != nil {
		return response.InternalError(c, "Failed to initialize voucher stock")
	}

	return response.SuccessWithMessage(c, nil, "Voucher stock initialized")
}
