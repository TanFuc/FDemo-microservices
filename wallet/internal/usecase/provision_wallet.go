package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"microservices/wallet/internal/domain"
	"microservices/wallet/internal/port"
)

// ProvisionWalletUseCase handles wallet provisioning for new users
type ProvisionWalletUseCase struct {
	repo   port.WalletRepository
	logger *slog.Logger
}

// NewProvisionWalletUseCase creates a new provision wallet use case
func NewProvisionWalletUseCase(repo port.WalletRepository, logger *slog.Logger) *ProvisionWalletUseCase {
	return &ProvisionWalletUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute provisions a wallet for a user. It is idempotent: if wallet already exists, returns nil.
func (uc *ProvisionWalletUseCase) Execute(ctx context.Context, userID uuid.UUID, email string) error {
	// Check if wallet already exists
	exists, err := uc.repo.ExistsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("check wallet exists: %w", err)
	}
	if exists {
		uc.logger.Info("wallet already provisioned (idempotent)", "user_id", userID)
		return nil
	}

	// Create new wallet
	wallet := domain.NewWallet(userID)
	if err := uc.repo.CreateWallet(ctx, wallet); err != nil {
		// Handle race condition: another instance might have created it
		if errors.Is(err, domain.ErrWalletAlreadyExists) {
			uc.logger.Info("wallet already provisioned (race condition)", "user_id", userID)
			return nil
		}
		return fmt.Errorf("create wallet: %w", err)
	}

	uc.logger.Info("wallet provisioned", "user_id", userID, "wallet_id", wallet.ID)
	return nil
}
