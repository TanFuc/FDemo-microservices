package service

import (
	"context"
	"errors"
	"log"
	"microservices/review/internal/adapter"
	"microservices/review/internal/adapter/grpc"
	"microservices/review/internal/adapter/mongodb"
	rediscache "microservices/review/internal/adapter/redis"
	"microservices/review/internal/core/domain"
	"microservices/review/internal/core/dto"
	"microservices/review/internal/core/util"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidRating       = errors.New("rating must be between 1 and 5")
	ErrEmptyContent        = errors.New("review content cannot be empty")
	ErrDuplicateReview     = errors.New("you have already reviewed this product for this order")
	ErrVerificationFailed  = errors.New("purchase verification failed")
	ErrReviewNotFound      = errors.New("review not found")
)

type ReviewService struct {
	reviewRepo        *mongodb.ReviewRepository
	productRatingRepo *mongodb.ProductRatingRepository
	cacheRepo         *rediscache.CacheRepository
	orderClient       grpc.OrderServiceClient
	profileClient     adapter.ProfileClient
}

func NewReviewService(
	reviewRepo *mongodb.ReviewRepository,
	productRatingRepo *mongodb.ProductRatingRepository,
	cacheRepo *rediscache.CacheRepository,
	orderClient grpc.OrderServiceClient,
) *ReviewService {
	return &ReviewService{
		reviewRepo:        reviewRepo,
		productRatingRepo: productRatingRepo,
		cacheRepo:         cacheRepo,
		orderClient:       orderClient,
		profileClient:     nil, // No profile fetching by default
	}
}

// NewReviewServiceWithProfile creates a review service with profile client
func NewReviewServiceWithProfile(
	reviewRepo *mongodb.ReviewRepository,
	productRatingRepo *mongodb.ProductRatingRepository,
	cacheRepo *rediscache.CacheRepository,
	orderClient grpc.OrderServiceClient,
	profileClient adapter.ProfileClient,
) *ReviewService {
	return &ReviewService{
		reviewRepo:        reviewRepo,
		productRatingRepo: productRatingRepo,
		cacheRepo:         cacheRepo,
		orderClient:       orderClient,
		profileClient:     profileClient,
	}
}

// CreateReview creates a new review after verifying purchase
func (s *ReviewService) CreateReview(ctx context.Context, req *dto.CreateReviewRequest) (*domain.Review, error) {
	// Validate rating
	if req.Rating < 1 || req.Rating > 5 {
		return nil, ErrInvalidRating
	}

	// Sanitize content to prevent XSS
	sanitizedContent := util.StripHTMLTags(req.Content)
	if sanitizedContent == "" {
		return nil, ErrEmptyContent
	}

	// Validate image URLs
	validImages := util.ValidateImageURLs(req.Images)

	// Step 1: Verify purchase through Order Service
	order, err := s.orderClient.GetOrderDetail(ctx, req.OrderID, req.UserID)
	if err != nil {
		if errors.Is(err, grpc.ErrOrderServiceDown) {
			return nil, ErrVerificationFailed
		}
		return nil, err
	}

	// Verify the purchase is valid
	if err := grpc.VerifyPurchase(order, req.ProductID, req.UserID); err != nil {
		return nil, err
	}

	// Check for duplicate review
	exists, err := s.reviewRepo.ExistsByOrderAndProduct(ctx, req.OrderID, req.ProductID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateReview
	}

	// Step 2: Fetch user info from Profile Service
	userName := req.UserName
	userAvatar := req.UserAvatar
	if s.profileClient != nil {
		userInfo, err := s.profileClient.GetUserInfo(ctx, req.UserID)
		if err != nil {
			// Log error but continue with fallback values
			log.Printf("Warning: failed to fetch user info from profile service: %v", err)
		} else if userInfo != nil {
			// Use profile info if available
			if userInfo.DisplayName != "" {
				userName = userInfo.DisplayName
			}
			if userInfo.AvatarURL != "" {
				userAvatar = userInfo.AvatarURL
			}
		}
	}

	// Fallback to default display name if not provided
	if userName == "" {
		shortID := req.UserID
		if len(req.UserID) > 8 {
			shortID = req.UserID[:8]
		}
		userName = "User-" + shortID
	}

	// Step 3: Create the review
	now := time.Now()
	review := &domain.Review{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		UserName:    userName,
		UserAvatar:  userAvatar,
		ProductID:   req.ProductID,
		OrderID:     req.OrderID,
		Rating:      req.Rating,
		Content:     sanitizedContent,
		Images:      validImages,
		IsPurchased: true,
		Status:      domain.ReviewStatusVisible,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.reviewRepo.Create(ctx, review); err != nil {
		if errors.Is(err, mongodb.ErrDuplicateReview) {
			return nil, ErrDuplicateReview
		}
		return nil, err
	}

	// Step 4: Update product ratings (async in production, sync here for simplicity)
	if err := s.updateProductRating(ctx, req.ProductID, req.Rating); err != nil {
		// Log error but don't fail the request - review was created successfully
		// In production, this would be handled by a background job/queue
	}

	return review, nil
}

// updateProductRating updates the product rating aggregation and invalidates cache
func (s *ReviewService) updateProductRating(ctx context.Context, productID string, rating int) error {
	// Update MongoDB product_ratings collection
	if err := s.productRatingRepo.IncrementRating(ctx, productID, rating); err != nil {
		return err
	}

	// Invalidate Redis cache
	return s.cacheRepo.InvalidateRatingSummary(ctx, productID)
}

// GetProductReviews returns paginated reviews for a product
func (s *ReviewService) GetProductReviews(ctx context.Context, req *dto.GetReviewsRequest) (*dto.PaginatedResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	reviews, total, err := s.reviewRepo.GetByProductID(ctx, req.ProductID, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	totalPages := (total + int64(req.Limit) - 1) / int64(req.Limit)

	return &dto.PaginatedResponse{
		Data:       reviews,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

// GetRatingSummary returns the rating summary for a product (cached)
func (s *ReviewService) GetRatingSummary(ctx context.Context, productID string) (*domain.RatingSummary, error) {
	// Step 1: Check Redis cache
	summary, err := s.cacheRepo.GetRatingSummary(ctx, productID)
	if err == nil {
		return summary, nil
	}

	// Step 2: Cache miss - get from MongoDB
	if !errors.Is(err, rediscache.ErrCacheMiss) {
		// Log Redis error but continue to MongoDB
	}

	rating, err := s.productRatingRepo.Get(ctx, productID)
	if err != nil {
		if errors.Is(err, mongodb.ErrProductRatingNotFound) {
			// Return empty summary for products with no reviews
			return &domain.RatingSummary{
				Average:   0,
				Total:     0,
				Breakdown: map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
			}, nil
		}
		return nil, err
	}

	summary = rating.ToSummary()

	// Step 3: Set Redis cache (ignore errors)
	_ = s.cacheRepo.SetRatingSummary(ctx, productID, summary)

	return summary, nil
}

// ReplyReview allows a seller to reply to a review
func (s *ReviewService) ReplyReview(ctx context.Context, req *dto.ReplyReviewRequest) error {
	// Sanitize reply content
	sanitizedContent := util.StripHTMLTags(req.Content)
	if sanitizedContent == "" {
		return ErrEmptyContent
	}

	reply := &domain.Reply{
		Content:   sanitizedContent,
		RepliedAt: time.Now(),
	}

	err := s.reviewRepo.UpdateReply(ctx, req.ReviewID, reply)
	if errors.Is(err, mongodb.ErrReviewNotFound) {
		return ErrReviewNotFound
	}
	return err
}

// GetReviewByID returns a single review by ID
func (s *ReviewService) GetReviewByID(ctx context.Context, reviewID string) (*domain.Review, error) {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		if errors.Is(err, mongodb.ErrReviewNotFound) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return review, nil
}

// GetUserReviews returns all reviews by a user
func (s *ReviewService) GetUserReviews(ctx context.Context, userID string, page, limit int) (*dto.PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	reviews, total, err := s.reviewRepo.GetByUserID(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return &dto.PaginatedResponse{
		Data:       reviews,
		Page:       page,
		Limit:      limit,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

// UpdateReviewStatus allows admin to hide/show reviews
func (s *ReviewService) UpdateReviewStatus(ctx context.Context, reviewID string, status domain.ReviewStatus) error {
	err := s.reviewRepo.UpdateStatus(ctx, reviewID, status)
	if errors.Is(err, mongodb.ErrReviewNotFound) {
		return ErrReviewNotFound
	}
	return err
}
