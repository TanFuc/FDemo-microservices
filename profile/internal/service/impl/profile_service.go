package impl

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"microservices/profile/internal/model"
	"microservices/profile/internal/repository"
	"microservices/profile/internal/service"
	"microservices/profile/pkg/errors"
	"microservices/profile/pkg/logger"
)

type profileService struct {
	profileRepo repository.ProfileRepository
	addressRepo repository.AddressRepository
	mongoClient *mongo.Client
}

func NewProfileService(
	profileRepo repository.ProfileRepository,
	addressRepo repository.AddressRepository,
	mongoClient *mongo.Client,
) service.ProfileService {
	return &profileService{
		profileRepo: profileRepo,
		addressRepo: addressRepo,
		mongoClient: mongoClient,
	}
}

// Profile operations

func (s *profileService) GetOrCreateProfile(ctx context.Context, userID string) (*model.Profile, error) {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err == nil {
		return profile, nil
	}

	if err != mongo.ErrNoDocuments {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch profile", 500)
	}

	// Create default profile
	displayName := "User-" + userID[len(userID)-4:]
	profile = model.NewProfile(userID, displayName, "")

	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to create profile", 500)
	}

	return profile, nil
}

func (s *profileService) CreateInitialProfile(ctx context.Context, userID, displayName, email string) (*model.Profile, error) {
	// Check if profile already exists
	exists, err := s.profileRepo.ExistsByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to check profile existence", 500)
	}
	if exists {
		// Return existing profile
		return s.profileRepo.FindByUserID(ctx, userID)
	}

	profile := model.NewProfile(userID, displayName, email)
	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to create profile", 500)
	}

	logger.Info().
		Str("userId", userID).
		Str("displayName", displayName).
		Msg("Created initial profile")

	return profile, nil
}

func (s *profileService) GetProfileByUserID(ctx context.Context, userID string) (*model.Profile, error) {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrProfileNotFound
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch profile", 500)
	}
	return profile, nil
}

func (s *profileService) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) (*model.Profile, error) {
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

// Address operations

func (s *profileService) AddAddress(ctx context.Context, userID string, req *model.CreateAddressRequest) (*model.Address, error) {
	address := model.NewAddress(userID)
	address.ContactName = req.ContactName
	address.Phone = req.Phone
	address.CountryCode = "VN" // Default to Vietnam
	address.ProvinceCode = req.ProvinceCode
	address.DistrictCode = req.DistrictCode
	address.WardCode = req.WardCode
	address.StreetAddress = req.StreetLine
	address.Type = req.Type
	address.IsDefault = req.IsDefault

	if req.FullAddress != "" {
		address.FullAddress = req.FullAddress
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

func (s *profileService) GetAddresses(ctx context.Context, userID string) ([]*model.Address, error) {
	addresses, err := s.addressRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch addresses", 500)
	}
	return addresses, nil
}

func (s *profileService) GetAddressByID(ctx context.Context, userID string, addressID string) (*model.Address, error) {
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

func (s *profileService) UpdateAddress(ctx context.Context, userID, addressID string, req *model.UpdateAddressRequest) (*model.Address, error) {
	address, err := s.GetAddressByID(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.ContactName != "" {
		address.ContactName = req.ContactName
	}
	if req.Phone != "" {
		address.Phone = req.Phone
	}
	if req.ProvinceCode != "" {
		address.ProvinceCode = req.ProvinceCode
	}
	if req.DistrictCode != "" {
		address.DistrictCode = req.DistrictCode
	}
	if req.WardCode != "" {
		address.WardCode = req.WardCode
	}
	if req.StreetLine != "" {
		address.StreetAddress = req.StreetLine
	}
	if req.FullAddress != "" {
		address.FullAddress = req.FullAddress
	}
	if req.Type != "" {
		address.Type = req.Type
	}

	// Handle default flag
	if req.IsDefault && !address.IsDefault {
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

func (s *profileService) SetDefaultAddress(ctx context.Context, userID, addressID string) (*model.Address, error) {
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

func (s *profileService) DeleteAddress(ctx context.Context, userID, addressID string) error {
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

func (s *profileService) RegisterShop(ctx context.Context, userID string, req *model.RegisterShopRequest) (*model.Profile, error) {
	profile, err := s.GetOrCreateProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if user already has a shop
	if profile.HasShop() {
		return nil, errors.ErrAlreadyHasShop
	}

	// Check if shop name is unique
	exists, err := s.profileRepo.ExistsByShopName(ctx, req.ShopName, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to check shop name", 500)
	}
	if exists {
		return nil, errors.ErrShopNameExists
	}

	// Determine business type (default to INDIVIDUAL)
	businessType := req.BusinessType
	if businessType == "" {
		businessType = model.BusinessTypeIndividual
	}

	// Validate BUSINESS type requirements
	if businessType == model.BusinessTypeBusiness {
		// Must have at least one image
		if req.LogoURL == "" && req.BannerURL == "" {
			return nil, errors.ErrBusinessRequiresImage
		}
		// Must have complete shop address
		if req.ShopAddress == nil {
			return nil, errors.ErrBusinessRequiresAddress
		}
	}

	// Use MongoDB session for transaction (atomicity for BUSINESS type with address)
	if businessType == model.BusinessTypeBusiness && req.ShopAddress != nil {
		return s.registerBusinessShopWithTransaction(ctx, profile, req, businessType)
	}

	// For INDIVIDUAL type, no transaction needed
	profile.ShopConfig = &model.ShopConfig{
		ShopID:             uuid.New().String(),
		ShopName:           req.ShopName,
		ShopSlug:           generateSlug(req.ShopName),
		Description:        req.Description,
		LogoURL:            req.LogoURL,
		BannerURL:          req.BannerURL,
		BusinessType:       businessType,
		VerificationStatus: model.VerificationStatusPending,
		JoinedAt:           time.Now(),
	}

	profile.UpdatedAt = time.Now()
	profile.Version++

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to register shop", 500)
	}

	logger.Info().
		Str("userId", userID).
		Str("shopId", profile.ShopConfig.ShopID).
		Str("businessType", string(businessType)).
		Msg("Shop registered successfully")

	return profile, nil
}

func (s *profileService) registerBusinessShopWithTransaction(
	ctx context.Context,
	profile *model.Profile,
	req *model.RegisterShopRequest,
	businessType model.BusinessType,
) (*model.Profile, error) {
	session, err := s.mongoClient.StartSession()
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to start session", 500)
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		// Create address first
		req.ShopAddress.Type = model.AddressTypeOther
		req.ShopAddress.IsDefault = false
		address, err := s.AddAddress(sc, profile.UserID, req.ShopAddress)
		if err != nil {
			return nil, err
		}

		// Create shop config with the address ID
		profile.ShopConfig = &model.ShopConfig{
			ShopID:             uuid.New().String(),
			ShopName:           req.ShopName,
			ShopSlug:           generateSlug(req.ShopName),
			Description:        req.Description,
			LogoURL:            req.LogoURL,
			BannerURL:          req.BannerURL,
			PickupAddressID:    &address.ID,
			BusinessType:       businessType,
			VerificationStatus: model.VerificationStatusPending,
			JoinedAt:           time.Now(),
		}

		profile.UpdatedAt = time.Now()
		profile.Version++

		if err := s.profileRepo.Update(sc, profile); err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to register shop", 500)
		}

		return profile, nil
	})

	if err != nil {
		return nil, err
	}

	logger.Info().
		Str("userId", profile.UserID).
		Str("shopId", profile.ShopConfig.ShopID).
		Str("businessType", string(businessType)).
		Msg("Business shop registered successfully with address")

	return result.(*model.Profile), nil
}

func (s *profileService) UpdateShop(ctx context.Context, userID string, req *model.UpdateShopRequest) (*model.Profile, error) {
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
	if req.ShopName != "" && req.ShopName != profile.ShopConfig.ShopName {
		exists, err := s.profileRepo.ExistsByShopName(ctx, req.ShopName, userID)
		if err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to check shop name", 500)
		}
		if exists {
			return nil, errors.ErrShopNameExists
		}
		profile.ShopConfig.ShopName = req.ShopName
		profile.ShopConfig.ShopSlug = generateSlug(req.ShopName)
	}

	if req.Description != "" {
		profile.ShopConfig.Description = req.Description
	}
	if req.LogoURL != "" {
		profile.ShopConfig.LogoURL = req.LogoURL
	}

	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to update shop", 500)
	}

	return profile, nil
}

// Affiliate operations

func (s *profileService) RegisterAffiliate(ctx context.Context, userID string, req *model.RegisterAffiliateRequest) (*model.Profile, error) {
	profile, err := s.GetOrCreateProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if user is already an affiliate
	if profile.IsAffiliate() {
		return nil, errors.ErrAlreadyAffiliate
	}

	// Validate referral code if provided
	if req.ReferredBy != "" {
		// Prevent self-referral
		if profile.AffiliateConfig != nil && profile.AffiliateConfig.AffiliateCode == req.ReferredBy {
			return nil, errors.ErrSelfReferral
		}

		// Check if referral code exists
		referrerProfile, err := s.profileRepo.FindByAffiliateCode(ctx, req.ReferredBy)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, errors.ErrInvalidReferralCode
			}
			return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to verify referral code", 500)
		}

		// Prevent referring yourself by checking userID
		if referrerProfile.UserID == userID {
			return nil, errors.ErrSelfReferral
		}
	}

	// Generate unique affiliate code with collision handling
	affiliateCode, err := s.generateUniqueAffiliateCode(ctx)
	if err != nil {
		return nil, err
	}

	// Create affiliate config
	profile.AffiliateConfig = &model.AffiliateConfig{
		AffiliateCode:   affiliateCode,
		Status:          model.AffiliateStatusActive,
		CommissionRate:  0.05, // Default 5% commission
		TotalReferrals:  0,
		TotalEarnings:   0,
		PendingEarnings: 0,
		PayoutThreshold: 100000, // Default payout threshold in VND
		ReferredBy:      req.ReferredBy,
		JoinedAt:        time.Now(),
	}

	profile.UpdatedAt = time.Now()
	profile.Version++

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to register affiliate", 500)
	}

	logger.Info().
		Str("userId", userID).
		Str("affiliateCode", affiliateCode).
		Str("referredBy", req.ReferredBy).
		Msg("User registered as affiliate")

	return profile, nil
}

func (s *profileService) generateUniqueAffiliateCode(ctx context.Context) (string, error) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		code := generateAffiliateCode()
		exists, err := s.profileRepo.ExistsByAffiliateCode(ctx, code)
		if err != nil {
			return "", errors.Wrap(err, "DATABASE_ERROR", "Failed to check affiliate code", 500)
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.New("AFFILIATE_CODE_GENERATION_FAILED", "Failed to generate unique affiliate code after retries", 500)
}

// generateAffiliateCode generates a short unique code like REF-A9K2B
func generateAffiliateCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	// Use base32 without padding for cleaner codes
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
	// Take first 5 characters and uppercase
	if len(code) > 5 {
		code = code[:5]
	}
	return "REF-" + strings.ToUpper(code)
}

// Internal API operations

func (s *profileService) GetUserInfo(ctx context.Context, userID string) (*model.UserInfoResponse, error) {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default info
			return &model.UserInfoResponse{
				UserID:      userID,
				DisplayName: "User-" + userID[len(userID)-4:],
			}, nil
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to fetch profile", 500)
	}

	return profile.ToUserInfoResponse(), nil
}

// Helper functions

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
