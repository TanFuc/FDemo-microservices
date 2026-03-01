package gateway

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"microservices/payment/internal/domain"
	"microservices/payment/internal/port"
)

// VNPayConfig holds VNPay configuration
type VNPayConfig struct {
	TmnCode    string // vnp_TmnCode — terminal code
	HashSecret string // HMAC-SHA512 key
	PayURL     string // e.g., https://sandbox.vnpayment.vn/paymentv2/vpcpay.html
	ReturnURL  string // browser redirect after payment
	APIURL     string // e.g., https://sandbox.vnpayment.vn/merchant_webapi/api/transaction
}

// VNPayAdapter implements port.PaymentGateway for VNPay
type VNPayAdapter struct {
	config VNPayConfig
	client *http.Client
}

// NewVNPayAdapter creates a new VNPay adapter
func NewVNPayAdapter(config VNPayConfig) *VNPayAdapter {
	if config.PayURL == "" {
		config.PayURL = "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html"
	}
	if config.APIURL == "" {
		config.APIURL = "https://sandbox.vnpayment.vn/merchant_webapi/api/transaction"
	}

	return &VNPayAdapter{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Ensure VNPayAdapter implements port.PaymentGateway
var _ port.PaymentGateway = (*VNPayAdapter)(nil)

// Provider returns the provider type
func (a *VNPayAdapter) Provider() domain.Provider {
	return domain.ProviderVNPay
}

// CreatePayment creates a VNPay payment
func (a *VNPayAdapter) CreatePayment(ctx context.Context, req *port.PaymentRequest) (*port.PaymentResponse, error) {
	// VNPay amount is in VND (no cents), multiply by 100 for their format
	amount := req.Amount.IntPart() * 100

	// Build params map
	params := map[string]string{
		"vnp_Version":    "2.1.0",
		"vnp_Command":    "pay",
		"vnp_TmnCode":    a.config.TmnCode,
		"vnp_Amount":     fmt.Sprintf("%d", amount),
		"vnp_CurrCode":   "VND",
		"vnp_TxnRef":     req.OrderID,
		"vnp_OrderInfo":  req.Description,
		"vnp_OrderType":  "other",
		"vnp_Locale":     "vn",
		"vnp_IpAddr":     "127.0.0.1",
		"vnp_CreateDate": time.Now().Format("20060102150405"),
	}

	// Use provided return URL or default
	returnURL := req.ReturnURL
	if returnURL == "" {
		returnURL = a.config.ReturnURL
	}
	params["vnp_ReturnUrl"] = returnURL

	// Sort keys alphabetically and build hash data
	hashData := a.buildHashData(params)

	// Sign with HMAC-SHA512
	signature := a.signHMACSHA512(hashData)
	params["vnp_SecureHash"] = signature

	// Build payment URL
	paymentURL := a.config.PayURL + "?" + a.encodeParams(params)

	return &port.PaymentResponse{
		PaymentURL:   paymentURL,
		ProviderTxID: req.OrderID, // VNPay uses our OrderID as reference
		RawData: map[string]interface{}{
			"vnp_TxnRef": req.OrderID,
		},
	}, nil
}

// VerifyWebhook verifies the VNPay webhook signature and parses the payload
func (a *VNPayAdapter) VerifyWebhook(r *http.Request) (bool, *port.WebhookData, error) {
	// VNPay sends params via query string, not JSON body
	query := r.URL.Query()

	// Extract and remove signature params
	receivedHash := query.Get("vnp_SecureHash")
	query.Del("vnp_SecureHash")
	query.Del("vnp_SecureHashType")

	// Build hash data from remaining params
	params := make(map[string]string)
	for key := range query {
		params[key] = query.Get(key)
	}

	hashData := a.buildHashData(params)
	expectedHash := a.signHMACSHA512(hashData)

	// Verify signature
	if receivedHash != expectedHash {
		return false, nil, nil
	}

	// Parse transaction status
	status := domain.StatusFailed
	if query.Get("vnp_TransactionStatus") == "00" {
		status = domain.StatusSuccess
	}

	// Parse amount (VNPay sends amount * 100)
	amountStr := query.Get("vnp_Amount")
	amountInt := int64(0)
	fmt.Sscanf(amountStr, "%d", &amountInt)
	amount := decimal.NewFromInt(amountInt / 100)

	return true, &port.WebhookData{
		ProviderTxID: query.Get("vnp_TxnRef"),
		Status:       status,
		Amount:       amount,
		Currency:     domain.CurrencyVND,
		RawPayload:   []byte(r.URL.RawQuery),
		Metadata: map[string]interface{}{
			"vnp_TransactionNo":     query.Get("vnp_TransactionNo"),
			"vnp_TransactionStatus": query.Get("vnp_TransactionStatus"),
			"vnp_ResponseCode":      query.Get("vnp_ResponseCode"),
			"vnp_BankCode":          query.Get("vnp_BankCode"),
		},
	}, nil
}

// QueryStatus queries the payment status from VNPay
func (a *VNPayAdapter) QueryStatus(ctx context.Context, providerTxID string) (*port.QueryStatusResponse, error) {
	params := map[string]string{
		"vnp_Version":      "2.1.0",
		"vnp_Command":      "querydr",
		"vnp_TmnCode":      a.config.TmnCode,
		"vnp_TxnRef":       providerTxID,
		"vnp_OrderInfo":    "Query transaction",
		"vnp_TransactionNo": "",
		"vnp_TransDate":    time.Now().Add(-24 * time.Hour).Format("20060102150405"),
		"vnp_CreateDate":   time.Now().Format("20060102150405"),
		"vnp_IpAddr":       "127.0.0.1",
	}

	// Sign request
	hashData := a.buildHashData(params)
	params["vnp_SecureHash"] = a.signHMACSHA512(hashData)

	// Send POST request
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse status
	status := domain.StatusPending
	if txStatus, ok := result["vnp_TransactionStatus"].(string); ok {
		if txStatus == "00" {
			status = domain.StatusSuccess
		} else if txStatus != "" {
			status = domain.StatusFailed
		}
	}

	return &port.QueryStatusResponse{
		ProviderTxID: providerTxID,
		Status:       status,
		Amount:       decimal.Zero,
		Currency:     domain.CurrencyVND,
		RawData:      result,
	}, nil
}

// Refund initiates a refund with VNPay
func (a *VNPayAdapter) Refund(ctx context.Context, providerTxID string, amount decimal.Decimal) (string, error) {
	params := map[string]string{
		"vnp_Version":     "2.1.0",
		"vnp_Command":     "refund",
		"vnp_TmnCode":     a.config.TmnCode,
		"vnp_TxnRef":      providerTxID,
		"vnp_Amount":      fmt.Sprintf("%d", amount.IntPart()*100),
		"vnp_OrderInfo":   "Refund transaction",
		"vnp_TransDate":   time.Now().Format("20060102150405"),
		"vnp_CreateDate":  time.Now().Format("20060102150405"),
		"vnp_IpAddr":      "127.0.0.1",
		"vnp_TransactionType": "02", // Partial refund
	}

	// Sign request
	hashData := a.buildHashData(params)
	params["vnp_SecureHash"] = a.signHMACSHA512(hashData)

	// Send POST request
	jsonData, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response code
	if respCode, ok := result["vnp_ResponseCode"].(string); ok && respCode != "00" {
		return "", fmt.Errorf("VNPay refund failed: %v", result["vnp_Message"])
	}

	refundID := ""
	if txNo, ok := result["vnp_TransactionNo"].(string); ok {
		refundID = txNo
	}

	return refundID, nil
}

// buildHashData builds the hash data string from params
func (a *VNPayAdapter) buildHashData(params map[string]string) string {
	// Get sorted keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build hash data
	var parts []string
	for _, k := range keys {
		if params[k] != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
		}
	}

	return strings.Join(parts, "&")
}

// encodeParams encodes params for URL
func (a *VNPayAdapter) encodeParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values.Encode()
}

// signHMACSHA512 creates HMAC-SHA512 signature
func (a *VNPayAdapter) signHMACSHA512(data string) string {
	mac := hmac.New(sha512.New, []byte(a.config.HashSecret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
