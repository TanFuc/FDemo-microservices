package entity

import (
	"time"
)

const (
	RoleSuperAdmin = "SUPER_ADMIN"
	RoleAdmin      = "ADMIN"
	RoleManager    = "MANAGER"
	RoleSale       = "SALE"
	RoleSeller     = "SELLER"
	RoleUser       = "USER"
)

type Role struct {
	ID              int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string           `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	DisplayName     string           `gorm:"type:varchar(100);not null" json:"displayName"`
	Description     *string          `gorm:"type:text" json:"description,omitempty"`
	IsSystem        bool             `gorm:"default:false" json:"isSystem"`
	IsDefault       bool             `gorm:"default:false;index" json:"isDefault"`
	Priority        int              `gorm:"default:0;index" json:"priority"`
	Metadata        map[string]any   `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt       time.Time        `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	RolePermissions []RolePermission `gorm:"foreignKey:RoleID" json:"rolePermissions,omitempty"`
	UserRoles       []UserRole       `gorm:"foreignKey:RoleID" json:"-"`
}

func (Role) TableName() string {
	return "roles"
}

func (r *Role) IsSuperAdmin() bool {
	return r.Name == RoleSuperAdmin
}

var DefaultRoles = []Role{
	{
		Name:        RoleSuperAdmin,
		DisplayName: "Super Administrator",
		Description: strPtr("Full system access with all permissions"),
		IsSystem:    true,
		IsDefault:   false,
		Priority:    100,
	},
	{
		Name:        RoleAdmin,
		DisplayName: "Administrator",
		Description: strPtr("Platform administration and management"),
		IsSystem:    true,
		IsDefault:   false,
		Priority:    80,
	},
	{
		Name:        RoleManager,
		DisplayName: "Manager",
		Description: strPtr("Operations management including orders, shops, and reviews"),
		IsSystem:    true,
		IsDefault:   false,
		Priority:    60,
	},
	{
		Name:        RoleSale,
		DisplayName: "Sale",
		Description: strPtr("Marketing, campaigns, vouchers, and promotions"),
		IsSystem:    true,
		IsDefault:   false,
		Priority:    40,
	},
	{
		Name:        RoleSeller,
		DisplayName: "Seller",
		Description: strPtr("Merchant/shop owner account"),
		IsSystem:    true,
		IsDefault:   false,
		Priority:    30,
	},
	{
		Name:        RoleUser,
		DisplayName: "User",
		Description: strPtr("Regular buyer account"),
		IsSystem:    true,
		IsDefault:   true,
		Priority:    10,
	},
}

func strPtr(s string) *string {
	return &s
}
