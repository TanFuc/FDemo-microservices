package service

import (
	"context"

	"microservices/profile/internal/model"
)

// ProfileService defines the interface for profile business logic
type ProfileService interface {
	// GetOrCreateProfile gets or creates a profile for a user
	GetOrCreateProfile(ctx context.Context, userID string) (*model.Profile, error)

	// CreateInitialProfile creates an initial profile for a user
	CreateInitialProfile(ctx context.Context, userID, displayName, email string) (*model.Profile, error)

	// GetProfileByUserID retrieves a profile by user ID
	GetProfileByUserID(ctx context.Context, userID string) (*model.Profile, error)

	// UpdateProfile updates a user's profile
	UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) (*model.Profile, error)

	// AddAddress adds a new address for a user
	AddAddress(ctx context.Context, userID string, req *model.CreateAddressRequest) (*model.Address, error)

	// GetAddresses retrieves all addresses for a user
	GetAddresses(ctx context.Context, userID string) ([]*model.Address, error)

	// GetAddressByID retrieves an address by ID
	GetAddressByID(ctx context.Context, userID string, addressID string) (*model.Address, error)

	// UpdateAddress updates an address
	UpdateAddress(ctx context.Context, userID, addressID string, req *model.UpdateAddressRequest) (*model.Address, error)

	// SetDefaultAddress sets an address as default
	SetDefaultAddress(ctx context.Context, userID, addressID string) (*model.Address, error)

	// DeleteAddress deletes an address
	DeleteAddress(ctx context.Context, userID, addressID string) error

	// RegisterShop registers a shop for a user
	RegisterShop(ctx context.Context, userID string, req *model.RegisterShopRequest) (*model.Profile, error)

	// UpdateShop updates shop details
	UpdateShop(ctx context.Context, userID string, req *model.UpdateShopRequest) (*model.Profile, error)

	// GetUserInfo retrieves user info for internal API
	GetUserInfo(ctx context.Context, userID string) (*model.UserInfoResponse, error)
}
