package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"microservices/profile/internal/model"
	"microservices/profile/internal/repository"
)

const AddressCollection = "addresses"

type addressRepository struct {
	collection *mongo.Collection
}

func NewAddressRepository(db *MongoDB) repository.AddressRepository {
	return &addressRepository{
		collection: db.Collection(AddressCollection),
	}
}

func (r *addressRepository) Create(ctx context.Context, address *model.Address) error {
	address.CreatedAt = time.Now()
	address.UpdatedAt = time.Now()
	address.ComputeFullAddress()

	result, err := r.collection.InsertOne(ctx, address)
	if err != nil {
		return err
	}

	address.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *addressRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Address, error) {
	var address model.Address
	err := r.collection.FindOne(ctx, bson.M{
		"_id":       id,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&address)
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) FindByUserID(ctx context.Context, userID string) ([]*model.Address, error) {
	opts := options.Find().SetSort(bson.D{
		{Key: "isDefault", Value: -1},
		{Key: "createdAt", Value: -1},
	})

	cursor, err := r.collection.Find(ctx, bson.M{
		"userId":    userID,
		"isDeleted": bson.M{"$ne": true},
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var addresses []*model.Address
	if err := cursor.All(ctx, &addresses); err != nil {
		return nil, err
	}

	return addresses, nil
}

func (r *addressRepository) FindByIDAndUserID(ctx context.Context, id primitive.ObjectID, userID string) (*model.Address, error) {
	var address model.Address
	err := r.collection.FindOne(ctx, bson.M{
		"_id":       id,
		"userId":    userID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&address)
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) Update(ctx context.Context, address *model.Address) error {
	address.UpdatedAt = time.Now()
	address.ComputeFullAddress()

	_, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"_id": address.ID},
		address,
	)
	return err
}

func (r *addressRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *addressRepository) SetDefault(ctx context.Context, userID string, addressID primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": addressID, "userId": userID},
		bson.M{
			"$set": bson.M{
				"isDefault": true,
				"updatedAt": time.Now(),
			},
		},
	)
	return err
}

func (r *addressRepository) UnsetAllDefaults(ctx context.Context, userID string) error {
	_, err := r.collection.UpdateMany(
		ctx,
		bson.M{
			"userId":    userID,
			"isDeleted": bson.M{"$ne": true},
		},
		bson.M{
			"$set": bson.M{
				"isDefault": false,
				"updatedAt": time.Now(),
			},
		},
	)
	return err
}

func (r *addressRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"userId":    userID,
		"isDeleted": bson.M{"$ne": true},
	})
}

func (r *addressRepository) FindDefaultByUserID(ctx context.Context, userID string) (*model.Address, error) {
	var address model.Address
	err := r.collection.FindOne(ctx, bson.M{
		"userId":    userID,
		"isDefault": true,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&address)
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) FindMostRecentByUserID(ctx context.Context, userID string) (*model.Address, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	var address model.Address
	err := r.collection.FindOne(ctx, bson.M{
		"userId":    userID,
		"isDeleted": bson.M{"$ne": true},
	}, opts).Decode(&address)
	if err != nil {
		return nil, err
	}
	return &address, nil
}
