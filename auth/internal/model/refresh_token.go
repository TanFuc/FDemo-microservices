package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeviceType string

const (
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeTablet  DeviceType = "tablet"
)

type RefreshToken struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash     string     `gorm:"type:varchar(255);not null;index" json:"-"`
	DeviceID      *string    `gorm:"type:varchar(100);index" json:"deviceId,omitempty"`
	DeviceName    *string    `gorm:"type:varchar(255)" json:"deviceName,omitempty"`
	DeviceType    *string    `gorm:"type:varchar(20)" json:"deviceType,omitempty"`
	Browser       *string    `gorm:"type:varchar(100)" json:"browser,omitempty"`
	OS            *string    `gorm:"type:varchar(100)" json:"os,omitempty"`
	IPAddress     string     `gorm:"type:varchar(45);not null" json:"ipAddress"`
	Location      *string    `gorm:"type:varchar(100)" json:"location,omitempty"`
	ExpiresAt     time.Time  `gorm:"type:timestamptz;not null;index" json:"expiresAt"`
	IsRevoked     bool       `gorm:"default:false;index" json:"isRevoked"`
	RevokedAt     *time.Time `gorm:"type:timestamptz" json:"revokedAt,omitempty"`
	RevokedReason *string    `gorm:"type:varchar(255)" json:"revokedReason,omitempty"`
	CreatedAt     time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`

	// Relations
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (rt *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	return nil
}

func (rt *RefreshToken) IsValid() bool {
	if rt.IsRevoked {
		return false
	}
	return time.Now().Before(rt.ExpiresAt)
}

func (rt *RefreshToken) Revoke(reason string) {
	now := time.Now()
	rt.IsRevoked = true
	rt.RevokedAt = &now
	rt.RevokedReason = &reason
}
