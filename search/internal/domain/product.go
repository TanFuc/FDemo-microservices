package domain

import "time"

// Product represents a product document for Elasticsearch indexing
type Product struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Slug       string                 `json:"slug"`
	CategoryID string                 `json:"categoryId"`
	BrandID    string                 `json:"brandId"`
	Price      float64                `json:"price"`
	Thumbnail  string                 `json:"thumbnail"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"createdAt"`
	Specs      map[string]interface{} `json:"specs,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SearchParams represents search query parameters
type SearchParams struct {
	Keyword    string                 `json:"keyword"`
	CategoryID string                 `json:"categoryId,omitempty"`
	BrandID    string                 `json:"brandId,omitempty"`
	PriceMin   *float64               `json:"priceMin,omitempty"`
	PriceMax   *float64               `json:"priceMax,omitempty"`
	Specs      map[string]interface{} `json:"specs,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	SortBy     string                 `json:"sortBy,omitempty"`
	SortOrder  string                 `json:"sortOrder,omitempty"`
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
