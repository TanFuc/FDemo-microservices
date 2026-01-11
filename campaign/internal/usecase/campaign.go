package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tafu/campaign-service/internal/domain"
)

type CampaignUsecase struct {
	voucherRepo      domain.VoucherRepository
	userVoucherRepo  domain.UserVoucherRepository
	voucherCache     domain.VoucherCacheRepository
}

func NewCampaignUsecase(
	voucherRepo domain.VoucherRepository,
	userVoucherRepo domain.UserVoucherRepository,
	voucherCache domain.VoucherCacheRepository,
) *CampaignUsecase {
	return &CampaignUsecase{
		voucherRepo:     voucherRepo,
		userVoucherRepo: userVoucherRepo,
		voucherCache:    voucherCache,
	}
}

// ClaimVoucher handles atomic voucher claiming using Redis + Postgres
// Step 1: Use Redis Lua script for atomic claim (source of truth for stock)
// Step 2: Persist to Postgres asynchronously for durability
func (uc *CampaignUsecase) ClaimVoucher(ctx context.Context, userID uuid.UUID, code string) error {
	// Validate voucher exists and is active
	voucher, err := uc.voucherRepo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if voucher.Status != domain.VoucherStatusActive {
		return domain.ErrVoucherInactive
	}

	// Step 1: Atomic claim via Redis Lua script
	if err := uc.voucherCache.AtomicClaim(ctx, code, userID); err != nil {
		return err
	}

	// Step 2: Persist to Postgres (Redis is source of truth, this is for durability)
	userVoucher := domain.NewUserVoucher(userID, voucher.ID)
	if err := uc.userVoucherRepo.Create(ctx, userVoucher); err != nil {
		// Log error but don't fail - Redis is source of truth
		// In production, use message queue for retry
		fmt.Printf("warning: failed to persist user voucher to postgres: %v\n", err)
	}

	// Update used count in Postgres (for statistics only)
	if err := uc.voucherRepo.IncrementUsedCount(ctx, voucher.ID); err != nil {
		fmt.Printf("warning: failed to increment used count: %v\n", err)
	}

	return nil
}

// CalculateCart applies voucher rules to cart items and returns discount calculation
// This simulates the discount but DOES NOT save it - actual usage happens in Order Service
func (uc *CampaignUsecase) CalculateCart(ctx context.Context, items []domain.CartItem, voucherCode string) (*domain.CalculateCartResult, error) {
	// Calculate original total
	originalTotal := decimal.Zero
	for _, item := range items {
		originalTotal = originalTotal.Add(item.TotalPrice())
	}

	// If no voucher code provided, return without discount
	if voucherCode == "" {
		return domain.NewCalculateCartResultNoDiscount(originalTotal), nil
	}

	// Fetch voucher
	voucher, err := uc.voucherRepo.GetByCode(ctx, voucherCode)
	if err != nil {
		return domain.NewCalculateCartResultError(originalTotal, err.Error()), err
	}

	if voucher.Status != domain.VoucherStatusActive {
		return domain.NewCalculateCartResultError(originalTotal, domain.ErrVoucherInactive.Error()), domain.ErrVoucherInactive
	}

	// Apply Rule Engine validation
	if err := uc.validateVoucherConditions(voucher, items, originalTotal); err != nil {
		return domain.NewCalculateCartResultError(originalTotal, err.Error()), err
	}

	// Calculate discount
	discount := uc.calculateDiscount(voucher, originalTotal)

	return domain.NewCalculateCartResult(originalTotal, discount, voucherCode), nil
}

// validateVoucherConditions applies the rule engine to check voucher eligibility
func (uc *CampaignUsecase) validateVoucherConditions(voucher *domain.Voucher, items []domain.CartItem, total decimal.Decimal) error {
	conditions := voucher.Conditions

	// Rule 1: Check minimum order value
	if !conditions.MinOrderValue.IsZero() && total.LessThan(conditions.MinOrderValue) {
		return domain.ErrMinOrderNotMet
	}

	// Rule 2: Check allowed categories (if specified)
	if len(conditions.AllowedCategories) > 0 {
		hasAllowedCategory := false
		for _, item := range items {
			for _, allowedCat := range conditions.AllowedCategories {
				if item.Category == allowedCat {
					hasAllowedCategory = true
					break
				}
			}
			if hasAllowedCategory {
				break
			}
		}
		if !hasAllowedCategory {
			return domain.ErrCategoryNotAllowed
		}
	}

	// Rule 3: Check excluded products
	if len(conditions.ExcludedProducts) > 0 {
		for _, item := range items {
			for _, excludedSKU := range conditions.ExcludedProducts {
				if item.SKU == excludedSKU {
					return domain.ErrProductExcluded
				}
			}
		}
	}

	return nil
}

// calculateDiscount computes the discount amount based on voucher type and conditions
func (uc *CampaignUsecase) calculateDiscount(voucher *domain.Voucher, total decimal.Decimal) decimal.Decimal {
	var discount decimal.Decimal

	switch voucher.Type {
	case domain.VoucherTypePercentage:
		// Discount = Total * Value / 100
		discount = total.Mul(voucher.Value).Div(decimal.NewFromInt(100))
	case domain.VoucherTypeFixedAmount:
		discount = voucher.Value
	default:
		return decimal.Zero
	}

	// Apply max discount cap if specified
	if !voucher.Conditions.MaxDiscount.IsZero() && discount.GreaterThan(voucher.Conditions.MaxDiscount) {
		discount = voucher.Conditions.MaxDiscount
	}

	// Ensure discount doesn't exceed total
	if discount.GreaterThan(total) {
		discount = total
	}

	return discount
}

// InitializeVoucherStock initializes voucher stock in Redis cache
func (uc *CampaignUsecase) InitializeVoucherStock(ctx context.Context, code string) error {
	voucher, err := uc.voucherRepo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	return uc.voucherCache.InitializeStock(ctx, code, voucher.RemainingStock())
}

// GetUserVouchers retrieves all vouchers claimed by a user
func (uc *CampaignUsecase) GetUserVouchers(ctx context.Context, userID uuid.UUID) ([]*domain.UserVoucher, error) {
	return uc.userVoucherRepo.GetByUserID(ctx, userID)
}
