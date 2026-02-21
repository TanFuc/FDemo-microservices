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
			Keys: bson.D{{Key: "shopId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
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

func (r *ProductRepository) Create(ctx context.Context, product *model.Product) error {
	product.EnsureDefaults()
	result, err := r.collection.InsertOne(ctx, product)
	if err != nil {
		return err
	}
	product.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Product, error) {
	var product model.Product
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) GetBySlug(ctx context.Context, slug string) (*model.Product, error) {
	var product model.Product
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) List(ctx context.Context, filter *model.ProductFilter) ([]model.Product, int64, error) {
	query := bson.M{}

	if filter.CategoryID != "" {
		catID, err := primitive.ObjectIDFromHex(filter.CategoryID)
		if err == nil {
			query["categoryId"] = catID
		}
	}
	if filter.BrandID != "" {
		brandID, err := primitive.ObjectIDFromHex(filter.BrandID)
		if err == nil {
			query["brandId"] = brandID
		}
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.ShopID != "" {
		query["shopId"] = filter.ShopID
	}

	// Count total
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	if filter.Page > 0 && filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		opts.SetSkip(int64(offset))
	}

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *ProductRepository) Update(ctx context.Context, product *model.Product) error {
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

func (r *ProductRepository) ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": id})
	return count > 0, err
}

func (r *ProductRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"slug": slug})
	return count > 0, err
}
