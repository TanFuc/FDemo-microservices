package service

import (
	"context"
	"errors"
	"log"
	"microservices/review/internal/adapter"
	"microservices/review/internal/adapter/grpc"
	"microservices/review/internal/adapter/mongodb"
	natspub "microservices/review/internal/adapter/nats"
	rediscache "microservices/review/internal/adapter/redis"
	"microservices/review/internal/core/domain"
	"microservices/review/internal/core/dto"
	"microservices/review/internal/core/port"
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
	ErrDuplicateRequest    = errors.New("duplicate review submission detected")
)

type ReviewService struct {
	reviewRepo        *mongodb.ReviewRepository
	productRatingRepo *mongodb.ProductRatingRepository
	cacheRepo         *rediscache.CacheRepository
	orderClient       grpc.OrderServiceClient
	profileClient     adapter.ProfileClient
	publisher         port.EventPublisher
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

// NewReviewServiceWithPublisher creates a review service with event publisher
func NewReviewServiceWithPublisher(
	reviewRepo *mongodb.ReviewRepository,
	productRatingRepo *mongodb.ProductRatingRepository,
	cacheRepo *rediscache.CacheRepository,
	orderClient grpc.OrderServiceClient,
	publisher port.EventPublisher,
) *ReviewService {
	return &ReviewService{
		reviewRepo:        reviewRepo,
		productRatingRepo: productRatingRepo,
		cacheRepo:         cacheRepo,
		orderClient:       orderClient,
		publisher:         publisher,
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

	// Step 0: Idempotency check - prevent duplicate submissions (e.g., double-click, network retries)
	// Check if a review for this order+user combination is already being processed or was recently created
	if s.cacheRepo != nil {
		existingReviewID, err := s.cacheRepo.GetExistingReviewID(ctx, req.OrderID, req.UserID)
		if err != nil {
			log.Printf("WARN: idempotency check failed: %v", err)
			// Continue without idempotency protection if Redis is down
		} else if existingReviewID != "" {
			// A review was already created for this order+user - return the existing review
			existingReview, err := s.reviewRepo.GetByID(ctx, existingReviewID)
			if err == nil {
				log.Printf("INFO: idempotency hit - returning existing review %s for order %s", existingReviewID, req.OrderID)
				return existingReview, nil
			}
			// If we can't find the review in DB but key exists in Redis, it might be a race condition
			// Clear the key and proceed with creation
			_ = s.cacheRepo.ClearReviewIdempotency(ctx, req.OrderID, req.UserID)
		}
	}

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

	// Step 3b: Set idempotency key after successful creation
	// This prevents duplicate submissions in a race condition window
	if s.cacheRepo != nil {
		if _, err := s.cacheRepo.CheckAndSetReviewIdempotency(ctx, req.OrderID, req.UserID, review.ID); err != nil {
			log.Printf("WARN: failed to set idempotency key: %v", err)
			// Continue - the review was created successfully, idempotency is a safeguard not a requirement
		}
	}

	// Step 4: Publish review.created event asynchronously (do NOT block the HTTP response)
	if s.publisher != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			content := review.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}

			evt := &natspub.ReviewCreatedEvent{
				EventID:    uuid.New().String(),
				ReviewID:   review.ID,
				ProductID:  review.ProductID,
				OrderID:    review.OrderID,
				UserID:     review.UserID,
				UserName:   review.UserName,
				Rating:     review.Rating,
				Content:    content,
				Images:     review.Images,
				IsEdited:   false,
				OccurredAt: time.Now(),
			}

			if err := s.publisher.PublishReviewCreated(bgCtx, evt); err != nil {
				log.Printf("ERROR: failed to publish review.created event: reviewId=%s err=%v", review.ID, err)
			}
		}()
	}

	// Step 5: Update product ratings asynchronously with retry
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var ratingErr error
		for attempt := 1; attempt <= 3; attempt++ {
			ratingErr = s.updateProductRating(bgCtx, req.ProductID, req.Rating)
			if ratingErr == nil {
				break
			}
			log.Printf("WARN: rating update attempt %d failed for product %s: %v", attempt, req.ProductID, ratingErr)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		if ratingErr != nil {
			log.Printf("ERROR: rating update exhausted retries for product %s: %v", req.ProductID, ratingErr)
			return
		}

		// Publish RatingUpdatedEvent for Catalog Service sync
		if s.publisher != nil {
			updatedRating, err := s.productRatingRepo.Get(bgCtx, req.ProductID)
			if err == nil {
				_ = s.publisher.PublishRatingUpdated(bgCtx, &natspub.RatingUpdatedEvent{
					EventID:       uuid.New().String(),
					ProductID:     req.ProductID,
					AverageRating: updatedRating.AverageRating,
					TotalReviews:  updatedRating.TotalReviews,
					OccurredAt:    time.Now(),
				})
			}
		}
	}()

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
	if err != nil {
		return err
	}

	// Notify user that their review has a reply
	if s.publisher != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			review, err := s.reviewRepo.GetByID(bgCtx, req.ReviewID)
			if err != nil {
				log.Printf("WARN: failed to fetch review for reply event: %v", err)
				return
			}

			evt := &natspub.ReviewCreatedEvent{
				EventID:    uuid.New().String(),
				ReviewID:   review.ID,
				ProductID:  review.ProductID,
				UserID:     review.UserID,
				UserName:   review.UserName,
				Rating:     review.Rating,
				EventType:  "review.updated",
				OccurredAt: time.Now(),
			}

			if err := s.publisher.PublishReviewUpdated(bgCtx, evt); err != nil {
				log.Printf("ERROR: failed to publish review.updated event: reviewId=%s err=%v", review.ID, err)
			}
		}()
	}

	return nil
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
