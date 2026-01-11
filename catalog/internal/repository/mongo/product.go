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

var _ repository.ProductRepository = (*ProductRepository)(nil)

type ProductRepository struct {
	collection *mongo.Collection
}

func NewProductRepository(db *mongo.Database) (*ProductRepository, error) {
	collection := db.Collection("products")

	// Create indexes
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "name", Value: "text"}},
		},
		{
			Keys: bson.D{{Key: "categoryId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "brandId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "metadata.$**", Value: 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(context.Background(), indexes)
	if err != nil {
		return nil, err
	}

	return &ProductRepository{collection: collection}, nil
}

func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	product.EnsureDefaults()
	result, err := r.collection.InsertOne(ctx, product)
	if err != nil {
		return err
	}
	product.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Product, error) {
	var product domain.Product
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	var product domain.Product
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) GetAll(ctx context.Context, filter repository.ProductFilter) ([]*domain.Product, error) {
	query := bson.M{}

	if filter.CategoryID != nil {
		query["categoryId"] = filter.CategoryID
	}
	if filter.BrandID != nil {
		query["brandId"] = filter.BrandID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}

	opts := options.Find()
	if filter.Limit > 0 {
		opts.SetLimit(filter.Limit)
	}
	if filter.Offset > 0 {
		opts.SetSkip(filter.Offset)
	}

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*domain.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}
	return products, nil
}

func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	product.EnsureDefaults()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": product.ID}, product)
	return err
}

func (r *ProductRepository) UpdateMetadata(ctx context.Context, id primitive.ObjectID, metadata map[string]interface{}) error {
	update := bson.M{
		"$set": bson.M{
			"metadata": metadata,
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *ProductRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
