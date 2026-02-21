package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Category Status constants
const (
	CategoryStatusActive   = "ACTIVE"
	CategoryStatusInactive = "INACTIVE"
	CategoryStatusHidden   = "HIDDEN"
)

// Attribute Types
const (
	AttributeTypeText        = "TEXT"
	AttributeTypeNumber      = "NUMBER"
	AttributeTypeSelect      = "SELECT"
	AttributeTypeMultiSelect = "MULTI_SELECT"
	AttributeTypeBoolean     = "BOOLEAN"
	AttributeTypeColor       = "COLOR"
	AttributeTypeSize        = "SIZE"
)

// AttributeOption represents a selectable option for an attribute
type AttributeOption struct {
	Value string            `bson:"value" json:"value"`
	Label map[string]string `bson:"label" json:"label"`
	Color string            `bson:"color,omitempty" json:"color,omitempty"`
	Order int               `bson:"order" json:"order"`
}

// AttributeValidation defines validation rules for attributes
type AttributeValidation struct {
	Min       *float64 `bson:"min,omitempty" json:"min,omitempty"`
	Max       *float64 `bson:"max,omitempty" json:"max,omitempty"`
	Pattern   string   `bson:"pattern,omitempty" json:"pattern,omitempty"`
	MinLength *int     `bson:"minLength,omitempty" json:"minLength,omitempty"`
	MaxLength *int     `bson:"maxLength,omitempty" json:"maxLength,omitempty"`
}

// AttributeDefinition defines a product attribute for a category
type AttributeDefinition struct {
	Key          string               `bson:"key" json:"key"`
	Name         map[string]string    `bson:"name" json:"name"`
	Type         string               `bson:"type" json:"type"`
	Unit         string               `bson:"unit,omitempty" json:"unit,omitempty"`
	IsRequired   bool                 `bson:"isRequired" json:"isRequired"`
	IsFilterable bool                 `bson:"isFilterable" json:"isFilterable"`
	IsSearchable bool                 `bson:"isSearchable" json:"isSearchable"`
	IsVariant    bool                 `bson:"isVariant" json:"isVariant"`
	Options      []AttributeOption    `bson:"options,omitempty" json:"options,omitempty"`
	Validation   *AttributeValidation `bson:"validation,omitempty" json:"validation,omitempty"`
	Order        int                  `bson:"order" json:"order"`
}

// Category represents a product category in the catalog
type Category struct {
	ID       primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ParentID *primitive.ObjectID `bson:"parentId,omitempty" json:"parentId"`
	Path     string              `bson:"path" json:"path"`
	Level    int                 `bson:"level" json:"level"`
	Position int                 `bson:"position" json:"position"`

	// Basic Info (multilingual)
	Name        map[string]string `bson:"name" json:"name"`
	Slug        string            `bson:"slug" json:"slug"`
	Description map[string]string `bson:"description,omitempty" json:"description,omitempty"`

	// Media
	ImageURL  string `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	IconURL   string `bson:"iconUrl,omitempty" json:"iconUrl,omitempty"`
	BannerURL string `bson:"bannerUrl,omitempty" json:"bannerUrl,omitempty"`

	// SEO (multilingual)
	MetaTitle       map[string]string `bson:"metaTitle,omitempty" json:"metaTitle,omitempty"`
	MetaDescription map[string]string `bson:"metaDescription,omitempty" json:"metaDescription,omitempty"`
	MetaKeywords    map[string]string `bson:"metaKeywords,omitempty" json:"metaKeywords,omitempty"`

	// Attributes
	AttributeDefinitions []AttributeDefinition `bson:"attributeDefinitions" json:"attributeDefinitions"`

	// Commission
	CommissionRate float64 `bson:"commissionRate" json:"commissionRate"`

	// Status
	Status     string `bson:"status" json:"status"`
	IsVisible  bool   `bson:"isVisible" json:"isVisible"`
	IsFeatured bool   `bson:"isFeatured" json:"isFeatured"`

	// Stats
	ProductCount   int `bson:"productCount" json:"productCount"`
	ActiveProducts int `bson:"activeProducts" json:"activeProducts"`

	// Timestamps
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
}

// EnsureDefaults initializes default values for Category
func (c *Category) EnsureDefaults() {
	if c.Name == nil {
		c.Name = make(map[string]string)
	}
	if c.Description == nil {
		c.Description = make(map[string]string)
	}
	if c.AttributeDefinitions == nil {
		c.AttributeDefinitions = []AttributeDefinition{}
	}
	if c.Status == "" {
		c.Status = CategoryStatusActive
	}
	c.IsVisible = true
}

// GetName returns the category name in the specified language
func (c *Category) GetName(lang string) string {
	if name, ok := c.Name[lang]; ok {
		return name
	}
	if name, ok := c.Name["en"]; ok {
		return name
	}
	return ""
}

// IsActive returns true if the category is active and visible
func (c *Category) IsActive() bool {
	return c.Status == CategoryStatusActive && c.IsVisible && c.DeletedAt == nil
}

// BuildPath builds the category path from parent categories
func (c *Category) BuildPath(parentPath string) {
	if parentPath == "" {
		c.Path = "/" + c.Slug
	} else {
		c.Path = parentPath + "/" + c.Slug
	}
}

// TableName returns the MongoDB collection name
func (Category) TableName() string {
	return "categories"
}
