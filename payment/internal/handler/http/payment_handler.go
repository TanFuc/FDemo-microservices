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
	OrderID     string            `json:"order_id"`
	UserID      string            `json:"user_id"`
	Amount      string            `json:"amount"`
	Currency    string            `json:"currency"`
	Provider    string            `json:"provider"`
	Description string            `json:"description"`
	CallbackURL string            `json:"callback_url"`
	ReturnURL   string            `json:"return_url"`
	Metadata    map[string]string `json:"metadata,omitempty"`
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
		OrderID:     orderID,
		UserID:      userID,
		Amount:      amount,
		Currency:    currency,
		Provider:    provider,
		Description: req.Description,
		CallbackURL: req.CallbackURL,
		ReturnURL:   req.ReturnURL,
		Metadata:    req.Metadata,
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
