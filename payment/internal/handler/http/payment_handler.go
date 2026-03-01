package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"microservices/payment/internal/domain"
	"microservices/payment/internal/usecase"
	"microservices/pkg/response"
)

// PaymentHandler handles payment HTTP requests
type PaymentHandler struct {
	uc *usecase.PaymentUseCase
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(uc *usecase.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{
		uc: uc,
	}
}

// CreatePaymentRequest represents the request body for creating a payment
type CreatePaymentRequest struct {
	OrderID       string            `json:"order_id"`
	UserID        string            `json:"user_id"`
	Amount        string            `json:"amount"`
	Currency      string            `json:"currency"`
	Provider      string            `json:"provider"`
	Description   string            `json:"description"`
	CallbackURL   string            `json:"callback_url"`
	ReturnURL     string            `json:"return_url"`
	IsWalletTopup bool              `json:"is_wallet_topup"`
	WalletTxID    string            `json:"wallet_tx_id"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// CreatePaymentResponse represents the response for creating a payment
type CreatePaymentResponse struct {
	TransactionID string `json:"transaction_id"`
	PaymentURL    string `json:"payment_url"`
	ProviderTxID  string `json:"provider_tx_id"`
}

// CreatePayment handles POST /api/v1/payments
func (h *PaymentHandler) CreatePayment(c *fiber.Ctx) error {
	var req CreatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Parse and validate order ID
	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		return response.BadRequest(c, "Invalid order_id")
	}

	// Parse and validate user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return response.BadRequest(c, "Invalid user_id")
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return response.BadRequest(c, "Invalid amount")
	}

	// Parse currency
	currency := domain.Currency(req.Currency)
	if !currency.IsValid() {
		return response.BadRequest(c, "Invalid currency")
	}

	// Parse provider
	provider := domain.Provider(req.Provider)
	if !provider.IsValid() {
		return response.BadRequest(c, "Invalid provider")
	}

	// Call use case
	result, err := h.uc.InitiatePayment(c.Context(), &usecase.InitiatePaymentRequest{
		OrderID:       orderID,
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		Provider:      provider,
		Description:   req.Description,
		CallbackURL:   req.CallbackURL,
		ReturnURL:     req.ReturnURL,
		IsWalletTopup: req.IsWalletTopup,
		WalletTxID:    req.WalletTxID,
		Metadata:      req.Metadata,
	})

	if err != nil {
		if errors.Is(err, usecase.ErrInvalidProvider) {
			return response.BadRequest(c, "Invalid provider")
		}
		if errors.Is(err, usecase.ErrInvalidAmount) {
			return response.BadRequest(c, "Invalid amount")
		}
		if errors.Is(err, usecase.ErrInvalidCurrency) {
			return response.BadRequest(c, "Invalid currency")
		}
		return response.InternalServerError(c, "Failed to create payment")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": CreatePaymentResponse{
			TransactionID: result.TransactionID.String(),
			PaymentURL:    result.PaymentURL,
			ProviderTxID:  result.ProviderTxID,
		},
	})
}

// GetPayment handles GET /api/v1/payments/:id
func (h *PaymentHandler) GetPayment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "Invalid transaction ID")
	}

	tx, err := h.uc.GetTransaction(c.Context(), id)
	if err != nil {
		return response.NotFound(c, "Transaction not found")
	}

	return response.Success(c, tx)
}

// GetPaymentsByOrder handles GET /api/v1/payments/order/:orderId
func (h *PaymentHandler) GetPaymentsByOrder(c *fiber.Ctx) error {
	orderIDStr := c.Params("orderId")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return response.BadRequest(c, "Invalid order ID")
	}

	txs, err := h.uc.GetTransactionsByOrderID(c.Context(), orderID)
	if err != nil {
		return response.InternalServerError(c, "Failed to get transactions")
	}

	return response.Success(c, fiber.Map{
		"transactions": txs,
		"count":        len(txs),
	})
}

// RefundPaymentRequest represents the request body for refunding a payment
type RefundPaymentRequest struct {
	Amount string `json:"amount"`
	Reason string `json:"reason"`
}

// RefundPayment handles POST /api/v1/payments/:id/refund
func (h *PaymentHandler) RefundPayment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "Invalid transaction ID")
	}

	var req RefundPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return response.BadRequest(c, "Invalid amount")
	}

	// Call use case
	result, err := h.uc.RefundPayment(c.Context(), &usecase.RefundRequest{
		TransactionID: id,
		Amount:        amount,
		Reason:        req.Reason,
	})

	if err != nil {
		if errors.Is(err, usecase.ErrTransactionNotFound) {
			return response.NotFound(c, "Transaction not found")
		}
		// Check if it's a validation error (not SUCCESS state)
		if err.Error() != "" && (errors.Is(err, usecase.ErrInvalidProvider) ||
			err.Error() == "transaction is not in SUCCESS state" ||
			err.Error()[:10] == "transaction") {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalServerError(c, "Failed to process refund")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"transaction_id": result.TransactionID.String(),
			"status":         result.Status.String(),
			"amount":         result.Amount.String(),
			"currency":       result.Currency.String(),
			"provider":       result.Provider.String(),
			"refund_id":      result.RefundID,
		},
	})
}

// VNPayReturn handles GET /api/v1/payments/vnpay-return (browser redirect)
// This is a read-only endpoint that does NOT update the database
func (h *PaymentHandler) VNPayReturn(c *fiber.Ctx) error {
	// Read vnp_ResponseCode from query params
	responseCode := c.Query("vnp_ResponseCode")

	// Return status based on response code (00 = success)
	if responseCode == "00" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"status":  "success",
			"message": "Payment completed successfully",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"status":  "failed",
		"message": "Payment was not successful",
		"code":    responseCode,
	})
}
