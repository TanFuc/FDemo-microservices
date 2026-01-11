package mongo

import (
	"context"
	"errors"

	"microservices/catalog/internal/domain"
	"microservices/catalog/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ repository.BrandRepository = (*BrandRepository)(nil)

type BrandRepository struct {
	collection *mongo.Collection
}

func NewBrandRepository(db *mongo.Database) (*BrandRepository, error) {
	collection := db.Collection("brands")

	// Create unique index on slug
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	return &BrandRepository{collection: collection}, nil
}

func (r *BrandRepository) Create(ctx context.Context, brand *domain.Brand) error {
	brand.EnsureDefaults()
	result, err := r.collection.InsertOne(ctx, brand)
	if err != nil {
		return err
	}
	brand.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *BrandRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Brand, error) {
	var brand domain.Brand
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&brand)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &brand, nil
}

func (r *BrandRepository) GetBySlug(ctx context.Context, slug string) (*domain.Brand, error) {
	var brand domain.Brand
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&brand)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &brand, nil
}

func (r *BrandRepository) GetAll(ctx context.Context) ([]*domain.Brand, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var brands []*domain.Brand
	if err := cursor.All(ctx, &brands); err != nil {
		return nil, err
	}
	return brands, nil
}

func (r *BrandRepository) Update(ctx context.Context, brand *domain.Brand) error {
	brand.EnsureDefaults()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": brand.ID}, brand)
	return err
}

func (r *BrandRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
