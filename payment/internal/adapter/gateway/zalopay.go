package gateway

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"microservices/payment/internal/domain"
	"microservices/payment/internal/port"
)

// ZaloPayConfig holds ZaloPay configuration
type ZaloPayConfig struct {
	AppID    int
	Key1     string // sign outgoing requests
	Key2     string // verify incoming callbacks
	Endpoint string // https://sb-openapi.zalopay.vn/v2
}

// ZaloPayAdapter implements port.PaymentGateway for ZaloPay
type ZaloPayAdapter struct {
	config ZaloPayConfig
	client *http.Client
}

// NewZaloPayAdapter creates a new ZaloPay adapter
func NewZaloPayAdapter(config ZaloPayConfig) *ZaloPayAdapter {
	if config.Endpoint == "" {
		config.Endpoint = "https://sb-openapi.zalopay.vn/v2"
	}

	return &ZaloPayAdapter{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Ensure ZaloPayAdapter implements port.PaymentGateway
var _ port.PaymentGateway = (*ZaloPayAdapter)(nil)

// Provider returns the provider type
func (a *ZaloPayAdapter) Provider() domain.Provider {
	return domain.ProviderZaloPay
}

// CreatePayment creates a ZaloPay payment
func (a *ZaloPayAdapter) CreatePayment(ctx context.Context, req *port.PaymentRequest) (*port.PaymentResponse, error) {
	// Format app_trans_id: YYMMDD_<orderID>
	now := time.Now()
	appTransID := fmt.Sprintf("%s_%s", now.Format("060102"), req.OrderID)

	amount := req.Amount.IntPart()
	appTime := now.UnixMilli()

	// Build embed_data and item (empty for now)
	embedData := "{}"
	item := "[]"

	// Build MAC: app_id|app_trans_id|app_user|amount|app_time|embed_data|item
	macData := fmt.Sprintf("%d|%s|user|%d|%d|%s|%s",
		a.config.AppID,
		appTransID,
		amount,
		appTime,
		embedData,
		item,
	)
	mac := a.signHMACSHA256(macData, a.config.Key1)

	// Build request body
	params := url.Values{}
	params.Set("app_id", fmt.Sprintf("%d", a.config.AppID))
	params.Set("app_trans_id", appTransID)
	params.Set("app_user", "user")
	params.Set("app_time", fmt.Sprintf("%d", appTime))
	params.Set("amount", fmt.Sprintf("%d", amount))
	params.Set("description", req.Description)
	params.Set("embed_data", embedData)
	params.Set("item", item)
	params.Set("mac", mac)

	if req.CallbackURL != "" {
		params.Set("callback_url", req.CallbackURL)
	}

	// Send POST request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		a.config.Endpoint+"/create",
		bytes.NewBufferString(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		ReturnCode   int    `json:"return_code"`
		ReturnMessage string `json:"return_message"`
		OrderURL     string `json:"order_url"`
		ZpTransToken string `json:"zp_trans_token"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.ReturnCode != 1 {
		return nil, fmt.Errorf("ZaloPay error: %s (code: %d)", result.ReturnMessage, result.ReturnCode)
	}

	return &port.PaymentResponse{
		PaymentURL:   result.OrderURL,
		ProviderTxID: appTransID,
		RawData: map[string]interface{}{
			"app_trans_id":   appTransID,
			"zp_trans_token": result.ZpTransToken,
		},
	}, nil
}

// VerifyWebhook verifies the ZaloPay webhook signature and parses the payload
func (a *ZaloPayAdapter) VerifyWebhook(r *http.Request) (bool, *port.WebhookData, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read body: %w", err)
	}

	// ZaloPay callback format: { "data": "<JSON string>", "mac": "<hex>" }
	var callback struct {
		Data string `json:"data"`
		Mac  string `json:"mac"`
	}

	if err := json.Unmarshal(body, &callback); err != nil {
		return false, nil, fmt.Errorf("failed to parse callback: %w", err)
	}

	// Verify MAC using key2
	expectedMac := a.signHMACSHA256(callback.Data, a.config.Key2)
	if callback.Mac != expectedMac {
		return false, nil, nil // Signature mismatch
	}

	// Parse data JSON
	var data struct {
		AppID       int    `json:"app_id"`
		AppTransID  string `json:"app_trans_id"`
		AppUser     string `json:"app_user"`
		Amount      int64  `json:"amount"`
		DiscountAmount int64 `json:"discount_amount"`
		ZpTransID   int64  `json:"zp_trans_id"`
	}

	if err := json.Unmarshal([]byte(callback.Data), &data); err != nil {
		return false, nil, fmt.Errorf("failed to parse data: %w", err)
	}

	return true, &port.WebhookData{
		ProviderTxID: data.AppTransID,
		Status:       domain.StatusSuccess, // Callback is only sent on success
		Amount:       decimal.NewFromInt(data.Amount),
		Currency:     domain.CurrencyVND,
		RawPayload:   body,
		Metadata: map[string]interface{}{
			"zp_trans_id":     data.ZpTransID,
			"app_trans_id":    data.AppTransID,
			"discount_amount": data.DiscountAmount,
		},
	}, nil
}

// QueryStatus queries the payment status from ZaloPay
func (a *ZaloPayAdapter) QueryStatus(ctx context.Context, providerTxID string) (*port.QueryStatusResponse, error) {
	appTime := time.Now().UnixMilli()

	// Build MAC: app_id|app_trans_id|key1
	macData := fmt.Sprintf("%d|%s|%s", a.config.AppID, providerTxID, a.config.Key1)
	mac := a.signHMACSHA256(macData, a.config.Key1)

	// Build request
	params := url.Values{}
	params.Set("app_id", fmt.Sprintf("%d", a.config.AppID))
	params.Set("app_trans_id", providerTxID)
	params.Set("mac", mac)

	// Send POST request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		a.config.Endpoint+"/query",
		bytes.NewBufferString(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		ReturnCode    int    `json:"return_code"`
		ReturnMessage string `json:"return_message"`
		Amount        int64  `json:"amount"`
		ZpTransID     int64  `json:"zp_trans_id"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Map return code to status
	status := domain.StatusPending
	switch result.ReturnCode {
	case 1:
		status = domain.StatusSuccess
	case 2:
		status = domain.StatusFailed
	case 3:
		status = domain.StatusPending // Processing
	}

	_ = appTime // Used for logging if needed

	return &port.QueryStatusResponse{
		ProviderTxID: providerTxID,
		Status:       status,
		Amount:       decimal.NewFromInt(result.Amount),
		Currency:     domain.CurrencyVND,
		RawData: map[string]interface{}{
			"return_code":    result.ReturnCode,
			"return_message": result.ReturnMessage,
			"zp_trans_id":    result.ZpTransID,
		},
	}, nil
}

// Refund initiates a refund with ZaloPay
func (a *ZaloPayAdapter) Refund(ctx context.Context, providerTxID string, amount decimal.Decimal) (string, error) {
	now := time.Now()
	mRefundID := fmt.Sprintf("%s_%s", now.Format("060102"), uuid.New().String()[:8])

	// Build MAC: app_id|zp_trans_id|amount|description|timestamp
	timestamp := now.UnixMilli()
	description := "Refund transaction"
	macData := fmt.Sprintf("%d|%s|%d|%s|%d",
		a.config.AppID,
		providerTxID, // This should be zp_trans_id, but we may only have app_trans_id
		amount.IntPart(),
		description,
		timestamp,
	)
	mac := a.signHMACSHA256(macData, a.config.Key1)

	// Build request
	params := url.Values{}
	params.Set("app_id", fmt.Sprintf("%d", a.config.AppID))
	params.Set("m_refund_id", mRefundID)
	params.Set("zp_trans_id", providerTxID)
	params.Set("amount", fmt.Sprintf("%d", amount.IntPart()))
	params.Set("description", description)
	params.Set("timestamp", fmt.Sprintf("%d", timestamp))
	params.Set("mac", mac)

	// Send POST request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		a.config.Endpoint+"/refund",
		bytes.NewBufferString(params.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		ReturnCode    int    `json:"return_code"`
		ReturnMessage string `json:"return_message"`
		RefundID      int64  `json:"refund_id"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.ReturnCode != 1 && result.ReturnCode != 2 {
		return "", fmt.Errorf("ZaloPay refund failed: %s (code: %d)", result.ReturnMessage, result.ReturnCode)
	}

	return fmt.Sprintf("%d", result.RefundID), nil
}

// signHMACSHA256 creates HMAC-SHA256 signature
func (a *ZaloPayAdapter) signHMACSHA256(data string, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
