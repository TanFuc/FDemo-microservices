package usecase

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"microservices/wallet/internal/domain"
	"microservices/wallet/internal/port"
)

// GetWalletUseCase handles wallet retrieval
type GetWalletUseCase struct {
	repo   port.WalletRepository
	logger *slog.Logger
}

// NewGetWalletUseCase creates a new get wallet use case
func NewGetWalletUseCase(repo port.WalletRepository, logger *slog.Logger) *GetWalletUseCase {
	return &GetWalletUseCase{
		repo:   repo,
		logger: logger,
	}
}

// WalletResponse represents the wallet response
type WalletResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Balance   string    `json:"balance"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

// TransactionResponse represents a transaction in the response
type TransactionResponse struct {
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	Amount        string    `json:"amount"`
	BalanceBefore string    `json:"balance_before"`
	BalanceAfter  string    `json:"balance_after"`
	Currency      string    `json:"currency"`
	ReferenceID   string    `json:"reference_id,omitempty"`
	ReferenceType string    `json:"reference_type,omitempty"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     string    `json:"created_at"`
}

// HistoryResponse represents the transaction history response
type HistoryResponse struct {
	Transactions []*TransactionResponse `json:"transactions"`
	Total        int64                  `json:"total"`
	Limit        int                    `json:"limit"`
	Offset       int                    `json:"offset"`
}

// GetByUserID retrieves a wallet by user ID
func (uc *GetWalletUseCase) GetByUserID(ctx context.Context, userID uuid.UUID) (*WalletResponse, error) {
	wallet, err := uc.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &WalletResponse{
		ID:        wallet.ID,
		UserID:    wallet.UserID,
		Balance:   wallet.Balance.String(),
		Currency:  wallet.Currency,
		Status:    string(wallet.Status),
		CreatedAt: wallet.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: wallet.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// GetHistory retrieves transaction history for a wallet
func (uc *GetWalletUseCase) GetHistory(ctx context.Context, userID uuid.UUID, limit, offset int) (*HistoryResponse, error) {
	// Get wallet first to validate ownership
	wallet, err := uc.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply defaults
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Get transactions
	transactions, total, err := uc.repo.ListTransactions(ctx, wallet.ID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert to response
	txResponses := make([]*TransactionResponse, len(transactions))
	for i, tx := range transactions {
		txResponses[i] = toTransactionResponse(tx)
	}

	return &HistoryResponse{
		Transactions: txResponses,
		Total:        total,
		Limit:        limit,
		Offset:       offset,
	}, nil
}

func toTransactionResponse(tx *domain.WalletTransaction) *TransactionResponse {
	return &TransactionResponse{
		ID:            tx.ID,
		Type:          string(tx.Type),
		Status:        string(tx.Status),
		Amount:        tx.Amount.String(),
		BalanceBefore: tx.BalanceBefore.String(),
		BalanceAfter:  tx.BalanceAfter.String(),
		Currency:      tx.Currency,
		ReferenceID:   tx.ReferenceID,
		ReferenceType: tx.ReferenceType,
		Description:   tx.Description,
		CreatedAt:     tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
