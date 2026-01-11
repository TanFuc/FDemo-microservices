package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"tafu-profile/internal/domain/entity"
)

type ProfileRepository interface {
	Create(ctx context.Context, profile *entity.Profile) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*entity.Profile, error)
	FindByUserID(ctx context.Context, userID string) (*entity.Profile, error)
	Update(ctx context.Context, profile *entity.Profile) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	ExistsByUserID(ctx context.Context, userID string) (bool, error)
	ExistsByShopName(ctx context.Context, shopName string, excludeUserID string) (bool, error)
}

type AddressRepository interface {
	Create(ctx context.Context, address *entity.Address) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*entity.Address, error)
	FindByUserID(ctx context.Context, userID string) ([]*entity.Address, error)
	FindByIDAndUserID(ctx context.Context, id primitive.ObjectID, userID string) (*entity.Address, error)
	Update(ctx context.Context, address *entity.Address) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	SetDefault(ctx context.Context, userID string, addressID primitive.ObjectID) error
	UnsetAllDefaults(ctx context.Context, userID string) error
	CountByUserID(ctx context.Context, userID string) (int64, error)
	FindDefaultByUserID(ctx context.Context, userID string) (*entity.Address, error)
	FindMostRecentByUserID(ctx context.Context, userID string) (*entity.Address, error)
}
