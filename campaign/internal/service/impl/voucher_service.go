package impl

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"microservices/campaign/internal/model"
	"microservices/campaign/internal/repository"
)

var (
	ErrVoucherNotFound = errors.New("voucher not found")
)

// VoucherService implements voucher business logic
type VoucherService struct {
	voucherRepo     repository.VoucherRepository
	userVoucherRepo repository.UserVoucherRepository
	voucherCache    repository.VoucherCacheRepository
}

// NewVoucherService creates a new voucher service
func NewVoucherService(
	voucherRepo repository.VoucherRepository,
	userVoucherRepo repository.UserVoucherRepository,
	voucherCache repository.VoucherCacheRepository,
) *VoucherService {
	return &VoucherService{
		voucherRepo:     voucherRepo,
		userVoucherRepo: userVoucherRepo,
		voucherCache:    voucherCache,
	}
}

// CreateVoucher creates a new voucher
func (s *VoucherService) CreateVoucher(ctx context.Context, req *model.CreateVoucherRequest) (*model.VoucherResponse, error) {
	// Custom validation
	if err := req.Validate(); err != nil {
		return nil, err
	}

	campaignID, err := uuid.Parse(req.CampaignID)
	if err != nil {
		return nil, ErrInvalidID
	}

	voucher := model.NewVoucher(
		req.Code,
		campaignID,
		req.TotalCount,
		model.VoucherType(req.Type),
		req.Value,
		req.Conditions,
		model.VoucherAssignType(req.AssignType),
		req.AssignedUserIDs,
	)

	if err := s.voucherRepo.Create(ctx, voucher); err != nil {
		return nil, err
	}

	return voucher.ToResponse(), nil
}

// GetVoucher retrieves a voucher by ID
func (s *VoucherService) GetVoucher(ctx context.Context, id string) (*model.VoucherResponse, error) {
	voucherID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	voucher, err := s.voucherRepo.GetByID(ctx, voucherID)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, ErrVoucherNotFound
	}

	return voucher.ToResponse(), nil
}

// GetVoucherByCode retrieves a voucher by code
func (s *VoucherService) GetVoucherByCode(ctx context.Context, code string) (*model.VoucherResponse, error) {
	voucher, err := s.voucherRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, ErrVoucherNotFound
	}

	return voucher.ToResponse(), nil
}

// ClaimVoucher handles atomic voucher claiming using Redis + Postgres
func (s *VoucherService) ClaimVoucher(ctx context.Context, userID uuid.UUID, code string) error {
	// Validate voucher exists and is active
	voucher, err := s.voucherRepo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if voucher == nil {
		return model.ErrVoucherNotFound
	}
	if voucher.Status != model.VoucherStatusActive {
		return model.ErrVoucherInactive
	}

	// Check user eligibility based on assignment type
	if !voucher.IsUserEligible(userID.String()) {
		return model.ErrUserNotEligible
	}

	// Step 1: Atomic claim via Redis Lua script
	if err := s.voucherCache.AtomicClaim(ctx, code, userID); err != nil {
		return err
	}

	// Step 2: Persist to Postgres (Redis is source of truth, this is for durability)
	userVoucher := model.NewUserVoucher(userID, voucher.ID)
	if err := s.userVoucherRepo.Create(ctx, userVoucher); err != nil {
		// Log error but don't fail - Redis is source of truth
		log.Printf("warning: failed to persist user voucher to postgres: %v", err)
	}

	// Update used count in Postgres (for statistics only)
	if err := s.voucherRepo.IncrementUsedCount(ctx, voucher.ID); err != nil {
		log.Printf("warning: failed to increment used count: %v", err)
	}

	return nil
}

// CalculateCart applies voucher rules to cart items and returns discount calculation
func (s *VoucherService) CalculateCart(ctx context.Context, items []model.CartItem, voucherCode string, userID string) (*model.CalculateCartResult, error) {
	// Calculate original total
	originalTotal := decimal.Zero
	for _, item := range items {
		originalTotal = originalTotal.Add(item.TotalPrice())
	}

	// If no voucher code provided, return without discount
	if voucherCode == "" {
		return model.NewCalculateCartResultNoDiscount(originalTotal), nil
	}

	// Fetch voucher
	voucher, err := s.voucherRepo.GetByCode(ctx, voucherCode)
	if err != nil {
		return model.NewCalculateCartResultError(originalTotal, err.Error()), err
	}
	if voucher == nil {
		return model.NewCalculateCartResultError(originalTotal, model.ErrVoucherNotFound.Error()), model.ErrVoucherNotFound
	}

	if voucher.Status != model.VoucherStatusActive {
		return model.NewCalculateCartResultError(originalTotal, model.ErrVoucherInactive.Error()), model.ErrVoucherInactive
	}

	// Check user eligibility if userID is provided
	if userID != "" && !voucher.IsUserEligible(userID) {
		return model.NewCalculateCartResultError(originalTotal, model.ErrUserNotEligible.Error()), model.ErrUserNotEligible
	}

	// Apply Rule Engine validation
	if err := s.validateVoucherConditions(voucher, items, originalTotal); err != nil {
		return model.NewCalculateCartResultError(originalTotal, err.Error()), err
	}

	// Calculate discount
	discount := s.calculateDiscount(voucher, originalTotal)

	return model.NewCalculateCartResult(originalTotal, discount, voucherCode), nil
}

// ValidateVoucherForCart validates a voucher for Cart Service and returns detailed result
func (s *VoucherService) ValidateVoucherForCart(ctx context.Context, req *model.ValidateVoucherRequest) (*model.VoucherValidationResult, error) {
	// Convert items to model.CartItem
	items := make([]model.CartItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = item.ToModel()
	}

	// Calculate original total
	originalTotal := decimal.Zero
	for _, item := range items {
		originalTotal = originalTotal.Add(item.TotalPrice())
	}

	// Fetch voucher
	voucher, err := s.voucherRepo.GetByCode(ctx, req.VoucherCode)
	if err != nil {
		return &model.VoucherValidationResult{
			Valid:         false,
			OriginalTotal: originalTotal,
			FinalPrice:    originalTotal,
			ErrorCode:     "VOUCHER_ERROR",
			ErrorMessage:  err.Error(),
		}, err
	}
	if voucher == nil {
		return &model.VoucherValidationResult{
			Valid:         false,
			OriginalTotal: originalTotal,
			FinalPrice:    originalTotal,
			ErrorCode:     "VOUCHER_NOT_FOUND",
			ErrorMessage:  model.ErrVoucherNotFound.Error(),
		}, model.ErrVoucherNotFound
	}

	// Check voucher is active
	if voucher.Status != model.VoucherStatusActive {
		return &model.VoucherValidationResult{
			Valid:         false,
			OriginalTotal: originalTotal,
			FinalPrice:    originalTotal,
			ErrorCode:     "VOUCHER_INACTIVE",
			ErrorMessage:  model.ErrVoucherInactive.Error(),
		}, nil
	}

	// Check voucher has stock
	if !voucher.IsAvailable() {
		return &model.VoucherValidationResult{
			Valid:         false,
			OriginalTotal: originalTotal,
			FinalPrice:    originalTotal,
			ErrorCode:     "VOUCHER_OUT_OF_STOCK",
			ErrorMessage:  model.ErrVoucherOutOfStock.Error(),
		}, nil
	}

	// Check user eligibility
	if !voucher.IsUserEligible(req.UserID) {
		return &model.VoucherValidationResult{
			Valid:         false,
			OriginalTotal: originalTotal,
			FinalPrice:    originalTotal,
			ErrorCode:     "USER_NOT_ELIGIBLE",
			ErrorMessage:  model.ErrUserNotEligible.Error(),
		}, nil
	}

	// Apply Rule Engine validation
	if err := s.validateVoucherConditions(voucher, items, originalTotal); err != nil {
		errorCode := "VALIDATION_FAILED"
		if errors.Is(err, model.ErrMinOrderNotMet) {
			errorCode = "MIN_ORDER_NOT_MET"
		} else if errors.Is(err, model.ErrCategoryNotAllowed) {
			errorCode = "CATEGORY_NOT_ALLOWED"
		} else if errors.Is(err, model.ErrProductExcluded) {
			errorCode = "PRODUCT_EXCLUDED"
		}
		return &model.VoucherValidationResult{
			Valid:         false,
			OriginalTotal: originalTotal,
			FinalPrice:    originalTotal,
			ErrorCode:     errorCode,
			ErrorMessage:  err.Error(),
		}, nil
	}

	// Calculate discount
	discount := s.calculateDiscount(voucher, originalTotal)
	finalPrice := originalTotal.Sub(discount)

	return &model.VoucherValidationResult{
		Valid:          true,
		VoucherID:      voucher.ID.String(),
		CampaignID:     voucher.CampaignID.String(),
		VoucherCode:    voucher.Code,
		DiscountType:   string(voucher.Type),
		DiscountValue:  voucher.Value,
		DiscountAmount: discount,
		OriginalTotal:  originalTotal,
		FinalPrice:     finalPrice,
	}, nil
}

// validateVoucherConditions applies the rule engine to check voucher eligibility
func (s *VoucherService) validateVoucherConditions(voucher *model.Voucher, items []model.CartItem, total decimal.Decimal) error {
	conditions := voucher.Conditions

	// Rule 1: Check minimum order value
	if !conditions.MinOrderValue.IsZero() && total.LessThan(conditions.MinOrderValue) {
		return model.ErrMinOrderNotMet
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
			return model.ErrCategoryNotAllowed
		}
	}

	// Rule 3: Check excluded products
	if len(conditions.ExcludedProducts) > 0 {
		for _, item := range items {
			for _, excludedSKU := range conditions.ExcludedProducts {
				if item.SKU == excludedSKU {
					return model.ErrProductExcluded
				}
			}
		}
	}

	return nil
}

// calculateDiscount computes the discount amount based on voucher type and conditions
func (s *VoucherService) calculateDiscount(voucher *model.Voucher, total decimal.Decimal) decimal.Decimal {
	var discount decimal.Decimal

	switch voucher.Type {
	case model.VoucherTypePercentage:
		discount = total.Mul(voucher.Value).Div(decimal.NewFromInt(100))
	case model.VoucherTypeFixedAmount:
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
func (s *VoucherService) InitializeVoucherStock(ctx context.Context, code string) error {
	voucher, err := s.voucherRepo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if voucher == nil {
		return fmt.Errorf("voucher not found: %s", code)
	}
	return s.voucherCache.InitializeStock(ctx, code, voucher.RemainingStock())
}

// GetUserVouchers retrieves all vouchers claimed by a user
func (s *VoucherService) GetUserVouchers(ctx context.Context, userID uuid.UUID) ([]*model.UserVoucherResponse, error) {
	userVouchers, err := s.userVoucherRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*model.UserVoucherResponse, len(userVouchers))
	for i, uv := range userVouchers {
		responses[i] = uv.ToResponse()
	}

	return responses, nil
}
