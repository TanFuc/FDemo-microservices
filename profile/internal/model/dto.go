package model

// CreateAddressRequest represents the request to create an address
type CreateAddressRequest struct {
	ContactName  string      `json:"contactName" validate:"required,max=100"`
	Phone        string      `json:"phone" validate:"required,max=20"`
	ProvinceCode string      `json:"provinceCode" validate:"required"`
	DistrictCode string      `json:"districtCode" validate:"required"`
	WardCode     string      `json:"wardCode" validate:"required"`
	StreetLine   string      `json:"streetLine" validate:"required"`
	FullAddress  string      `json:"fullAddress,omitempty"`
	IsDefault    bool        `json:"isDefault,omitempty"`
	Type         AddressType `json:"type,omitempty"`
}

// UpdateAddressRequest represents the request to update an address
type UpdateAddressRequest struct {
	ContactName  string      `json:"contactName,omitempty"`
	Phone        string      `json:"phone,omitempty"`
	ProvinceCode string      `json:"provinceCode,omitempty"`
	DistrictCode string      `json:"districtCode,omitempty"`
	WardCode     string      `json:"wardCode,omitempty"`
	StreetLine   string      `json:"streetLine,omitempty"`
	FullAddress  string      `json:"fullAddress,omitempty"`
	IsDefault    bool        `json:"isDefault,omitempty"`
	Type         AddressType `json:"type,omitempty"`
}

// RegisterShopRequest represents the request to register a shop
type RegisterShopRequest struct {
	ShopName    string `json:"shopName" validate:"required"`
	Description string `json:"description,omitempty"`
	LogoURL     string `json:"logoUrl,omitempty" validate:"omitempty,url"`
}

// UpdateShopRequest represents the request to update shop details
type UpdateShopRequest struct {
	ShopName    string `json:"shopName,omitempty"`
	Description string `json:"description,omitempty"`
	LogoURL     string `json:"logoUrl,omitempty" validate:"omitempty,url"`
}

// UserInfoResponse represents the user info response for internal API
type UserInfoResponse struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Email       string `json:"email,omitempty"`
}

// ProfileResponse represents the profile response
type ProfileResponse struct {
	ID                  string           `json:"id"`
	UserID              string           `json:"userId"`
	DisplayName         string           `json:"displayName"`
	Email               string           `json:"email,omitempty"`
	Phone               string           `json:"phone,omitempty"`
	AvatarURL           string           `json:"avatarUrl,omitempty"`
	CoverURL            string           `json:"coverUrl,omitempty"`
	Bio                 string           `json:"bio,omitempty"`
	Gender              Gender           `json:"gender,omitempty"`
	MembershipTier      MembershipTier   `json:"membershipTier"`
	LoyaltyPoints       int              `json:"loyaltyPoints"`
	IsPhoneVerified     bool             `json:"isPhoneVerified"`
	IsEmailVerified     bool             `json:"isEmailVerified"`
	IsIdentityVerified  bool             `json:"isIdentityVerified"`
	Preferences         *UserPreferences `json:"preferences,omitempty"`
	ShopConfig          *ShopConfig      `json:"shopConfig,omitempty"`
	Stats               *ProfileStats    `json:"stats,omitempty"`
	CreatedAt           string           `json:"createdAt"`
	UpdatedAt           string           `json:"updatedAt"`
}

// AddressResponse represents the address response
type AddressResponse struct {
	ID           string      `json:"id"`
	UserID       string      `json:"userId"`
	ContactName  string      `json:"contactName"`
	Phone        string      `json:"phone"`
	CountryCode  string      `json:"countryCode"`
	ProvinceCode string      `json:"provinceCode"`
	DistrictCode string      `json:"districtCode"`
	WardCode     string      `json:"wardCode"`
	StreetAddress string     `json:"streetAddress"`
	FullAddress  string      `json:"fullAddress"`
	Type         AddressType `json:"type"`
	IsDefault    bool        `json:"isDefault"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
}

// ToResponse converts Profile entity to ProfileResponse
func (p *Profile) ToResponse() *ProfileResponse {
	return &ProfileResponse{
		ID:                 p.ID.Hex(),
		UserID:             p.UserID,
		DisplayName:        p.DisplayName,
		Email:              p.Email,
		Phone:              p.Phone,
		AvatarURL:          p.AvatarURL,
		CoverURL:           p.CoverURL,
		Bio:                p.Bio,
		Gender:             p.Gender,
		MembershipTier:     p.MembershipTier,
		LoyaltyPoints:      p.LoyaltyPoints,
		IsPhoneVerified:    p.IsPhoneVerified,
		IsEmailVerified:    p.IsEmailVerified,
		IsIdentityVerified: p.IsIdentityVerified,
		Preferences:        p.Preferences,
		ShopConfig:         p.ShopConfig,
		Stats:              p.Stats,
		CreatedAt:          p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ToResponse converts Address entity to AddressResponse
func (a *Address) ToResponse() *AddressResponse {
	return &AddressResponse{
		ID:            a.ID.Hex(),
		UserID:        a.UserID,
		ContactName:   a.ContactName,
		Phone:         a.Phone,
		CountryCode:   a.CountryCode,
		ProvinceCode:  a.ProvinceCode,
		DistrictCode:  a.DistrictCode,
		WardCode:      a.WardCode,
		StreetAddress: a.StreetAddress,
		FullAddress:   a.FullAddress,
		Type:          a.Type,
		IsDefault:     a.IsDefault,
		CreatedAt:     a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ToUserInfoResponse converts Profile entity to UserInfoResponse
func (p *Profile) ToUserInfoResponse() *UserInfoResponse {
	return &UserInfoResponse{
		UserID:      p.UserID,
		DisplayName: p.DisplayName,
		AvatarURL:   p.AvatarURL,
		Email:       p.Email,
	}
}
