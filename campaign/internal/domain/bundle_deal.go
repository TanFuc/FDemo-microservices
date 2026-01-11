package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BundleDealStatus represents the status of a bundle deal
type BundleDealStatus string

const (
	BundleDealStatusDraft     BundleDealStatus = "DRAFT"
	BundleDealStatusActive    BundleDealStatus = "ACTIVE"
	BundleDealStatusInactive  BundleDealStatus = "INACTIVE"
	BundleDealStatusExpired   BundleDealStatus = "EXPIRED"
)

// BundleDiscountType represents how the bundle discount is calculated
type BundleDiscountType string

const (
	BundleDiscountTypePercentage   BundleDiscountType = "PERCENTAGE"   // % off bundle total
	BundleDiscountTypeFixedAmount  BundleDiscountType = "FIXED_AMOUNT" // Fixed amount off
	BundleDiscountTypeFixedPrice   BundleDiscountType = "FIXED_PRICE"  // Fixed bundle price
)

// BundleDeal represents a bundle deal promotion
type BundleDeal struct {
	ID              uuid.UUID          `json:"id"`
	CampaignID      *uuid.UUID         `json:"campaign_id,omitempty"`
	ShopID          uuid.UUID          `json:"shop_id"`

	// Basic info
	Name            string             `json:"name"`
	Slug            string             `json:"slug"`
	Description     string             `json:"description"`
	ImageURL        string             `json:"image_url,omitempty"`

	// Bundle items
	Items           []BundleItem       `json:"items"`
	MinItems        int                `json:"min_items"` // Min items to activate bundle
	MaxItems        int                `json:"max_items"` // Max items in bundle (0 = no limit)

	// Pricing
	DiscountType    BundleDiscountType `json:"discount_type"`
	DiscountValue   decimal.Decimal    `json:"discount_value"` // % or fixed amount
	BundlePrice     decimal.Decimal    `json:"bundle_price,omitempty"` // For FIXED_PRICE type
	MaxDiscount     decimal.Decimal    `json:"max_discount,omitempty"` // Cap for percentage discount

	// Calculated prices (denormalized)
	OriginalTotal   decimal.Decimal    `json:"original_total"`
	FinalPrice      decimal.Decimal    `json:"final_price"`
	SavingsAmount   decimal.Decimal    `json:"savings_amount"`
	SavingsPercent  decimal.Decimal    `json:"savings_percent"`

	// Limits
	TotalStock      int                `json:"total_stock"` // 0 = unlimited
	SoldCount       int                `json:"sold_count"`
	MaxPerUser      int                `json:"max_per_user"` // 0 = no limit

	// Validity
	ValidFrom       *time.Time         `json:"valid_from,omitempty"`
	ValidUntil      *time.Time         `json:"valid_until,omitempty"`

	// Status
	Status          BundleDealStatus   `json:"status"`
	IsStackable     bool               `json:"is_stackable"` // Can stack with other promotions
	IsVisible       bool               `json:"is_visible"`

	// Conditions
	MinOrderValue   decimal.Decimal    `json:"min_order_value,omitempty"`
	UserSegments    []string           `json:"user_segments,omitempty"`

	// Stats
	ViewCount       int64              `json:"view_count"`
	TotalRevenue    decimal.Decimal    `json:"total_revenue"`

	// Metadata
	Metadata        json.RawMessage    `json:"metadata,omitempty"`

	// Audit
	CreatedBy       string             `json:"created_by"`
	UpdatedBy       string             `json:"updated_by,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// BundleItem represents an item in a bundle
type BundleItem struct {
	ProductID       string          `json:"product_id"`
	SkuID           string          `json:"sku_id,omitempty"`
	Quantity        int             `json:"quantity"`
	IsRequired      bool            `json:"is_required"` // Must be in bundle
	IsSubstitutable bool            `json:"is_substitutable"` // Can be replaced with alternatives
	AlternativeIDs  []string        `json:"alternative_ids,omitempty"`

	// Price snapshot
	UnitPrice       decimal.Decimal `json:"unit_price"`
	TotalPrice      decimal.Decimal `json:"total_price"`

	// Display
	ProductName     string          `json:"product_name"`
	ProductImage    string          `json:"product_image"`
	SortOrder       int             `json:"sort_order"`
}

// NewBundleDeal creates a new bundle deal
func NewBundleDeal(shopID uuid.UUID, name string, discountType BundleDiscountType, discountValue decimal.Decimal, createdBy string) *BundleDeal {
	now := time.Now()
	return &BundleDeal{
		ID:             uuid.New(),
		ShopID:         shopID,
		Name:           name,
		Slug:           GenerateSlug(name),
		DiscountType:   discountType,
		DiscountValue:  discountValue,
		Items:          make([]BundleItem, 0),
		MinItems:       2,
		MaxItems:       0,
		TotalStock:     0,
		SoldCount:      0,
		MaxPerUser:     0,
		Status:         BundleDealStatusDraft,
		IsStackable:    false,
		IsVisible:      true,
		ViewCount:      0,
		TotalRevenue:   decimal.Zero,
		UserSegments:   make([]string, 0),
		Metadata:       json.RawMessage("{}"),
		CreatedBy:      createdBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// AddItem adds an item to the bundle
func (bd *BundleDeal) AddItem(item BundleItem) {
	item.SortOrder = len(bd.Items)
	bd.Items = append(bd.Items, item)
	bd.RecalculatePricing()
	bd.UpdatedAt = time.Now()
}

// RemoveItem removes an item from the bundle
func (bd *BundleDeal) RemoveItem(productID string) {
	for i, item := range bd.Items {
		if item.ProductID == productID {
			bd.Items = append(bd.Items[:i], bd.Items[i+1:]...)
			break
		}
	}
	bd.RecalculatePricing()
	bd.UpdatedAt = time.Now()
}

// RecalculatePricing recalculates the bundle pricing
func (bd *BundleDeal) RecalculatePricing() {
	bd.OriginalTotal = decimal.Zero
	for _, item := range bd.Items {
		bd.OriginalTotal = bd.OriginalTotal.Add(item.TotalPrice)
	}

	switch bd.DiscountType {
	case BundleDiscountTypePercentage:
		discount := bd.OriginalTotal.Mul(bd.DiscountValue).Div(decimal.NewFromInt(100))
		if !bd.MaxDiscount.IsZero() && discount.GreaterThan(bd.MaxDiscount) {
			discount = bd.MaxDiscount
		}
		bd.FinalPrice = bd.OriginalTotal.Sub(discount)
		bd.SavingsAmount = discount

	case BundleDiscountTypeFixedAmount:
		bd.FinalPrice = bd.OriginalTotal.Sub(bd.DiscountValue)
		bd.SavingsAmount = bd.DiscountValue

	case BundleDiscountTypeFixedPrice:
		bd.FinalPrice = bd.BundlePrice
		bd.SavingsAmount = bd.OriginalTotal.Sub(bd.BundlePrice)
	}

	if bd.FinalPrice.LessThan(decimal.Zero) {
		bd.FinalPrice = decimal.Zero
	}

	if !bd.OriginalTotal.IsZero() {
		bd.SavingsPercent = bd.SavingsAmount.Div(bd.OriginalTotal).Mul(decimal.NewFromInt(100))
	}
}

// SetValidity sets the validity period
func (bd *BundleDeal) SetValidity(from, until *time.Time) {
	bd.ValidFrom = from
	bd.ValidUntil = until
	bd.UpdatedAt = time.Now()
}

// Activate activates the bundle deal
func (bd *BundleDeal) Activate() {
	bd.Status = BundleDealStatusActive
	bd.UpdatedAt = time.Now()
}

// Deactivate deactivates the bundle deal
func (bd *BundleDeal) Deactivate() {
	bd.Status = BundleDealStatusInactive
	bd.UpdatedAt = time.Now()
}

// Expire marks the bundle deal as expired
func (bd *BundleDeal) Expire() {
	bd.Status = BundleDealStatusExpired
	bd.UpdatedAt = time.Now()
}

// IsValid checks if the bundle deal is currently valid
func (bd *BundleDeal) IsValid() bool {
	if bd.Status != BundleDealStatusActive {
		return false
	}
	now := time.Now()
	if bd.ValidFrom != nil && now.Before(*bd.ValidFrom) {
		return false
	}
	if bd.ValidUntil != nil && now.After(*bd.ValidUntil) {
		return false
	}
	return true
}

// IsAvailable checks if the bundle is available for purchase
func (bd *BundleDeal) IsAvailable() bool {
	if !bd.IsValid() {
		return false
	}
	if bd.TotalStock > 0 && bd.SoldCount >= bd.TotalStock {
		return false
	}
	return true
}

// RecordSale records a sale
func (bd *BundleDeal) RecordSale(revenue decimal.Decimal) {
	bd.SoldCount++
	bd.TotalRevenue = bd.TotalRevenue.Add(revenue)
	bd.UpdatedAt = time.Now()
}

// IncrementViewCount increments the view count
func (bd *BundleDeal) IncrementViewCount() {
	bd.ViewCount++
	bd.UpdatedAt = time.Now()
}

// GetRequiredItems returns only the required items
func (bd *BundleDeal) GetRequiredItems() []BundleItem {
	var required []BundleItem
	for _, item := range bd.Items {
		if item.IsRequired {
			required = append(required, item)
		}
	}
	return required
}

// GetOptionalItems returns only the optional items
func (bd *BundleDeal) GetOptionalItems() []BundleItem {
	var optional []BundleItem
	for _, item := range bd.Items {
		if !item.IsRequired {
			optional = append(optional, item)
		}
	}
	return optional
}

// ValidateItemSelection validates that a selection of items meets bundle requirements
func (bd *BundleDeal) ValidateItemSelection(selectedProductIDs []string) bool {
	// Check required items
	for _, item := range bd.Items {
		if item.IsRequired {
			found := false
			for _, id := range selectedProductIDs {
				if id == item.ProductID {
					found = true
					break
				}
				// Check alternatives
				if item.IsSubstitutable {
					for _, altID := range item.AlternativeIDs {
						if altID == id {
							found = true
							break
						}
					}
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check min/max items
	if len(selectedProductIDs) < bd.MinItems {
		return false
	}
	if bd.MaxItems > 0 && len(selectedProductIDs) > bd.MaxItems {
		return false
	}

	return true
}
