package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"microservices/wallet/internal/domain"
	"microservices/wallet/internal/port"
)

// TopUpUseCase handles wallet top-up operations
type TopUpUseCase struct {
	repo              port.WalletRepository
	publisher         port.EventPublisher
	paymentServiceURL string
	minTopUp          decimal.Decimal
	maxTopUp          decimal.Decimal
	maxBalance        decimal.Decimal
	logger            *slog.Logger
	httpClient        *http.Client
}

// TopUpConfig holds top-up configuration
type TopUpConfig struct {
	PaymentServiceURL string
	MinTopUpVND       int64
	MaxTopUpVND       int64
	MaxBalanceVND     int64
}

// NewTopUpUseCase creates a new top-up use case
func NewTopUpUseCase(repo port.WalletRepository, publisher port.EventPublisher, cfg TopUpConfig, logger *slog.Logger) *TopUpUseCase {
	minTopUp := decimal.NewFromInt(cfg.MinTopUpVND)
	if cfg.MinTopUpVND == 0 {
		minTopUp = decimal.NewFromInt(10000)
	}
	maxTopUp := decimal.NewFromInt(cfg.MaxTopUpVND)
	if cfg.MaxTopUpVND == 0 {
		maxTopUp = decimal.NewFromInt(50000000)
	}
	maxBalance := decimal.NewFromInt(cfg.MaxBalanceVND)
	if cfg.MaxBalanceVND == 0 {
		maxBalance = decimal.NewFromInt(200000000)
	}

	return &TopUpUseCase{
		repo:              repo,
		publisher:         publisher,
		paymentServiceURL: cfg.PaymentServiceURL,
		minTopUp:          minTopUp,
		maxTopUp:          maxTopUp,
		maxBalance:        maxBalance,
		logger:            logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// InitiateTopUpRequest represents the request to initiate a top-up
type InitiateTopUpRequest struct {
	UserID         uuid.UUID
	Amount         decimal.Decimal
	Provider       string // "VNPAY", "ZALOPAY", "MOMO", "STRIPE"
	Description    string
	IdempotencyKey string // from X-Idempotency-Key header
	ReturnURL      string
}

// InitiateTopUpResponse represents the response from initiating a top-up
type InitiateTopUpResponse struct {
	WalletTxID  uuid.UUID `json:"wallet_tx_id"`
	PaymentURL  string    `json:"payment_url"`
	PaymentTxID string    `json:"payment_tx_id"`
	Amount      string    `json:"amount"`
}

// InitiateTopUp creates a PENDING wallet transaction and calls the Payment Service
func (uc *TopUpUseCase) InitiateTopUp(ctx context.Context, req *InitiateTopUpRequest) (*InitiateTopUpResponse, error) {
	// Validate amount
	if req.Amount.LessThan(uc.minTopUp) {
		return nil, domain.ErrAmountTooSmall
	}
	if req.Amount.GreaterThan(uc.maxTopUp) {
		return nil, domain.ErrAmountTooLarge
	}

	// Check idempotency key - if PENDING wallet tx exists, return its linked info
	if req.IdempotencyKey != "" {
		existingTx, err := uc.repo.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil && existingTx != nil {
			// Return existing transaction info
			return &InitiateTopUpResponse{
				WalletTxID: existingTx.ID,
				Amount:     existingTx.Amount.String(),
			}, nil
		}
	}

	// Get wallet
	wallet, err := uc.repo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	// Check if balance would exceed limit
	if wallet.Balance.Add(req.Amount).GreaterThan(uc.maxBalance) {
		return nil, domain.ErrBalanceExceedsLimit
	}

	// Create wallet transaction with PENDING status
	walletTx := domain.NewWalletTransaction(
		wallet.ID, wallet.UserID,
		domain.TxTypeTopUp,
		req.Amount, wallet.Balance, wallet.Balance, // balance unchanged until completed
		wallet.Currency,
	)
	walletTx.SetIdempotencyKey(req.IdempotencyKey)
	walletTx.SetDescription(req.Description)

	// Call Payment Service to create payment
	paymentResp, err := uc.createPayment(ctx, walletTx.ID, req)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	// Link payment transaction ID
	paymentTxID, _ := uuid.Parse(paymentResp.TransactionID)
	walletTx.SetPaymentTxID(paymentTxID)
	walletTx.SetReference(paymentResp.TransactionID, "payment")

	// Save wallet transaction
	if err := uc.repo.CreateTransaction(ctx, walletTx); err != nil {
		if errors.Is(err, domain.ErrDuplicateTx) {
			// Idempotent - return existing
			existingTx, _ := uc.repo.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
			if existingTx != nil {
				return &InitiateTopUpResponse{
					WalletTxID: existingTx.ID,
					Amount:     existingTx.Amount.String(),
				}, nil
			}
		}
		return nil, fmt.Errorf("create wallet transaction: %w", err)
	}

	return &InitiateTopUpResponse{
		WalletTxID:  walletTx.ID,
		PaymentURL:  paymentResp.PaymentURL,
		PaymentTxID: paymentResp.TransactionID,
		Amount:      req.Amount.String(),
	}, nil
}

// CompleteTopUp is called when payment.processed event is received
func (uc *TopUpUseCase) CompleteTopUp(ctx context.Context, paymentTxID uuid.UUID, amount decimal.Decimal) error {
	// Find the PENDING wallet transaction
	walletTx, err := uc.repo.GetTransactionByPaymentTxID(ctx, paymentTxID)
	if err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return domain.ErrPaymentTxNotFound
		}
		return fmt.Errorf("get transaction by payment_tx_id: %w", err)
	}

	// Idempotent: if already COMPLETED, skip
	if walletTx.Status == domain.TxStatusCompleted {
		uc.logger.Info("top-up already completed (idempotent)", "tx_id", walletTx.ID)
		return nil
	}

	// Get wallet
	wallet, err := uc.repo.GetByID(ctx, walletTx.WalletID)
	if err != nil {
		return fmt.Errorf("get wallet: %w", err)
	}

	// Credit wallet
	balanceBefore, balanceAfter, err := wallet.Credit(amount)
	if err != nil {
		return fmt.Errorf("credit wallet: %w", err)
	}

	// Update wallet transaction balances
	walletTx.BalanceBefore = balanceBefore
	walletTx.BalanceAfter = balanceAfter

	// Create outbox event
	event := domain.NewWalletEvent(domain.SubjectWalletToppedUp, walletTx)
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	outboxEvent := domain.NewOutboxEvent(wallet.ID.String(), domain.SubjectWalletToppedUp, eventPayload)

	// Execute in transaction with retry on version conflict
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = uc.repo.RunInTransaction(ctx, func(tx port.TxContext) error {
			// Update balance with optimistic locking
			if err := tx.UpdateBalanceOptimistic(ctx, wallet.ID, balanceAfter, wallet.Version, wallet.Version-1); err != nil {
				return err
			}

			// Update transaction status
			if err := tx.UpdateTransactionStatus(ctx, walletTx.ID, domain.TxStatusCompleted); err != nil {
				return err
			}

			// Create outbox event
			if err := tx.CreateOutboxEvent(ctx, outboxEvent); err != nil {
				return err
			}

			return nil
		})

		if err == nil {
			break
		}

		if !errors.Is(err, domain.ErrVersionConflict) {
			return fmt.Errorf("run transaction: %w", err)
		}

		// Refresh wallet for retry
		wallet, err = uc.repo.GetByID(ctx, walletTx.WalletID)
		if err != nil {
			return fmt.Errorf("refresh wallet: %w", err)
		}
		balanceBefore, balanceAfter, err = wallet.Credit(amount)
		if err != nil {
			return fmt.Errorf("credit wallet (retry): %w", err)
		}

		time.Sleep(time.Duration(attempt*10) * time.Millisecond)
	}

	// Best effort: attempt direct NATS publish
	if uc.publisher != nil {
		if pubErr := uc.publisher.Publish(ctx, domain.SubjectWalletToppedUp, event); pubErr != nil {
			uc.logger.Warn("failed to publish top-up event directly (outbox will retry)", "error", pubErr)
		}
	}

	uc.logger.Info("top-up completed", "wallet_id", wallet.ID, "tx_id", walletTx.ID, "amount", amount)
	return nil
}

// Execute implements the CompleteTopUpUseCase interface for payment_listener
func (uc *TopUpUseCase) Execute(ctx context.Context, paymentTxID uuid.UUID, amount decimal.Decimal) error {
	return uc.CompleteTopUp(ctx, paymentTxID, amount)
}

// PaymentServiceRequest represents the request to the payment service
type PaymentServiceRequest struct {
	OrderID       string `json:"order_id"`
	UserID        string `json:"user_id"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Provider      string `json:"provider"`
	Description   string `json:"description"`
	ReturnURL     string `json:"return_url"`
	IsWalletTopup bool   `json:"is_wallet_topup"`
	WalletTxID    string `json:"wallet_tx_id"`
}

// PaymentServiceResponse represents the response from the payment service
type PaymentServiceResponse struct {
	TransactionID string `json:"transaction_id"`
	PaymentURL    string `json:"payment_url"`
	ProviderTxID  string `json:"provider_tx_id"`
}

func (uc *TopUpUseCase) createPayment(ctx context.Context, walletTxID uuid.UUID, req *InitiateTopUpRequest) (*PaymentServiceResponse, error) {
	paymentReq := PaymentServiceRequest{
		OrderID:       walletTxID.String(),
		UserID:        req.UserID.String(),
		Amount:        req.Amount.String(),
		Currency:      "VND",
		Provider:      req.Provider,
		Description:   req.Description,
		ReturnURL:     req.ReturnURL,
		IsWalletTopup: true,
		WalletTxID:    walletTxID.String(),
	}

	body, err := json.Marshal(paymentReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/payments", uc.paymentServiceURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := uc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	var result struct {
		Success bool                   `json:"success"`
		Data    PaymentServiceResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result.Data, nil
}
