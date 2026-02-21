package service

import (
	"context"

	"github.com/google/uuid"

	"microservices/campaign/internal/model"
)

// CampaignService defines the interface for campaign business logic
type CampaignService interface {
	// CreateCampaign creates a new campaign
	CreateCampaign(ctx context.Context, req *model.CreateCampaignRequest) (*model.CampaignResponse, error)

	// GetCampaign retrieves a campaign by ID
	GetCampaign(ctx context.Context, id string) (*model.CampaignResponse, error)

	// ListCampaigns retrieves campaigns with pagination
	ListCampaigns(ctx context.Context, page, limit int) ([]*model.CampaignResponse, int64, error)

	// ListActiveCampaigns retrieves all active campaigns
	ListActiveCampaigns(ctx context.Context) ([]*model.CampaignResponse, error)
}

// VoucherService defines the interface for voucher business logic
type VoucherService interface {
	// CreateVoucher creates a new voucher
	CreateVoucher(ctx context.Context, req *model.CreateVoucherRequest) (*model.VoucherResponse, error)

	// GetVoucher retrieves a voucher by ID
	GetVoucher(ctx context.Context, id string) (*model.VoucherResponse, error)

	// GetVoucherByCode retrieves a voucher by code
	GetVoucherByCode(ctx context.Context, code string) (*model.VoucherResponse, error)

	// ClaimVoucher claims a voucher for a user
	ClaimVoucher(ctx context.Context, userID uuid.UUID, code string) error

	// CalculateCart applies voucher rules to cart items and returns discount calculation
	CalculateCart(ctx context.Context, items []model.CartItem, voucherCode string) (*model.CalculateCartResult, error)

	// InitializeVoucherStock initializes voucher stock in Redis cache
	InitializeVoucherStock(ctx context.Context, code string) error

	// GetUserVouchers retrieves all vouchers claimed by a user
	GetUserVouchers(ctx context.Context, userID uuid.UUID) ([]*model.UserVoucherResponse, error)
}
