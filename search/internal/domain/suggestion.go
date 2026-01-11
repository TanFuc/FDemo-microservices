package domain

import "time"

// SuggestionType represents the type of search suggestion
type SuggestionType string

const (
	SuggestionTypeKeyword   SuggestionType = "KEYWORD"    // Popular search terms
	SuggestionTypeProduct   SuggestionType = "PRODUCT"    // Product name
	SuggestionTypeCategory  SuggestionType = "CATEGORY"   // Category name
	SuggestionTypeBrand     SuggestionType = "BRAND"      // Brand name
	SuggestionTypeShop      SuggestionType = "SHOP"       // Shop name
	SuggestionTypeTrending  SuggestionType = "TRENDING"   // Trending searches
	SuggestionTypeRecent    SuggestionType = "RECENT"     // User's recent searches
	SuggestionTypeCorrection SuggestionType = "CORRECTION" // Spelling correction
)

// SearchSuggestion represents a search suggestion/autocomplete entry
type SearchSuggestion struct {
	ID            string         `json:"id"`
	Text          string         `json:"text"`           // The suggestion text
	DisplayText   string         `json:"displayText"`    // How it's displayed (with highlighting)
	Type          SuggestionType `json:"type"`

	// Entity reference (for product, category, brand, shop suggestions)
	EntityID      string         `json:"entityId,omitempty"`
	EntityType    string         `json:"entityType,omitempty"` // product, category, brand, shop

	// Additional context
	ImageURL      string         `json:"imageUrl,omitempty"`
	Category      string         `json:"category,omitempty"`
	CategoryID    string         `json:"categoryId,omitempty"`

	// Popularity metrics
	SearchCount   int64          `json:"searchCount"`    // How many times searched
	ClickCount    int64          `json:"clickCount"`     // How many times clicked
	ConversionCount int64        `json:"conversionCount"` // How many led to purchase
	Score         float64        `json:"score"`          // Relevance/popularity score

	// Weighting
	Weight        int            `json:"weight"`         // Manual boost weight
	IsPromoted    bool           `json:"isPromoted"`     // Manually promoted
	IsTrending    bool           `json:"isTrending"`     // Currently trending

	// Completion input (for Elasticsearch completion suggester)
	Suggest       *SuggestInput  `json:"suggest,omitempty"`

	// Language
	Language      string         `json:"language"`       // vi, en

	// Status
	IsActive      bool           `json:"isActive"`

	// Timestamps
	LastSearchedAt *time.Time    `json:"lastSearchedAt,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// SuggestInput represents the input for Elasticsearch completion suggester
type SuggestInput struct {
	Input  []string `json:"input"`  // Possible input strings
	Weight int      `json:"weight"` // Suggestion weight
}

// NewSearchSuggestion creates a new search suggestion
func NewSearchSuggestion(text string, suggestType SuggestionType, language string) *SearchSuggestion {
	now := time.Now()
	return &SearchSuggestion{
		Text:         text,
		DisplayText:  text,
		Type:         suggestType,
		SearchCount:  0,
		ClickCount:   0,
		ConversionCount: 0,
		Score:        0,
		Weight:       1,
		IsPromoted:   false,
		IsTrending:   false,
		Language:     language,
		IsActive:     true,
		Suggest: &SuggestInput{
			Input:  generateInputVariations(text),
			Weight: 1,
		},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// generateInputVariations generates input variations for autocomplete
func generateInputVariations(text string) []string {
	variations := []string{text}

	// Add lowercase version
	lower := ""
	for _, r := range text {
		if r >= 'A' && r <= 'Z' {
			lower += string(r + 32)
		} else {
			lower += string(r)
		}
	}
	if lower != text {
		variations = append(variations, lower)
	}

	// Could add more variations (without diacritics, etc.)
	return variations
}

// SetEntity sets the entity reference
func (ss *SearchSuggestion) SetEntity(entityID, entityType string) {
	ss.EntityID = entityID
	ss.EntityType = entityType
	ss.UpdatedAt = time.Now()
}

// SetImage sets the image URL
func (ss *SearchSuggestion) SetImage(imageURL string) {
	ss.ImageURL = imageURL
	ss.UpdatedAt = time.Now()
}

// SetCategory sets the category
func (ss *SearchSuggestion) SetCategory(category, categoryID string) {
	ss.Category = category
	ss.CategoryID = categoryID
	ss.UpdatedAt = time.Now()
}

// IncrementSearchCount increments the search count
func (ss *SearchSuggestion) IncrementSearchCount() {
	now := time.Now()
	ss.SearchCount++
	ss.LastSearchedAt = &now
	ss.UpdatedAt = now
	ss.updateScore()
}

// IncrementClickCount increments the click count
func (ss *SearchSuggestion) IncrementClickCount() {
	ss.ClickCount++
	ss.UpdatedAt = time.Now()
	ss.updateScore()
}

// IncrementConversionCount increments the conversion count
func (ss *SearchSuggestion) IncrementConversionCount() {
	ss.ConversionCount++
	ss.UpdatedAt = time.Now()
	ss.updateScore()
}

// updateScore recalculates the suggestion score
func (ss *SearchSuggestion) updateScore() {
	// Simple scoring: searches * 1 + clicks * 2 + conversions * 5
	ss.Score = float64(ss.SearchCount) + float64(ss.ClickCount)*2 + float64(ss.ConversionCount)*5

	// Apply weight multiplier
	ss.Score *= float64(ss.Weight)

	// Boost for promoted suggestions
	if ss.IsPromoted {
		ss.Score *= 2
	}

	// Boost for trending
	if ss.IsTrending {
		ss.Score *= 1.5
	}

	// Update suggester weight
	if ss.Suggest != nil {
		ss.Suggest.Weight = int(ss.Score/10) + 1
	}
}

// Promote promotes the suggestion
func (ss *SearchSuggestion) Promote(weight int) {
	ss.IsPromoted = true
	ss.Weight = weight
	ss.UpdatedAt = time.Now()
	ss.updateScore()
}

// Unpromote removes promotion
func (ss *SearchSuggestion) Unpromote() {
	ss.IsPromoted = false
	ss.Weight = 1
	ss.UpdatedAt = time.Now()
	ss.updateScore()
}

// MarkTrending marks as trending
func (ss *SearchSuggestion) MarkTrending() {
	ss.IsTrending = true
	ss.UpdatedAt = time.Now()
	ss.updateScore()
}

// UnmarkTrending removes trending status
func (ss *SearchSuggestion) UnmarkTrending() {
	ss.IsTrending = false
	ss.UpdatedAt = time.Now()
	ss.updateScore()
}

// Deactivate deactivates the suggestion
func (ss *SearchSuggestion) Deactivate() {
	ss.IsActive = false
	ss.UpdatedAt = time.Now()
}

// Activate activates the suggestion
func (ss *SearchSuggestion) Activate() {
	ss.IsActive = true
	ss.UpdatedAt = time.Now()
}

// SuggestionRequest represents a request for suggestions
type SuggestionRequest struct {
	Query      string           `json:"query"`
	Types      []SuggestionType `json:"types,omitempty"` // Filter by types
	Language   string           `json:"language,omitempty"`
	CategoryID string           `json:"categoryId,omitempty"`
	Limit      int              `json:"limit"`
	UserID     string           `json:"userId,omitempty"` // For recent searches
}

// SuggestionResponse represents suggestion results
type SuggestionResponse struct {
	Suggestions []SearchSuggestion `json:"suggestions"`
	Query       string             `json:"query"`
	DidYouMean  string             `json:"didYouMean,omitempty"` // Spelling correction
}

// RecentSearch represents a user's recent search
type RecentSearch struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Query     string    `json:"query"`
	ResultCount int     `json:"resultCount"`
	Clicked   bool      `json:"clicked"`
	CreatedAt time.Time `json:"createdAt"`
}

// TrendingSearch represents a trending search query
type TrendingSearch struct {
	Query          string    `json:"query"`
	SearchCount    int64     `json:"searchCount"`
	GrowthRate     float64   `json:"growthRate"` // % increase
	TrendStartedAt time.Time `json:"trendStartedAt"`
	Category       string    `json:"category,omitempty"`
}

// PopularSearch represents popular searches (for display)
type PopularSearch struct {
	Query       string `json:"query"`
	SearchCount int64  `json:"searchCount"`
	Category    string `json:"category,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
}
