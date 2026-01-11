package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CatalogClient interface for interacting with Catalog Service
type CatalogClient interface {
	GetProductPrice(ctx context.Context, skuID string) (float64, error)
	ValidateProduct(ctx context.Context, skuID string) (*ProductInfo, error)
}

// ProductInfo contains product information from Catalog Service
type ProductInfo struct {
	SkuID     string  `json:"sku_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Available bool    `json:"available"`
	Thumbnail string  `json:"thumbnail"`
}

// HTTPCatalogClient implements CatalogClient using HTTP
type HTTPCatalogClient struct {
	baseURL    string
	httpClient *http.Client
	serviceKey string
}

// CatalogClientConfig holds configuration for catalog client
type CatalogClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	ServiceKey string
}

// NewHTTPCatalogClient creates a new HTTP catalog client
func NewHTTPCatalogClient(cfg CatalogClientConfig) *HTTPCatalogClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	return &HTTPCatalogClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		serviceKey: cfg.ServiceKey,
	}
}

// GetProductPrice fetches the current price for a SKU from Catalog Service
func (c *HTTPCatalogClient) GetProductPrice(ctx context.Context, skuID string) (float64, error) {
	info, err := c.ValidateProduct(ctx, skuID)
	if err != nil {
		return 0, err
	}
	return info.Price, nil
}

// ValidateProduct validates a product exists and returns its info
func (c *HTTPCatalogClient) ValidateProduct(ctx context.Context, skuID string) (*ProductInfo, error) {
	url := fmt.Sprintf("%s/internal/products/sku/%s", c.baseURL, skuID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add internal service authentication header
	if c.serviceKey != "" {
		req.Header.Set("X-Internal-Service-Key", c.serviceKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call catalog service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("product not found: %s", skuID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog service returned status %d", resp.StatusCode)
	}

	var info ProductInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &info, nil
}

// Ensure HTTPCatalogClient implements CatalogClient
var _ CatalogClient = (*HTTPCatalogClient)(nil)

// MockCatalogClient is a mock implementation for development
type MockCatalogClient struct{}

// GetProductPrice returns a mock price
func (c *MockCatalogClient) GetProductPrice(ctx context.Context, skuID string) (float64, error) {
	// In mock mode, always return the requested price (no validation)
	return 0, nil
}

// ValidateProduct returns mock product info
func (c *MockCatalogClient) ValidateProduct(ctx context.Context, skuID string) (*ProductInfo, error) {
	return &ProductInfo{
		SkuID:     skuID,
		Name:      "Mock Product",
		Price:     100.0,
		Available: true,
		Thumbnail: "",
	}, nil
}

// Ensure MockCatalogClient implements CatalogClient
var _ CatalogClient = (*MockCatalogClient)(nil)

// NoCatalogClient is a no-op implementation when price validation is disabled
type NoCatalogClient struct{}

// GetProductPrice always returns 0 (no validation)
func (c *NoCatalogClient) GetProductPrice(ctx context.Context, skuID string) (float64, error) {
	return 0, nil
}

// ValidateProduct always returns nil (no validation)
func (c *NoCatalogClient) ValidateProduct(ctx context.Context, skuID string) (*ProductInfo, error) {
	return nil, nil
}

// Ensure NoCatalogClient implements CatalogClient
var _ CatalogClient = (*NoCatalogClient)(nil)
