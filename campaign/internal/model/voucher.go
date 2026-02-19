package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type VoucherType string

const (
	VoucherTypePercentage  VoucherType = "PERCENTAGE"
	VoucherTypeFixedAmount VoucherType = "FIXED_AMOUNT"
)

type VoucherStatus string

const (
	VoucherStatusActive   VoucherStatus = "ACTIVE"
	VoucherStatusInactive VoucherStatus = "INACTIVE"
)

var (
	ErrVoucherNotFound       = errors.New("voucher not found")
	ErrVoucherInactive       = errors.New("voucher is inactive")
	ErrVoucherOutOfStock     = errors.New("voucher out of stock")
	ErrVoucherAlreadyClaimed = errors.New("voucher already claimed by user")
	ErrMinOrderNotMet        = errors.New("minimum order value not met")
	ErrCategoryNotAllowed    = errors.New("cart items do not match allowed categories")
	ErrProductExcluded       = errors.New("cart contains excluded products")
)

// VoucherConditions represents the JSONB rule set for voucher eligibility
type VoucherConditions struct {
	MinOrderValue     decimal.Decimal `json:"min_order_value,omitempty"`
	MaxDiscount       decimal.Decimal `json:"max_discount,omitempty"`
	AllowedCategories []string        `json:"allowed_categories,omitempty"`
	ExcludedProducts  []string        `json:"excluded_products,omitempty"`
}

func (vc *VoucherConditions) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan VoucherConditions")
	}
	return json.Unmarshal(bytes, vc)
}

func (vc VoucherConditions) Value() ([]byte, error) {
	return json.Marshal(vc)
}

type Voucher struct {
	ID         uuid.UUID
	Code       string
	CampaignID uuid.UUID
	TotalCount int
	UsedCount  int
	Type       VoucherType
	Value      decimal.Decimal
	Conditions VoucherConditions
	Status     VoucherStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewVoucher(code string, campaignID uuid.UUID, totalCount int, vType VoucherType, value decimal.Decimal, conditions VoucherConditions) *Voucher {
	return &Voucher{
		ID:         uuid.New(),
		Code:       code,
		CampaignID: campaignID,
		TotalCount: totalCount,
		UsedCount:  0,
		Type:       vType,
		Value:      value,
		Conditions: conditions,
		Status:     VoucherStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func (v *Voucher) IsAvailable() bool {
	return v.Status == VoucherStatusActive && v.UsedCount < v.TotalCount
}

func (v *Voucher) RemainingStock() int {
	return v.TotalCount - v.UsedCount
}
