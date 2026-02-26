package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CreateDraftOrderRequest represents the request to create a draft order
type CreateDraftOrderRequest struct {
	UserID          string                   `json:"userId"`
	Items           []DraftOrderItem         `json:"items"`
	ShippingAddress DraftOrderShippingAddr   `json:"shippingAddress"`
	PaymentMethod   string                   `json:"paymentMethod"`
	CustomerNote    string                   `json:"customerNote,omitempty"`
	VoucherCode     string                   `json:"voucherCode,omitempty"`
	VoucherID       string                   `json:"voucherId,omitempty"`
	CampaignID      string                   `json:"campaignId,omitempty"`
	OriginalAmount  float64                  `json:"originalAmount"`
	DiscountAmount  float64                  `json:"discountAmount"`
	ShippingFee     float64                  `json:"shippingFee"`
	FinalAmount     float64                  `json:"finalAmount"`
}

// DraftOrderItem represents an item in the draft order
type DraftOrderItem struct {
	SkuID       string  `json:"skuId"`
	ProductID   string  `json:"productId,omitempty"`
	ProductName string  `json:"productName"`
	SkuCode     string  `json:"skuCode,omitempty"`
	Thumbnail   string  `json:"thumbnail,omitempty"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
}

// DraftOrderShippingAddr represents the shipping address for a draft order
type DraftOrderShippingAddr struct {
	FullName   string `json:"fullName"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	Ward       string `json:"ward,omitempty"`
	District   string `json:"district"`
	City       string `json:"city"`
	Country    string `json:"country"`
	PostalCode string `json:"postalCode,omitempty"`
}

// DraftOrderResponse represents the response after creating a draft order
type DraftOrderResponse struct {
	DraftOrderID string  `json:"id"`
	OrderNumber  string  `json:"orderNumber"`
	Status       string  `json:"status"` // "DRAFT"
	FinalAmount  float64 `json:"finalAmount"`
	CreatedAt    string  `json:"createdAt"`
}

// OrderClient interface for interacting with Order Service
type OrderClient interface {
	CreateDraftOrder(ctx context.Context, req *CreateDraftOrderRequest) (*DraftOrderResponse, error)
}

// OrderClientConfig holds configuration for order client
type OrderClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	ServiceKey string
}

// HTTPOrderClient implements OrderClient using HTTP
type HTTPOrderClient struct {
	baseURL    string
	httpClient *http.Client
	serviceKey string
}

// NewHTTPOrderClient creates a new HTTP order client
func NewHTTPOrderClient(cfg OrderClientConfig) *HTTPOrderClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &HTTPOrderClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		serviceKey: cfg.ServiceKey,
	}
}

// CreateDraftOrder creates a draft order in the Order Service
func (c *HTTPOrderClient) CreateDraftOrder(ctx context.Context, req *CreateDraftOrderRequest) (*DraftOrderResponse, error) {
	url := fmt.Sprintf("%s/api/v1/orders/draft", c.baseURL)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceKey != "" {
		httpReq.Header.Set("X-Internal-Service-Key", c.serviceKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call order service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("order service unavailable: status %d", resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Message != "" {
			return nil, fmt.Errorf("order service error: %s", errResp.Message)
		}
		return nil, fmt.Errorf("order service returned status %d", resp.StatusCode)
	}

	var result DraftOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Ensure HTTPOrderClient implements OrderClient
var _ OrderClient = (*HTTPOrderClient)(nil)

// MockOrderClient is a mock implementation for development/testing
type MockOrderClient struct{}

// CreateDraftOrder returns a mock draft order response
func (c *MockOrderClient) CreateDraftOrder(ctx context.Context, req *CreateDraftOrderRequest) (*DraftOrderResponse, error) {
	return &DraftOrderResponse{
		DraftOrderID: "mock-draft-order-id",
		OrderNumber:  "MOCK-ORD-001",
		Status:       "DRAFT",
		FinalAmount:  req.FinalAmount,
		CreatedAt:    time.Now().Format(time.RFC3339),
	}, nil
}

// Ensure MockOrderClient implements OrderClient
var _ OrderClient = (*MockOrderClient)(nil)

// NoOrderClient is a no-op implementation when order service is not configured
type NoOrderClient struct{}

// CreateDraftOrder returns an error indicating order service is unavailable
func (c *NoOrderClient) CreateDraftOrder(ctx context.Context, req *CreateDraftOrderRequest) (*DraftOrderResponse, error) {
	return nil, fmt.Errorf("order service is not configured")
}

// Ensure NoOrderClient implements OrderClient
var _ OrderClient = (*NoOrderClient)(nil)
