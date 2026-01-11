package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusInactive  UserStatus = "INACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
	UserStatusBanned    UserStatus = "BANNED"
	UserStatusPending   UserStatus = "PENDING"
)

type Gender string

const (
	GenderMale   Gender = "MALE"
	GenderFemale Gender = "FEMALE"
	GenderOther  Gender = "OTHER"
)

type User struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email               string         `gorm:"type:varchar(255);uniqueIndex:idx_users_email;not null" json:"email"`
	EmailVerified       bool           `gorm:"default:false" json:"emailVerified"`
	EmailVerifiedAt     *time.Time     `gorm:"type:timestamptz" json:"emailVerifiedAt,omitempty"`
	Phone               *string        `gorm:"type:varchar(20);uniqueIndex:idx_users_phone,where:phone IS NOT NULL" json:"phone,omitempty"`
	PhoneVerified       bool           `gorm:"default:false" json:"phoneVerified"`
	PasswordHash        string         `gorm:"type:varchar(255);not null" json:"-"`
	PasswordChangedAt   *time.Time     `gorm:"type:timestamptz" json:"passwordChangedAt,omitempty"`
	FullName            string         `gorm:"type:varchar(255);not null" json:"fullName"`
	DisplayName         *string        `gorm:"type:varchar(100)" json:"displayName,omitempty"`
	AvatarURL           *string        `gorm:"type:varchar(500)" json:"avatarUrl,omitempty"`
	Birthday            *time.Time     `gorm:"type:date" json:"birthday,omitempty"`
	Gender              *Gender        `gorm:"type:varchar(10)" json:"gender,omitempty"`
	Status              UserStatus     `gorm:"type:varchar(20);default:'ACTIVE';index:idx_users_status_created_at" json:"status"`
	SuspensionReason    *string        `gorm:"type:text" json:"suspensionReason,omitempty"`
	SuspendedUntil      *time.Time     `gorm:"type:timestamptz" json:"suspendedUntil,omitempty"`
	TwoFactorEnabled    bool           `gorm:"default:false" json:"twoFactorEnabled"`
	TwoFactorSecret     *string        `gorm:"type:varchar(255)" json:"-"`
	BackupCodes         []string       `gorm:"type:jsonb" json:"-"`
	FailedLoginAttempts int            `gorm:"default:0" json:"-"`
	LockedUntil         *time.Time     `gorm:"type:timestamptz" json:"-"`
	LastLoginAt         *time.Time     `gorm:"type:timestamptz" json:"lastLoginAt,omitempty"`
	LastLoginIP         *string        `gorm:"type:varchar(45)" json:"lastLoginIp,omitempty"`
	LoginCount          int            `gorm:"default:0" json:"loginCount"`
	Language            string         `gorm:"type:varchar(10);default:'vi'" json:"language"`
	Currency            string         `gorm:"type:varchar(3);default:'VND'" json:"currency"`
	Timezone            string         `gorm:"type:varchar(50);default:'Asia/Ho_Chi_Minh'" json:"timezone"`
	AcceptsMarketing    bool           `gorm:"default:false" json:"acceptsMarketing"`
	MarketingOptedInAt  *time.Time     `gorm:"type:timestamptz" json:"marketingOptedInAt,omitempty"`
	ReferralCode        string         `gorm:"type:varchar(20);uniqueIndex:idx_users_referral_code" json:"referralCode"`
	ReferredBy          *uuid.UUID     `gorm:"type:uuid" json:"referredBy,omitempty"`
	GoogleID            *string        `gorm:"type:varchar(100);index:idx_users_google_id,where:google_id IS NOT NULL" json:"-"`
	FacebookID          *string        `gorm:"type:varchar(100);index:idx_users_facebook_id,where:facebook_id IS NOT NULL" json:"-"`
	AppleID             *string        `gorm:"type:varchar(100);index:idx_users_apple_id,where:apple_id IS NOT NULL" json:"-"`
	Metadata            map[string]any `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt           time.Time      `gorm:"type:timestamptz;not null;default:now();index:idx_users_status_created_at" json:"createdAt"`
	UpdatedAt           time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"`
	Version             int            `gorm:"default:1" json:"-"`

	// Relations
	UserRoles     []UserRole     `gorm:"foreignKey:UserID" json:"userRoles,omitempty"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID" json:"-"`
	LoginHistory  []LoginHistory `gorm:"foreignKey:UserID" json:"-"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.ReferralCode == "" {
		u.ReferralCode = generateReferralCode()
	}
	return nil
}

func generateReferralCode() string {
	return uuid.New().String()[:8]
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

func (u *User) GetRoleNames() []string {
	roles := make([]string, 0, len(u.UserRoles))
	for _, ur := range u.UserRoles {
		if ur.Role.Name != "" {
			roles = append(roles, ur.Role.Name)
		}
	}
	return roles
}
