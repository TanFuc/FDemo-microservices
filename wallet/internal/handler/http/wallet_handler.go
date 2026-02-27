package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"microservices/pkg/response"
	"microservices/wallet/internal/domain"
	"microservices/wallet/internal/usecase"
)

// WalletHandler handles wallet HTTP requests
type WalletHandler struct {
	getUC    *usecase.GetWalletUseCase
	topupUC  *usecase.TopUpUseCase
	payUC    *usecase.PayUseCase
	refundUC *usecase.RefundUseCase
}

// NewWalletHandler creates a new wallet handler
func NewWalletHandler(
	getUC *usecase.GetWalletUseCase,
	topupUC *usecase.TopUpUseCase,
	payUC *usecase.PayUseCase,
	refundUC *usecase.RefundUseCase,
) *WalletHandler {
	return &WalletHandler{
		getUC:    getUC,
		topupUC:  topupUC,
		payUC:    payUC,
		refundUC: refundUC,
	}
}

// GetWallet handles GET /api/v1/wallet
func (h *WalletHandler) GetWallet(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Authentication required")
	}

	wallet, err := h.getUC.GetByUserID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			return response.NotFound(c, "Wallet not found")
		}
		return response.InternalServerError(c, "Failed to get wallet")
	}

	return response.Success(c, wallet)
}

// GetHistory handles GET /api/v1/wallet/history
func (h *WalletHandler) GetHistory(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Authentication required")
	}

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	history, err := h.getUC.GetHistory(c.Context(), userID, limit, offset)
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			return response.NotFound(c, "Wallet not found")
		}
		return response.InternalServerError(c, "Failed to get transaction history")
	}

	return response.Success(c, history)
}

// InitiateTopUpRequest represents the request body for initiating a top-up
type InitiateTopUpRequest struct {
	Amount      string `json:"amount"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
	ReturnURL   string `json:"return_url"`
}

// InitiateTopUp handles POST /api/v1/wallet/topup
func (h *WalletHandler) InitiateTopUp(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Authentication required")
	}

	var req InitiateTopUpRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return response.BadRequest(c, "Invalid amount")
	}

	if req.Provider == "" {
		return response.BadRequest(c, "Provider is required")
	}

	idempotencyKey := c.Get("X-Idempotency-Key")

	result, err := h.topupUC.InitiateTopUp(c.Context(), &usecase.InitiateTopUpRequest{
		UserID:         userID,
		Amount:         amount,
		Provider:       req.Provider,
		Description:    req.Description,
		IdempotencyKey: idempotencyKey,
		ReturnURL:      req.ReturnURL,
	})
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			return response.NotFound(c, "Wallet not found")
		}
		if errors.Is(err, domain.ErrAmountTooSmall) {
			return response.BadRequest(c, "Amount is below minimum limit")
		}
		if errors.Is(err, domain.ErrAmountTooLarge) {
			return response.BadRequest(c, "Amount exceeds maximum limit")
		}
		if errors.Is(err, domain.ErrBalanceExceedsLimit) {
			return response.BadRequest(c, "Balance would exceed maximum limit")
		}
		return response.InternalServerError(c, "Failed to initiate top-up")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// PayWithWalletRequest represents the request body for paying with wallet
type PayWithWalletRequest struct {
	OrderID     string `json:"order_id"`
	Amount      string `json:"amount"`
	Description string `json:"description"`
}

// PayWithWallet handles POST /api/v1/wallet/pay
func (h *WalletHandler) PayWithWallet(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Authentication required")
	}

	var req PayWithWalletRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		return response.BadRequest(c, "Invalid order_id")
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return response.BadRequest(c, "Invalid amount")
	}

	idempotencyKey := c.Get("X-Idempotency-Key")

	result, err := h.payUC.Execute(c.Context(), &usecase.WalletPayRequest{
		UserID:         userID,
		OrderID:        orderID,
		Amount:         amount,
		Description:    req.Description,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			return response.NotFound(c, "Wallet not found")
		}
		if errors.Is(err, domain.ErrInsufficientBalance) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success": false,
				"code":    "INSUFFICIENT_BALANCE",
				"message": "Insufficient wallet balance",
			})
		}
		if errors.Is(err, domain.ErrInvalidAmount) {
			return response.BadRequest(c, "Invalid amount")
		}
		if errors.Is(err, domain.ErrWalletSuspended) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "WALLET_SUSPENDED",
				"message": "Wallet is suspended",
			})
		}
		if errors.Is(err, domain.ErrWalletFrozen) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "WALLET_FROZEN",
				"message": "Wallet is frozen",
			})
		}
		return response.InternalServerError(c, "Failed to process payment")
	}

	return response.Success(c, fiber.Map{
		"wallet_tx_id":   result.WalletTxID.String(),
		"balance_before": result.BalanceBefore.String(),
		"balance_after":  result.BalanceAfter.String(),
	})
}

// RefundToWalletRequest represents the request body for refunding to wallet
type RefundToWalletRequest struct {
	UserID  string `json:"user_id"`
	OrderID string `json:"order_id"`
	Amount  string `json:"amount"`
	Reason  string `json:"reason"`
}

// RefundToWallet handles POST /api/v1/wallet/refund
func (h *WalletHandler) RefundToWallet(c *fiber.Ctx) error {
	var req RefundToWalletRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return response.BadRequest(c, "Invalid user_id")
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		return response.BadRequest(c, "Invalid order_id")
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return response.BadRequest(c, "Invalid amount")
	}

	idempotencyKey := c.Get("X-Idempotency-Key")

	result, err := h.refundUC.Execute(c.Context(), &usecase.WalletRefundRequest{
		UserID:         userID,
		OrderID:        orderID,
		Amount:         amount,
		Reason:         req.Reason,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			return response.NotFound(c, "Wallet not found")
		}
		if errors.Is(err, domain.ErrInvalidAmount) {
			return response.BadRequest(c, "Invalid amount")
		}
		return response.InternalServerError(c, "Failed to process refund")
	}

	return response.Success(c, fiber.Map{
		"wallet_tx_id":   result.WalletTxID.String(),
		"balance_before": result.BalanceBefore.String(),
		"balance_after":  result.BalanceAfter.String(),
	})
}

// Helper functions

func getUserID(c *fiber.Ctx) (uuid.UUID, error) {
	// Try both formats for compatibility
	userIDStr, ok := c.Locals("userId").(string)
	if !ok || userIDStr == "" {
		userIDStr, ok = c.Locals("user_id").(string)
		if !ok || userIDStr == "" {
			return uuid.Nil, errors.New("user_id not found in context")
		}
	}
	return uuid.Parse(userIDStr)
}
