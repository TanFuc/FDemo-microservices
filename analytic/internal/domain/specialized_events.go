package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProductEvent represents a product-related analytics event
type ProductEvent struct {
	EventID      uuid.UUID `json:"event_id"`
	UserID       string    `json:"user_id"`
	SessionID    string    `json:"session_id"`
	ProductID    string    `json:"product_id"`
	SkuID        string    `json:"sku_id,omitempty"`
	ShopID       string    `json:"shop_id"`
	CategoryID   string    `json:"category_id"`
	BrandID      string    `json:"brand_id,omitempty"`

	// Event type
	EventType    string    `json:"event_type"` // view, add_to_cart, remove_from_cart, purchase, wishlist_add, wishlist_remove

	// Product context
	Price        float64   `json:"price"`
	Quantity     int       `json:"quantity"`
	Currency     string    `json:"currency"`

	// Source/attribution
	Source       string    `json:"source"`       // search, category, recommendation, direct
	RefProductID string    `json:"ref_product_id,omitempty"` // Referring product (for recommendations)
	SearchQuery  string    `json:"search_query,omitempty"`   // If from search
	Position     int       `json:"position,omitempty"`       // Position in list

	// User context
	DeviceType   string    `json:"device_type"`  // mobile, desktop, tablet
	Platform     string    `json:"platform"`     // ios, android, web

	// Request info
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Country      string    `json:"country,omitempty"`
	City         string    `json:"city,omitempty"`

	// Timestamp
	CreatedAt    time.Time `json:"created_at"`
}

// SearchQuery represents a search query analytics event
type SearchQuery struct {
	EventID        uuid.UUID `json:"event_id"`
	UserID         string    `json:"user_id"`
	SessionID      string    `json:"session_id"`

	// Query info
	Query          string    `json:"query"`
	NormalizedQuery string   `json:"normalized_query"` // Lowercase, trimmed
	QueryLength    int       `json:"query_length"`
	QueryType      string    `json:"query_type"` // keyword, category, brand, autocomplete

	// Results
	TotalResults   int       `json:"total_results"`
	ResultsPage    int       `json:"results_page"`
	ResultsLimit   int       `json:"results_limit"`
	HasResults     bool      `json:"has_results"`

	// Filters applied
	CategoryFilter string    `json:"category_filter,omitempty"`
	BrandFilter    string    `json:"brand_filter,omitempty"`
	PriceMinFilter float64   `json:"price_min_filter,omitempty"`
	PriceMaxFilter float64   `json:"price_max_filter,omitempty"`
	SortBy         string    `json:"sort_by,omitempty"`

	// Interaction
	ClickedProductID string  `json:"clicked_product_id,omitempty"` // First clicked product
	ClickPosition    int     `json:"click_position,omitempty"`
	TimeToClick      int64   `json:"time_to_click_ms,omitempty"` // Milliseconds until first click
	ClickedCount     int     `json:"clicked_count"`              // Total products clicked

	// Conversion
	AddedToCart     bool     `json:"added_to_cart"`
	Purchased       bool     `json:"purchased"`

	// Performance
	ResponseTimeMs int64     `json:"response_time_ms"`

	// User context
	DeviceType     string    `json:"device_type"`
	Platform       string    `json:"platform"`
	IPAddress      string    `json:"ip_address"`
	Country        string    `json:"country,omitempty"`

	// Timestamp
	CreatedAt      time.Time `json:"created_at"`
}

// OrderEvent represents an order-related analytics event
type OrderEvent struct {
	EventID       uuid.UUID `json:"event_id"`
	OrderID       uuid.UUID `json:"order_id"`
	UserID        string    `json:"user_id"`
	SessionID     string    `json:"session_id"`

	// Event type
	EventType     string    `json:"event_type"` // created, paid, shipped, delivered, cancelled, refunded

	// Order details
	TotalAmount   float64   `json:"total_amount"`
	SubtotalAmount float64  `json:"subtotal_amount"`
	ShippingAmount float64  `json:"shipping_amount"`
	DiscountAmount float64  `json:"discount_amount"`
	TaxAmount     float64   `json:"tax_amount"`
	Currency      string    `json:"currency"`

	// Item counts
	ItemCount     int       `json:"item_count"`
	UniqueItems   int       `json:"unique_items"`

	// Payment
	PaymentMethod string    `json:"payment_method"`
	PaymentStatus string    `json:"payment_status"`

	// Shipping
	ShippingMethod string   `json:"shipping_method"`
	ShippingProvider string `json:"shipping_provider,omitempty"`

	// Promotions
	VoucherCode   string    `json:"voucher_code,omitempty"`
	VoucherDiscount float64 `json:"voucher_discount,omitempty"`
	CampaignID    string    `json:"campaign_id,omitempty"`

	// Attribution
	Source        string    `json:"source"`        // direct, search, social, email, affiliate
	UtmSource     string    `json:"utm_source,omitempty"`
	UtmMedium     string    `json:"utm_medium,omitempty"`
	UtmCampaign   string    `json:"utm_campaign,omitempty"`
	ReferrerURL   string    `json:"referrer_url,omitempty"`

	// User behavior
	CartDuration  int64     `json:"cart_duration_min,omitempty"` // Minutes from first item added to checkout
	CheckoutSteps int       `json:"checkout_steps,omitempty"`
	IsFirstOrder  bool      `json:"is_first_order"`
	IsReturning   bool      `json:"is_returning_customer"`

	// User context
	DeviceType    string    `json:"device_type"`
	Platform      string    `json:"platform"`
	IPAddress     string    `json:"ip_address"`
	Country       string    `json:"country,omitempty"`
	City          string    `json:"city,omitempty"`

	// Timestamps
	CreatedAt     time.Time `json:"created_at"`
}

// RevenueEvent represents a revenue tracking event (aggregated per shop/day)
type RevenueEvent struct {
	EventID       uuid.UUID `json:"event_id"`
	ShopID        string    `json:"shop_id"`
	Date          time.Time `json:"date"` // Date only (no time)

	// Revenue metrics
	GrossRevenue      float64 `json:"gross_revenue"`
	NetRevenue        float64 `json:"net_revenue"`
	RefundAmount      float64 `json:"refund_amount"`
	DiscountAmount    float64 `json:"discount_amount"`
	ShippingRevenue   float64 `json:"shipping_revenue"`
	CommissionAmount  float64 `json:"commission_amount"` // Platform fee
	Currency          string  `json:"currency"`

	// Order counts
	TotalOrders       int     `json:"total_orders"`
	CompletedOrders   int     `json:"completed_orders"`
	CancelledOrders   int     `json:"cancelled_orders"`
	RefundedOrders    int     `json:"refunded_orders"`

	// Item metrics
	TotalItemsSold    int     `json:"total_items_sold"`
	UniqueProducts    int     `json:"unique_products"`
	UniqueCustomers   int     `json:"unique_customers"`
	NewCustomers      int     `json:"new_customers"`
	ReturningCustomers int    `json:"returning_customers"`

	// Average metrics
	AvgOrderValue     float64 `json:"avg_order_value"`
	AvgItemsPerOrder  float64 `json:"avg_items_per_order"`

	// Category breakdown (top 5)
	TopCategories     []CategoryRevenue `json:"top_categories,omitempty"`

	// Payment method breakdown
	PaymentBreakdown  map[string]float64 `json:"payment_breakdown,omitempty"`

	// Timestamps
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CategoryRevenue represents revenue for a category
type CategoryRevenue struct {
	CategoryID string  `json:"category_id"`
	Revenue    float64 `json:"revenue"`
	ItemCount  int     `json:"item_count"`
}

// NewProductEvent creates a new product event
func NewProductEvent(userID, sessionID, productID, eventType string) ProductEvent {
	return ProductEvent{
		EventID:   uuid.New(),
		UserID:    userID,
		SessionID: sessionID,
		ProductID: productID,
		EventType: eventType,
		Quantity:  1,
		Currency:  "VND",
		CreatedAt: time.Now().UTC(),
	}
}

// NewSearchQuery creates a new search query event
func NewSearchQuery(userID, sessionID, query string) SearchQuery {
	return SearchQuery{
		EventID:         uuid.New(),
		UserID:          userID,
		SessionID:       sessionID,
		Query:           query,
		NormalizedQuery: normalizeQuery(query),
		QueryLength:     len(query),
		QueryType:       "keyword",
		ResultsPage:     1,
		ResultsLimit:    20,
		ClickedCount:    0,
		AddedToCart:     false,
		Purchased:       false,
		CreatedAt:       time.Now().UTC(),
	}
}

// normalizeQuery normalizes a search query
func normalizeQuery(query string) string {
	// Simple normalization - in production use proper text processing
	normalized := ""
	for _, r := range query {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			normalized += string(r)
		} else if r >= 'A' && r <= 'Z' {
			normalized += string(r + 32)
		}
	}
	return normalized
}

// NewOrderEvent creates a new order event
func NewOrderEvent(orderID uuid.UUID, userID, sessionID, eventType string) OrderEvent {
	return OrderEvent{
		EventID:   uuid.New(),
		OrderID:   orderID,
		UserID:    userID,
		SessionID: sessionID,
		EventType: eventType,
		Currency:  "VND",
		CreatedAt: time.Now().UTC(),
	}
}

// NewRevenueEvent creates a new revenue event
func NewRevenueEvent(shopID string, date time.Time) RevenueEvent {
	now := time.Now().UTC()
	return RevenueEvent{
		EventID:          uuid.New(),
		ShopID:           shopID,
		Date:             date,
		Currency:         "VND",
		TopCategories:    make([]CategoryRevenue, 0),
		PaymentBreakdown: make(map[string]float64),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// PageViewEvent represents a page view analytics event
type PageViewEvent struct {
	EventID     uuid.UUID `json:"event_id"`
	UserID      string    `json:"user_id"`
	SessionID   string    `json:"session_id"`

	// Page info
	PageType    string    `json:"page_type"` // home, category, product, cart, checkout, search, shop
	PageURL     string    `json:"page_url"`
	PageTitle   string    `json:"page_title,omitempty"`

	// Entity context
	ProductID   string    `json:"product_id,omitempty"`
	CategoryID  string    `json:"category_id,omitempty"`
	ShopID      string    `json:"shop_id,omitempty"`
	SearchQuery string    `json:"search_query,omitempty"`

	// Navigation
	ReferrerURL string    `json:"referrer_url,omitempty"`
	EntryPage   bool      `json:"entry_page"`
	ExitPage    bool      `json:"exit_page"`
	PageDepth   int       `json:"page_depth"` // Pages viewed in session

	// Engagement
	TimeOnPage  int64     `json:"time_on_page_ms,omitempty"` // Milliseconds
	ScrollDepth int       `json:"scroll_depth,omitempty"`    // Percentage

	// User context
	DeviceType  string    `json:"device_type"`
	Platform    string    `json:"platform"`
	Browser     string    `json:"browser,omitempty"`
	IPAddress   string    `json:"ip_address"`
	Country     string    `json:"country,omitempty"`
	City        string    `json:"city,omitempty"`

	// Timestamp
	CreatedAt   time.Time `json:"created_at"`
}

// SessionEvent represents a user session analytics event
type SessionEvent struct {
	SessionID      string    `json:"session_id"`
	UserID         string    `json:"user_id"`

	// Session info
	SessionStart   time.Time `json:"session_start"`
	SessionEnd     *time.Time `json:"session_end,omitempty"`
	Duration       int64     `json:"duration_seconds,omitempty"`

	// Page metrics
	PageViews      int       `json:"page_views"`
	UniquePages    int       `json:"unique_pages"`
	EntryPage      string    `json:"entry_page"`
	ExitPage       string    `json:"exit_page,omitempty"`

	// Product interactions
	ProductsViewed     int   `json:"products_viewed"`
	ProductsCarted     int   `json:"products_carted"`
	SearchesPerformed  int   `json:"searches_performed"`

	// Conversion
	Converted      bool      `json:"converted"` // Made a purchase
	ConversionValue float64  `json:"conversion_value,omitempty"`
	OrderID        string    `json:"order_id,omitempty"`

	// Attribution
	Source         string    `json:"source"` // direct, organic, paid, social, email, referral
	Medium         string    `json:"medium,omitempty"`
	Campaign       string    `json:"campaign,omitempty"`
	LandingURL     string    `json:"landing_url"`

	// User context
	IsNewUser      bool      `json:"is_new_user"`
	UserType       string    `json:"user_type"` // guest, registered
	DeviceType     string    `json:"device_type"`
	Platform       string    `json:"platform"`
	Browser        string    `json:"browser,omitempty"`
	Country        string    `json:"country,omitempty"`
	City           string    `json:"city,omitempty"`

	// Timestamps
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
