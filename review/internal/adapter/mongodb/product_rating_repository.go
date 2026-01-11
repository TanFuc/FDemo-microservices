package mongodb

import (
	"context"
	"errors"
	"tafu-review/internal/core/domain"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrProductRatingNotFound = errors.New("product rating not found")

type ProductRatingRepository struct {
	collection *mongo.Collection
}

func NewProductRatingRepository(db *mongo.Database) *ProductRatingRepository {
	return &ProductRatingRepository{
		collection: db.Collection("product_ratings"),
	}
}

func (r *ProductRatingRepository) Get(ctx context.Context, productID string) (*domain.ProductRating, error) {
	var rating domain.ProductRating
	err := r.collection.FindOne(ctx, bson.M{"_id": productID}).Decode(&rating)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrProductRatingNotFound
		}
		return nil, err
	}
	return &rating, nil
}

func (r *ProductRatingRepository) Upsert(ctx context.Context, rating *domain.ProductRating) error {
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": rating.ProductID},
		bson.M{
			"$set": bson.M{
				"averageRating": rating.AverageRating,
				"totalReviews":  rating.TotalReviews,
				"starCounts":    rating.StarCounts,
				"updatedAt":     time.Now(),
			},
		},
		opts,
	)
	return err
}

func (r *ProductRatingRepository) IncrementRating(ctx context.Context, productID string, rating int) error {
	// First, try to get existing rating
	existing, err := r.Get(ctx, productID)
	if err != nil && !errors.Is(err, ErrProductRatingNotFound) {
		return err
	}

	var newRating *domain.ProductRating
	if existing == nil {
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

	return r.Upsert(ctx, newRating)
}
