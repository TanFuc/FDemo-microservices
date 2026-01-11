package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tafu/payment-service/internal/adapter/repository"
	"github.com/tafu/payment-service/internal/domain"
	"github.com/tafu/payment-service/internal/port"
)

var (
	ErrInvalidProvider    = errors.New("invalid payment provider")
	ErrInvalidAmount      = errors.New("amount must be positive")
	ErrInvalidCurrency    = errors.New("invalid currency")
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrWebhookVerification = errors.New("webhook verification failed")
)

// PaymentUseCase handles payment business logic
type PaymentUseCase struct {
	repo      port.PaymentRepository
	publisher port.EventPublisher
	gateways  map[domain.Provider]port.PaymentGateway
	logger    *slog.Logger
}

// NewPaymentUseCase creates a new payment use case
func NewPaymentUseCase(
	repo port.PaymentRepository,
	publisher port.EventPublisher,
	gateways []port.PaymentGateway,
	logger *slog.Logger,
) *PaymentUseCase {
	gwMap := make(map[domain.Provider]port.PaymentGateway)
	for _, gw := range gateways {
		gwMap[gw.Provider()] = gw
	}

	return &PaymentUseCase{
		repo:      repo,
		publisher: publisher,
		gateways:  gwMap,
		logger:    logger,
	}
}

// InitiatePaymentRequest represents a request to initiate a payment
type InitiatePaymentRequest struct {
	OrderID     uuid.UUID
	UserID      uuid.UUID
	Amount      decimal.Decimal
	Currency    domain.Currency
	Provider    domain.Provider
	Description string
	CallbackURL string
	ReturnURL   string
	Metadata    map[string]string
}

// InitiatePaymentResponse represents the response from initiating a payment
type InitiatePaymentResponse struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	PaymentURL    string    `json:"payment_url"`
	ProviderTxID  string    `json:"provider_tx_id"`
}

// InitiatePayment creates a new payment with the specified gateway
func (uc *PaymentUseCase) InitiatePayment(ctx context.Context, req *InitiatePaymentRequest) (*InitiatePaymentResponse, error) {
	// Validate request
	if !req.Provider.IsValid() {
		return nil, ErrInvalidProvider
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidAmount
	}
	if !req.Currency.IsValid() {
		return nil, ErrInvalidCurrency
	}

	// Get the gateway
	gateway, ok := uc.gateways[req.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, req.Provider)
	}

	// Create transaction record (PENDING)
	tx := domain.NewPaymentTransaction(
		req.OrderID,
		req.UserID,
		req.Amount,
		req.Currency,
		req.Provider,
	)

	// Save initial transaction
	if err := uc.repo.Create(ctx, tx); err != nil {
		uc.logger.Error("failed to create transaction",
			"error", err,
			"order_id", req.OrderID,
		)
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Call gateway to create payment
	gwReq := &port.PaymentRequest{
		OrderID:     req.OrderID.String(),
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		CallbackURL: req.CallbackURL,
		ReturnURL:   req.ReturnURL,
		Metadata:    req.Metadata,
	}

	gwResp, err := gateway.CreatePayment(ctx, gwReq)
	if err != nil {
		uc.logger.Error("failed to create payment with gateway",
			"error", err,
			"provider", req.Provider,
			"order_id", req.OrderID,
		)
		// Update transaction status to FAILED
		tx.UpdateStatus(domain.StatusFailed)
		_ = tx.SetMetadata(map[string]interface{}{"error": err.Error()})
		_ = uc.repo.Update(ctx, tx)
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Update transaction with provider TX ID
	tx.SetProviderTxID(gwResp.ProviderTxID)
	if err := tx.SetMetadata(map[string]interface{}{
		"gateway_response": gwResp.RawData,
	}); err != nil {
		uc.logger.Warn("failed to set metadata", "error", err)
	}

	if err := uc.repo.Update(ctx, tx); err != nil {
		uc.logger.Error("failed to update transaction",
			"error", err,
			"transaction_id", tx.ID,
		)
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	uc.logger.Info("payment initiated",
		"transaction_id", tx.ID,
		"provider", req.Provider,
		"provider_tx_id", gwResp.ProviderTxID,
		"order_id", req.OrderID,
	)

	return &InitiatePaymentResponse{
		TransactionID: tx.ID,
		PaymentURL:    gwResp.PaymentURL,
		ProviderTxID:  gwResp.ProviderTxID,
	}, nil
}

// HandleWebhookResult represents the result of handling a webhook
type HandleWebhookResult struct {
	Processed     bool
	TransactionID uuid.UUID
	Status        domain.Status
	IsIdempotent  bool
}

// HandleWebhook processes a webhook from a payment provider
func (uc *PaymentUseCase) HandleWebhook(ctx context.Context, provider domain.Provider, r *http.Request, ipAddress string) (*HandleWebhookResult, error) {
	// Get the gateway
	gateway, ok := uc.gateways[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, provider)
	}

	// CRITICAL: Verify webhook signature FIRST
	valid, webhookData, err := gateway.VerifyWebhook(r)
	if err != nil {
		uc.logger.Error("webhook verification error",
			"error", err,
			"provider", provider,
		)
		return nil, fmt.Errorf("webhook verification error: %w", err)
	}

	if !valid {
		uc.logger.Warn("webhook signature verification failed",
			"provider", provider,
			"ip_address", ipAddress,
		)
		return nil, ErrWebhookVerification
	}

	if webhookData == nil {
		// Valid webhook but not a type we handle
		uc.logger.Debug("webhook event type not handled", "provider", provider)
		return &HandleWebhookResult{Processed: false}, nil
	}

	// Log the raw webhook
	logEntry := domain.NewPaymentLog(provider, "webhook", webhookData.RawPayload, ipAddress)
	if err := uc.repo.CreateLog(ctx, logEntry); err != nil {
		uc.logger.Error("failed to create webhook log",
			"error", err,
			"provider", provider,
		)
		// Don't fail the webhook processing due to logging failure
	}

	// Find the transaction
	tx, err := uc.repo.GetByProviderTxID(ctx, provider, webhookData.ProviderTxID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			uc.logger.Warn("transaction not found for webhook",
				"provider", provider,
				"provider_tx_id", webhookData.ProviderTxID,
			)
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Update log with transaction ID
	logEntry.SetTransactionID(tx.ID)
	_ = uc.repo.CreateLog(ctx, logEntry) // Best effort update

	// IDEMPOTENCY CHECK: If already in a final state, do nothing
	if tx.Status.IsFinal() {
		uc.logger.Info("idempotent webhook - transaction already final",
			"transaction_id", tx.ID,
			"current_status", tx.Status,
			"webhook_status", webhookData.Status,
		)
		return &HandleWebhookResult{
			Processed:     true,
			TransactionID: tx.ID,
			Status:        tx.Status,
			IsIdempotent:  true,
		}, nil
	}

	// Update transaction status
	previousStatus := tx.Status
	tx.UpdateStatus(webhookData.Status)

	// Add webhook data to metadata
	currentMeta, _ := tx.GetMetadata()
	if currentMeta == nil {
		currentMeta = make(map[string]interface{})
	}
	currentMeta["last_webhook"] = map[string]interface{}{
		"status":     webhookData.Status,
		"timestamp":  time.Now().UTC(),
		"ip_address": ipAddress,
	}
	_ = tx.SetMetadata(currentMeta)

	if err := uc.repo.Update(ctx, tx); err != nil {
		uc.logger.Error("failed to update transaction from webhook",
			"error", err,
			"transaction_id", tx.ID,
		)
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	// Publish event to NATS
	event := domain.NewPaymentEvent(tx)
	subject := port.SubjectPaymentProcessed
	if webhookData.Status == domain.StatusFailed {
		subject = port.SubjectPaymentFailed
	} else if webhookData.Status == domain.StatusRefunded {
		subject = port.SubjectPaymentRefunded
	}

	if err := uc.publisher.Publish(ctx, subject, event); err != nil {
		uc.logger.Error("failed to publish payment event",
			"error", err,
			"transaction_id", tx.ID,
			"subject", subject,
		)
		// Don't fail - the reconciliation job will catch this
	} else {
		uc.logger.Info("payment event published",
			"transaction_id", tx.ID,
			"subject", subject,
			"previous_status", previousStatus,
			"new_status", tx.Status,
		)
	}

	return &HandleWebhookResult{
		Processed:     true,
		TransactionID: tx.ID,
		Status:        tx.Status,
		IsIdempotent:  false,
	}, nil
}

// GetTransaction retrieves a transaction by ID
func (uc *PaymentUseCase) GetTransaction(ctx context.Context, id uuid.UUID) (*domain.PaymentTransaction, error) {
	return uc.repo.GetByID(ctx, id)
}

// GetTransactionsByOrderID retrieves transactions for an order
func (uc *PaymentUseCase) GetTransactionsByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domain.PaymentTransaction, error) {
	return uc.repo.GetByOrderID(ctx, orderID)
}

// ReconcilePendingPayments reconciles pending payments with their gateways
func (uc *PaymentUseCase) ReconcilePendingPayments(ctx context.Context, olderThan time.Duration) (int, int, error) {
	cutoff := time.Now().Add(-olderThan)

	pending, err := uc.repo.GetPendingTransactions(ctx, cutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get pending transactions: %w", err)
	}

	var synced, failed int

	for _, tx := range pending {
		gateway, ok := uc.gateways[tx.Provider]
		if !ok {
			uc.logger.Warn("no gateway for provider",
				"provider", tx.Provider,
				"transaction_id", tx.ID,
			)
			continue
		}

		result, err := gateway.QueryStatus(ctx, tx.ProviderTxID)
		if err != nil {
			uc.logger.Error("failed to query gateway status",
				"error", err,
				"provider", tx.Provider,
				"transaction_id", tx.ID,
			)
			failed++
			continue
		}

		if result.Status == tx.Status {
			// No change
			continue
		}

		// Update transaction
		previousStatus := tx.Status
		tx.UpdateStatus(result.Status)

		meta, _ := tx.GetMetadata()
		if meta == nil {
			meta = make(map[string]interface{})
		}
		meta["reconciliation"] = map[string]interface{}{
			"timestamp":       time.Now().UTC(),
			"previous_status": previousStatus,
			"gateway_data":    result.RawData,
		}
		_ = tx.SetMetadata(meta)

		if err := uc.repo.Update(ctx, tx); err != nil {
			uc.logger.Error("failed to update transaction during reconciliation",
				"error", err,
				"transaction_id", tx.ID,
			)
			failed++
			continue
		}

		// Publish event
		event := domain.NewPaymentEvent(tx)
		subject := port.SubjectPaymentProcessed
		if result.Status == domain.StatusFailed {
			subject = port.SubjectPaymentFailed
		}

		if err := uc.publisher.Publish(ctx, subject, event); err != nil {
			uc.logger.Error("failed to publish reconciliation event",
				"error", err,
				"transaction_id", tx.ID,
			)
		}

		uc.logger.Info("transaction reconciled",
			"transaction_id", tx.ID,
			"previous_status", previousStatus,
			"new_status", tx.Status,
		)

		synced++
	}

	return synced, failed, nil
}

// logWebhookPayload logs the raw webhook payload for debugging
func (uc *PaymentUseCase) logWebhookPayload(provider domain.Provider, payload []byte, ipAddress string) {
	log := domain.NewPaymentLog(provider, "webhook_raw", json.RawMessage(payload), ipAddress)
	if err := uc.repo.CreateLog(context.Background(), log); err != nil {
		uc.logger.Error("failed to log webhook payload", "error", err)
	}
}
