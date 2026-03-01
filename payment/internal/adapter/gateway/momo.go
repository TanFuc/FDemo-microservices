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
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"microservices/payment/internal/domain"
	"microservices/payment/internal/port"
)

// MoMoConfig holds MoMo configuration
type MoMoConfig struct {
	PartnerCode string
	AccessKey   string
	SecretKey   string
	Endpoint    string // e.g., https://test-payment.momo.vn/v2/gateway/api
}

// MoMoAdapter implements port.PaymentGateway for MoMo
type MoMoAdapter struct {
	config MoMoConfig
	client *http.Client
}

// NewMoMoAdapter creates a new MoMo adapter
func NewMoMoAdapter(config MoMoConfig) *MoMoAdapter {
	return &MoMoAdapter{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Ensure MoMoAdapter implements port.PaymentGateway
var _ port.PaymentGateway = (*MoMoAdapter)(nil)

// Provider returns the provider type
func (a *MoMoAdapter) Provider() domain.Provider {
	return domain.ProviderMoMo
}

// momoCreateRequest represents MoMo create payment request
type momoCreateRequest struct {
	PartnerCode string `json:"partnerCode"`
	RequestID   string `json:"requestId"`
	Amount      int64  `json:"amount"`
	OrderID     string `json:"orderId"`
	OrderInfo   string `json:"orderInfo"`
	RedirectURL string `json:"redirectUrl"`
	IpnURL      string `json:"ipnUrl"`
	RequestType string `json:"requestType"`
	ExtraData   string `json:"extraData"`
	Lang        string `json:"lang"`
	Signature   string `json:"signature"`
}

// momoCreateResponse represents MoMo create payment response
type momoCreateResponse struct {
	PartnerCode string `json:"partnerCode"`
	RequestID   string `json:"requestId"`
	OrderID     string `json:"orderId"`
	Amount      int64  `json:"amount"`
	ResponseTime int64  `json:"responseTime"`
	Message     string `json:"message"`
	ResultCode  int    `json:"resultCode"`
	PayURL      string `json:"payUrl"`
	DeepLink    string `json:"deeplink"`
	QRCodeURL   string `json:"qrCodeUrl"`
}

// momoIPNRequest represents MoMo IPN (webhook) request
type momoIPNRequest struct {
	PartnerCode  string `json:"partnerCode"`
	OrderID      string `json:"orderId"`
	RequestID    string `json:"requestId"`
	Amount       int64  `json:"amount"`
	OrderInfo    string `json:"orderInfo"`
	OrderType    string `json:"orderType"`
	TransID      int64  `json:"transId"`
	ResultCode   int    `json:"resultCode"`
	Message      string `json:"message"`
	PayType      string `json:"payType"`
	ResponseTime int64  `json:"responseTime"`
	ExtraData    string `json:"extraData"`
	Signature    string `json:"signature"`
}

// CreatePayment creates a MoMo payment
func (a *MoMoAdapter) CreatePayment(ctx context.Context, req *port.PaymentRequest) (*port.PaymentResponse, error) {
	requestID := uuid.New().String()
	amount := req.Amount.IntPart() // MoMo uses whole VND amounts

	// Build signature
	// accessKey=$accessKey&amount=$amount&extraData=$extraData&ipnUrl=$ipnUrl&orderId=$orderId&orderInfo=$orderInfo&partnerCode=$partnerCode&redirectUrl=$redirectUrl&requestId=$requestId&requestType=$requestType
	rawSignature := fmt.Sprintf(
		"accessKey=%s&amount=%d&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=%s",
		a.config.AccessKey,
		amount,
		"", // extraData
		req.CallbackURL,
		req.OrderID,
		req.Description,
		a.config.PartnerCode,
		req.ReturnURL,
		requestID,
		"payWithMethod",
	)

	signature := a.signHMAC(rawSignature)

	momoReq := momoCreateRequest{
		PartnerCode: a.config.PartnerCode,
		RequestID:   requestID,
		Amount:      amount,
		OrderID:     req.OrderID,
		OrderInfo:   req.Description,
		RedirectURL: req.ReturnURL,
		IpnURL:      req.CallbackURL,
		RequestType: "payWithMethod",
		ExtraData:   "",
		Lang:        "vi",
		Signature:   signature,
	}

	jsonData, err := json.Marshal(momoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.config.Endpoint+"/create", bytes.NewBuffer(jsonData))
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

	var momoResp momoCreateResponse
	if err := json.Unmarshal(body, &momoResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if momoResp.ResultCode != 0 {
		return nil, fmt.Errorf("MoMo error: %s (code: %d)", momoResp.Message, momoResp.ResultCode)
	}

	return &port.PaymentResponse{
		PaymentURL:   momoResp.PayURL,
		ProviderTxID: momoResp.OrderID, // MoMo uses orderId as the transaction reference
		RawData: map[string]interface{}{
			"request_id": momoResp.RequestID,
			"order_id":   momoResp.OrderID,
			"deeplink":   momoResp.DeepLink,
			"qr_code":    momoResp.QRCodeURL,
		},
	}, nil
}

// VerifyWebhook verifies the MoMo webhook signature and parses the payload
func (a *MoMoAdapter) VerifyWebhook(r *http.Request) (bool, *port.WebhookData, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read body: %w", err)
	}

	var ipn momoIPNRequest
	if err := json.Unmarshal(body, &ipn); err != nil {
		return false, nil, fmt.Errorf("failed to parse IPN: %w", err)
	}

	// Verify signature
	// accessKey=$accessKey&amount=$amount&extraData=$extraData&message=$message&orderId=$orderId&orderInfo=$orderInfo&orderType=$orderType&partnerCode=$partnerCode&payType=$payType&requestId=$requestId&responseTime=$responseTime&resultCode=$resultCode&transId=$transId
	rawSignature := fmt.Sprintf(
		"accessKey=%s&amount=%d&extraData=%s&message=%s&orderId=%s&orderInfo=%s&orderType=%s&partnerCode=%s&payType=%s&requestId=%s&responseTime=%d&resultCode=%d&transId=%d",
		a.config.AccessKey,
		ipn.Amount,
		ipn.ExtraData,
		ipn.Message,
		ipn.OrderID,
		ipn.OrderInfo,
		ipn.OrderType,
		ipn.PartnerCode,
		ipn.PayType,
		ipn.RequestID,
		ipn.ResponseTime,
		ipn.ResultCode,
		ipn.TransID,
	)

	expectedSignature := a.signHMAC(rawSignature)
	if ipn.Signature != expectedSignature {
		return false, nil, nil // Signature mismatch
	}

	// Map result code to status
	status := domain.StatusFailed
	if ipn.ResultCode == 0 {
		status = domain.StatusSuccess
	}

	return true, &port.WebhookData{
		ProviderTxID: ipn.OrderID,
		Status:       status,
		Amount:       decimal.NewFromInt(ipn.Amount),
		Currency:     domain.CurrencyVND, // MoMo only supports VND
		RawPayload:   body,
		Metadata: map[string]interface{}{
			"trans_id":    ipn.TransID,
			"request_id":  ipn.RequestID,
			"result_code": ipn.ResultCode,
			"message":     ipn.Message,
		},
	}, nil
}

// QueryStatus queries the payment status from MoMo
func (a *MoMoAdapter) QueryStatus(ctx context.Context, providerTxID string) (*port.QueryStatusResponse, error) {
	requestID := uuid.New().String()

	// Build signature for query
	rawSignature := fmt.Sprintf(
		"accessKey=%s&orderId=%s&partnerCode=%s&requestId=%s",
		a.config.AccessKey,
		providerTxID,
		a.config.PartnerCode,
		requestID,
	)

	signature := a.signHMAC(rawSignature)

	queryReq := map[string]interface{}{
		"partnerCode": a.config.PartnerCode,
		"requestId":   requestID,
		"orderId":     providerTxID,
		"lang":        "vi",
		"signature":   signature,
	}

	jsonData, err := json.Marshal(queryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.config.Endpoint+"/query", bytes.NewBuffer(jsonData))
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

	var queryResp struct {
		ResultCode int    `json:"resultCode"`
		Message    string `json:"message"`
		Amount     int64  `json:"amount"`
	}
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := domain.StatusPending
	switch queryResp.ResultCode {
	case 0:
		status = domain.StatusSuccess
	case 1006: // Transaction not found or expired
		status = domain.StatusFailed
	}

	return &port.QueryStatusResponse{
		ProviderTxID: providerTxID,
		Status:       status,
		Amount:       decimal.NewFromInt(queryResp.Amount),
		Currency:     domain.CurrencyVND,
		RawData: map[string]interface{}{
			"result_code": queryResp.ResultCode,
			"message":     queryResp.Message,
		},
	}, nil
}

// Refund initiates a refund with MoMo
func (a *MoMoAdapter) Refund(ctx context.Context, providerTxID string, amount decimal.Decimal) (string, error) {
	requestID := uuid.New().String()
	amountInt := amount.IntPart()

	// Build signature for refund
	// accessKey=$accessKey&amount=$amount&description=$description&orderId=$orderId&partnerCode=$partnerCode&requestId=$requestId
	rawSignature := fmt.Sprintf(
		"accessKey=%s&amount=%d&description=%s&orderId=%s&partnerCode=%s&requestId=%s&transId=%s",
		a.config.AccessKey,
		amountInt,
		"Refund transaction",
		providerTxID,
		a.config.PartnerCode,
		requestID,
		"0", // Original transId would be needed here in production
	)

	signature := a.signHMAC(rawSignature)

	refundReq := map[string]interface{}{
		"partnerCode": a.config.PartnerCode,
		"orderId":     providerTxID,
		"requestId":   requestID,
		"amount":      amountInt,
		"transId":     0, // Would need actual transId from original payment
		"lang":        "vi",
		"description": "Refund transaction",
		"signature":   signature,
	}

	jsonData, err := json.Marshal(refundReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.config.Endpoint+"/refund", bytes.NewBuffer(jsonData))
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

	var refundResp struct {
		ResultCode int    `json:"resultCode"`
		Message    string `json:"message"`
		TransId    int64  `json:"transId"`
	}
	if err := json.Unmarshal(body, &refundResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if refundResp.ResultCode != 0 {
		return "", fmt.Errorf("MoMo refund error: %s (code: %d)", refundResp.Message, refundResp.ResultCode)
	}

	return fmt.Sprintf("%d", refundResp.TransId), nil
}

// signHMAC creates HMAC-SHA256 signature
func (a *MoMoAdapter) signHMAC(data string) string {
	h := hmac.New(sha256.New, []byte(a.config.SecretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
