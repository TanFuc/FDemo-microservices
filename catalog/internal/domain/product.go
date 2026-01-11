package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product Status constants
const (
	ProductStatusDraft          = "DRAFT"
	ProductStatusPendingReview  = "PENDING_REVIEW"
	ProductStatusActive         = "ACTIVE"
	ProductStatusRejected       = "REJECTED"
	ProductStatusArchived       = "ARCHIVED"
)

// Product Visibility constants
const (
	ProductVisibilityPublic  = "PUBLIC"
	ProductVisibilityPrivate = "PRIVATE"
	ProductVisibilityHidden  = "HIDDEN"
)

// Shipping Class constants
const (
	ShippingClassStandard = "STANDARD"
	ShippingClassBulky    = "BULKY"
	ShippingClassFragile  = "FRAGILE"
	ShippingClassHazmat   = "HAZMAT"
)

// ProductImage represents a product image with metadata
type ProductImage struct {
	URL       string `bson:"url" json:"url"`
	Alt       string `bson:"alt,omitempty" json:"alt,omitempty"`
	Position  int    `bson:"position" json:"position"`
	Width     int    `bson:"width,omitempty" json:"width,omitempty"`
	Height    int    `bson:"height,omitempty" json:"height,omitempty"`
	IsPrimary bool   `bson:"isPrimary" json:"isPrimary"`
}

// ProductVideo represents a product video
type ProductVideo struct {
	URL          string `bson:"url" json:"url"`
	ThumbnailURL string `bson:"thumbnailUrl,omitempty" json:"thumbnailUrl,omitempty"`
	Duration     int    `bson:"duration,omitempty" json:"duration,omitempty"` // seconds
	Provider     string `bson:"provider,omitempty" json:"provider,omitempty"` // youtube, vimeo, hosted
}

// Dimensions represents product dimensions
type Dimensions struct {
	Length float64 `bson:"length" json:"length"` // cm
	Width  float64 `bson:"width" json:"width"`   // cm
	Height float64 `bson:"height" json:"height"` // cm
}

// Specification represents a product specification
type Specification struct {
	Group string `bson:"group,omitempty" json:"group,omitempty"` // Group name for display
	Key   string `bson:"key" json:"key"`
	Name  string `bson:"name" json:"name"`
	Value string `bson:"value" json:"value"`
	Unit  string `bson:"unit,omitempty" json:"unit,omitempty"`
	Order int    `bson:"order" json:"order"`
}

// VariantOption defines options for creating variants (e.g., Color, Size)
type VariantOption struct {
	Name    string   `bson:"name" json:"name"`       // e.g., "Color", "Size"
	Options []string `bson:"options" json:"options"` // e.g., ["Red", "Blue", "Green"]
	Images  []string `bson:"images,omitempty" json:"images,omitempty"` // Optional images per option
}

// Variation represents a product SKU/variant
type Variation struct {
	SKU     string `bson:"sku" json:"sku"`
	Barcode string `bson:"barcode,omitempty" json:"barcode,omitempty"` // EAN, UPC

	// Variant Attributes
	Attributes map[string]string `bson:"attributes" json:"attributes"` // {"color": "Red", "size": "M"}
	TierIndex  []int             `bson:"tierIndex" json:"tierIndex"`   // [0, 1] for color[0] + size[1]

	// Pricing
	Price     decimal.Decimal `bson:"price" json:"price"`
	CompareAt decimal.Decimal `bson:"compareAt,omitempty" json:"compareAt,omitempty"`
	CostPrice decimal.Decimal `bson:"costPrice,omitempty" json:"costPrice,omitempty"`

	// Inventory (denormalized from Inventory Service)
	Stock     int `bson:"stock" json:"stock"`
	Reserved  int `bson:"reserved" json:"reserved"`
	Available int `bson:"available" json:"available"` // stock - reserved

	// Media
	ImageURL string `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`

	// Shipping overrides
	Weight     float64     `bson:"weight,omitempty" json:"weight,omitempty"` // grams
	Dimensions *Dimensions `bson:"dimensions,omitempty" json:"dimensions,omitempty"`

	// Status
	IsActive bool `bson:"isActive" json:"isActive"`

	// Metadata
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// EnsureDefaults initializes default values for Variation
func (v *Variation) EnsureDefaults() {
	if v.Attributes == nil {
		v.Attributes = make(map[string]string)
	}
	if v.Metadata == nil {
		v.Metadata = make(map[string]interface{})
	}
	if v.TierIndex == nil {
		v.TierIndex = []int{}
	}
	v.Available = v.Stock - v.Reserved
	if v.Price.IsZero() {
		v.Price = decimal.Zero
	}
}

// Product represents a full e-commerce product
type Product struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// Ownership
	ShopID   string `bson:"shopId" json:"shopId"`
	ShopName string `bson:"shopName" json:"shopName"` // Denormalized

	// Basic Info
	Name             string `bson:"name" json:"name"`
	Slug             string `bson:"slug" json:"slug"`
	Description      string `bson:"description" json:"description"`
	ShortDescription string `bson:"shortDescription,omitempty" json:"shortDescription,omitempty"`

	// Categorization
	CategoryID   primitive.ObjectID `bson:"categoryId" json:"categoryId"`
	CategoryPath string             `bson:"categoryPath" json:"categoryPath"` // For filtering
	BrandID      primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	BrandName    string             `bson:"brandName,omitempty" json:"brandName,omitempty"` // Denormalized

	// Media
	Thumbnail string         `bson:"thumbnail" json:"thumbnail"`
	Images    []ProductImage `bson:"images" json:"images"`
	Videos    []ProductVideo `bson:"videos,omitempty" json:"videos,omitempty"`
	Model3D   string         `bson:"model3d,omitempty" json:"model3d,omitempty"` // AR/3D preview

	// Pricing
	BasePrice      decimal.Decimal `bson:"basePrice" json:"basePrice"`
	CompareAtPrice decimal.Decimal `bson:"compareAtPrice,omitempty" json:"compareAtPrice,omitempty"` // Original price
	CostPrice      decimal.Decimal `bson:"costPrice,omitempty" json:"costPrice,omitempty"`           // For profit calculation
	MinPrice       decimal.Decimal `bson:"minPrice" json:"minPrice"`                                  // Lowest SKU price
	MaxPrice       decimal.Decimal `bson:"maxPrice" json:"maxPrice"`                                  // Highest SKU price
	Currency       string          `bson:"currency" json:"currency"`
	TaxRate        decimal.Decimal `bson:"taxRate,omitempty" json:"taxRate,omitempty"`
	IsTaxInclusive bool            `bson:"isTaxInclusive" json:"isTaxInclusive"`

	// Variants
	VariantOptions []VariantOption `bson:"variantOptions,omitempty" json:"variantOptions,omitempty"`
	Variations     []Variation     `bson:"variations" json:"variations"`
	HasVariants    bool            `bson:"hasVariants" json:"hasVariants"`

	// Inventory
	TotalStock        int  `bson:"totalStock" json:"totalStock"` // Sum of all SKUs
	TrackInventory    bool `bson:"trackInventory" json:"trackInventory"`
	AllowBackorder    bool `bson:"allowBackorder" json:"allowBackorder"`
	LowStockThreshold int  `bson:"lowStockThreshold" json:"lowStockThreshold"`

	// Attributes & Specs
	Attributes     map[string]interface{} `bson:"attributes,omitempty" json:"attributes,omitempty"`
	Specifications []Specification        `bson:"specifications,omitempty" json:"specifications,omitempty"`

	// Shipping
	Weight        float64     `bson:"weight" json:"weight"` // grams
	Dimensions    *Dimensions `bson:"dimensions,omitempty" json:"dimensions,omitempty"`
	ShippingClass string      `bson:"shippingClass" json:"shippingClass"` // STANDARD, BULKY, FRAGILE
	IsFreeShipping bool       `bson:"isFreeShipping" json:"isFreeShipping"`

	// Digital Products
	IsDigital      bool   `bson:"isDigital" json:"isDigital"`
	DigitalFileURL string `bson:"digitalFileUrl,omitempty" json:"digitalFileUrl,omitempty"`
	MaxDownloads   int    `bson:"maxDownloads,omitempty" json:"maxDownloads,omitempty"`

	// Pre-order
	IsPreOrder      bool       `bson:"isPreOrder" json:"isPreOrder"`
	PreOrderMessage string     `bson:"preOrderMessage,omitempty" json:"preOrderMessage,omitempty"`
	ReleaseDate     *time.Time `bson:"releaseDate,omitempty" json:"releaseDate,omitempty"`

	// SEO
	MetaTitle       string `bson:"metaTitle,omitempty" json:"metaTitle,omitempty"`
	MetaDescription string `bson:"metaDescription,omitempty" json:"metaDescription,omitempty"`
	MetaKeywords    string `bson:"metaKeywords,omitempty" json:"metaKeywords,omitempty"`
	CanonicalURL    string `bson:"canonicalUrl,omitempty" json:"canonicalUrl,omitempty"`

	// Status & Visibility
	Status          string `bson:"status" json:"status"`         // DRAFT, PENDING_REVIEW, ACTIVE, REJECTED, ARCHIVED
	Visibility      string `bson:"visibility" json:"visibility"` // PUBLIC, PRIVATE, HIDDEN
	RejectionReason string `bson:"rejectionReason,omitempty" json:"rejectionReason,omitempty"`

	// Scheduling
	PublishedAt *time.Time `bson:"publishedAt,omitempty" json:"publishedAt,omitempty"`
	ScheduledAt *time.Time `bson:"scheduledAt,omitempty" json:"scheduledAt,omitempty"`

	// Promotions & Badges
	Badges   []string               `bson:"badges,omitempty" json:"badges,omitempty"` // NEW, BESTSELLER, SALE, LIMITED
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Tags     []string               `bson:"tags,omitempty" json:"tags,omitempty"`

	// Collections/Groups
	CollectionIDs []primitive.ObjectID `bson:"collectionIds,omitempty" json:"collectionIds,omitempty"`

	// Related Products
	RelatedProductIDs   []primitive.ObjectID `bson:"relatedProductIds,omitempty" json:"relatedProductIds,omitempty"`
	CrossSellProductIDs []primitive.ObjectID `bson:"crossSellProductIds,omitempty" json:"crossSellProductIds,omitempty"`
	UpSellProductIDs    []primitive.ObjectID `bson:"upSellProductIds,omitempty" json:"upSellProductIds,omitempty"`

	// Stats (Denormalized for performance)
	ViewCount     int64   `bson:"viewCount" json:"viewCount"`
	SoldCount     int64   `bson:"soldCount" json:"soldCount"`
	WishlistCount int64   `bson:"wishlistCount" json:"wishlistCount"`
	Rating        float64 `bson:"rating" json:"rating"`
	ReviewCount   int64   `bson:"reviewCount" json:"reviewCount"`

	// Scoring (for search ranking)
	PopularityScore float64 `bson:"popularityScore" json:"popularityScore"`
	QualityScore    float64 `bson:"qualityScore" json:"qualityScore"`

	// Audit
	CreatedBy  string     `bson:"createdBy" json:"createdBy"`
	UpdatedBy  string     `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	ApprovedBy string     `bson:"approvedBy,omitempty" json:"approvedBy,omitempty"`
	ApprovedAt *time.Time `bson:"approvedAt,omitempty" json:"approvedAt,omitempty"`

	// Timestamps
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
}

// EnsureDefaults initializes default values for Product
func (p *Product) EnsureDefaults() {
	if p.Images == nil {
		p.Images = []ProductImage{}
	}
	if p.Videos == nil {
		p.Videos = []ProductVideo{}
	}
	if p.Attributes == nil {
		p.Attributes = make(map[string]interface{})
	}
	if p.Specifications == nil {
		p.Specifications = []Specification{}
	}
	if p.VariantOptions == nil {
		p.VariantOptions = []VariantOption{}
	}
	if p.Variations == nil {
		p.Variations = []Variation{}
	}
	if p.Metadata == nil {
		p.Metadata = make(map[string]interface{})
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.Badges == nil {
		p.Badges = []string{}
	}
	if p.CollectionIDs == nil {
		p.CollectionIDs = []primitive.ObjectID{}
	}
	if p.RelatedProductIDs == nil {
		p.RelatedProductIDs = []primitive.ObjectID{}
	}
	if p.CrossSellProductIDs == nil {
		p.CrossSellProductIDs = []primitive.ObjectID{}
	}
	if p.UpSellProductIDs == nil {
		p.UpSellProductIDs = []primitive.ObjectID{}
	}
	if p.Status == "" {
		p.Status = ProductStatusDraft
	}
	if p.Visibility == "" {
		p.Visibility = ProductVisibilityPublic
	}
	if p.ShippingClass == "" {
		p.ShippingClass = ShippingClassStandard
	}
	if p.Currency == "" {
		p.Currency = "VND"
	}
	p.HasVariants = len(p.Variations) > 0
	for i := range p.Variations {
		p.Variations[i].EnsureDefaults()
	}
}

// CalculatePriceRange calculates min and max prices from variations
func (p *Product) CalculatePriceRange() {
	if len(p.Variations) == 0 {
		p.MinPrice = p.BasePrice
		p.MaxPrice = p.BasePrice
		return
	}

	p.MinPrice = p.Variations[0].Price
	p.MaxPrice = p.Variations[0].Price

	for _, v := range p.Variations {
		if v.Price.LessThan(p.MinPrice) {
			p.MinPrice = v.Price
		}
		if v.Price.GreaterThan(p.MaxPrice) {
			p.MaxPrice = v.Price
		}
	}
}

// CalculateTotalStock calculates total stock from all variations
func (p *Product) CalculateTotalStock() {
	total := 0
	for _, v := range p.Variations {
		total += v.Stock
	}
	p.TotalStock = total
}

// IsAvailable returns true if the product is available for purchase
func (p *Product) IsAvailable() bool {
	if p.Status != ProductStatusActive {
		return false
	}
	if p.Visibility != ProductVisibilityPublic {
		return false
	}
	if p.DeletedAt != nil {
		return false
	}
	if p.TrackInventory && p.TotalStock <= 0 && !p.AllowBackorder {
		return false
	}
	return true
}

// GetDefaultVariation returns the first active variation
func (p *Product) GetDefaultVariation() *Variation {
	for i := range p.Variations {
		if p.Variations[i].IsActive {
			return &p.Variations[i]
		}
	}
	if len(p.Variations) > 0 {
		return &p.Variations[0]
	}
	return nil
}
