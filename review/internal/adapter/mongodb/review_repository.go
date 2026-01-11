package mongodb

import (
	"context"
	"errors"
	"microservices/review/internal/core/domain"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrReviewNotFound = errors.New("review not found")
	ErrDuplicateReview = errors.New("review already exists for this order and product")
)

type ReviewRepository struct {
	collection *mongo.Collection
}

func NewReviewRepository(db *mongo.Database) *ReviewRepository {
	collection := db.Collection("reviews")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "productId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "userId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{
				{Key: "orderId", Value: 1},
				{Key: "productId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	collection.Indexes().CreateMany(ctx, indexes)

	return &ReviewRepository{collection: collection}
}

func (r *ReviewRepository) Create(ctx context.Context, review *domain.Review) error {
	_, err := r.collection.InsertOne(ctx, review)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateReview
		}
		return err
	}
	return nil
}

func (r *ReviewRepository) GetByID(ctx context.Context, id string) (*domain.Review, error) {
	var review domain.Review
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&review)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *ReviewRepository) GetByProductID(ctx context.Context, productID string, page, limit int) ([]*domain.Review, int64, error) {
	filter := bson.M{
		"productId": productID,
		"status":    domain.ReviewStatusVisible,
	}

	// Count total documents
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var reviews []*domain.Review
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *ReviewRepository) GetByUserID(ctx context.Context, userID string, page, limit int) ([]*domain.Review, int64, error) {
	filter := bson.M{"userId": userID}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var reviews []*domain.Review
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *ReviewRepository) ExistsByOrderAndProduct(ctx context.Context, orderID, productID string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{
		"orderId":   orderID,
		"productId": productID,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ReviewRepository) UpdateReply(ctx context.Context, reviewID string, reply *domain.Reply) error {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": reviewID},
		bson.M{
			"$set": bson.M{
				"reply":     reply,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrReviewNotFound
	}
	return nil
}

func (r *ReviewRepository) UpdateStatus(ctx context.Context, reviewID string, status domain.ReviewStatus) error {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": reviewID},
		bson.M{
			"$set": bson.M{
				"status":    status,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrReviewNotFound
	}
	return nil
}
