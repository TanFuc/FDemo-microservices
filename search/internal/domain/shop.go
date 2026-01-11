package domain

import "time"

// Shop represents a shop document for Elasticsearch indexing
type Shop struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Slug          string                 `json:"slug"`
	Description   string                 `json:"description,omitempty"`
	OwnerID       string                 `json:"ownerId"`

	// Display
	LogoURL       string                 `json:"logoUrl,omitempty"`
	BannerURL     string                 `json:"bannerUrl,omitempty"`

	// Location
	Address       string                 `json:"address,omitempty"`
	City          string                 `json:"city,omitempty"`
	Province      string                 `json:"province,omitempty"`
	Country       string                 `json:"country"`
	Location      *GeoLocation           `json:"location,omitempty"`

	// Categorization
	CategoryIDs   []string               `json:"categoryIds,omitempty"`
	Tags          []string               `json:"tags,omitempty"`

	// Ratings and reviews
	Rating        float64                `json:"rating"`
	ReviewCount   int64                  `json:"reviewCount"`
	ResponseRate  float64                `json:"responseRate,omitempty"` // % of queries responded

	// Stats
	ProductCount  int64                  `json:"productCount"`
	SoldCount     int64                  `json:"soldCount"`
	FollowerCount int64                  `json:"followerCount"`

	// Status
	Status        string                 `json:"status"` // ACTIVE, INACTIVE, VERIFIED
	IsVerified    bool                   `json:"isVerified"`
	IsOfficialStore bool                 `json:"isOfficialStore"`

	// Scoring
	PopularityScore float64              `json:"popularityScore"`
	QualityScore    float64              `json:"qualityScore"`

	// Business info
	BusinessType  string                 `json:"businessType,omitempty"` // INDIVIDUAL, BUSINESS
	YearsActive   int                    `json:"yearsActive,omitempty"`

	// Shipping
	FreeShipping  bool                   `json:"freeShipping"`
	ShipFrom      string                 `json:"shipFrom,omitempty"`

	// Metadata
	Metadata      map[string]interface{} `json:"metadata,omitempty"`

	// Timestamps
	JoinedAt      time.Time              `json:"joinedAt"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// GeoLocation represents a geographic location
type GeoLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// ShopSearchParams represents search query parameters for shops
type ShopSearchParams struct {
	Keyword       string                 `json:"keyword"`
	CategoryID    string                 `json:"categoryId,omitempty"`
	City          string                 `json:"city,omitempty"`
	Province      string                 `json:"province,omitempty"`
	MinRating     *float64               `json:"minRating,omitempty"`
	IsVerified    *bool                  `json:"isVerified,omitempty"`
	IsOfficial    *bool                  `json:"isOfficialStore,omitempty"`
	FreeShipping  *bool                  `json:"freeShipping,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Location      *GeoLocation           `json:"location,omitempty"`
	Distance      string                 `json:"distance,omitempty"` // e.g., "10km"
	SortBy        string                 `json:"sortBy,omitempty"`   // relevance, rating, soldCount, newest
	SortOrder     string                 `json:"sortOrder,omitempty"`
	Page          int                    `json:"page"`
	Limit         int                    `json:"limit"`
}

// ShopSearchResult represents paginated search results for shops
type ShopSearchResult struct {
	Shops      []Shop `json:"shops"`
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int    `json:"totalPages"`
	Facets     *ShopFacets `json:"facets,omitempty"`
}

// ShopFacets represents aggregation facets for shop search
type ShopFacets struct {
	Categories   []FacetBucket `json:"categories,omitempty"`
	Cities       []FacetBucket `json:"cities,omitempty"`
	RatingRanges []FacetBucket `json:"ratingRanges,omitempty"`
	Tags         []FacetBucket `json:"tags,omitempty"`
}

// FacetBucket represents a bucket in facet aggregation
type FacetBucket struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// ShopEvent represents a NATS event payload for shop changes
type ShopEvent struct {
	EventType string `json:"eventType"` // created, updated, deleted
	Shop      *Shop  `json:"shop,omitempty"`
	ShopID    string `json:"shopId,omitempty"`
}

// ShopAutocomplete represents autocomplete suggestion for shops
type ShopAutocomplete struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	LogoURL    string   `json:"logoUrl,omitempty"`
	Rating     float64  `json:"rating"`
	IsVerified bool     `json:"isVerified"`
	Category   string   `json:"category,omitempty"`
}
