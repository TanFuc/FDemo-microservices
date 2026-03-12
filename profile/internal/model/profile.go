package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Gender string

const (
	GenderMale           Gender = "MALE"
	GenderFemale         Gender = "FEMALE"
	GenderOther          Gender = "OTHER"
	GenderPreferNotToSay Gender = "PREFER_NOT_TO_SAY"
)

type MembershipTier string

const (
	MembershipTierBronze   MembershipTier = "BRONZE"
	MembershipTierSilver   MembershipTier = "SILVER"
	MembershipTierGold     MembershipTier = "GOLD"
	MembershipTierPlatinum MembershipTier = "PLATINUM"
	MembershipTierDiamond  MembershipTier = "DIAMOND"
)

type VerificationStatus string

const (
	VerificationStatusPending  VerificationStatus = "PENDING"
	VerificationStatusVerified VerificationStatus = "VERIFIED"
	VerificationStatusRejected VerificationStatus = "REJECTED"
)

type BusinessType string

const (
	BusinessTypeIndividual BusinessType = "INDIVIDUAL"
	BusinessTypeBusiness   BusinessType = "BUSINESS"
)

type MeasurementUnit string

const (
	MeasurementUnitMetric   MeasurementUnit = "METRIC"
	MeasurementUnitImperial MeasurementUnit = "IMPERIAL"
)

type AllowMessages string

const (
	AllowMessagesEveryone  AllowMessages = "EVERYONE"
	AllowMessagesFollowing AllowMessages = "FOLLOWING"
	AllowMessagesNobody    AllowMessages = "NOBODY"
)

type Profile struct {
	ID                  primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID              string                 `bson:"userId" json:"userId"`
	DisplayName         string                 `bson:"displayName" json:"displayName"`
	Email               string                 `bson:"email,omitempty" json:"email,omitempty"`
	Phone               string                 `bson:"phone,omitempty" json:"phone,omitempty"`
	AvatarURL           string                 `bson:"avatarUrl,omitempty" json:"avatarUrl,omitempty"`
	CoverURL            string                 `bson:"coverUrl,omitempty" json:"coverUrl,omitempty"`
	Bio                 string                 `bson:"bio,omitempty" json:"bio,omitempty"`
	DateOfBirth         *time.Time             `bson:"dateOfBirth,omitempty" json:"dateOfBirth,omitempty"`
	Gender              Gender                 `bson:"gender,omitempty" json:"gender,omitempty"`
	CountryCode         string                 `bson:"countryCode,omitempty" json:"countryCode,omitempty"`
	City                string                 `bson:"city,omitempty" json:"city,omitempty"`
	Preferences         *UserPreferences       `bson:"preferences,omitempty" json:"preferences,omitempty"`
	ShopConfig          *ShopConfig            `bson:"shopConfig,omitempty" json:"shopConfig,omitempty"`
	AffiliateConfig     *AffiliateConfig       `bson:"affiliateConfig,omitempty" json:"affiliateConfig,omitempty"`
	Stats               *ProfileStats          `bson:"stats,omitempty" json:"stats,omitempty"`
	Identity            *IdentityDocument      `bson:"identity,omitempty" json:"identity,omitempty"`
	MembershipTier      MembershipTier         `bson:"membershipTier" json:"membershipTier"`
	LoyaltyPoints       int                    `bson:"loyaltyPoints" json:"loyaltyPoints"`
	MembershipExpiresAt *time.Time             `bson:"membershipExpiresAt,omitempty" json:"membershipExpiresAt,omitempty"`
	IsPhoneVerified     bool                   `bson:"isPhoneVerified" json:"isPhoneVerified"`
	IsEmailVerified     bool                   `bson:"isEmailVerified" json:"isEmailVerified"`
	IsIdentityVerified  bool                   `bson:"isIdentityVerified" json:"isIdentityVerified"`
	FollowingShopIDs    []string               `bson:"followingShopIds" json:"followingShopIds"`
	FollowersCount      int                    `bson:"followersCount" json:"followersCount"`
	Metadata            map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	LastActiveAt        *time.Time             `bson:"lastActiveAt,omitempty" json:"lastActiveAt,omitempty"`
	IsDeleted           bool                   `bson:"isDeleted" json:"isDeleted"`
	DeletedAt           *time.Time             `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	Version             int64                  `bson:"version" json:"version"`
	CreatedAt           time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt           time.Time              `bson:"updatedAt" json:"updatedAt"`
}

type UserPreferences struct {
	Language        string                   `bson:"language" json:"language"`
	Currency        string                   `bson:"currency" json:"currency"`
	Timezone        string                   `bson:"timezone" json:"timezone"`
	MeasurementUnit MeasurementUnit          `bson:"measurementUnit" json:"measurementUnit"`
	Notifications   *NotificationPreferences `bson:"notifications,omitempty" json:"notifications,omitempty"`
	Privacy         *PrivacySettings         `bson:"privacy,omitempty" json:"privacy,omitempty"`
}

type NotificationPreferences struct {
	Email *EmailNotifications `bson:"email,omitempty" json:"email,omitempty"`
	Push  *PushNotifications  `bson:"push,omitempty" json:"push,omitempty"`
	SMS   *SMSNotifications   `bson:"sms,omitempty" json:"sms,omitempty"`
}

type EmailNotifications struct {
	Orders     bool `bson:"orders" json:"orders"`
	Promotions bool `bson:"promotions" json:"promotions"`
	News       bool `bson:"news" json:"news"`
}

type PushNotifications struct {
	Orders     bool `bson:"orders" json:"orders"`
	Promotions bool `bson:"promotions" json:"promotions"`
	Chat       bool `bson:"chat" json:"chat"`
}

type SMSNotifications struct {
	Orders bool `bson:"orders" json:"orders"`
	OTP    bool `bson:"otp" json:"otp"`
}

type PrivacySettings struct {
	ShowOnlineStatus bool          `bson:"showOnlineStatus" json:"showOnlineStatus"`
	ShowLastSeen     bool          `bson:"showLastSeen" json:"showLastSeen"`
	AllowMessages    AllowMessages `bson:"allowMessages" json:"allowMessages"`
}

type ShopConfig struct {
	ShopID             string                     `bson:"shopId" json:"shopId"`
	ShopName           string                     `bson:"shopName" json:"shopName"`
	ShopSlug           string                     `bson:"shopSlug" json:"shopSlug"`
	Description        string                     `bson:"description,omitempty" json:"description,omitempty"`
	LogoURL            string                     `bson:"logoUrl,omitempty" json:"logoUrl,omitempty"`
	BannerURL          string                     `bson:"bannerUrl,omitempty" json:"bannerUrl,omitempty"`
	PickupAddressID    *primitive.ObjectID        `bson:"pickupAddressId,omitempty" json:"pickupAddressId,omitempty"`
	BusinessType       BusinessType               `bson:"businessType" json:"businessType"`
	BusinessLicense    string                     `bson:"businessLicense,omitempty" json:"businessLicense,omitempty"`
	TaxID              string                     `bson:"taxId,omitempty" json:"taxId,omitempty"`
	BankAccount        *BankAccount               `bson:"bankAccount,omitempty" json:"bankAccount,omitempty"`
	Policies           *ShopPolicies              `bson:"policies,omitempty" json:"policies,omitempty"`
	OperatingHours     map[string]*OperatingHours `bson:"operatingHours,omitempty" json:"operatingHours,omitempty"`
	SocialLinks        *SocialLinks               `bson:"socialLinks,omitempty" json:"socialLinks,omitempty"`
	VerificationStatus VerificationStatus         `bson:"verificationStatus" json:"verificationStatus"`
	VerifiedAt         *time.Time                 `bson:"verifiedAt,omitempty" json:"verifiedAt,omitempty"`
	Rating             float64                    `bson:"rating" json:"rating"`
	TotalReviews       int                        `bson:"totalReviews" json:"totalReviews"`
	TotalProducts      int                        `bson:"totalProducts" json:"totalProducts"`
	TotalOrders        int                        `bson:"totalOrders" json:"totalOrders"`
	TotalFollowers     int                        `bson:"totalFollowers" json:"totalFollowers"`
	JoinedAt           time.Time                  `bson:"joinedAt" json:"joinedAt"`
}

type BankAccount struct {
	BankName      string `bson:"bankName" json:"bankName"`
	AccountNumber string `bson:"accountNumber" json:"accountNumber"`
	AccountHolder string `bson:"accountHolder" json:"accountHolder"`
	Branch        string `bson:"branch,omitempty" json:"branch,omitempty"`
}

type ShopPolicies struct {
	ReturnPolicy   string `bson:"returnPolicy,omitempty" json:"returnPolicy,omitempty"`
	ShippingPolicy string `bson:"shippingPolicy,omitempty" json:"shippingPolicy,omitempty"`
}

type OperatingHours struct {
	Open   string `bson:"open" json:"open"`
	Close  string `bson:"close" json:"close"`
	Closed bool   `bson:"closed" json:"closed"`
}

type SocialLinks struct {
	Facebook  string `bson:"facebook,omitempty" json:"facebook,omitempty"`
	Instagram string `bson:"instagram,omitempty" json:"instagram,omitempty"`
	TikTok    string `bson:"tiktok,omitempty" json:"tiktok,omitempty"`
	YouTube   string `bson:"youtube,omitempty" json:"youtube,omitempty"`
	Website   string `bson:"website,omitempty" json:"website,omitempty"`
}

type AffiliateStatus string

const (
	AffiliateStatusActive   AffiliateStatus = "ACTIVE"
	AffiliateStatusInactive AffiliateStatus = "INACTIVE"
	AffiliateStatusPending  AffiliateStatus = "PENDING"
)

type AffiliateConfig struct {
	AffiliateCode     string          `bson:"affiliateCode" json:"affiliateCode"`
	Status            AffiliateStatus `bson:"status" json:"status"`
	CommissionRate    float64         `bson:"commissionRate" json:"commissionRate"`
	TotalReferrals    int             `bson:"totalReferrals" json:"totalReferrals"`
	TotalEarnings     float64         `bson:"totalEarnings" json:"totalEarnings"`
	PendingEarnings   float64         `bson:"pendingEarnings" json:"pendingEarnings"`
	PayoutThreshold   float64         `bson:"payoutThreshold" json:"payoutThreshold"`
	ReferredBy        string          `bson:"referredBy,omitempty" json:"referredBy,omitempty"`
	JoinedAt          time.Time       `bson:"joinedAt" json:"joinedAt"`
	LastPayoutAt      *time.Time      `bson:"lastPayoutAt,omitempty" json:"lastPayoutAt,omitempty"`
}

type ProfileStats struct {
	TotalOrders       int     `bson:"totalOrders" json:"totalOrders"`
	CompletedOrders   int     `bson:"completedOrders" json:"completedOrders"`
	CancelledOrders   int     `bson:"cancelledOrders" json:"cancelledOrders"`
	TotalSpent        float64 `bson:"totalSpent" json:"totalSpent"`
	AverageOrderValue float64 `bson:"averageOrderValue" json:"averageOrderValue"`
	TotalReviews      int     `bson:"totalReviews" json:"totalReviews"`
	WishlistCount     int     `bson:"wishlistCount" json:"wishlistCount"`
	CartItemCount     int     `bson:"cartItemCount" json:"cartItemCount"`
}

type IdentityDocument struct {
	Type        string     `bson:"type,omitempty" json:"type,omitempty"`
	Number      string     `bson:"number,omitempty" json:"number,omitempty"`
	FrontImage  string     `bson:"frontImage,omitempty" json:"frontImage,omitempty"`
	BackImage   string     `bson:"backImage,omitempty" json:"backImage,omitempty"`
	SelfieImage string     `bson:"selfieImage,omitempty" json:"selfieImage,omitempty"`
	VerifiedAt  *time.Time `bson:"verifiedAt,omitempty" json:"verifiedAt,omitempty"`
}

func NewProfile(userID, displayName, email string) *Profile {
	now := time.Now()
	return &Profile{
		UserID:         userID,
		DisplayName:    displayName,
		Email:          email,
		MembershipTier: MembershipTierBronze,
		LoyaltyPoints:  0,
		Preferences: &UserPreferences{
			Language:        "vi",
			Currency:        "VND",
			Timezone:        "Asia/Ho_Chi_Minh",
			MeasurementUnit: MeasurementUnitMetric,
			Privacy: &PrivacySettings{
				ShowOnlineStatus: true,
				ShowLastSeen:     true,
				AllowMessages:    AllowMessagesEveryone,
			},
		},
		Stats:            &ProfileStats{},
		FollowingShopIDs: []string{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (p *Profile) HasShop() bool {
	return p.ShopConfig != nil && p.ShopConfig.ShopID != ""
}

func (p *Profile) IsAffiliate() bool {
	return p.AffiliateConfig != nil && p.AffiliateConfig.AffiliateCode != ""
}
