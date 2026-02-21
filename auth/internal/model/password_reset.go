package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PasswordReset struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash string     `gorm:"type:varchar(255);not null" json:"-"`
	ExpiresAt time.Time  `gorm:"type:timestamptz;not null" json:"expiresAt"`
	UsedAt    *time.Time `gorm:"type:timestamptz" json:"usedAt,omitempty"`
	IPAddress string     `gorm:"type:varchar(45);not null" json:"ipAddress"`
	UserAgent *string    `gorm:"type:varchar(500)" json:"userAgent,omitempty"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
}

func (PasswordReset) TableName() string {
	return "password_resets"
}

func (pr *PasswordReset) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	return nil
}

func (pr *PasswordReset) IsValid() bool {
	if pr.UsedAt != nil {
		return false
	}
	return time.Now().Before(pr.ExpiresAt)
}

func (pr *PasswordReset) MarkUsed() {
	now := time.Now()
	pr.UsedAt = &now
}
