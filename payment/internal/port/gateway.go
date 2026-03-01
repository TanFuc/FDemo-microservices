package port

import (
	"context"
	"net/http"

	"github.com/shopspring/decimal"
	"microservices/payment/internal/domain"
)

// PaymentRequest represents a request to create a payment
type PaymentRequest struct {
	OrderID     string          `json:"order_id"`
	Amount      decimal.Decimal `json:"amount"`
	Currency    domain.Currency `json:"currency"`
	Description string          `json:"description"`
	CallbackURL string          `json:"callback_url"`
	ReturnURL   string          `json:"return_url"`
	Metadata    map[string]string
}

// PaymentResponse represents the response from creating a payment
type PaymentResponse struct {
	PaymentURL   string                 `json:"payment_url"`
	ProviderTxID string                 `json:"provider_tx_id"`
	RawData      map[string]interface{} `json:"raw_data,omitempty"`
}

// WebhookData represents parsed webhook data from a provider
type WebhookData struct {
	ProviderTxID string
	Status       domain.Status
	Amount       decimal.Decimal
	Currency     domain.Currency
	RawPayload   []byte
	Metadata     map[string]interface{}
}

// QueryStatusResponse represents the response from querying payment status
type QueryStatusResponse struct {
	ProviderTxID string
	Status       domain.Status
	Amount       decimal.Decimal
	Currency     domain.Currency
	RawData      map[string]interface{}
}

// PaymentGateway defines the interface for payment gateway adapters
type PaymentGateway interface {
	// Provider returns the provider type
	Provider() domain.Provider

	// CreatePayment creates a payment with the gateway
	CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error)

	// VerifyWebhook verifies the webhook signature and parses the payload
	// Returns (isValid, webhookData, error)
	VerifyWebhook(r *http.Request) (bool, *WebhookData, error)

	// QueryStatus queries the payment status from the gateway
	QueryStatus(ctx context.Context, providerTxID string) (*QueryStatusResponse, error)

	// Refund initiates a refund at the provider level.
	// Returns the provider-side refund reference ID.
	// Returns error if provider does not support programmatic refunds.
	Refund(ctx context.Context, providerTxID string, amount decimal.Decimal) (refundID string, err error)
}
