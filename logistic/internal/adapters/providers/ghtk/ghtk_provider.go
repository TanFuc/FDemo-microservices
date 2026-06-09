package ghtk

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

// Config holds GHTK API configuration
type Config struct {
	APIURL string
	Token  string
}

// Provider implements Provider interface for GiaoHangTietKiem
type Provider struct {
	config Config
	client *http.Client
}

// NewProvider creates a new GHTK provider
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://services.giaohangtietkiem.vn"
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
	return domain.ProviderGHTK
}

// CalculateFee calculates shipping fee via GHTK API
func (p *Provider) CalculateFee(ctx context.Context, req *ports.RateRequest) (float64, error) {
	url := fmt.Sprintf("%s/services/shipment/fee?pick_district=%d&district=%d&weight=%d&value=%d&transport=road",
		p.config.APIURL,
		req.FromDistrictID,
		req.ToDistrictID,
		req.WeightGram,
		req.InsuranceValue,
	)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Token", p.config.Token)

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
		Success bool   `json:"success"`
		Message string `json:"message"`
		Fee     struct {
			Fee int `json:"fee"`
		} `json:"fee"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return 0, fmt.Errorf("GHTK API error: %s", result.Message)
	}

	return float64(result.Fee.Fee), nil
}

// CreateOrder creates a shipping order via GHTK API
func (p *Provider) CreateOrder(ctx context.Context, req *ports.ShipRequest) (*ports.ShipResponse, error) {
	totalWeight := domain.TotalWeight(req.Parcels)
	totalValue := domain.TotalValue(req.Parcels)

	// Build products array
	products := make([]map[string]interface{}, len(req.Parcels))
	for i, parcel := range req.Parcels {
		products[i] = map[string]interface{}{
			"name":     parcel.Name,
			"quantity": parcel.Quantity,
			"weight":   float64(parcel.WeightGram) / 1000, // GHTK uses kg
		}
	}

	payload := map[string]interface{}{
		"products": products,
		"order": map[string]interface{}{
			"id":            req.InternalOrderID,
			"pick_name":     req.Sender.Name,
			"pick_tel":      req.Sender.Phone,
			"pick_address":  req.Sender.Address,
			"pick_province": req.Sender.ProvinceID,
			"pick_district": req.Sender.DistrictID,
			"pick_ward":     req.Sender.WardCode,
			"name":          req.Receiver.Name,
			"tel":           req.Receiver.Phone,
			"address":       req.Receiver.Address,
			"province":      req.Receiver.ProvinceID,
			"district":      req.Receiver.DistrictID,
			"ward":          req.Receiver.WardCode,
			"hamlet":        "Khác",
			"is_freeship":   0,
			"pick_money":    int(req.CODAmount),
			"value":         int(totalValue),
			"weight":        float64(totalWeight) / 1000, // GHTK uses kg
			"note":          req.Note,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.config.APIURL+"/services/shipment/order",
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Token", p.config.Token)

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
		Success bool   `json:"success"`
		Message string `json:"message"`
		Order   struct {
			Label      string `json:"label"`
			Fee        int    `json:"fee"`
			TrackingID string `json:"tracking_id"`
		} `json:"order"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("GHTK API error: %s", result.Message)
	}

	return &ports.ShipResponse{
		TrackingCode: result.Order.TrackingID,
		LabelURL:     result.Order.Label,
		ShippingFee:  float64(result.Order.Fee),
	}, nil
}

// GHTKWebhookPayload represents GHTK webhook structure
type GHTKWebhookPayload struct {
	LabelID   string `json:"label_id"`
	PartnerID string `json:"partner_id"`
	StatusID  int    `json:"status_id"`
}

// ParseWebhook parses GHTK webhook requests
func (p *Provider) ParseWebhook(r *http.Request) (*ports.WebhookPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var payload GHTKWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	return &ports.WebhookPayload{
		TrackingCode:    payload.LabelID,
		InternalOrderID: payload.PartnerID,
		CarrierStatus:   fmt.Sprintf("%d", payload.StatusID),
		RawPayload:      body,
	}, nil
}
