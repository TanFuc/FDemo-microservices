package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// VoucherValidationResult represents the result of voucher validation from Campaign Service
type VoucherValidationResult struct {
	Valid          bool    `json:"valid"`
	VoucherID      string  `json:"voucherId,omitempty"`
	CampaignID     string  `json:"campaignId,omitempty"`
	VoucherCode    string  `json:"voucherCode,omitempty"`
	DiscountType   string  `json:"discountType,omitempty"`   // "PERCENTAGE" or "FIXED_AMOUNT"
	DiscountValue  float64 `json:"discountValue,omitempty"`  // percentage or fixed amount
	DiscountAmount float64 `json:"discountAmount"`           // actual deducted amount
	OriginalTotal  float64 `json:"originalTotal"`
	FinalPrice     float64 `json:"finalPrice"`
	ErrorCode      string  `json:"errorCode,omitempty"`
	ErrorMessage   string  `json:"errorMessage,omitempty"`
}

// CartItemForVoucher represents a cart item for voucher validation
type CartItemForVoucher struct {
	SkuID    string  `json:"sku"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Category string  `json:"category"`
}

// CampaignClient interface for interacting with Campaign Service
type CampaignClient interface {
	ValidateVoucherForCart(ctx context.Context, voucherCode, userID string, items []CartItemForVoucher) (*VoucherValidationResult, error)
}

// CampaignClientConfig holds configuration for campaign client
type CampaignClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	ServiceKey string
}

// HTTPCampaignClient implements CampaignClient using HTTP
type HTTPCampaignClient struct {
	baseURL    string
	httpClient *http.Client
	serviceKey string
}

// NewHTTPCampaignClient creates a new HTTP campaign client
func NewHTTPCampaignClient(cfg CampaignClientConfig) *HTTPCampaignClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	return &HTTPCampaignClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		serviceKey: cfg.ServiceKey,
	}
}

// validateVoucherRequest is the request payload for voucher validation
type validateVoucherRequest struct {
	VoucherCode string               `json:"voucherCode"`
	UserID      string               `json:"userId"`
	Items       []CartItemForVoucher `json:"items"`
}

// ValidateVoucherForCart validates a voucher against cart items
func (c *HTTPCampaignClient) ValidateVoucherForCart(ctx context.Context, voucherCode, userID string, items []CartItemForVoucher) (*VoucherValidationResult, error) {
	url := fmt.Sprintf("%s/api/v1/vouchers/validate", c.baseURL)

	reqBody := validateVoucherRequest{
		VoucherCode: voucherCode,
		UserID:      userID,
		Items:       items,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.serviceKey != "" {
		req.Header.Set("X-Internal-Service-Key", c.serviceKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call campaign service: %w", err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("campaign service unavailable: status %d", resp.StatusCode)
	}

	var result VoucherValidationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Ensure HTTPCampaignClient implements CampaignClient
var _ CampaignClient = (*HTTPCampaignClient)(nil)

// MockCampaignClient is a mock implementation for development/testing
type MockCampaignClient struct{}

// ValidateVoucherForCart returns a mock validation result
func (c *MockCampaignClient) ValidateVoucherForCart(ctx context.Context, voucherCode, userID string, items []CartItemForVoucher) (*VoucherValidationResult, error) {
	// Calculate total
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}

	// Mock 10% discount
	discount := total * 0.1

	return &VoucherValidationResult{
		Valid:          true,
		VoucherID:      "mock-voucher-id",
		CampaignID:     "mock-campaign-id",
		VoucherCode:    voucherCode,
		DiscountType:   "PERCENTAGE",
		DiscountValue:  10,
		DiscountAmount: discount,
		OriginalTotal:  total,
		FinalPrice:     total - discount,
	}, nil
}

// Ensure MockCampaignClient implements CampaignClient
var _ CampaignClient = (*MockCampaignClient)(nil)

// NoCampaignClient is a no-op implementation when voucher validation is disabled
type NoCampaignClient struct{}

// ValidateVoucherForCart always returns invalid voucher (no campaign service configured)
func (c *NoCampaignClient) ValidateVoucherForCart(ctx context.Context, voucherCode, userID string, items []CartItemForVoucher) (*VoucherValidationResult, error) {
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}

	return &VoucherValidationResult{
		Valid:         false,
		OriginalTotal: total,
		FinalPrice:    total,
		ErrorCode:     "VOUCHER_SERVICE_UNAVAILABLE",
		ErrorMessage:  "Voucher service is not configured",
	}, nil
}

// Ensure NoCampaignClient implements CampaignClient
var _ CampaignClient = (*NoCampaignClient)(nil)
