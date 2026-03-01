package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"microservices/logistic/internal/core/domain"
	"microservices/logistic/internal/core/ports"
)

// MockProvider implements Provider interface for testing/development
type MockProvider struct {
	rng *rand.Rand
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetName returns the provider name
func (p *MockProvider) GetName() domain.ProviderName {
	return domain.ProviderMock
}

// CalculateFee returns a random fee for testing
func (p *MockProvider) CalculateFee(ctx context.Context, req *ports.RateRequest) (float64, error) {
	// Base fee based on weight
	baseFee := float64(req.WeightGram) * 0.02 // 20 VND per gram

	// Add distance-based fee (mock calculation)
	distanceFee := float64(p.rng.Intn(20000) + 10000) // 10,000 - 30,000 VND

	// Insurance fee (0.5% of value)
	insuranceFee := float64(req.InsuranceValue) * 0.005

	totalFee := baseFee + distanceFee + insuranceFee

	// Round to nearest 1000
	totalFee = float64(int(totalFee/1000)) * 1000

	return totalFee, nil
}

// CreateOrder creates a mock shipment
func (p *MockProvider) CreateOrder(ctx context.Context, req *ports.ShipRequest) (*ports.ShipResponse, error) {
	// Generate mock tracking code
	trackingCode := fmt.Sprintf("MOCK-%d-%d", time.Now().Unix(), p.rng.Intn(99999))

	// Calculate mock fee
	totalWeight := domain.TotalWeight(req.Parcels)
	fee, _ := p.CalculateFee(ctx, &ports.RateRequest{
		FromDistrictID: req.Sender.DistrictID,
		ToDistrictID:   req.Receiver.DistrictID,
		WeightGram:     totalWeight,
		InsuranceValue: int(domain.TotalValue(req.Parcels)),
	})

	return &ports.ShipResponse{
		TrackingCode: trackingCode,
		LabelURL:     fmt.Sprintf("https://mock-label.example.com/%s.pdf", trackingCode),
		ShippingFee:  fee,
	}, nil
}

// MockWebhookPayload represents the expected mock webhook structure
type MockWebhookPayload struct {
	TrackingCode    string `json:"tracking_code"`
	InternalOrderID string `json:"internal_order_id"`
	Status          string `json:"status"`
}

// CancelOrder cancels a mock shipment (always succeeds)
func (p *MockProvider) CancelOrder(ctx context.Context, trackingCode string) error {
	// Mock provider always returns success
	return nil
}

// ParseWebhook parses mock webhook requests
func (p *MockProvider) ParseWebhook(r *http.Request) (*ports.WebhookPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var payload MockWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	return &ports.WebhookPayload{
		TrackingCode:    payload.TrackingCode,
		InternalOrderID: payload.InternalOrderID,
		CarrierStatus:   payload.Status,
		RawPayload:      body,
	}, nil
}
