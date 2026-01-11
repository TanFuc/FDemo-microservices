package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"microservices/payment/internal/domain"
	"microservices/payment/internal/port"
)

// CODAdapter implements port.PaymentGateway for Cash on Delivery
type CODAdapter struct{}

// NewCODAdapter creates a new COD adapter
func NewCODAdapter() *CODAdapter {
	return &CODAdapter{}
}

// Ensure CODAdapter implements port.PaymentGateway
var _ port.PaymentGateway = (*CODAdapter)(nil)

// Provider returns the provider type
func (a *CODAdapter) Provider() domain.Provider {
	return domain.ProviderCOD
}

// CreatePayment creates a COD payment (immediately returns PENDING)
func (a *CODAdapter) CreatePayment(ctx context.Context, req *port.PaymentRequest) (*port.PaymentResponse, error) {
	// COD doesn't need an external payment URL
	// Generate a unique transaction ID
	txID := fmt.Sprintf("COD-%s", uuid.New().String())

	return &port.PaymentResponse{
		PaymentURL:   "", // No payment URL for COD
		ProviderTxID: txID,
		RawData: map[string]interface{}{
			"order_id": req.OrderID,
			"amount":   req.Amount.String(),
			"currency": req.Currency.String(),
			"status":   "PENDING_DELIVERY",
			"message":  "Payment will be collected on delivery",
		},
	}, nil
}

// VerifyWebhook verifies the COD webhook (manual confirmation)
// In practice, this would be called by an internal system when delivery is confirmed
func (a *CODAdapter) VerifyWebhook(r *http.Request) (bool, *port.WebhookData, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read body: %w", err)
	}

	var payload struct {
		ProviderTxID string          `json:"provider_tx_id"`
		Status       string          `json:"status"` // "SUCCESS" or "FAILED"
		Amount       decimal.Decimal `json:"amount"`
		Currency     string          `json:"currency"`
		Secret       string          `json:"secret"` // Internal verification secret
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return false, nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// In a real implementation, verify the internal secret
	// For now, we accept any valid JSON payload

	status := domain.StatusFailed
	if payload.Status == "SUCCESS" {
		status = domain.StatusSuccess
	}

	currency := domain.CurrencyVND
	if payload.Currency == "USD" {
		currency = domain.CurrencyUSD
	}

	return true, &port.WebhookData{
		ProviderTxID: payload.ProviderTxID,
		Status:       status,
		Amount:       payload.Amount,
		Currency:     currency,
		RawPayload:   body,
		Metadata:     map[string]interface{}{},
	}, nil
}

// QueryStatus queries the COD payment status
// In practice, this would query an internal delivery system
func (a *CODAdapter) QueryStatus(ctx context.Context, providerTxID string) (*port.QueryStatusResponse, error) {
	// COD status is typically managed internally
	// Return PENDING unless manually confirmed
	return &port.QueryStatusResponse{
		ProviderTxID: providerTxID,
		Status:       domain.StatusPending,
		Amount:       decimal.Zero,
		Currency:     domain.CurrencyVND,
		RawData: map[string]interface{}{
			"message": "COD payment status managed by delivery system",
		},
	}, nil
}
