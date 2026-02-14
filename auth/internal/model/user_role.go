package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole struct {
	ID        int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_user_role" json:"userId"`
	RoleID    int64      `gorm:"not null;uniqueIndex:idx_user_role" json:"roleId"`
	GrantedBy *uuid.UUID `gorm:"type:uuid" json:"grantedBy,omitempty"`
	ExpiresAt *time.Time `gorm:"type:timestamptz;index" json:"expiresAt,omitempty"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`

	// Relations
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Role Role `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"role,omitempty"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

func (ur *UserRole) IsExpired() bool {
	if ur.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ur.ExpiresAt)
}
