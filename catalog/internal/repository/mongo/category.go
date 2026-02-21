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

var _ repository.CategoryRepository = (*CategoryRepository)(nil)

type CategoryRepository struct {
	collection *mongo.Collection
}

func NewCategoryRepository(db *mongo.Database) (*CategoryRepository, error) {
	collection := db.Collection("categories")

	// Create unique index on slug
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	return &CategoryRepository{collection: collection}, nil
}

func (r *CategoryRepository) Create(ctx context.Context, category *model.Category) error {
	category.EnsureDefaults()
	result, err := r.collection.InsertOne(ctx, category)
	if err != nil {
		return err
	}
	category.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Category, error) {
	var category model.Category
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&category)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*model.Category, error) {
	var category model.Category
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&category)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) List(ctx context.Context) ([]model.Category, error) {
	opts := options.Find().SetSort(bson.D{{Key: "position", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var categories []model.Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) Update(ctx context.Context, category *model.Category) error {
	category.EnsureDefaults()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": category.ID}, category)
	return err
}

func (r *CategoryRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *CategoryRepository) ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": id})
	return count > 0, err
}

func (r *CategoryRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"slug": slug})
	return count > 0, err
}
