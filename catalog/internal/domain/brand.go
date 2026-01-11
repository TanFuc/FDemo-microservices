package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Brand Status constants
const (
	BrandStatusActive   = "ACTIVE"
	BrandStatusInactive = "INACTIVE"
	BrandStatusPending  = "PENDING"
)

// Brand represents a product brand in the catalog
type Brand struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// Basic Info
	Name        string            `bson:"name" json:"name"`
	Slug        string            `bson:"slug" json:"slug"`
	Description map[string]string `bson:"description,omitempty" json:"description,omitempty"` // Multilingual

	// Media
	LogoURL   string `bson:"logoUrl,omitempty" json:"logoUrl,omitempty"`
	BannerURL string `bson:"bannerUrl,omitempty" json:"bannerUrl,omitempty"`

	// Details
	Website     string `bson:"website,omitempty" json:"website,omitempty"`
	Country     string `bson:"country,omitempty" json:"country,omitempty"`
	FoundedYear int    `bson:"foundedYear,omitempty" json:"foundedYear,omitempty"`

	// SEO
	MetaTitle       string `bson:"metaTitle,omitempty" json:"metaTitle,omitempty"`
	MetaDescription string `bson:"metaDescription,omitempty" json:"metaDescription,omitempty"`

	// Status & Visibility
	Status     string `bson:"status" json:"status"` // ACTIVE, INACTIVE, PENDING
	IsVerified bool   `bson:"isVerified" json:"isVerified"`
	IsFeatured bool   `bson:"isFeatured" json:"isFeatured"`

	// Stats (denormalized)
	ProductCount   int     `bson:"productCount" json:"productCount"`
	ActiveProducts int     `bson:"activeProducts" json:"activeProducts"`
	AverageRating  float64 `bson:"averageRating" json:"averageRating"`
	TotalReviews   int64   `bson:"totalReviews" json:"totalReviews"`

	// Position for sorting
	Position int `bson:"position" json:"position"`

	// Metadata
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
}

// EnsureDefaults initializes default values for Brand
func (b *Brand) EnsureDefaults() {
	if b.Status == "" {
		b.Status = BrandStatusActive
	}
	if b.Description == nil {
		b.Description = make(map[string]string)
	}
	if b.Metadata == nil {
		b.Metadata = make(map[string]interface{})
	}
}

// IsActive returns true if the brand is active
func (b *Brand) IsActive() bool {
	return b.Status == BrandStatusActive && b.DeletedAt == nil
}

// GetDescription returns the brand description in the specified language
func (b *Brand) GetDescription(lang string) string {
	if desc, ok := b.Description[lang]; ok {
		return desc
	}
	if desc, ok := b.Description["en"]; ok {
		return desc
	}
	return ""
}

// Collection represents a product collection/group
type Collection struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// Basic Info
	Name        map[string]string `bson:"name" json:"name"` // Multilingual
	Slug        string            `bson:"slug" json:"slug"`
	Description map[string]string `bson:"description,omitempty" json:"description,omitempty"`

	// Media
	ImageURL  string `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	BannerURL string `bson:"bannerUrl,omitempty" json:"bannerUrl,omitempty"`

	// Type
	Type string `bson:"type" json:"type"` // MANUAL, AUTOMATED

	// Automated Collection Rules
	Rules []CollectionRule `bson:"rules,omitempty" json:"rules,omitempty"`

	// Manual Product IDs
	ProductIDs []primitive.ObjectID `bson:"productIds,omitempty" json:"productIds,omitempty"`

	// SEO
	MetaTitle       string `bson:"metaTitle,omitempty" json:"metaTitle,omitempty"`
	MetaDescription string `bson:"metaDescription,omitempty" json:"metaDescription,omitempty"`

	// Status
	Status     string `bson:"status" json:"status"` // ACTIVE, INACTIVE
	IsVisible  bool   `bson:"isVisible" json:"isVisible"`
	IsFeatured bool   `bson:"isFeatured" json:"isFeatured"`

	// Scheduling
	StartDate *time.Time `bson:"startDate,omitempty" json:"startDate,omitempty"`
	EndDate   *time.Time `bson:"endDate,omitempty" json:"endDate,omitempty"`

	// Stats
	ProductCount int `bson:"productCount" json:"productCount"`

	// Position
	Position int `bson:"position" json:"position"`

	// Timestamps
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
}

// CollectionRule defines rules for automated collections
type CollectionRule struct {
	Field    string      `bson:"field" json:"field"`       // category, brand, price, tags, etc.
	Operator string      `bson:"operator" json:"operator"` // equals, contains, greater_than, less_than, in
	Value    interface{} `bson:"value" json:"value"`
}

// EnsureDefaults initializes default values for Collection
func (c *Collection) EnsureDefaults() {
	if c.Name == nil {
		c.Name = make(map[string]string)
	}
	if c.Description == nil {
		c.Description = make(map[string]string)
	}
	if c.Rules == nil {
		c.Rules = []CollectionRule{}
	}
	if c.ProductIDs == nil {
		c.ProductIDs = []primitive.ObjectID{}
	}
	if c.Status == "" {
		c.Status = "ACTIVE"
	}
	if c.Type == "" {
		c.Type = "MANUAL"
	}
	c.IsVisible = true
}

// IsActive returns true if the collection is currently active
func (c *Collection) IsActive() bool {
	if c.Status != "ACTIVE" || !c.IsVisible || c.DeletedAt != nil {
		return false
	}
	now := time.Now()
	if c.StartDate != nil && now.Before(*c.StartDate) {
		return false
	}
	if c.EndDate != nil && now.After(*c.EndDate) {
		return false
	}
	return true
}
