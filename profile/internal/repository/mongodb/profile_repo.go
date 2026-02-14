package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"microservices/profile/internal/model"
	"microservices/profile/internal/repository"
)

const ProfileCollection = "profiles"

type profileRepository struct {
	collection *mongo.Collection
}

func NewProfileRepository(db *MongoDB) repository.ProfileRepository {
	return &profileRepository{
		collection: db.Collection(ProfileCollection),
	}
}

func (r *profileRepository) Create(ctx context.Context, profile *model.Profile) error {
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, profile)
	if err != nil {
		return err
	}

	profile.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *profileRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Profile, error) {
	var profile model.Profile
	err := r.collection.FindOne(ctx, bson.M{
		"_id":       id,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) FindByUserID(ctx context.Context, userID string) (*model.Profile, error) {
	var profile model.Profile
	err := r.collection.FindOne(ctx, bson.M{
		"userId":    userID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) Update(ctx context.Context, profile *model.Profile) error {
	profile.UpdatedAt = time.Now()

	_, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"_id": profile.ID},
		profile,
	)
	return err
}

func (r *profileRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"isDeleted": true,
				"deletedAt": now,
				"updatedAt": now,
			},
		},
	)
	return err
}

func (r *profileRepository) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{
		"userId":    userID,
		"isDeleted": bson.M{"$ne": true},
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *profileRepository) ExistsByShopName(ctx context.Context, shopName string, excludeUserID string) (bool, error) {
	filter := bson.M{
		"shopConfig.shopName": shopName,
		"isDeleted":           bson.M{"$ne": true},
	}

	if excludeUserID != "" {
		filter["userId"] = bson.M{"$ne": excludeUserID}
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
