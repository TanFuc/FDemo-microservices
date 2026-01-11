package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// GWPStatus represents the status of a gift with purchase promotion
type GWPStatus string

const (
	GWPStatusDraft    GWPStatus = "DRAFT"
	GWPStatusActive   GWPStatus = "ACTIVE"
	GWPStatusInactive GWPStatus = "INACTIVE"
	GWPStatusExpired  GWPStatus = "EXPIRED"
)

// GWPTriggerType represents what triggers the gift
type GWPTriggerType string

const (
	GWPTriggerTypeMinSpend     GWPTriggerType = "MIN_SPEND"     // Minimum order value
	GWPTriggerTypeProductBuy   GWPTriggerType = "PRODUCT_BUY"   // Buy specific product(s)
	GWPTriggerTypeCategoryBuy  GWPTriggerType = "CATEGORY_BUY"  // Buy from category
	GWPTriggerTypeQuantity     GWPTriggerType = "QUANTITY"      // Buy X quantity
	GWPTriggerTypeTiered       GWPTriggerType = "TIERED"        // Different gifts at different thresholds
)

// GWPSelectionType represents how users select their gift
type GWPSelectionType string

const (
	GWPSelectionTypeAutomatic   GWPSelectionType = "AUTOMATIC"   // Gift added automatically
	GWPSelectionTypeUserChoice  GWPSelectionType = "USER_CHOICE" // User selects from options
)

// GiftWithPurchase represents a gift with purchase promotion
type GiftWithPurchase struct {
	ID              uuid.UUID        `json:"id"`
	CampaignID      *uuid.UUID       `json:"campaign_id,omitempty"`
	ShopID          *uuid.UUID       `json:"shop_id,omitempty"` // nil = platform-wide

	// Basic info
	Name            string           `json:"name"`
	Slug            string           `json:"slug"`
	Description     string           `json:"description"`
	BannerURL       string           `json:"banner_url,omitempty"`
	PromoBadge      string           `json:"promo_badge,omitempty"` // e.g., "FREE GIFT"

	// Trigger configuration
	TriggerType     GWPTriggerType   `json:"trigger_type"`
	MinSpendAmount  decimal.Decimal  `json:"min_spend_amount,omitempty"`
	TriggerProducts []string         `json:"trigger_products,omitempty"` // Product IDs
	TriggerCategories []string       `json:"trigger_categories,omitempty"`
	TriggerQuantity int              `json:"trigger_quantity,omitempty"` // Buy X get Y

	// Tiered thresholds (for TIERED trigger type)
	Tiers           []GWPTier        `json:"tiers,omitempty"`

	// Gift selection
	SelectionType   GWPSelectionType `json:"selection_type"`
	GiftOptions     []GiftOption     `json:"gift_options"`
	MaxGiftsPerOrder int             `json:"max_gifts_per_order"` // 0 = based on trigger

	// Stock management
	TotalGiftStock  int              `json:"total_gift_stock"` // 0 = unlimited
	ClaimedCount    int              `json:"claimed_count"`

	// Limits
	MaxClaimsPerUser int             `json:"max_claims_per_user"` // 0 = no limit
	MaxTotalClaims   int             `json:"max_total_claims"` // 0 = no limit

	// Validity
	ValidFrom       *time.Time       `json:"valid_from,omitempty"`
	ValidUntil      *time.Time       `json:"valid_until,omitempty"`

	// Status
	Status          GWPStatus        `json:"status"`
	IsStackable     bool             `json:"is_stackable"` // Can stack with other promotions
	IsVisible       bool             `json:"is_visible"`

	// Priority (higher = applied first)
	Priority        int              `json:"priority"`

	// Conditions
	UserSegments    []string         `json:"user_segments,omitempty"`
	ExcludedProducts []string        `json:"excluded_products,omitempty"`
	NewUsersOnly    bool             `json:"new_users_only"`
	FirstOrderOnly  bool             `json:"first_order_only"`

	// Stats
	ViewCount       int64            `json:"view_count"`
	ConversionCount int64            `json:"conversion_count"` // Orders with gift

	// Metadata
	Metadata        json.RawMessage  `json:"metadata,omitempty"`

	// Audit
	CreatedBy       string           `json:"created_by"`
	UpdatedBy       string           `json:"updated_by,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// GWPTier represents a tier in tiered gift with purchase
type GWPTier struct {
	MinAmount       decimal.Decimal `json:"min_amount"`
	MinQuantity     int             `json:"min_quantity,omitempty"`
	GiftOptionIDs   []uuid.UUID     `json:"gift_option_ids"`
	GiftQuantity    int             `json:"gift_quantity"` // How many gifts at this tier
	Description     string          `json:"description,omitempty"`
}

// GiftOption represents a gift that can be given
type GiftOption struct {
	ID              uuid.UUID       `json:"id"`
	ProductID       string          `json:"product_id"`
	SkuID           string          `json:"sku_id,omitempty"`
	Quantity        int             `json:"quantity"` // Quantity per gift claim

	// Stock for this specific gift
	TotalStock      int             `json:"total_stock"` // 0 = unlimited
	ClaimedCount    int             `json:"claimed_count"`

	// Value (for display)
	RetailValue     decimal.Decimal `json:"retail_value"`

	// Display info
	ProductName     string          `json:"product_name"`
	ProductImage    string          `json:"product_image"`
	Description     string          `json:"description,omitempty"`

	// Availability
	IsActive        bool            `json:"is_active"`
	SortOrder       int             `json:"sort_order"`
}

// GWPClaim represents a user's claim of a gift
type GWPClaim struct {
	ID              uuid.UUID  `json:"id"`
	GWPID           uuid.UUID  `json:"gwp_id"`
	UserID          string     `json:"user_id"`
	OrderID         uuid.UUID  `json:"order_id"`
	GiftOptionID    uuid.UUID  `json:"gift_option_id"`
	ProductID       string     `json:"product_id"`
	Quantity        int        `json:"quantity"`
	TierIndex       *int       `json:"tier_index,omitempty"` // Which tier triggered
	TriggerAmount   decimal.Decimal `json:"trigger_amount,omitempty"`
	ClaimedAt       time.Time  `json:"claimed_at"`
}

// NewGiftWithPurchase creates a new gift with purchase promotion
func NewGiftWithPurchase(name string, triggerType GWPTriggerType, selectionType GWPSelectionType, createdBy string) *GiftWithPurchase {
	now := time.Now()
	return &GiftWithPurchase{
		ID:               uuid.New(),
		Name:             name,
		Slug:             GenerateSlug(name),
		TriggerType:      triggerType,
		SelectionType:    selectionType,
		GiftOptions:      make([]GiftOption, 0),
		Tiers:            make([]GWPTier, 0),
		TriggerProducts:  make([]string, 0),
		TriggerCategories: make([]string, 0),
		MaxGiftsPerOrder: 1,
		TotalGiftStock:   0,
		ClaimedCount:     0,
		MaxClaimsPerUser: 0,
		MaxTotalClaims:   0,
		Status:           GWPStatusDraft,
		IsStackable:      true,
		IsVisible:        true,
		Priority:         0,
		UserSegments:     make([]string, 0),
		ExcludedProducts: make([]string, 0),
		NewUsersOnly:     false,
		FirstOrderOnly:   false,
		ViewCount:        0,
		ConversionCount:  0,
		Metadata:         json.RawMessage("{}"),
		CreatedBy:        createdBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// SetMinSpendTrigger sets minimum spend trigger
func (gwp *GiftWithPurchase) SetMinSpendTrigger(amount decimal.Decimal) {
	gwp.TriggerType = GWPTriggerTypeMinSpend
	gwp.MinSpendAmount = amount
	gwp.UpdatedAt = time.Now()
}

// SetProductTrigger sets product purchase trigger
func (gwp *GiftWithPurchase) SetProductTrigger(productIDs []string) {
	gwp.TriggerType = GWPTriggerTypeProductBuy
	gwp.TriggerProducts = productIDs
	gwp.UpdatedAt = time.Now()
}

// SetCategoryTrigger sets category purchase trigger
func (gwp *GiftWithPurchase) SetCategoryTrigger(categoryIDs []string) {
	gwp.TriggerType = GWPTriggerTypeCategoryBuy
	gwp.TriggerCategories = categoryIDs
	gwp.UpdatedAt = time.Now()
}

// AddGiftOption adds a gift option
func (gwp *GiftWithPurchase) AddGiftOption(option GiftOption) {
	option.ID = uuid.New()
	option.SortOrder = len(gwp.GiftOptions)
	option.IsActive = true
	option.ClaimedCount = 0
	gwp.GiftOptions = append(gwp.GiftOptions, option)
	gwp.UpdatedAt = time.Now()
}

// RemoveGiftOption removes a gift option
func (gwp *GiftWithPurchase) RemoveGiftOption(optionID uuid.UUID) {
	for i, opt := range gwp.GiftOptions {
		if opt.ID == optionID {
			gwp.GiftOptions = append(gwp.GiftOptions[:i], gwp.GiftOptions[i+1:]...)
			break
		}
	}
	gwp.UpdatedAt = time.Now()
}

// AddTier adds a tier for tiered GWP
func (gwp *GiftWithPurchase) AddTier(tier GWPTier) {
	gwp.Tiers = append(gwp.Tiers, tier)
	// Sort tiers by min amount
	for i := len(gwp.Tiers) - 1; i > 0; i-- {
		if gwp.Tiers[i].MinAmount.LessThan(gwp.Tiers[i-1].MinAmount) {
			gwp.Tiers[i], gwp.Tiers[i-1] = gwp.Tiers[i-1], gwp.Tiers[i]
		}
	}
	gwp.UpdatedAt = time.Now()
}

// SetValidity sets the validity period
func (gwp *GiftWithPurchase) SetValidity(from, until *time.Time) {
	gwp.ValidFrom = from
	gwp.ValidUntil = until
	gwp.UpdatedAt = time.Now()
}

// Activate activates the promotion
func (gwp *GiftWithPurchase) Activate() {
	gwp.Status = GWPStatusActive
	gwp.UpdatedAt = time.Now()
}

// Deactivate deactivates the promotion
func (gwp *GiftWithPurchase) Deactivate() {
	gwp.Status = GWPStatusInactive
	gwp.UpdatedAt = time.Now()
}

// Expire marks as expired
func (gwp *GiftWithPurchase) Expire() {
	gwp.Status = GWPStatusExpired
	gwp.UpdatedAt = time.Now()
}

// IsValid checks if the promotion is currently valid
func (gwp *GiftWithPurchase) IsValid() bool {
	if gwp.Status != GWPStatusActive {
		return false
	}
	now := time.Now()
	if gwp.ValidFrom != nil && now.Before(*gwp.ValidFrom) {
		return false
	}
	if gwp.ValidUntil != nil && now.After(*gwp.ValidUntil) {
		return false
	}
	return true
}

// HasAvailableGifts checks if there are available gifts
func (gwp *GiftWithPurchase) HasAvailableGifts() bool {
	if gwp.MaxTotalClaims > 0 && gwp.ClaimedCount >= gwp.MaxTotalClaims {
		return false
	}
	for _, opt := range gwp.GiftOptions {
		if opt.IsActive && (opt.TotalStock == 0 || opt.ClaimedCount < opt.TotalStock) {
			return true
		}
	}
	return false
}

// GetAvailableGiftOptions returns available gift options
func (gwp *GiftWithPurchase) GetAvailableGiftOptions() []GiftOption {
	var available []GiftOption
	for _, opt := range gwp.GiftOptions {
		if opt.IsActive && (opt.TotalStock == 0 || opt.ClaimedCount < opt.TotalStock) {
			available = append(available, opt)
		}
	}
	return available
}

// CheckTrigger checks if the trigger condition is met
func (gwp *GiftWithPurchase) CheckTrigger(orderTotal decimal.Decimal, productIDs []string, categoryIDs []string, quantity int) (bool, *int) {
	switch gwp.TriggerType {
	case GWPTriggerTypeMinSpend:
		return orderTotal.GreaterThanOrEqual(gwp.MinSpendAmount), nil

	case GWPTriggerTypeProductBuy:
		for _, triggerID := range gwp.TriggerProducts {
			for _, productID := range productIDs {
				if triggerID == productID {
					return true, nil
				}
			}
		}
		return false, nil

	case GWPTriggerTypeCategoryBuy:
		for _, triggerCat := range gwp.TriggerCategories {
			for _, catID := range categoryIDs {
				if triggerCat == catID {
					return true, nil
				}
			}
		}
		return false, nil

	case GWPTriggerTypeQuantity:
		return quantity >= gwp.TriggerQuantity, nil

	case GWPTriggerTypeTiered:
		for i := len(gwp.Tiers) - 1; i >= 0; i-- {
			tier := gwp.Tiers[i]
			if orderTotal.GreaterThanOrEqual(tier.MinAmount) {
				return true, &i
			}
		}
		return false, nil
	}

	return false, nil
}

// RecordClaim records a gift claim
func (gwp *GiftWithPurchase) RecordClaim(giftOptionID uuid.UUID) {
	gwp.ClaimedCount++
	gwp.ConversionCount++
	for i := range gwp.GiftOptions {
		if gwp.GiftOptions[i].ID == giftOptionID {
			gwp.GiftOptions[i].ClaimedCount++
			break
		}
	}
	gwp.UpdatedAt = time.Now()
}

// IncrementViewCount increments view count
func (gwp *GiftWithPurchase) IncrementViewCount() {
	gwp.ViewCount++
	gwp.UpdatedAt = time.Now()
}

// NewGWPClaim creates a new claim record
func NewGWPClaim(gwpID uuid.UUID, userID string, orderID uuid.UUID, giftOption GiftOption, tierIndex *int, triggerAmount decimal.Decimal) *GWPClaim {
	return &GWPClaim{
		ID:            uuid.New(),
		GWPID:         gwpID,
		UserID:        userID,
		OrderID:       orderID,
		GiftOptionID:  giftOption.ID,
		ProductID:     giftOption.ProductID,
		Quantity:      giftOption.Quantity,
		TierIndex:     tierIndex,
		TriggerAmount: triggerAmount,
		ClaimedAt:     time.Now(),
	}
}
