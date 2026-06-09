package ghn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"microservices/logistic/internal/core/domain"
	"microservices/logistic/internal/core/ports"
)

// Config holds GHN API configuration
type Config struct {
	APIURL string
	Token  string
	ShopID string
}

// Provider implements Provider interface for GiaoHangNhanh
type Provider struct {
	config Config
	client *http.Client
}

// NewProvider creates a new GHN provider
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://dev-online-gateway.ghn.vn"
	}

	return &Provider{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GetName returns the provider name
func (p *Provider) GetName() domain.ProviderName {
	return domain.ProviderGHN
}

// CalculateFee calculates shipping fee via GHN API
func (p *Provider) CalculateFee(ctx context.Context, req *ports.RateRequest) (float64, error) {
	payload := map[string]interface{}{
		"service_type_id":  2, // Standard delivery
		"from_district_id": req.FromDistrictID,
		"to_district_id":   req.ToDistrictID,
		"weight":           req.WeightGram,
		"insurance_value":  req.InsuranceValue,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.config.APIURL+"/shiip/public-api/v2/shipping-order/fee",
		bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Token", p.config.Token)
	httpReq.Header.Set("ShopId", p.config.ShopID)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Total int `json:"total"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != 200 {
		return 0, fmt.Errorf("GHN API error: %s", result.Message)
	}

	return float64(result.Data.Total), nil
}

// CreateOrder creates a shipping order via GHN API
func (p *Provider) CreateOrder(ctx context.Context, req *ports.ShipRequest) (*ports.ShipResponse, error) {
	totalWeight := domain.TotalWeight(req.Parcels)

	// Build items array
	items := make([]map[string]interface{}, len(req.Parcels))
	for i, parcel := range req.Parcels {
		items[i] = map[string]interface{}{
			"name":     parcel.Name,
			"quantity": parcel.Quantity,
			"weight":   parcel.WeightGram,
		}
	}

	payload := map[string]interface{}{
		"service_type_id":   2, // Standard delivery
		"payment_type_id":   1, // Sender pays
		"required_note":     "KHONGCHOXEMHANG",
		"client_order_code": req.InternalOrderID,
		"from_name":         req.Sender.Name,
		"from_phone":        req.Sender.Phone,
		"from_address":      req.Sender.Address,
		"from_ward_code":    req.Sender.WardCode,
		"from_district_id":  req.Sender.DistrictID,
		"to_name":           req.Receiver.Name,
		"to_phone":          req.Receiver.Phone,
		"to_address":        req.Receiver.Address,
		"to_ward_code":      req.Receiver.WardCode,
		"to_district_id":    req.Receiver.DistrictID,
		"weight":            totalWeight,
		"cod_amount":        int(req.CODAmount),
		"content":           req.Note,
		"items":             items,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.config.APIURL+"/shiip/public-api/v2/shipping-order/create",
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Token", p.config.Token)
	httpReq.Header.Set("ShopId", p.config.ShopID)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			OrderCode string `json:"order_code"`
			TotalFee  int    `json:"total_fee"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("GHN API error: %s", result.Message)
	}

	// Generate print URL
	labelURL := fmt.Sprintf("%s/shiip/public-api/v2/a5/gen-token?order_codes=%s",
		p.config.APIURL, result.Data.OrderCode)

	return &ports.ShipResponse{
		TrackingCode: result.Data.OrderCode,
		LabelURL:     labelURL,
		ShippingFee:  float64(result.Data.TotalFee),
	}, nil
}

// GHNWebhookPayload represents GHN webhook structure
type GHNWebhookPayload struct {
	OrderCode       string `json:"OrderCode"`
	ClientOrderCode string `json:"ClientOrderCode"`
	Status          string `json:"Status"`
}

// ParseWebhook parses GHN webhook requests
func (p *Provider) ParseWebhook(r *http.Request) (*ports.WebhookPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var payload GHNWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	return &ports.WebhookPayload{
		TrackingCode:    payload.OrderCode,
		InternalOrderID: payload.ClientOrderCode,
		CarrierStatus:   payload.Status,
		RawPayload:      body,
	}, nil
}
