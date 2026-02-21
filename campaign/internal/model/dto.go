package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CartItem represents an item in the shopping cart for discount calculation
type CartItem struct {
	SKU      string
	Name     string
	Price    decimal.Decimal
	Quantity int
	Category string
}

func (ci *CartItem) TotalPrice() decimal.Decimal {
	return ci.Price.Mul(decimal.NewFromInt(int64(ci.Quantity)))
}

// CalculateCartResult holds the result of cart calculation with voucher applied
type CalculateCartResult struct {
	OriginalTotal  decimal.Decimal `json:"originalTotal"`
	DiscountAmount decimal.Decimal `json:"discountAmount"`
	FinalPrice     decimal.Decimal `json:"finalPrice"`
	VoucherApplied bool            `json:"voucherApplied"`
	VoucherCode    string          `json:"voucherCode,omitempty"`
	ErrorMessage   string          `json:"errorMessage,omitempty"`
}

func NewCalculateCartResult(originalTotal, discountAmount decimal.Decimal, voucherCode string) *CalculateCartResult {
	return &CalculateCartResult{
		OriginalTotal:  originalTotal,
		DiscountAmount: discountAmount,
		FinalPrice:     originalTotal.Sub(discountAmount),
		VoucherApplied: true,
		VoucherCode:    voucherCode,
	}
}

func NewCalculateCartResultNoDiscount(originalTotal decimal.Decimal) *CalculateCartResult {
	return &CalculateCartResult{
		OriginalTotal:  originalTotal,
		DiscountAmount: decimal.Zero,
		FinalPrice:     originalTotal,
		VoucherApplied: false,
	}
}

func NewCalculateCartResultError(originalTotal decimal.Decimal, errMsg string) *CalculateCartResult {
	return &CalculateCartResult{
		OriginalTotal:  originalTotal,
		DiscountAmount: decimal.Zero,
		FinalPrice:     originalTotal,
		VoucherApplied: false,
		ErrorMessage:   errMsg,
	}
}

// CreateCampaignRequest represents the request to create a campaign
type CreateCampaignRequest struct {
	Name      string    `json:"name" validate:"required,min=1,max=255"`
	StartTime time.Time `json:"startTime" validate:"required"`
	EndTime   time.Time `json:"endTime" validate:"required,gtfield=StartTime"`
}

// CampaignResponse represents the response for a campaign
type CampaignResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Status    string    `json:"status"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (c *Campaign) ToResponse() *CampaignResponse {
	return &CampaignResponse{
		ID:        c.ID.String(),
		Name:      c.Name,
		StartTime: c.StartTime,
		EndTime:   c.EndTime,
		Status:    string(c.Status),
		IsActive:  c.IsActive(),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// CreateVoucherRequest represents the request to create a voucher
type CreateVoucherRequest struct {
	Code       string            `json:"code" validate:"required,min=3,max=50"`
	CampaignID string            `json:"campaignId" validate:"required,uuid"`
	TotalCount int               `json:"totalCount" validate:"required,gt=0"`
	Type       string            `json:"type" validate:"required,oneof=PERCENTAGE FIXED_AMOUNT"`
	Value      decimal.Decimal   `json:"value" validate:"required"`
	Conditions VoucherConditions `json:"conditions"`
}

// VoucherResponse represents the response for a voucher
type VoucherResponse struct {
	ID             string            `json:"id"`
	Code           string            `json:"code"`
	CampaignID     string            `json:"campaignId"`
	TotalCount     int               `json:"totalCount"`
	UsedCount      int               `json:"usedCount"`
	RemainingStock int               `json:"remainingStock"`
	Type           string            `json:"type"`
	Value          decimal.Decimal   `json:"value"`
	Conditions     VoucherConditions `json:"conditions"`
	Status         string            `json:"status"`
	IsAvailable    bool              `json:"isAvailable"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

func (v *Voucher) ToResponse() *VoucherResponse {
	return &VoucherResponse{
		ID:             v.ID.String(),
		Code:           v.Code,
		CampaignID:     v.CampaignID.String(),
		TotalCount:     v.TotalCount,
		UsedCount:      v.UsedCount,
		RemainingStock: v.RemainingStock(),
		Type:           string(v.Type),
		Value:          v.Value,
		Conditions:     v.Conditions,
		Status:         string(v.Status),
		IsAvailable:    v.IsAvailable(),
		CreatedAt:      v.CreatedAt,
		UpdatedAt:      v.UpdatedAt,
	}
}

// ClaimVoucherRequest represents the request to claim a voucher
type ClaimVoucherRequest struct {
	UserID string `json:"userId" validate:"required,uuid"`
	Code   string `json:"code" validate:"required"`
}

// CalculateCartRequest represents the request to calculate cart discount
type CalculateCartRequest struct {
	Items       []CartItemRequest `json:"items" validate:"required,min=1,dive"`
	VoucherCode string            `json:"voucherCode"`
}

type CartItemRequest struct {
	SKU      string  `json:"sku" validate:"required"`
	Name     string  `json:"name" validate:"required"`
	Price    float64 `json:"price" validate:"required,gt=0"`
	Quantity int     `json:"quantity" validate:"required,gt=0"`
	Category string  `json:"category"`
}

func (r *CartItemRequest) ToModel() CartItem {
	return CartItem{
		SKU:      r.SKU,
		Name:     r.Name,
		Price:    decimal.NewFromFloat(r.Price),
		Quantity: r.Quantity,
		Category: r.Category,
	}
}

// UserVoucherResponse represents the response for a user voucher
type UserVoucherResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	VoucherID string     `json:"voucherId"`
	IsUsed    bool       `json:"isUsed"`
	ClaimedAt time.Time  `json:"claimedAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
}

func (uv *UserVoucher) ToResponse() *UserVoucherResponse {
	return &UserVoucherResponse{
		ID:        uv.ID.String(),
		UserID:    uv.UserID.String(),
		VoucherID: uv.VoucherID.String(),
		IsUsed:    uv.IsUsed,
		ClaimedAt: uv.ClaimedAt,
		UsedAt:    uv.UsedAt,
	}
}

// GetUserVouchersRequest represents the request to get user vouchers
type GetUserVouchersRequest struct {
	UserID uuid.UUID
}
