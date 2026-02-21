package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoginStatus string

const (
	LoginStatusSuccess     LoginStatus = "SUCCESS"
	LoginStatusFailed      LoginStatus = "FAILED"
	LoginStatusBlocked     LoginStatus = "BLOCKED"
	LoginStatus2FARequired LoginStatus = "2FA_REQUIRED"
	LoginStatus2FASuccess  LoginStatus = "2FA_SUCCESS"
	LoginStatus2FAFailed   LoginStatus = "2FA_FAILED"
)

type AuthMethod string

const (
	AuthMethodPassword AuthMethod = "PASSWORD"
	AuthMethodGoogle   AuthMethod = "GOOGLE"
	AuthMethodFacebook AuthMethod = "FACEBOOK"
	AuthMethodApple    AuthMethod = "APPLE"
	AuthMethod2FA      AuthMethod = "2FA"
)

type FailureReason string

const (
	FailureReasonInvalidPassword  FailureReason = "INVALID_PASSWORD"
	FailureReasonAccountLocked    FailureReason = "ACCOUNT_LOCKED"
	FailureReasonAccountSuspended FailureReason = "ACCOUNT_SUSPENDED"
	FailureReasonAccountBanned    FailureReason = "ACCOUNT_BANNED"
	FailureReasonAccountInactive  FailureReason = "ACCOUNT_INACTIVE"
	FailureReason2FAInvalid       FailureReason = "2FA_INVALID"
)

type LoginHistory struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index:idx_login_history_user_created" json:"userId"`
	Status        LoginStatus    `gorm:"type:varchar(20);not null;index" json:"status"`
	FailureReason *string        `gorm:"type:varchar(100)" json:"failureReason,omitempty"`
	IPAddress     string         `gorm:"type:varchar(45);not null;index" json:"ipAddress"`
	UserAgent     *string        `gorm:"type:varchar(500)" json:"userAgent,omitempty"`
	Location      *string        `gorm:"type:varchar(100)" json:"location,omitempty"`
	AuthMethod    AuthMethod     `gorm:"type:varchar(20);not null" json:"authMethod"`
	DeviceType    *string        `gorm:"type:varchar(20)" json:"deviceType,omitempty"`
	Browser       *string        `gorm:"type:varchar(100)" json:"browser,omitempty"`
	OS            *string        `gorm:"type:varchar(100)" json:"os,omitempty"`
	Metadata      map[string]any `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;not null;default:now();index:idx_login_history_user_created;index:idx_login_history_created" json:"createdAt"`

	// Relations
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

func (LoginHistory) TableName() string {
	return "login_history"
}

func (lh *LoginHistory) BeforeCreate(tx *gorm.DB) error {
	if lh.ID == uuid.Nil {
		lh.ID = uuid.New()
	}
	return nil
}

func NewLoginHistory(userID uuid.UUID, status LoginStatus, authMethod AuthMethod, ipAddress string, userAgent *string) *LoginHistory {
	return &LoginHistory{
		UserID:     userID,
		Status:     status,
		AuthMethod: authMethod,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}
}

func (lh *LoginHistory) SetFailure(reason FailureReason) {
	r := string(reason)
	lh.FailureReason = &r
}
