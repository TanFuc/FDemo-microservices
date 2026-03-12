package usercontext

import (
	"time"
)

// UserRole represents the user's role in the system
type UserRole string

const (
	RoleCustomer UserRole = "CUSTOMER"
	RoleSeller   UserRole = "SELLER"
	RoleAdmin    UserRole = "ADMIN"
)

// UserStatus represents the user's account status
type UserStatus string

const (
	StatusActive   UserStatus = "ACTIVE"
	StatusInactive UserStatus = "INACTIVE"
	StatusBanned   UserStatus = "BANNED"
	StatusPending  UserStatus = "PENDING"
)

// UserContext represents the federated user meta-data model
// This struct should be replicated across all downstream services
type UserContext struct {
	UserID      string     `bson:"userId" json:"userId" gorm:"column:user_id;primaryKey;size:255"`
	Role        UserRole   `bson:"role" json:"role" gorm:"column:role;size:50;not null"`
	ShopID      string     `bson:"shopId,omitempty" json:"shopId,omitempty" gorm:"column:shop_id;size:255"`
	DisplayName string     `bson:"displayName,omitempty" json:"displayName,omitempty" gorm:"column:display_name;size:255"`
	AvatarURL   string     `bson:"avatarUrl,omitempty" json:"avatarUrl,omitempty" gorm:"column:avatar_url;size:500"`
	Email       string     `bson:"email,omitempty" json:"email,omitempty" gorm:"column:email;size:255"`
	Status      UserStatus `bson:"status" json:"status" gorm:"column:status;size:50;default:ACTIVE"`
	Version     int64      `bson:"version" json:"version" gorm:"column:version;default:0"`
	CreatedAt   time.Time  `bson:"createdAt" json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `bson:"updatedAt" json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for GORM (PostgreSQL)
func (UserContext) TableName() string {
	return "users"
}

// IsSeller checks if the user is a seller
func (u *UserContext) IsSeller() bool {
	return u.Role == RoleSeller && u.ShopID != ""
}

// IsAdmin checks if the user is an admin
func (u *UserContext) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsActive checks if the user account is active
func (u *UserContext) IsActive() bool {
	return u.Status == StatusActive
}

// CanManageResource checks if the user can manage a resource owned by ownerID
func (u *UserContext) CanManageResource(ownerID string) bool {
	if u.IsAdmin() {
		return true
	}
	return u.UserID == ownerID
}

// CanManageShopResource checks if the user can manage a shop-owned resource
func (u *UserContext) CanManageShopResource(shopID string) bool {
	if u.IsAdmin() {
		return true
	}
	return u.IsSeller() && u.ShopID == shopID
}

// ShopContext represents minimal shop information for ownership validation
type ShopContext struct {
	ShopID             string     `bson:"shopId" json:"shopId" gorm:"column:shop_id;primaryKey;size:255"`
	OwnerUserID        string     `bson:"ownerUserId" json:"ownerUserId" gorm:"column:owner_user_id;size:255;not null"`
	ShopName           string     `bson:"shopName" json:"shopName" gorm:"column:shop_name;size:255"`
	Status             UserStatus `bson:"status" json:"status" gorm:"column:status;size:50;default:PENDING"`
	VerificationStatus string     `bson:"verificationStatus" json:"verificationStatus" gorm:"column:verification_status;size:50;default:PENDING"`
	Version            int64      `bson:"version" json:"version" gorm:"column:version;default:0"`
	CreatedAt          time.Time  `bson:"createdAt" json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `bson:"updatedAt" json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for GORM (PostgreSQL)
func (ShopContext) TableName() string {
	return "shops"
}
