package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"microservices/order/internal/domain"
	"microservices/order/internal/util"
)

// CreateDraftOrderUseCase handles draft order creation from cart checkout
type CreateDraftOrderUseCase struct {
	orderRepo domain.OrderRepository
}

// NewCreateDraftOrderUseCase creates a new CreateDraftOrderUseCase
func NewCreateDraftOrderUseCase(orderRepo domain.OrderRepository) *CreateDraftOrderUseCase {
	return &CreateDraftOrderUseCase{
		orderRepo: orderRepo,
	}
}

// Execute creates a new draft order from cart checkout
// Draft orders do NOT reserve stock - stock is reserved when user confirms the draft
func (uc *CreateDraftOrderUseCase) Execute(ctx context.Context, req *CreateDraftOrderRequest) (*DraftOrderResponse, error) {
	// Step 1: Validate request
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Step 2: Convert shipping address to domain format
	shippingAddr := domain.ShippingAddress{
		ContactName:   req.ShippingAddress.FullName,
		Phone:         req.ShippingAddress.Phone,
		StreetAddress: req.ShippingAddress.Address,
		WardName:      req.ShippingAddress.Ward,
		DistrictName:  req.ShippingAddress.District,
		ProvinceName:  req.ShippingAddress.City,
		CountryCode:   req.ShippingAddress.Country,
		PostalCode:    req.ShippingAddress.PostalCode,
		FullAddress:   buildFullAddress(req.ShippingAddress),
	}

	// Step 3: Create draft order entity
	order := domain.NewDraftOrder(req.UserID, req.PaymentMethod, shippingAddr)

	// Step 4: Set voucher information
	order.VoucherCode = req.VoucherCode
	order.VoucherID = req.VoucherID
	order.CampaignID = req.CampaignID
	order.VoucherDiscount = req.DiscountAmount

	// Step 5: Set customer note
	order.CustomerNote = req.CustomerNote

	// Step 6: Create order items (without stock reservation)
	for _, itemDTO := range req.Items {
		item := domain.NewOrderItem(
			itemDTO.ProductID,
			itemDTO.SkuID,
			itemDTO.ProductName,
			itemDTO.SkuCode,
			itemDTO.Thumbnail,
			itemDTO.Quantity,
			itemDTO.UnitPrice,
		)
		order.AddItem(*item)
	}

	// Step 7: Set pre-calculated amounts from Cart Service
	order.SubTotal = req.OriginalAmount
	order.ShippingFee = req.ShippingFee
	order.VoucherDiscount = req.DiscountAmount
	order.FinalAmount = req.FinalAmount

	// Step 8: Persist to database (no stock reservation, no events)
	if err := uc.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Step 9: Return response
	return uc.toDraftOrderResponse(order), nil
}

// validateRequest validates the create draft order request
func (uc *CreateDraftOrderUseCase) validateRequest(req *CreateDraftOrderRequest) error {
	if req.UserID.String() == "" || req.UserID.String() == "00000000-0000-0000-0000-000000000000" {
		return domain.ErrInvalidUserID
	}

	if len(req.Items) == 0 {
		return domain.ErrEmptyOrderItems
	}

	validPaymentMethods := map[string]bool{
		"MOMO":   true,
		"COD":    true,
		"STRIPE": true,
		"VNPAY":  true,
	}
	if !validPaymentMethods[req.PaymentMethod] {
		return domain.ErrInvalidPaymentMethod
	}

	// finalAmount must be > 0
	if req.FinalAmount.LessThanOrEqual(decimalZero) {
		return domain.ErrInvalidPrice
	}

	// Validate VND price integrity: finalAmount = originalAmount - discountAmount + shippingFee
	calculatedFinal := req.OriginalAmount.
		Add(req.ShippingFee).
		Sub(req.DiscountAmount)

	// Allow ±1 VND tolerance for rounding differences
	diff := req.FinalAmount.Sub(calculatedFinal).Abs()
	if diff.GreaterThan(decimal.NewFromInt(1)) {
		return domain.ErrInvalidFinalAmount
	}

	// Amount must not be negative
	if req.FinalAmount.IsNegative() {
		return domain.ErrInvalidPrice
	}

	// Discount must not exceed original amount
	if req.DiscountAmount.GreaterThanOrEqual(req.OriginalAmount) && req.OriginalAmount.IsPositive() {
		return domain.ErrInvalidPrice
	}

	// Validate each item
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return domain.ErrInvalidQuantity
		}
		if item.UnitPrice.LessThanOrEqual(decimalZero) {
			return domain.ErrInvalidPrice
		}
		// VND must be whole number (no decimal places)
		if !util.IsWholeNumber(item.UnitPrice) {
			return domain.ErrInvalidPrice
		}
	}

	// Warning: VoucherCode present but discountAmount = 0
	if req.VoucherCode != "" && req.DiscountAmount.IsZero() {
		slog.Warn("VoucherCode present but discountAmount is zero",
			"voucherCode", req.VoucherCode,
			"userId", req.UserID.String(),
		)
		// Do not fail the request, just log a warning
	}

	return nil
}

// toDraftOrderResponse converts Order entity to DraftOrderResponse
func (uc *CreateDraftOrderUseCase) toDraftOrderResponse(order *domain.Order) *DraftOrderResponse {
	items := make([]DraftOrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = DraftOrderItemResponse{
			ID:          item.ID,
			SkuID:       item.SkuID,
			ProductName: item.ProductName,
			Thumbnail:   item.Thumbnail,
			Quantity:    item.Quantity,
			UnitPrice:   util.RoundVND(item.UnitPrice),
			SubTotal:    util.RoundVND(item.SubTotal),
		}
	}

	// Build price breakdown for transparent pricing display
	priceBreakdown := &PriceBreakdown{
		SubTotal:          util.RoundVND(order.SubTotal),
		ShippingFee:       util.RoundVND(order.ShippingFee),
		ShippingDiscount:  util.RoundVND(order.ShippingDiscount),
		VoucherDiscount:   util.RoundVND(order.VoucherDiscount),
		PromotionDiscount: util.RoundVND(order.DiscountAmount),
		FinalAmount:       util.RoundVND(order.FinalAmount),
		Currency:          order.Currency,
	}

	// Calculate expiresAt = createdAt + 24h (soft TTL for display purposes)
	expiresAt := order.CreatedAt.Add(24 * time.Hour)

	return &DraftOrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		OrderNumber:     order.OrderNumber,
		Status:          string(order.Status),
		Currency:        order.Currency,
		OriginalAmount:  util.RoundVND(order.SubTotal),
		DiscountAmount:  util.RoundVND(order.VoucherDiscount),
		ShippingFee:     util.RoundVND(order.ShippingFee),
		FinalAmount:     util.RoundVND(order.FinalAmount),
		VoucherCode:     order.VoucherCode,
		Items:           items,
		ShippingAddress: order.ShippingAddress,
		PaymentMethod:   order.PaymentMethod,
		CustomerNote:    order.CustomerNote,
		PriceBreakdown:  priceBreakdown,
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
		ExpiresAt:       expiresAt.Format(time.RFC3339),
	}
}

// buildFullAddress constructs a full address string from components
func buildFullAddress(addr ShippingAddressDTO) string {
	parts := []string{addr.Address}
	if addr.Ward != "" {
		parts = append(parts, addr.Ward)
	}
	parts = append(parts, addr.District, addr.City, addr.Country)

	result := ""
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i > 0 && result != "" {
			result += ", "
		}
		result += part
	}
	return result
}
