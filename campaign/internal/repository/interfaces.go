package repository

import (
	"context"

	"github.com/google/uuid"

	"microservices/campaign/internal/model"
)

// CampaignRepository defines the interface for campaign persistence
type CampaignRepository interface {
	Create(ctx context.Context, campaign *model.Campaign) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Campaign, error)
	Update(ctx context.Context, campaign *model.Campaign) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListActive(ctx context.Context) ([]*model.Campaign, error)
	List(ctx context.Context, offset, limit int) ([]*model.Campaign, int64, error)
}

// VoucherRepository defines the interface for voucher persistence
type VoucherRepository interface {
	Create(ctx context.Context, voucher *model.Voucher) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Voucher, error)
	GetByCode(ctx context.Context, code string) (*model.Voucher, error)
	Update(ctx context.Context, voucher *model.Voucher) error
	IncrementUsedCount(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*model.Voucher, error)
}

// UserVoucherRepository defines the interface for user-voucher relationship persistence
type UserVoucherRepository interface {
	Create(ctx context.Context, userVoucher *model.UserVoucher) error
	GetByUserAndVoucher(ctx context.Context, userID, voucherID uuid.UUID) (*model.UserVoucher, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.UserVoucher, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, userID, voucherID uuid.UUID) (bool, error)
}

// VoucherCacheRepository defines the interface for Redis caching operations
type VoucherCacheRepository interface {
	// InitializeStock sets up the voucher stock in cache
	InitializeStock(ctx context.Context, code string, stock int) error
	// AtomicClaim attempts to claim a voucher atomically using Lua script
	// Returns nil if successful, error if already claimed or out of stock
	AtomicClaim(ctx context.Context, code string, userID uuid.UUID) error
	// GetStock returns current stock from cache
	GetStock(ctx context.Context, code string) (int, error)
	// HasUserClaimed checks if user has already claimed
	HasUserClaimed(ctx context.Context, code string, userID uuid.UUID) (bool, error)
	// InvalidateVoucher removes voucher data from cache
	InvalidateVoucher(ctx context.Context, code string) error
}
