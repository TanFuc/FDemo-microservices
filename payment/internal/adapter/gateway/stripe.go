package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
	"microservices/payment/internal/domain"
	"microservices/payment/internal/port"
)

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
}

// StripeAdapter implements port.PaymentGateway for Stripe
type StripeAdapter struct {
	config StripeConfig
}

// NewStripeAdapter creates a new Stripe adapter
func NewStripeAdapter(config StripeConfig) *StripeAdapter {
	stripe.Key = config.SecretKey
	return &StripeAdapter{config: config}
}

// Ensure StripeAdapter implements port.PaymentGateway
var _ port.PaymentGateway = (*StripeAdapter)(nil)

// Provider returns the provider type
func (a *StripeAdapter) Provider() domain.Provider {
	return domain.ProviderStripe
}

// CreatePayment creates a Stripe Checkout Session
func (a *StripeAdapter) CreatePayment(ctx context.Context, req *port.PaymentRequest) (*port.PaymentResponse, error) {
	// Convert amount to cents (Stripe uses smallest currency unit)
	amountCents := req.Amount.Mul(decimal.NewFromInt(100)).IntPart()

	// Map currency
	currency := string(stripe.CurrencyUSD)
	if req.Currency == domain.CurrencyVND {
		currency = string(stripe.CurrencyVND)
		// VND doesn't have cents, use whole amount
		amountCents = req.Amount.IntPart()
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(req.Description),
						Description: stripe.String(fmt.Sprintf("Order: %s", req.OrderID)),
					},
					UnitAmount: stripe.Int64(amountCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(req.ReturnURL + "?status=success&session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(req.ReturnURL + "?status=cancelled"),
		Metadata: map[string]string{
			"order_id": req.OrderID,
		},
	}

	// Add custom metadata if provided
	for k, v := range req.Metadata {
		params.Metadata[k] = v
	}

	s, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe session: %w", err)
	}

	return &port.PaymentResponse{
		PaymentURL:   s.URL,
		ProviderTxID: s.ID,
		RawData: map[string]interface{}{
			"session_id":     s.ID,
			"payment_status": string(s.PaymentStatus),
		},
	}, nil
}

// VerifyWebhook verifies the Stripe webhook signature and parses the payload
func (a *StripeAdapter) VerifyWebhook(r *http.Request) (bool, *port.WebhookData, error) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(nil, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read request body: %w", err)
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sigHeader, a.config.WebhookSecret)
	if err != nil {
		return false, nil, nil // Signature verification failed
	}

	// Parse the event based on type
	var webhookData *port.WebhookData

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return true, nil, fmt.Errorf("failed to parse checkout session: %w", err)
		}

		status := domain.StatusSuccess
		if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
			status = domain.StatusPending
		}

		webhookData = &port.WebhookData{
			ProviderTxID: session.ID,
			Status:       status,
			Amount:       decimal.NewFromInt(session.AmountTotal).Div(decimal.NewFromInt(100)),
			Currency:     mapStripeCurrency(session.Currency),
			RawPayload:   payload,
			Metadata:     make(map[string]interface{}),
		}
		for k, v := range session.Metadata {
			webhookData.Metadata[k] = v
		}

	case "checkout.session.expired":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return true, nil, fmt.Errorf("failed to parse checkout session: %w", err)
		}

		webhookData = &port.WebhookData{
			ProviderTxID: session.ID,
			Status:       domain.StatusFailed,
			RawPayload:   payload,
		}

	case "charge.refunded":
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
			return true, nil, fmt.Errorf("failed to parse charge: %w", err)
		}

		webhookData = &port.WebhookData{
			ProviderTxID: charge.PaymentIntent.ID,
			Status:       domain.StatusRefunded,
			Amount:       decimal.NewFromInt(charge.AmountRefunded).Div(decimal.NewFromInt(100)),
			Currency:     mapStripeCurrency(charge.Currency),
			RawPayload:   payload,
		}

	default:
		// Event type not handled
		return true, nil, nil
	}

	return true, webhookData, nil
}

// QueryStatus queries the payment status from Stripe
func (a *StripeAdapter) QueryStatus(ctx context.Context, providerTxID string) (*port.QueryStatusResponse, error) {
	s, err := session.Get(providerTxID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe session: %w", err)
	}

	status := domain.StatusPending
	switch s.PaymentStatus {
	case stripe.CheckoutSessionPaymentStatusPaid:
		status = domain.StatusSuccess
	case stripe.CheckoutSessionPaymentStatusUnpaid:
		if s.Status == stripe.CheckoutSessionStatusExpired {
			status = domain.StatusFailed
		}
	}

	return &port.QueryStatusResponse{
		ProviderTxID: s.ID,
		Status:       status,
		Amount:       decimal.NewFromInt(s.AmountTotal).Div(decimal.NewFromInt(100)),
		Currency:     mapStripeCurrency(s.Currency),
		RawData: map[string]interface{}{
			"session_status":  string(s.Status),
			"payment_status":  string(s.PaymentStatus),
			"payment_intent":  s.PaymentIntent,
			"customer_email":  s.CustomerDetails.Email,
		},
	}, nil
}

// mapStripeCurrency maps Stripe currency to domain currency
func mapStripeCurrency(c stripe.Currency) domain.Currency {
	switch c {
	case stripe.CurrencyVND:
		return domain.CurrencyVND
	default:
		return domain.CurrencyUSD
	}
}
