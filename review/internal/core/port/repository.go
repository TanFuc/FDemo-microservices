package port

import (
	"context"
	"tafu-review/internal/core/domain"
)

type ReviewRepository interface {
	Create(ctx context.Context, review *domain.Review) error
	GetByID(ctx context.Context, id string) (*domain.Review, error)
	GetByProductID(ctx context.Context, productID string, page, limit int) ([]*domain.Review, int64, error)
	GetByUserID(ctx context.Context, userID string, page, limit int) ([]*domain.Review, int64, error)
	ExistsByOrderAndProduct(ctx context.Context, orderID, productID string) (bool, error)
	UpdateReply(ctx context.Context, reviewID string, reply *domain.Reply) error
	UpdateStatus(ctx context.Context, reviewID string, status domain.ReviewStatus) error
}

type ProductRatingRepository interface {
	Get(ctx context.Context, productID string) (*domain.ProductRating, error)
	Upsert(ctx context.Context, rating *domain.ProductRating) error
	IncrementRating(ctx context.Context, productID string, rating int) error
}

type CacheRepository interface {
	GetRatingSummary(ctx context.Context, productID string) (*domain.RatingSummary, error)
	SetRatingSummary(ctx context.Context, productID string, summary *domain.RatingSummary) error
	InvalidateRatingSummary(ctx context.Context, productID string) error
}
