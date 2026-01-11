package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DevicePlatform represents the platform of the device
type DevicePlatform string

const (
	DevicePlatformIOS     DevicePlatform = "IOS"
	DevicePlatformAndroid DevicePlatform = "ANDROID"
	DevicePlatformWeb     DevicePlatform = "WEB"
)

// DeviceTokenStatus represents the status of a device token
type DeviceTokenStatus string

const (
	DeviceTokenStatusActive   DeviceTokenStatus = "ACTIVE"
	DeviceTokenStatusInactive DeviceTokenStatus = "INACTIVE"
	DeviceTokenStatusExpired  DeviceTokenStatus = "EXPIRED"
	DeviceTokenStatusInvalid  DeviceTokenStatus = "INVALID"
)

// DeviceToken represents a push notification device token
type DeviceToken struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"userId" json:"userId"`

	// Token information
	Token       string             `bson:"token" json:"token"`
	Platform    DevicePlatform     `bson:"platform" json:"platform"`
	Provider    string             `bson:"provider" json:"provider"` // FCM, APNS, WEB_PUSH

	// Device information
	DeviceID    string             `bson:"deviceId,omitempty" json:"deviceId,omitempty"`
	DeviceName  string             `bson:"deviceName,omitempty" json:"deviceName,omitempty"`
	DeviceModel string             `bson:"deviceModel,omitempty" json:"deviceModel,omitempty"`
	OSVersion   string             `bson:"osVersion,omitempty" json:"osVersion,omitempty"`
	AppVersion  string             `bson:"appVersion,omitempty" json:"appVersion,omitempty"`

	// Browser info (for web push)
	Browser     string             `bson:"browser,omitempty" json:"browser,omitempty"`
	BrowserVersion string          `bson:"browserVersion,omitempty" json:"browserVersion,omitempty"`

	// APNS specific
	APNSToken   string             `bson:"apnsToken,omitempty" json:"-"` // Hidden in JSON
	APNSEnvironment string         `bson:"apnsEnvironment,omitempty" json:"apnsEnvironment,omitempty"` // sandbox or production

	// Status
	Status      DeviceTokenStatus  `bson:"status" json:"status"`

	// Activity tracking
	LastUsedAt  *time.Time         `bson:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
	LastPushAt  *time.Time         `bson:"lastPushAt,omitempty" json:"lastPushAt,omitempty"`
	PushCount   int64              `bson:"pushCount" json:"pushCount"`
	FailCount   int64              `bson:"failCount" json:"failCount"`
	LastError   string             `bson:"lastError,omitempty" json:"lastError,omitempty"`

	// Timezone for scheduling
	Timezone    string             `bson:"timezone,omitempty" json:"timezone,omitempty"`

	// Metadata
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
	ExpiresAt   *time.Time         `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
}

// NewDeviceToken creates a new device token
func NewDeviceToken(userID, token string, platform DevicePlatform, provider string) *DeviceToken {
	now := time.Now()
	return &DeviceToken{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Token:     token,
		Platform:  platform,
		Provider:  provider,
		Status:    DeviceTokenStatusActive,
		PushCount: 0,
		FailCount: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetDeviceInfo sets the device information
func (dt *DeviceToken) SetDeviceInfo(deviceID, deviceName, deviceModel, osVersion, appVersion string) {
	dt.DeviceID = deviceID
	dt.DeviceName = deviceName
	dt.DeviceModel = deviceModel
	dt.OSVersion = osVersion
	dt.AppVersion = appVersion
	dt.UpdatedAt = time.Now()
}

// SetBrowserInfo sets the browser information for web push
func (dt *DeviceToken) SetBrowserInfo(browser, browserVersion string) {
	dt.Browser = browser
	dt.BrowserVersion = browserVersion
	dt.UpdatedAt = time.Now()
}

// RecordPush records a push notification attempt
func (dt *DeviceToken) RecordPush(success bool, errorMsg string) {
	now := time.Now()
	dt.LastPushAt = &now
	dt.PushCount++
	if !success {
		dt.FailCount++
		dt.LastError = errorMsg
	}
	dt.UpdatedAt = now
}

// RecordUsage records device usage
func (dt *DeviceToken) RecordUsage() {
	now := time.Now()
	dt.LastUsedAt = &now
	dt.UpdatedAt = now
}

// Deactivate deactivates the token
func (dt *DeviceToken) Deactivate() {
	dt.Status = DeviceTokenStatusInactive
	dt.UpdatedAt = time.Now()
}

// MarkInvalid marks the token as invalid
func (dt *DeviceToken) MarkInvalid(reason string) {
	dt.Status = DeviceTokenStatusInvalid
	dt.LastError = reason
	dt.UpdatedAt = time.Now()
}

// MarkExpired marks the token as expired
func (dt *DeviceToken) MarkExpired() {
	dt.Status = DeviceTokenStatusExpired
	dt.UpdatedAt = time.Now()
}

// Activate activates the token
func (dt *DeviceToken) Activate() {
	dt.Status = DeviceTokenStatusActive
	dt.FailCount = 0
	dt.LastError = ""
	dt.UpdatedAt = time.Now()
}

// IsActive returns true if the token is active
func (dt *DeviceToken) IsActive() bool {
	if dt.Status != DeviceTokenStatusActive {
		return false
	}
	if dt.ExpiresAt != nil && time.Now().After(*dt.ExpiresAt) {
		return false
	}
	return true
}

// ShouldRetry checks if push should be retried based on failure count
func (dt *DeviceToken) ShouldRetry(maxFailures int64) bool {
	return dt.FailCount < maxFailures
}

// ResetFailCount resets the failure count (e.g., after successful push)
func (dt *DeviceToken) ResetFailCount() {
	dt.FailCount = 0
	dt.LastError = ""
	dt.UpdatedAt = time.Now()
}
