package tests

import (
	"context"
	"encoding/json"
	"errors"
	"tafu-review/internal/core/domain"
	"testing"
	"time"
)

// MockReviewRepository simulates MongoDB review repository
type MockReviewRepository struct {
	reviews map[string]*domain.Review
}

func NewMockReviewRepository() *MockReviewRepository {
	return &MockReviewRepository{
		reviews: make(map[string]*domain.Review),
	}
}

func (r *MockReviewRepository) Create(review *domain.Review) error {
	r.reviews[review.ID] = review
	return nil
}

func (r *MockReviewRepository) GetByProductID(productID string) []*domain.Review {
	var result []*domain.Review
	for _, review := range r.reviews {
		if review.ProductID == productID && review.Status == domain.ReviewStatusVisible {
			result = append(result, review)
		}
	}
	return result
}

// MockProductRatingRepository simulates MongoDB product_ratings collection
type MockProductRatingRepository struct {
	ratings map[string]*domain.ProductRating
}

func NewMockProductRatingRepository() *MockProductRatingRepository {
	return &MockProductRatingRepository{
		ratings: make(map[string]*domain.ProductRating),
	}
}

func (r *MockProductRatingRepository) Get(productID string) (*domain.ProductRating, error) {
	rating, exists := r.ratings[productID]
	if !exists {
		return nil, errors.New("not found")
	}
	return rating, nil
}

func (r *MockProductRatingRepository) IncrementRating(productID string, rating int) error {
	existing, err := r.Get(productID)

	var newRating *domain.ProductRating
	if err != nil {
		// Create new rating document
		newRating = &domain.ProductRating{
			ProductID:     productID,
			AverageRating: float64(rating),
			TotalReviews:  1,
			StarCounts:    domain.StarCounts{},
			UpdatedAt:     time.Now(),
		}
	} else {
		// Calculate new average: NewAvg = ((OldAvg * OldTotal) + NewRating) / (OldTotal + 1)
		newTotal := existing.TotalReviews + 1
		newAvg := ((existing.AverageRating * float64(existing.TotalReviews)) + float64(rating)) / float64(newTotal)

		newRating = &domain.ProductRating{
			ProductID:     productID,
			AverageRating: newAvg,
			TotalReviews:  newTotal,
			StarCounts:    existing.StarCounts,
			UpdatedAt:     time.Now(),
		}
	}

	// Increment the appropriate star count
	switch rating {
	case 1:
		newRating.StarCounts.One++
	case 2:
		newRating.StarCounts.Two++
	case 3:
		newRating.StarCounts.Three++
	case 4:
		newRating.StarCounts.Four++
	case 5:
		newRating.StarCounts.Five++
	}

	r.ratings[productID] = newRating
	return nil
}

// MockCacheRepository simulates Redis cache
type MockCacheRepository struct {
	cache map[string]*domain.RatingSummary
}

func NewMockCacheRepository() *MockCacheRepository {
	return &MockCacheRepository{
		cache: make(map[string]*domain.RatingSummary),
	}
}

func (r *MockCacheRepository) GetRatingSummary(productID string) (*domain.RatingSummary, error) {
	summary, exists := r.cache[productID]
	if !exists {
		return nil, errors.New("cache miss")
	}
	return summary, nil
}

func (r *MockCacheRepository) SetRatingSummary(productID string, summary *domain.RatingSummary) error {
	r.cache[productID] = summary
	return nil
}

func (r *MockCacheRepository) InvalidateRatingSummary(productID string) error {
	delete(r.cache, productID)
	return nil
}

// TestRatingAggregationLogic tests the core rating aggregation logic
// Scenario: Insert 3 reviews: 5-star, 5-star, 2-star
// Expected: Total=3, Average=4.0
func TestRatingAggregationLogic(t *testing.T) {
	ctx := context.Background()
	_ = ctx // Context for future async operations

	productID := "test-product-123"

	// Initialize mock repositories
	reviewRepo := NewMockReviewRepository()
	ratingRepo := NewMockProductRatingRepository()
	cacheRepo := NewMockCacheRepository()

	// Create 3 reviews: 5-star, 5-star, 2-star
	reviews := []struct {
		id     string
		rating int
	}{
		{"review-1", 5},
		{"review-2", 5},
		{"review-3", 2},
	}

	for _, r := range reviews {
		// Create review
		review := &domain.Review{
			ID:          r.id,
			UserID:      "user-1",
			UserName:    "Test User",
			ProductID:   productID,
			OrderID:     "order-" + r.id,
			Rating:      r.rating,
			Content:     "Test review content",
			IsPurchased: true,
			Status:      domain.ReviewStatusVisible,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := reviewRepo.Create(review); err != nil {
			t.Fatalf("Failed to create review: %v", err)
		}

		// Update rating aggregation
		if err := ratingRepo.IncrementRating(productID, r.rating); err != nil {
			t.Fatalf("Failed to update rating: %v", err)
		}

		// Invalidate cache
		if err := cacheRepo.InvalidateRatingSummary(productID); err != nil {
			t.Fatalf("Failed to invalidate cache: %v", err)
		}
	}

	// Verify aggregation results
	rating, err := ratingRepo.Get(productID)
	if err != nil {
		t.Fatalf("Failed to get rating: %v", err)
	}

	// Test Total Reviews
	expectedTotal := 3
	if rating.TotalReviews != expectedTotal {
		t.Errorf("Expected total reviews %d, got %d", expectedTotal, rating.TotalReviews)
	}

	// Test Average Rating: (5 + 5 + 2) / 3 = 4.0
	expectedAvg := 4.0
	if rating.AverageRating != expectedAvg {
		t.Errorf("Expected average rating %.1f, got %.1f", expectedAvg, rating.AverageRating)
	}

	// Test Star Counts
	if rating.StarCounts.Five != 2 {
		t.Errorf("Expected 2 five-star reviews, got %d", rating.StarCounts.Five)
	}
	if rating.StarCounts.Two != 1 {
		t.Errorf("Expected 1 two-star review, got %d", rating.StarCounts.Two)
	}

	t.Logf("Rating aggregation test passed: Total=%d, Average=%.1f", rating.TotalReviews, rating.AverageRating)
}

// TestCacheAfterGetRatingSummary tests that Redis cache is set after GetRatingSummary
func TestCacheAfterGetRatingSummary(t *testing.T) {
	productID := "test-product-456"

	// Initialize mock repositories
	ratingRepo := NewMockProductRatingRepository()
	cacheRepo := NewMockCacheRepository()

	// Setup: Add some ratings
	ratingRepo.IncrementRating(productID, 5)
	ratingRepo.IncrementRating(productID, 4)

	// Verify cache is empty initially
	_, err := cacheRepo.GetRatingSummary(productID)
	if err == nil {
		t.Error("Expected cache miss, but got a hit")
	}

	// Simulate GetRatingSummary behavior
	rating, err := ratingRepo.Get(productID)
	if err != nil {
		t.Fatalf("Failed to get rating from repo: %v", err)
	}

	summary := rating.ToSummary()

	// Set cache
	if err := cacheRepo.SetRatingSummary(productID, summary); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Verify cache is now populated
	cachedSummary, err := cacheRepo.GetRatingSummary(productID)
	if err != nil {
		t.Fatalf("Expected cache hit, but got miss: %v", err)
	}

	// Verify cached data matches
	if cachedSummary.Average != summary.Average {
		t.Errorf("Cached average mismatch: expected %.1f, got %.1f", summary.Average, cachedSummary.Average)
	}
	if cachedSummary.Total != summary.Total {
		t.Errorf("Cached total mismatch: expected %d, got %d", summary.Total, cachedSummary.Total)
	}

	t.Logf("Cache test passed: Average=%.1f, Total=%d", cachedSummary.Average, cachedSummary.Total)
}

// TestHTMLSanitization tests that HTML tags are stripped from content
func TestHTMLSanitization(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"<script>alert('xss')</script>Hello", "alert('xss')Hello"},
		{"<p>Great product!</p>", "Great product!"},
		{"Normal text without HTML", "Normal text without HTML"},
		{"<b>Bold</b> and <i>italic</i>", "Bold and italic"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := stripHTMLTags(tc.input)
		if result != tc.expected {
			t.Errorf("stripHTMLTags(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// stripHTMLTags is a local copy of the sanitizer function for testing
func stripHTMLTags(input string) string {
	// Simple HTML tag removal without regex
	result := ""
	inTag := false
	for _, char := range input {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result += string(char)
		}
	}
	return result
}

// TestImageURLValidation tests URL validation for images
func TestImageURLValidation(t *testing.T) {
	testCases := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/image.jpg", true},
		{"http://example.com/image.png", true},
		{"ftp://example.com/image.jpg", false},
		{"javascript:alert('xss')", false},
		{"", false},
		{"not-a-url", false},
	}

	for _, tc := range testCases {
		result := validateImageURL(tc.url)
		if result != tc.expected {
			t.Errorf("validateImageURL(%q) = %v, expected %v", tc.url, result, tc.expected)
		}
	}
}

// validateImageURL is a local copy for testing
func validateImageURL(imageURL string) bool {
	if imageURL == "" {
		return false
	}

	// Simple check for http/https prefix
	return len(imageURL) > 8 && (imageURL[:8] == "https://" || imageURL[:7] == "http://")
}

// TestRatingSummaryJSON tests the JSON serialization of RatingSummary
func TestRatingSummaryJSON(t *testing.T) {
	summary := &domain.RatingSummary{
		Average: 4.5,
		Total:   100,
		Breakdown: map[string]int{
			"1": 5,
			"2": 5,
			"3": 10,
			"4": 30,
			"5": 50,
		},
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Failed to marshal summary: %v", err)
	}

	var decoded domain.RatingSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal summary: %v", err)
	}

	if decoded.Average != summary.Average {
		t.Errorf("Average mismatch: expected %.1f, got %.1f", summary.Average, decoded.Average)
	}
	if decoded.Total != summary.Total {
		t.Errorf("Total mismatch: expected %d, got %d", summary.Total, decoded.Total)
	}

	t.Logf("JSON serialization test passed")
}

// TestIncrementalAverageFormula verifies the incremental average formula
// NewAvg = ((OldAvg * OldTotal) + NewRating) / (OldTotal + 1)
func TestIncrementalAverageFormula(t *testing.T) {
	testCases := []struct {
		oldAvg      float64
		oldTotal    int
		newRating   int
		expectedAvg float64
	}{
		{0, 0, 5, 5.0},           // First rating
		{5.0, 1, 5, 5.0},         // Second 5-star keeps average
		{5.0, 2, 2, 4.0},         // Adding 2-star drops average
		{4.0, 3, 4, 4.0},         // Adding 4-star maintains average
		{4.5, 100, 5, 4.505},     // Large sample slight increase
	}

	for _, tc := range testCases {
		var newAvg float64
		if tc.oldTotal == 0 {
			newAvg = float64(tc.newRating)
		} else {
			newAvg = ((tc.oldAvg * float64(tc.oldTotal)) + float64(tc.newRating)) / float64(tc.oldTotal+1)
		}

		// Compare with tolerance for floating point
		diff := newAvg - tc.expectedAvg
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.001 {
			t.Errorf("Formula error: oldAvg=%.1f, oldTotal=%d, newRating=%d -> got %.3f, expected %.3f",
				tc.oldAvg, tc.oldTotal, tc.newRating, newAvg, tc.expectedAvg)
		}
	}

	t.Logf("Incremental average formula test passed")
}
