package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationChannel represents a notification channel
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "EMAIL"
	NotificationChannelPush  NotificationChannel = "PUSH"
	NotificationChannelSMS   NotificationChannel = "SMS"
	NotificationChannelInApp NotificationChannel = "IN_APP"
)

// NotificationCategory represents a category of notifications
type NotificationCategory string

const (
	NotificationCategoryOrders      NotificationCategory = "ORDERS"
	NotificationCategoryPayments    NotificationCategory = "PAYMENTS"
	NotificationCategoryPromotions  NotificationCategory = "PROMOTIONS"
	NotificationCategoryReviews     NotificationCategory = "REVIEWS"
	NotificationCategoryChat        NotificationCategory = "CHAT"
	NotificationCategorySystem      NotificationCategory = "SYSTEM"
	NotificationCategorySecurity    NotificationCategory = "SECURITY"
	NotificationCategoryNewsletter  NotificationCategory = "NEWSLETTER"
)

// ChannelPreference represents preferences for a single channel
type ChannelPreference struct {
	Enabled    bool               `bson:"enabled" json:"enabled"`
	Categories map[string]bool    `bson:"categories" json:"categories"` // Category -> enabled
}

// QuietHours represents quiet hours configuration
type QuietHours struct {
	Enabled   bool   `bson:"enabled" json:"enabled"`
	StartTime string `bson:"startTime" json:"startTime"` // HH:MM format
	EndTime   string `bson:"endTime" json:"endTime"`     // HH:MM format
	Timezone  string `bson:"timezone" json:"timezone"`
}

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"userId" json:"userId"`

	// Global settings
	GlobalEnabled bool `bson:"globalEnabled" json:"globalEnabled"`

	// Channel preferences
	Email ChannelPreference `bson:"email" json:"email"`
	Push  ChannelPreference `bson:"push" json:"push"`
	SMS   ChannelPreference `bson:"sms" json:"sms"`
	InApp ChannelPreference `bson:"inApp" json:"inApp"`

	// Quiet hours
	QuietHours QuietHours `bson:"quietHours" json:"quietHours"`

	// Frequency settings
	EmailDigest    string `bson:"emailDigest" json:"emailDigest"`       // INSTANT, DAILY, WEEKLY, NONE
	PushFrequency  string `bson:"pushFrequency" json:"pushFrequency"`   // ALL, IMPORTANT_ONLY, MINIMAL

	// Language preference for notifications
	Language string `bson:"language" json:"language"`

	// Marketing preferences
	MarketingOptIn     bool       `bson:"marketingOptIn" json:"marketingOptIn"`
	MarketingOptInAt   *time.Time `bson:"marketingOptInAt,omitempty" json:"marketingOptInAt,omitempty"`
	MarketingOptOutAt  *time.Time `bson:"marketingOptOutAt,omitempty" json:"marketingOptOutAt,omitempty"`

	// Unsubscribe tokens
	UnsubscribeToken string `bson:"unsubscribeToken,omitempty" json:"-"`

	// Metadata
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// NewNotificationPreference creates a new notification preference with defaults
func NewNotificationPreference(userID string) *NotificationPreference {
	now := time.Now()

	// Default categories enabled
	defaultCategories := map[string]bool{
		string(NotificationCategoryOrders):     true,
		string(NotificationCategoryPayments):   true,
		string(NotificationCategoryPromotions): true,
		string(NotificationCategoryReviews):    true,
		string(NotificationCategoryChat):       true,
		string(NotificationCategorySystem):     true,
		string(NotificationCategorySecurity):   true,
		string(NotificationCategoryNewsletter): false,
	}

	return &NotificationPreference{
		ID:            primitive.NewObjectID(),
		UserID:        userID,
		GlobalEnabled: true,
		Email: ChannelPreference{
			Enabled:    true,
			Categories: defaultCategories,
		},
		Push: ChannelPreference{
			Enabled:    true,
			Categories: defaultCategories,
		},
		SMS: ChannelPreference{
			Enabled: false,
			Categories: map[string]bool{
				string(NotificationCategoryOrders):   true,
				string(NotificationCategorySecurity): true,
			},
		},
		InApp: ChannelPreference{
			Enabled:    true,
			Categories: defaultCategories,
		},
		QuietHours: QuietHours{
			Enabled:   false,
			StartTime: "22:00",
			EndTime:   "08:00",
			Timezone:  "Asia/Ho_Chi_Minh",
		},
		EmailDigest:   "INSTANT",
		PushFrequency: "ALL",
		Language:      "vi",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// SetChannelEnabled enables or disables a channel
func (np *NotificationPreference) SetChannelEnabled(channel NotificationChannel, enabled bool) {
	switch channel {
	case NotificationChannelEmail:
		np.Email.Enabled = enabled
	case NotificationChannelPush:
		np.Push.Enabled = enabled
	case NotificationChannelSMS:
		np.SMS.Enabled = enabled
	case NotificationChannelInApp:
		np.InApp.Enabled = enabled
	}
	np.UpdatedAt = time.Now()
}

// SetCategoryEnabled enables or disables a category for a channel
func (np *NotificationPreference) SetCategoryEnabled(channel NotificationChannel, category NotificationCategory, enabled bool) {
	var channelPref *ChannelPreference
	switch channel {
	case NotificationChannelEmail:
		channelPref = &np.Email
	case NotificationChannelPush:
		channelPref = &np.Push
	case NotificationChannelSMS:
		channelPref = &np.SMS
	case NotificationChannelInApp:
		channelPref = &np.InApp
	default:
		return
	}

	if channelPref.Categories == nil {
		channelPref.Categories = make(map[string]bool)
	}
	channelPref.Categories[string(category)] = enabled
	np.UpdatedAt = time.Now()
}

// IsChannelEnabled checks if a channel is enabled
func (np *NotificationPreference) IsChannelEnabled(channel NotificationChannel) bool {
	if !np.GlobalEnabled {
		return false
	}
	switch channel {
	case NotificationChannelEmail:
		return np.Email.Enabled
	case NotificationChannelPush:
		return np.Push.Enabled
	case NotificationChannelSMS:
		return np.SMS.Enabled
	case NotificationChannelInApp:
		return np.InApp.Enabled
	}
	return false
}

// IsCategoryEnabled checks if a category is enabled for a channel
func (np *NotificationPreference) IsCategoryEnabled(channel NotificationChannel, category NotificationCategory) bool {
	if !np.IsChannelEnabled(channel) {
		return false
	}

	var channelPref ChannelPreference
	switch channel {
	case NotificationChannelEmail:
		channelPref = np.Email
	case NotificationChannelPush:
		channelPref = np.Push
	case NotificationChannelSMS:
		channelPref = np.SMS
	case NotificationChannelInApp:
		channelPref = np.InApp
	default:
		return false
	}

	if enabled, ok := channelPref.Categories[string(category)]; ok {
		return enabled
	}
	return true // Default to enabled if not specified
}

// SetQuietHours sets quiet hours configuration
func (np *NotificationPreference) SetQuietHours(enabled bool, startTime, endTime, timezone string) {
	np.QuietHours = QuietHours{
		Enabled:   enabled,
		StartTime: startTime,
		EndTime:   endTime,
		Timezone:  timezone,
	}
	np.UpdatedAt = time.Now()
}

// IsInQuietHours checks if current time is within quiet hours
func (np *NotificationPreference) IsInQuietHours(currentTime time.Time) bool {
	if !np.QuietHours.Enabled {
		return false
	}

	// Parse quiet hours times
	startHour, startMin := parseTimeStr(np.QuietHours.StartTime)
	endHour, endMin := parseTimeStr(np.QuietHours.EndTime)

	// Get current hour and minute in user's timezone
	loc, err := time.LoadLocation(np.QuietHours.Timezone)
	if err != nil {
		loc = time.UTC
	}
	localTime := currentTime.In(loc)
	currentHour := localTime.Hour()
	currentMin := localTime.Minute()

	currentMins := currentHour*60 + currentMin
	startMins := startHour*60 + startMin
	endMins := endHour*60 + endMin

	// Handle overnight quiet hours (e.g., 22:00 - 08:00)
	if startMins > endMins {
		return currentMins >= startMins || currentMins < endMins
	}
	return currentMins >= startMins && currentMins < endMins
}

// parseTimeStr parses HH:MM format
func parseTimeStr(timeStr string) (int, int) {
	var hour, min int
	if len(timeStr) >= 5 {
		hour = int(timeStr[0]-'0')*10 + int(timeStr[1]-'0')
		min = int(timeStr[3]-'0')*10 + int(timeStr[4]-'0')
	}
	return hour, min
}

// OptInMarketing opts in to marketing
func (np *NotificationPreference) OptInMarketing() {
	now := time.Now()
	np.MarketingOptIn = true
	np.MarketingOptInAt = &now
	np.UpdatedAt = now
}

// OptOutMarketing opts out of marketing
func (np *NotificationPreference) OptOutMarketing() {
	now := time.Now()
	np.MarketingOptIn = false
	np.MarketingOptOutAt = &now
	np.UpdatedAt = now
}

// DisableAll disables all notifications
func (np *NotificationPreference) DisableAll() {
	np.GlobalEnabled = false
	np.UpdatedAt = time.Now()
}

// EnableAll enables all notifications
func (np *NotificationPreference) EnableAll() {
	np.GlobalEnabled = true
	np.UpdatedAt = time.Now()
}
