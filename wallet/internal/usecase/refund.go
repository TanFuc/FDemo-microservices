package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"microservices/wallet/internal/domain"
	"microservices/wallet/internal/port"
)

// RefundUseCase handles wallet refund operations
type RefundUseCase struct {
	repo      port.WalletRepository
	publisher port.EventPublisher
	logger    *slog.Logger
}

// NewRefundUseCase creates a new refund use case
func NewRefundUseCase(repo port.WalletRepository, publisher port.EventPublisher, logger *slog.Logger) *RefundUseCase {
	return &RefundUseCase{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

// WalletRefundRequest represents the request to refund to wallet
type WalletRefundRequest struct {
	UserID         uuid.UUID
	OrderID        uuid.UUID
	Amount         decimal.Decimal
	Reason         string
	IdempotencyKey string
}

// WalletRefundResponse represents the response from wallet refund
type WalletRefundResponse struct {
	WalletTxID    uuid.UUID       `json:"wallet_tx_id"`
	BalanceBefore decimal.Decimal `json:"balance_before"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
}

// Execute processes a wallet refund
func (uc *RefundUseCase) Execute(ctx context.Context, req *WalletRefundRequest) (*WalletRefundResponse, error) {
	// Build idempotency key
	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("refund:%s", req.OrderID.String())
	}

	// Check idempotency - if COMPLETED transaction exists, return it
	existingTx, err := uc.repo.GetTransactionByIdempotencyKey(ctx, idempotencyKey)
	if err == nil && existingTx != nil && existingTx.Status == domain.TxStatusCompleted {
		return &WalletRefundResponse{
			WalletTxID:    existingTx.ID,
			BalanceBefore: existingTx.BalanceBefore,
			BalanceAfter:  existingTx.BalanceAfter,
		}, nil
	}

	// Get wallet
	wallet, err := uc.repo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	// Credit wallet
	balanceBefore, balanceAfter, err := wallet.Credit(req.Amount)
	if err != nil {
		return nil, err
	}

	// Create wallet transaction
	walletTx := domain.NewWalletTransaction(
		wallet.ID, wallet.UserID,
		domain.TxTypeRefund,
		req.Amount, balanceBefore, balanceAfter,
		wallet.Currency,
	)
	walletTx.SetIdempotencyKey(idempotencyKey)
	walletTx.SetReference(req.OrderID.String(), "order")
	walletTx.SetDescription(req.Reason)
	walletTx.MarkCompleted()

	// Create outbox event
	event := domain.NewWalletEvent(domain.SubjectWalletRefunded, walletTx)
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}
	outboxEvent := domain.NewOutboxEvent(wallet.ID.String(), domain.SubjectWalletRefunded, eventPayload)

	// Execute in transaction with retry on version conflict
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = uc.repo.RunInTransaction(ctx, func(tx port.TxContext) error {
			// Update balance with optimistic locking
			if err := tx.UpdateBalanceOptimistic(ctx, wallet.ID, balanceAfter, wallet.Version, wallet.Version-1); err != nil {
				return err
			}

			// Create transaction
			if err := tx.CreateTransaction(ctx, walletTx); err != nil {
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

		if errors.Is(err, domain.ErrDuplicateTx) {
			// Idempotent - return existing
			existingTx, _ := uc.repo.GetTransactionByIdempotencyKey(ctx, idempotencyKey)
			if existingTx != nil {
				return &WalletRefundResponse{
					WalletTxID:    existingTx.ID,
					BalanceBefore: existingTx.BalanceBefore,
					BalanceAfter:  existingTx.BalanceAfter,
				}, nil
			}
		}

		if !errors.Is(err, domain.ErrVersionConflict) {
			return nil, fmt.Errorf("run transaction: %w", err)
		}

		// Refresh wallet for retry
		wallet, err = uc.repo.GetByUserID(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("refresh wallet: %w", err)
		}
		balanceBefore, balanceAfter, err = wallet.Credit(req.Amount)
		if err != nil {
			return nil, err
		}

		// Update transaction balances
		walletTx.BalanceBefore = balanceBefore
		walletTx.BalanceAfter = balanceAfter

		time.Sleep(time.Duration(attempt*10) * time.Millisecond)
	}

	// Best effort: attempt direct NATS publish
	if uc.publisher != nil {
		if pubErr := uc.publisher.Publish(ctx, domain.SubjectWalletRefunded, event); pubErr != nil {
			uc.logger.Warn("failed to publish refund event directly (outbox will retry)", "error", pubErr)
		}
	}

	uc.logger.Info("wallet refund completed",
		"wallet_id", wallet.ID,
		"tx_id", walletTx.ID,
		"order_id", req.OrderID,
		"amount", req.Amount,
	)

	return &WalletRefundResponse{
		WalletTxID:    walletTx.ID,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
	}, nil
}
