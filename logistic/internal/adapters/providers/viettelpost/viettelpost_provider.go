package viettelpost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"microservices/logistic/internal/core/domain"
	"microservices/logistic/internal/core/ports"
)

// Config holds ViettelPost API configuration
type Config struct {
	APIURL           string
	Username         string
	Password         string
	SenderProvinceID int
	SenderDistrictID int
}

// Provider implements Provider interface for ViettelPost
type Provider struct {
	config     Config
	client     *http.Client
	token      string
	tokenMutex sync.RWMutex
	userID     int64
}

// NewProvider creates a new ViettelPost provider
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://partner.viettelpost.vn/v2"
	}

	p := &Provider{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Perform initial authentication
	if err := p.authenticate(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to authenticate with ViettelPost: %w", err)
	}

	return p, nil
}

// authenticate logs in to ViettelPost and caches the token
func (p *Provider) authenticate(ctx context.Context) error {
	payload := map[string]string{
		"USERNAME": p.config.Username,
		"PASSWORD": p.config.Password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.config.APIURL+"/user/Login",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Token  string `json:"TOKEN"`
			UserID int64  `json:"USERID"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if result.Status != 200 || result.Data.Token == "" {
		return fmt.Errorf("ViettelPost login failed: %s", result.Message)
	}

	p.tokenMutex.Lock()
	p.token = result.Data.Token
	p.userID = result.Data.UserID
	p.tokenMutex.Unlock()

	return nil
}

// getToken returns the cached token (thread-safe)
func (p *Provider) getToken() string {
	p.tokenMutex.RLock()
	defer p.tokenMutex.RUnlock()
	return p.token
}

// doRequestWithAuth performs an HTTP request with authentication, re-authenticating on 401
func (p *Provider) doRequestWithAuth(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Token", p.getToken())

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Re-authenticate on 401
	if resp.StatusCode == http.StatusUnauthorized {
		if err := p.authenticate(ctx); err != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", err)
		}

		// Retry request with new token
		httpReq, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create retry request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Token", p.getToken())

		resp, err = p.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to send retry request: %w", err)
		}
		defer resp.Body.Close()
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, nil
}

// GetName returns the provider name
func (p *Provider) GetName() domain.ProviderName {
	return domain.ProviderViettelPost
}

// CalculateFee calculates shipping fee via ViettelPost API
func (p *Provider) CalculateFee(ctx context.Context, req *ports.RateRequest) (float64, error) {
	payload := map[string]interface{}{
		"SENDER_PROVINCE":   p.config.SenderProvinceID,
		"SENDER_DISTRICT":   p.config.SenderDistrictID,
		"RECEIVER_PROVINCE": req.ToDistrictID / 100, // Approximate province from district
		"RECEIVER_DISTRICT": req.ToDistrictID,
		"PRODUCT_WEIGHT":    req.WeightGram,
		"PRODUCT_PRICE":     req.InsuranceValue,
		"MONEY_COLLECTION":  0,
		"PRODUCT_TYPE":      "HH", // Goods type
		"ORDER_SERVICE":     "VCN", // Standard delivery
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := p.doRequestWithAuth(ctx, "POST", p.config.APIURL+"/order/getPriceAll", body)
	if err != nil {
		return 0, err
	}

	var result []struct {
		MoneyTotal float64 `json:"MONEY_TOTAL"`
		ServiceID  string  `json:"MA_DV_CHINH"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result) == 0 {
		return 0, fmt.Errorf("ViettelPost returned no price data")
	}

	return result[0].MoneyTotal, nil
}

// CreateOrder creates a shipping order via ViettelPost API
func (p *Provider) CreateOrder(ctx context.Context, req *ports.ShipRequest) (*ports.ShipResponse, error) {
	totalWeight := domain.TotalWeight(req.Parcels)
	totalValue := domain.TotalValue(req.Parcels)

	// Build product description
	productName := ""
	for i, parcel := range req.Parcels {
		if i > 0 {
			productName += ", "
		}
		productName += parcel.Name
	}

	payload := map[string]interface{}{
		"ORDER_NUMBER":       req.InternalOrderID,
		"SENDER_FULLNAME":    req.Sender.Name,
		"SENDER_PHONE":       req.Sender.Phone,
		"SENDER_ADDRESS":     req.Sender.Address,
		"SENDER_PROVINCE":    p.config.SenderProvinceID,
		"SENDER_DISTRICT":    req.Sender.DistrictID,
		"RECEIVER_FULLNAME":  req.Receiver.Name,
		"RECEIVER_PHONE":     req.Receiver.Phone,
		"RECEIVER_ADDRESS":   req.Receiver.Address,
		"RECEIVER_PROVINCE":  req.Receiver.ProvinceID,
		"RECEIVER_DISTRICT":  req.Receiver.DistrictID,
		"PRODUCT_NAME":       productName,
		"PRODUCT_WEIGHT":     totalWeight,
		"PRODUCT_PRICE":      int(totalValue),
		"MONEY_COLLECTION":   int(req.CODAmount),
		"ORDER_PAYMENT":      3, // Receiver pays
		"ORDER_SERVICE":      "VCN", // Standard delivery
		"ORDER_NOTE":         req.Note,
		"PRODUCT_TYPE":       "HH",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := p.doRequestWithAuth(ctx, "POST", p.config.APIURL+"/order/createOrder", body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    struct {
			OrderNumber string  `json:"ORDER_NUMBER"`
			MoneyTotal  float64 `json:"MONEY_TOTAL"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != 200 {
		return nil, fmt.Errorf("ViettelPost API error: %s", result.Message)
	}

	return &ports.ShipResponse{
		TrackingCode: result.Data.OrderNumber,
		LabelURL:     fmt.Sprintf("https://viettelpost.vn/tracking/%s", result.Data.OrderNumber),
		ShippingFee:  result.Data.MoneyTotal,
	}, nil
}

// CancelOrder cancels a ViettelPost shipment
func (p *Provider) CancelOrder(ctx context.Context, trackingCode string) error {
	payload := map[string]interface{}{
		"TYPE":         4, // Cancel order type
		"ORDER_NUMBER": trackingCode,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := p.doRequestWithAuth(ctx, "POST", p.config.APIURL+"/order/cancelorder", body)
	if err != nil {
		return err
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != 200 {
		return fmt.Errorf("ViettelPost API error: %s", result.Message)
	}

	return nil
}

// ViettelPostWebhookPayload represents ViettelPost webhook structure
type ViettelPostWebhookPayload struct {
	OrderNumber string `json:"ORDER_NUMBER"`
	Status      string `json:"STATUS"`
}

// ParseWebhook parses ViettelPost webhook requests
func (p *Provider) ParseWebhook(r *http.Request) (*ports.WebhookPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var payload ViettelPostWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	return &ports.WebhookPayload{
		TrackingCode:    payload.OrderNumber,
		InternalOrderID: payload.OrderNumber, // ViettelPost uses ORDER_NUMBER as both
		CarrierStatus:   payload.Status,
		RawPayload:      body,
	}, nil
}
