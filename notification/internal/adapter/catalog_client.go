package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CatalogClient retrieves product/shop metadata from Catalog Service
type CatalogClient interface {
	GetProductSeller(ctx context.Context, productID string) (*ProductSeller, error)
}

// ProductSeller contains the seller info for a product
type ProductSeller struct {
	ProductID   string `json:"productId"`
	ProductName string `json:"productName"`
	ShopID      string `json:"shopId"`
	SellerID    string `json:"sellerId"`
	ShopName    string `json:"shopName"`
}

// HTTPCatalogClient implements CatalogClient using HTTP
type HTTPCatalogClient struct {
	baseURL    string
	httpClient *http.Client
	serviceKey string
}

// NewHTTPCatalogClient creates a new HTTP-based catalog client
func NewHTTPCatalogClient(baseURL, serviceKey string) *HTTPCatalogClient {
	return &HTTPCatalogClient{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// GetProductSeller fetches the seller information for a product
func (c *HTTPCatalogClient) GetProductSeller(ctx context.Context, productID string) (*ProductSeller, error) {
	url := fmt.Sprintf("%s/internal/products/%s/seller", c.baseURL, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Service-Key", c.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("product %s not found", productID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog service returned %d", resp.StatusCode)
	}

	var seller ProductSeller
	if err := json.NewDecoder(resp.Body).Decode(&seller); err != nil {
		return nil, err
	}
	return &seller, nil
}
