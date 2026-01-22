package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"microservices/payment/internal/domain"
	"microservices/payment/internal/usecase"
	"microservices/pkg/authclient"
)

// PaymentHandler handles payment HTTP requests
type PaymentHandler struct {
	uc             *usecase.PaymentUseCase
	logger         *slog.Logger
	authMiddleware *authclient.ChiMiddleware
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(uc *usecase.PaymentUseCase, logger *slog.Logger, authMiddleware *authclient.ChiMiddleware) *PaymentHandler {
	return &PaymentHandler{
		uc:             uc,
		logger:         logger,
		authMiddleware: authMiddleware,
	}
}

// RegisterRoutes registers the payment routes
func (h *PaymentHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/payments", func(r chi.Router) {
		// Apply auth middleware if available
		if h.authMiddleware != nil {
			r.Use(h.authMiddleware.RequireAuth)
		}

		r.Post("/", h.CreatePayment)
		r.Get("/{id}", h.GetPayment)
		r.Get("/order/{orderId}", h.GetPaymentsByOrder)
	})
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
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Parse and validate order ID
	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid order_id")
		return
	}

	// Parse and validate user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid amount")
		return
	}

	// Parse currency
	currency := domain.Currency(req.Currency)
	if !currency.IsValid() {
		h.respondError(w, http.StatusBadRequest, "invalid currency")
		return
	}

	// Parse provider
	provider := domain.Provider(req.Provider)
	if !provider.IsValid() {
		h.respondError(w, http.StatusBadRequest, "invalid provider")
		return
	}

	// Call use case
	result, err := h.uc.InitiatePayment(r.Context(), &usecase.InitiatePaymentRequest{
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
		h.logger.Error("failed to initiate payment",
			"error", err,
			"order_id", req.OrderID,
		)

		if errors.Is(err, usecase.ErrInvalidProvider) {
			h.respondError(w, http.StatusBadRequest, "invalid provider")
			return
		}
		if errors.Is(err, usecase.ErrInvalidAmount) {
			h.respondError(w, http.StatusBadRequest, "invalid amount")
			return
		}
		if errors.Is(err, usecase.ErrInvalidCurrency) {
			h.respondError(w, http.StatusBadRequest, "invalid currency")
			return
		}

		h.respondError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	h.respondJSON(w, http.StatusCreated, CreatePaymentResponse{
		TransactionID: result.TransactionID.String(),
		PaymentURL:    result.PaymentURL,
		ProviderTxID:  result.ProviderTxID,
	})
}

// GetPayment handles GET /api/v1/payments/{id}
func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	tx, err := h.uc.GetTransaction(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get transaction",
			"error", err,
			"id", idStr,
		)
		h.respondError(w, http.StatusNotFound, "transaction not found")
		return
	}

	h.respondJSON(w, http.StatusOK, tx)
}

// GetPaymentsByOrder handles GET /api/v1/payments/order/{orderId}
func (h *PaymentHandler) GetPaymentsByOrder(w http.ResponseWriter, r *http.Request) {
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	txs, err := h.uc.GetTransactionsByOrderID(r.Context(), orderID)
	if err != nil {
		h.logger.Error("failed to get transactions by order",
			"error", err,
			"order_id", orderIDStr,
		)
		h.respondError(w, http.StatusInternalServerError, "failed to get transactions")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": txs,
		"count":        len(txs),
	})
}

// respondJSON writes a JSON response
func (h *PaymentHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// respondError writes an error JSON response
func (h *PaymentHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{
		"error": message,
	})
}
