package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/tafu/payment-service/internal/domain"
	"github.com/tafu/payment-service/internal/port"
)

// MockGateway implements port.PaymentGateway for testing
type MockGateway struct {
	provider     domain.Provider
	payments     map[string]*mockPayment
	mu           sync.RWMutex
	webhookValid bool
}

type mockPayment struct {
	ProviderTxID string
	Status       domain.Status
	Request      *port.PaymentRequest
}

// NewMockGateway creates a new mock gateway
func NewMockGateway(provider domain.Provider) *MockGateway {
	return &MockGateway{
		provider:     provider,
		payments:     make(map[string]*mockPayment),
		webhookValid: true,
	}
}

// Ensure MockGateway implements port.PaymentGateway
var _ port.PaymentGateway = (*MockGateway)(nil)

// Provider returns the provider type
func (m *MockGateway) Provider() domain.Provider {
	return m.provider
}

// SetWebhookValid sets whether webhook verification should succeed
func (m *MockGateway) SetWebhookValid(valid bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhookValid = valid
}

// SetPaymentStatus sets the status for a payment (for testing)
func (m *MockGateway) SetPaymentStatus(providerTxID string, status domain.Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.payments[providerTxID]; ok {
		p.Status = status
	}
}

// CreatePayment creates a mock payment
func (m *MockGateway) CreatePayment(ctx context.Context, req *port.PaymentRequest) (*port.PaymentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	txID := fmt.Sprintf("MOCK-%s-%s", m.provider, uuid.New().String())

	m.payments[txID] = &mockPayment{
		ProviderTxID: txID,
		Status:       domain.StatusPending,
		Request:      req,
	}

	return &port.PaymentResponse{
		PaymentURL:   fmt.Sprintf("https://mock-payment.test/pay/%s", txID),
		ProviderTxID: txID,
		RawData: map[string]interface{}{
			"mock":     true,
			"order_id": req.OrderID,
		},
	}, nil
}

// VerifyWebhook verifies the mock webhook
func (m *MockGateway) VerifyWebhook(r *http.Request) (bool, *port.WebhookData, error) {
	m.mu.RLock()
	valid := m.webhookValid
	m.mu.RUnlock()

	if !valid {
		return false, nil, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read body: %w", err)
	}

	var payload struct {
		ProviderTxID string `json:"provider_tx_id"`
		Status       string `json:"status"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return false, nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	m.mu.RLock()
	payment, exists := m.payments[payload.ProviderTxID]
	m.mu.RUnlock()

	if !exists {
		return true, nil, fmt.Errorf("payment not found: %s", payload.ProviderTxID)
	}

	status := domain.StatusPending
	switch payload.Status {
	case "SUCCESS":
		status = domain.StatusSuccess
	case "FAILED":
		status = domain.StatusFailed
	case "REFUNDED":
		status = domain.StatusRefunded
	}

	return true, &port.WebhookData{
		ProviderTxID: payload.ProviderTxID,
		Status:       status,
		Amount:       payment.Request.Amount,
		Currency:     payment.Request.Currency,
		RawPayload:   body,
		Metadata:     map[string]interface{}{},
	}, nil
}

// QueryStatus queries the mock payment status
func (m *MockGateway) QueryStatus(ctx context.Context, providerTxID string) (*port.QueryStatusResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	payment, exists := m.payments[providerTxID]
	if !exists {
		return nil, fmt.Errorf("payment not found: %s", providerTxID)
	}

	return &port.QueryStatusResponse{
		ProviderTxID: providerTxID,
		Status:       payment.Status,
		Amount:       payment.Request.Amount,
		Currency:     payment.Request.Currency,
		RawData: map[string]interface{}{
			"mock": true,
		},
	}, nil
}
