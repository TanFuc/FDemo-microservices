package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"microservices/profile/internal/model"
)

// ProfileRepository defines the interface for profile data access
type ProfileRepository interface {
	// Create creates a new profile
	Create(ctx context.Context, profile *model.Profile) error

	// FindByID retrieves a profile by ID
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.Profile, error)

	// FindByUserID retrieves a profile by user ID
	FindByUserID(ctx context.Context, userID string) (*model.Profile, error)

	// Update updates an existing profile
	Update(ctx context.Context, profile *model.Profile) error

	// Delete soft-deletes a profile by ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// ExistsByUserID checks if a profile exists for a user
	ExistsByUserID(ctx context.Context, userID string) (bool, error)

	// ExistsByShopName checks if a shop name already exists
	ExistsByShopName(ctx context.Context, shopName string, excludeUserID string) (bool, error)

	// FindByAffiliateCode retrieves a profile by affiliate code
	FindByAffiliateCode(ctx context.Context, affiliateCode string) (*model.Profile, error)

	// ExistsByAffiliateCode checks if affiliate code already exists
	ExistsByAffiliateCode(ctx context.Context, affiliateCode string) (bool, error)
}

// AddressRepository defines the interface for address data access
type AddressRepository interface {
	// Create creates a new address
	Create(ctx context.Context, address *model.Address) error

	// FindByID retrieves an address by ID
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.Address, error)

	// FindByUserID retrieves all addresses for a user
	FindByUserID(ctx context.Context, userID string) ([]*model.Address, error)

	// FindByIDAndUserID retrieves an address by ID and user ID
	FindByIDAndUserID(ctx context.Context, id primitive.ObjectID, userID string) (*model.Address, error)

	// Update updates an existing address
	Update(ctx context.Context, address *model.Address) error

	// Delete removes an address by ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// SetDefault sets an address as default
	SetDefault(ctx context.Context, userID string, addressID primitive.ObjectID) error

	// UnsetAllDefaults unsets all default addresses for a user
	UnsetAllDefaults(ctx context.Context, userID string) error

	// CountByUserID counts addresses for a user
	CountByUserID(ctx context.Context, userID string) (int64, error)

	// FindDefaultByUserID finds the default address for a user
	FindDefaultByUserID(ctx context.Context, userID string) (*model.Address, error)

	// FindMostRecentByUserID finds the most recent address for a user
	FindMostRecentByUserID(ctx context.Context, userID string) (*model.Address, error)
}
