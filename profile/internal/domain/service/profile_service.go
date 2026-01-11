package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"microservices/profile/internal/domain/entity"
	"microservices/profile/internal/domain/repository"
	"microservices/profile/pkg/errors"
	"microservices/profile/pkg/logger"
)

type ProfileService struct {
	profileRepo repository.ProfileRepository
	addressRepo repository.AddressRepository
}

func NewProfileService(
	profileRepo repository.ProfileRepository,
	addressRepo repository.AddressRepository,
) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		addressRepo: addressRepo,
	}
}

// Profile operations

func (s *ProfileService) GetOrCreateProfile(ctx context.Context, userID string) (*entity.Profile, error) {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err == nil {
		return profile, nil
	}

	if err != mongo.ErrNoDocuments {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch profile", 500)
	}

	// Create default profile
	displayName := "User-" + userID[len(userID)-4:]
	profile = entity.NewProfile(userID, displayName, "")

	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to create profile", 500)
	}

	return profile, nil
}

func (s *ProfileService) CreateInitialProfile(ctx context.Context, userID, displayName, email string) (*entity.Profile, error) {
	// Check if profile already exists
	exists, err := s.profileRepo.ExistsByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to check profile existence", 500)
	}
	if exists {
		// Return existing profile
		return s.profileRepo.FindByUserID(ctx, userID)
	}

	profile := entity.NewProfile(userID, displayName, email)
	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to create profile", 500)
	}

	logger.Info().
		Str("userId", userID).
		Str("displayName", displayName).
		Msg("Created initial profile")

	return profile, nil
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) (*entity.Profile, error) {
	profile, err := s.GetOrCreateProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if displayName, ok := updates["displayName"].(string); ok {
		profile.DisplayName = displayName
	}
	if avatarURL, ok := updates["avatarUrl"].(string); ok {
		profile.AvatarURL = avatarURL
	}
	if bio, ok := updates["bio"].(string); ok {
		profile.Bio = bio
	}
	if coverURL, ok := updates["coverUrl"].(string); ok {
		profile.CoverURL = coverURL
	}

	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to update profile", 500)
	}

	return profile, nil
}

func (s *ProfileService) GetProfileByUserID(ctx context.Context, userID string) (*entity.Profile, error) {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrProfileNotFound
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch profile", 500)
	}
	return profile, nil
}

// Address operations

func (s *ProfileService) AddAddress(ctx context.Context, userID string, input *CreateAddressInput) (*entity.Address, error) {
	address := entity.NewAddress(userID)
	address.ContactName = input.ContactName
	address.Phone = input.Phone
	address.CountryCode = "VN" // Default to Vietnam
	address.ProvinceCode = input.ProvinceCode
	address.DistrictCode = input.DistrictCode
	address.WardCode = input.WardCode
	address.StreetAddress = input.StreetLine
	address.Type = input.Type
	address.IsDefault = input.IsDefault

	if input.FullAddress != "" {
		address.FullAddress = input.FullAddress
	}

	// Check if this is the first address - make it default
	count, err := s.addressRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to count addresses", 500)
	}

	if count == 0 {
		address.IsDefault = true
	} else if address.IsDefault {
		// Unset other defaults
		if err := s.addressRepo.UnsetAllDefaults(ctx, userID); err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to unset defaults", 500)
		}
	}

	if err := s.addressRepo.Create(ctx, address); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to create address", 500)
	}

	return address, nil
}

func (s *ProfileService) GetAddresses(ctx context.Context, userID string) ([]*entity.Address, error) {
	addresses, err := s.addressRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch addresses", 500)
	}
	return addresses, nil
}

func (s *ProfileService) GetAddressByID(ctx context.Context, userID string, addressID string) (*entity.Address, error) {
	objID, err := primitive.ObjectIDFromHex(addressID)
	if err != nil {
		return nil, errors.New("INVALID_ADDRESS_ID", "Invalid address ID format", 400)
	}

	address, err := s.addressRepo.FindByIDAndUserID(ctx, objID, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrAddressNotFound
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch address", 500)
	}

	return address, nil
}

func (s *ProfileService) UpdateAddress(ctx context.Context, userID, addressID string, input *UpdateAddressInput) (*entity.Address, error) {
	address, err := s.GetAddressByID(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if input.ContactName != "" {
		address.ContactName = input.ContactName
	}
	if input.Phone != "" {
		address.Phone = input.Phone
	}
	if input.ProvinceCode != "" {
		address.ProvinceCode = input.ProvinceCode
	}
	if input.DistrictCode != "" {
		address.DistrictCode = input.DistrictCode
	}
	if input.WardCode != "" {
		address.WardCode = input.WardCode
	}
	if input.StreetLine != "" {
		address.StreetAddress = input.StreetLine
	}
	if input.FullAddress != "" {
		address.FullAddress = input.FullAddress
	}
	if input.Type != "" {
		address.Type = input.Type
	}

	// Handle default flag
	if input.IsDefault && !address.IsDefault {
		if err := s.addressRepo.UnsetAllDefaults(ctx, userID); err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to unset defaults", 500)
		}
		address.IsDefault = true
	}

	address.UpdatedAt = time.Now()

	if err := s.addressRepo.Update(ctx, address); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to update address", 500)
	}

	return address, nil
}

func (s *ProfileService) SetDefaultAddress(ctx context.Context, userID, addressID string) (*entity.Address, error) {
	objID, err := primitive.ObjectIDFromHex(addressID)
	if err != nil {
		return nil, errors.New("INVALID_ADDRESS_ID", "Invalid address ID format", 400)
	}

	// Verify address exists and belongs to user
	address, err := s.addressRepo.FindByIDAndUserID(ctx, objID, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrAddressNotFound
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch address", 500)
	}

	// Unset all defaults
	if err := s.addressRepo.UnsetAllDefaults(ctx, userID); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to unset defaults", 500)
	}

	// Set this one as default
	if err := s.addressRepo.SetDefault(ctx, userID, objID); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to set default", 500)
	}

	address.IsDefault = true
	return address, nil
}

func (s *ProfileService) DeleteAddress(ctx context.Context, userID, addressID string) error {
	objID, err := primitive.ObjectIDFromHex(addressID)
	if err != nil {
		return errors.New("INVALID_ADDRESS_ID", "Invalid address ID format", 400)
	}

	// Verify address exists and belongs to user
	address, err := s.addressRepo.FindByIDAndUserID(ctx, objID, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.ErrAddressNotFound
		}
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch address", 500)
	}

	wasDefault := address.IsDefault

	// Delete the address
	if err := s.addressRepo.Delete(ctx, objID); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to delete address", 500)
	}

	// If deleted address was default, promote most recent
	if wasDefault {
		recent, err := s.addressRepo.FindMostRecentByUserID(ctx, userID)
		if err == nil && recent != nil {
			s.addressRepo.SetDefault(ctx, userID, recent.ID)
		}
	}

	return nil
}

// Shop operations

func (s *ProfileService) RegisterShop(ctx context.Context, userID string, input *RegisterShopInput) (*entity.Profile, error) {
	profile, err := s.GetOrCreateProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if shop name is unique
	exists, err := s.profileRepo.ExistsByShopName(ctx, input.ShopName, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to check shop name", 500)
	}
	if exists {
		return nil, errors.ErrShopNameExists
	}

	// Create shop config
	profile.ShopConfig = &entity.ShopConfig{
		ShopID:             uuid.New().String(),
		ShopName:           input.ShopName,
		ShopSlug:           generateSlug(input.ShopName),
		Description:        input.Description,
		LogoURL:            input.LogoURL,
		BusinessType:       entity.BusinessTypeIndividual,
		VerificationStatus: entity.VerificationStatusPending,
		JoinedAt:           time.Now(),
	}

	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to register shop", 500)
	}

	return profile, nil
}

func (s *ProfileService) UpdateShop(ctx context.Context, userID string, input *UpdateShopInput) (*entity.Profile, error) {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrProfileNotFound
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch profile", 500)
	}

	if !profile.HasShop() {
		return nil, errors.ErrShopNotRegistered
	}

	// Check shop name uniqueness if changing
	if input.ShopName != "" && input.ShopName != profile.ShopConfig.ShopName {
		exists, err := s.profileRepo.ExistsByShopName(ctx, input.ShopName, userID)
		if err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to check shop name", 500)
		}
		if exists {
			return nil, errors.ErrShopNameExists
		}
		profile.ShopConfig.ShopName = input.ShopName
		profile.ShopConfig.ShopSlug = generateSlug(input.ShopName)
	}

	if input.Description != "" {
		profile.ShopConfig.Description = input.Description
	}
	if input.LogoURL != "" {
		profile.ShopConfig.LogoURL = input.LogoURL
	}

	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to update shop", 500)
	}

	return profile, nil
}

// Internal API operations

func (s *ProfileService) GetUserInfo(ctx context.Context, userID string) (*UserInfoResponse, error) {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default info
			return &UserInfoResponse{
				UserID:      userID,
				DisplayName: "User-" + userID[len(userID)-4:],
			}, nil
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch profile", 500)
	}

	return &UserInfoResponse{
		UserID:      profile.UserID,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Email:       profile.Email,
	}, nil
}

// Helper functions

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

// Input/Output types

type CreateAddressInput struct {
	ContactName  string             `json:"contactName" validate:"required,max=100"`
	Phone        string             `json:"phone" validate:"required,max=20"`
	ProvinceCode string             `json:"provinceCode" validate:"required"`
	DistrictCode string             `json:"districtCode" validate:"required"`
	WardCode     string             `json:"wardCode" validate:"required"`
	StreetLine   string             `json:"streetLine" validate:"required"`
	FullAddress  string             `json:"fullAddress,omitempty"`
	IsDefault    bool               `json:"isDefault,omitempty"`
	Type         entity.AddressType `json:"type,omitempty"`
}

type UpdateAddressInput struct {
	ContactName  string             `json:"contactName,omitempty"`
	Phone        string             `json:"phone,omitempty"`
	ProvinceCode string             `json:"provinceCode,omitempty"`
	DistrictCode string             `json:"districtCode,omitempty"`
	WardCode     string             `json:"wardCode,omitempty"`
	StreetLine   string             `json:"streetLine,omitempty"`
	FullAddress  string             `json:"fullAddress,omitempty"`
	IsDefault    bool               `json:"isDefault,omitempty"`
	Type         entity.AddressType `json:"type,omitempty"`
}

type RegisterShopInput struct {
	ShopName    string `json:"shopName" validate:"required"`
	Description string `json:"description,omitempty"`
	LogoURL     string `json:"logoUrl,omitempty" validate:"omitempty,url"`
}

type UpdateShopInput struct {
	ShopName    string `json:"shopName,omitempty"`
	Description string `json:"description,omitempty"`
	LogoURL     string `json:"logoUrl,omitempty" validate:"omitempty,url"`
}

type UserInfoResponse struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Email       string `json:"email,omitempty"`
}
