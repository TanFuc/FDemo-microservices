package mongo

import (
	"context"
	"errors"

	"microservices/catalog/internal/model"
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

func (r *BrandRepository) Create(ctx context.Context, brand *model.Brand) error {
	brand.EnsureDefaults()
	result, err := r.collection.InsertOne(ctx, brand)
	if err != nil {
		return err
	}
	brand.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *BrandRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Brand, error) {
	var brand model.Brand
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&brand)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &brand, nil
}

func (r *BrandRepository) GetBySlug(ctx context.Context, slug string) (*model.Brand, error) {
	var brand model.Brand
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&brand)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &brand, nil
}

func (r *BrandRepository) List(ctx context.Context) ([]model.Brand, error) {
	opts := options.Find().SetSort(bson.D{{Key: "position", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var brands []model.Brand
	if err := cursor.All(ctx, &brands); err != nil {
		return nil, err
	}
	return brands, nil
}

func (r *BrandRepository) Update(ctx context.Context, brand *model.Brand) error {
	brand.EnsureDefaults()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": brand.ID}, brand)
	return err
}

func (r *BrandRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *BrandRepository) ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": id})
	return count > 0, err
}

func (r *BrandRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"slug": slug})
	return count > 0, err
}
