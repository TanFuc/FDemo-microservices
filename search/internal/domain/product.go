package domain

import "time"

// Specification represents product specs for nested filtering
type Specification struct {
	Group string `json:"group,omitempty"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
	Order int    `json:"order"`
}

// Product represents a product document for Elasticsearch indexing
type Product struct {
	ID               string                 `json:"id"`
	ShopID           string                 `json:"shopId,omitempty"`
	ShopName         string                 `json:"shopName,omitempty"`
	Name             string                 `json:"name"`
	Slug             string                 `json:"slug"`
	Description      string                 `json:"description,omitempty"`
	ShortDescription string                 `json:"shortDescription,omitempty"`
	CategoryID       string                 `json:"categoryId"`
	CategoryName     string                 `json:"categoryName,omitempty"`
	CategoryPath     string                 `json:"categoryPath,omitempty"`
	BrandID          string                 `json:"brandId"`
	BrandName        string                 `json:"brandName,omitempty"`
	Thumbnail        string                 `json:"thumbnail"`
	Images           []string               `json:"images,omitempty"`
	BasePrice        float64                `json:"basePrice"`
	MinPrice         float64                `json:"minPrice"`
	MaxPrice         float64                `json:"maxPrice"`
	Currency         string                 `json:"currency,omitempty"`
	Status           string                 `json:"status"`
	Visibility       string                 `json:"visibility,omitempty"`
	TotalStock       int                    `json:"totalStock"`
	SoldCount        int                    `json:"soldCount"`
	ViewCount        int                    `json:"viewCount"`
	Rating           float64                `json:"rating"`
	ReviewCount      int                    `json:"reviewCount"`
	HasVariants      bool                   `json:"hasVariants"`
	IsDigital        bool                   `json:"isDigital"`
	IsFreeShipping   bool                   `json:"isFreeShipping"`
	Weight           float64                `json:"weight,omitempty"`
	Specifications   []Specification        `json:"specifications,omitempty"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
	PublishedAt      *time.Time             `json:"publishedAt,omitempty"`
}

// SearchParams represents search query parameters
type SearchParams struct {
	Keyword    string                 `json:"keyword"`
	CategoryID string                 `json:"categoryId,omitempty"`
	BrandID    string                 `json:"brandId,omitempty"`
	ShopID     string                 `json:"shopId,omitempty"`
	PriceMin   *float64               `json:"priceMin,omitempty"`
	PriceMax   *float64               `json:"priceMax,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Specs      map[string]interface{} `json:"specs,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	SortBy     string                 `json:"sortBy,omitempty"`  // price, createdAt, rating, soldCount
	SortOrder  string                 `json:"sortOrder,omitempty"` // asc, desc
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
}

// SearchResult represents paginated search results
type SearchResult struct {
	Products   []Product `json:"products"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	TotalPages int       `json:"totalPages"`
}

// ProductEvent represents a NATS event payload for product changes
type ProductEvent struct {
	EventType string   `json:"eventType"`
	Product   *Product `json:"product,omitempty"`
	ProductID string   `json:"productId,omitempty"`
}
