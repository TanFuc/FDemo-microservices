package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FlashSaleStatus represents the status of a flash sale
type FlashSaleStatus string

const (
	FlashSaleStatusDraft     FlashSaleStatus = "DRAFT"
	FlashSaleStatusScheduled FlashSaleStatus = "SCHEDULED"
	FlashSaleStatusActive    FlashSaleStatus = "ACTIVE"
	FlashSaleStatusEnded     FlashSaleStatus = "ENDED"
	FlashSaleStatusCancelled FlashSaleStatus = "CANCELLED"
)

// FlashSale represents a flash sale event
type FlashSale struct {
	ID              uuid.UUID       `json:"id"`
	CampaignID      *uuid.UUID      `json:"campaign_id,omitempty"` // Parent campaign

	// Basic info
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	Description     string          `json:"description"`
	BannerURL       string          `json:"banner_url,omitempty"`
	MobileBannerURL string          `json:"mobile_banner_url,omitempty"`

	// Timing
	StartTime       time.Time       `json:"start_time"`
	EndTime         time.Time       `json:"end_time"`
	Timezone        string          `json:"timezone"`

	// Status
	Status          FlashSaleStatus `json:"status"`

	// Limits
	MaxProductsPerUser int          `json:"max_products_per_user"` // 0 = no limit
	MaxTotalOrders     int          `json:"max_total_orders"`      // 0 = no limit

	// Stats (denormalized)
	TotalProducts   int             `json:"total_products"`
	TotalSoldQty    int             `json:"total_sold_qty"`
	TotalRevenue    decimal.Decimal `json:"total_revenue"`

	// Visibility
	IsVisible       bool            `json:"is_visible"`
	IsFeatured      bool            `json:"is_featured"`
	SortOrder       int             `json:"sort_order"`

	// Targeting
	ShopIDs         []uuid.UUID     `json:"shop_ids,omitempty"` // Empty = all shops
	CategoryIDs     []string        `json:"category_ids,omitempty"`
	UserSegments    []string        `json:"user_segments,omitempty"`

	// Metadata
	Metadata        json.RawMessage `json:"metadata,omitempty"`

	// Audit
	CreatedBy       string          `json:"created_by"`
	UpdatedBy       string          `json:"updated_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// FlashSaleProduct represents a product in a flash sale
type FlashSaleProduct struct {
	ID              uuid.UUID       `json:"id"`
	FlashSaleID     uuid.UUID       `json:"flash_sale_id"`

	// Product info
	ProductID       string          `json:"product_id"`
	SkuID           string          `json:"sku_id,omitempty"` // Specific SKU or empty for all
	ShopID          uuid.UUID       `json:"shop_id"`

	// Pricing
	OriginalPrice   decimal.Decimal `json:"original_price"`
	FlashPrice      decimal.Decimal `json:"flash_price"`
	DiscountPercent decimal.Decimal `json:"discount_percent"`

	// Inventory
	TotalStock      int             `json:"total_stock"`
	SoldQty         int             `json:"sold_qty"`
	AvailableQty    int             `json:"available_qty"`
	MaxQtyPerUser   int             `json:"max_qty_per_user"` // 0 = no limit

	// Limits
	MinQtyPerOrder  int             `json:"min_qty_per_order"`
	MaxQtyPerOrder  int             `json:"max_qty_per_order"`

	// Display
	SortOrder       int             `json:"sort_order"`
	IsHighlighted   bool            `json:"is_highlighted"`

	// Status
	IsActive        bool            `json:"is_active"`

	// Stats
	ViewCount       int64           `json:"view_count"`
	CartAddCount    int64           `json:"cart_add_count"`

	// Metadata (product snapshot)
	ProductName     string          `json:"product_name"`
	ProductImage    string          `json:"product_image"`
	ProductSlug     string          `json:"product_slug"`

	// Timestamps
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// NewFlashSale creates a new flash sale
func NewFlashSale(name string, startTime, endTime time.Time, createdBy string) *FlashSale {
	now := time.Now()
	return &FlashSale{
		ID:                 uuid.New(),
		Name:               name,
		Slug:               GenerateSlug(name),
		StartTime:          startTime,
		EndTime:            endTime,
		Timezone:           "Asia/Ho_Chi_Minh",
		Status:             FlashSaleStatusDraft,
		MaxProductsPerUser: 0,
		MaxTotalOrders:     0,
		TotalProducts:      0,
		TotalSoldQty:       0,
		TotalRevenue:       decimal.Zero,
		IsVisible:          true,
		IsFeatured:         false,
		SortOrder:          0,
		ShopIDs:            make([]uuid.UUID, 0),
		CategoryIDs:        make([]string, 0),
		UserSegments:       make([]string, 0),
		Metadata:           json.RawMessage("{}"),
		CreatedBy:          createdBy,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// GenerateSlug generates a URL-safe slug
func GenerateSlug(name string) string {
	slug := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			slug += string(r)
		} else if r >= 'A' && r <= 'Z' {
			slug += string(r + 32)
		} else if r == ' ' || r == '-' || r == '_' {
			slug += "-"
		}
	}
	return slug
}

// Schedule schedules the flash sale
func (fs *FlashSale) Schedule() {
	fs.Status = FlashSaleStatusScheduled
	fs.UpdatedAt = time.Now()
}

// Start starts the flash sale
func (fs *FlashSale) Start() {
	fs.Status = FlashSaleStatusActive
	fs.UpdatedAt = time.Now()
}

// End ends the flash sale
func (fs *FlashSale) End() {
	fs.Status = FlashSaleStatusEnded
	fs.UpdatedAt = time.Now()
}

// Cancel cancels the flash sale
func (fs *FlashSale) Cancel() {
	fs.Status = FlashSaleStatusCancelled
	fs.UpdatedAt = time.Now()
}

// IsLive checks if the flash sale is currently live
func (fs *FlashSale) IsLive() bool {
	now := time.Now()
	return fs.Status == FlashSaleStatusActive &&
		now.After(fs.StartTime) && now.Before(fs.EndTime)
}

// HasStarted checks if the flash sale has started
func (fs *FlashSale) HasStarted() bool {
	return time.Now().After(fs.StartTime)
}

// HasEnded checks if the flash sale has ended
func (fs *FlashSale) HasEnded() bool {
	return time.Now().After(fs.EndTime)
}

// TimeRemaining returns the time remaining until end
func (fs *FlashSale) TimeRemaining() time.Duration {
	if fs.HasEnded() {
		return 0
	}
	return fs.EndTime.Sub(time.Now())
}

// TimeUntilStart returns the time until start
func (fs *FlashSale) TimeUntilStart() time.Duration {
	if fs.HasStarted() {
		return 0
	}
	return fs.StartTime.Sub(time.Now())
}

// IncrementSales increments the sales count
func (fs *FlashSale) IncrementSales(qty int, revenue decimal.Decimal) {
	fs.TotalSoldQty += qty
	fs.TotalRevenue = fs.TotalRevenue.Add(revenue)
	fs.UpdatedAt = time.Now()
}

// NewFlashSaleProduct creates a new flash sale product
func NewFlashSaleProduct(flashSaleID uuid.UUID, productID string, shopID uuid.UUID, originalPrice, flashPrice decimal.Decimal, totalStock int) *FlashSaleProduct {
	now := time.Now()

	discountPercent := decimal.Zero
	if !originalPrice.IsZero() {
		discountPercent = originalPrice.Sub(flashPrice).Div(originalPrice).Mul(decimal.NewFromInt(100))
	}

	return &FlashSaleProduct{
		ID:              uuid.New(),
		FlashSaleID:     flashSaleID,
		ProductID:       productID,
		ShopID:          shopID,
		OriginalPrice:   originalPrice,
		FlashPrice:      flashPrice,
		DiscountPercent: discountPercent,
		TotalStock:      totalStock,
		SoldQty:         0,
		AvailableQty:    totalStock,
		MaxQtyPerUser:   0,
		MinQtyPerOrder:  1,
		MaxQtyPerOrder:  0,
		SortOrder:       0,
		IsHighlighted:   false,
		IsActive:        true,
		ViewCount:       0,
		CartAddCount:    0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// SetProductSnapshot sets the product snapshot data
func (fp *FlashSaleProduct) SetProductSnapshot(name, image, slug string) {
	fp.ProductName = name
	fp.ProductImage = image
	fp.ProductSlug = slug
	fp.UpdatedAt = time.Now()
}

// RecordSale records a sale
func (fp *FlashSaleProduct) RecordSale(qty int) bool {
	if qty > fp.AvailableQty {
		return false
	}
	fp.SoldQty += qty
	fp.AvailableQty = fp.TotalStock - fp.SoldQty
	fp.UpdatedAt = time.Now()
	return true
}

// RestoreStock restores stock (for cancelled orders)
func (fp *FlashSaleProduct) RestoreStock(qty int) {
	fp.SoldQty -= qty
	if fp.SoldQty < 0 {
		fp.SoldQty = 0
	}
	fp.AvailableQty = fp.TotalStock - fp.SoldQty
	fp.UpdatedAt = time.Now()
}

// IsAvailable checks if the product is available for purchase
func (fp *FlashSaleProduct) IsAvailable() bool {
	return fp.IsActive && fp.AvailableQty > 0
}

// IsSoldOut checks if the product is sold out
func (fp *FlashSaleProduct) IsSoldOut() bool {
	return fp.AvailableQty <= 0
}

// GetSoldPercentage returns the percentage of stock sold
func (fp *FlashSaleProduct) GetSoldPercentage() float64 {
	if fp.TotalStock == 0 {
		return 0
	}
	return float64(fp.SoldQty) / float64(fp.TotalStock) * 100
}

// IncrementViewCount increments the view count
func (fp *FlashSaleProduct) IncrementViewCount() {
	fp.ViewCount++
	fp.UpdatedAt = time.Now()
}

// IncrementCartAddCount increments the cart add count
func (fp *FlashSaleProduct) IncrementCartAddCount() {
	fp.CartAddCount++
	fp.UpdatedAt = time.Now()
}

// Deactivate deactivates the product
func (fp *FlashSaleProduct) Deactivate() {
	fp.IsActive = false
	fp.UpdatedAt = time.Now()
}

// Activate activates the product
func (fp *FlashSaleProduct) Activate() {
	fp.IsActive = true
	fp.UpdatedAt = time.Now()
}
