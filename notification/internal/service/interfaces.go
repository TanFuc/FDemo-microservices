package service

import (
	"context"

	"microservices/notification/internal/models"
)

// NotificationService defines the interface for notification business logic
type NotificationService interface {
	// SendNotification sends a notification through all configured channels
	SendNotification(ctx context.Context, notification *models.Notification) error

	// SendToUser sends a notification to a specific user
	SendToUser(ctx context.Context, userID string, notification *models.Notification) error

	// SendToUsers sends a notification to multiple users
	SendToUsers(ctx context.Context, userIDs []string, notification *models.Notification) error

	// BroadcastNotification broadcasts a notification to all connected users
	BroadcastNotification(ctx context.Context, notification *models.Notification) error

	// GetUserNotifications retrieves notifications for a user with pagination
	GetUserNotifications(ctx context.Context, userID string, page, limit int) (*models.NotificationListResponse, error)

	// MarkAsRead marks a notification as read
	MarkAsRead(ctx context.Context, notificationID, userID string) error

	// MarkAllAsRead marks all notifications as read for a user
	MarkAllAsRead(ctx context.Context, userID string) error

	// GetUnreadCount returns the count of unread notifications for a user
	GetUnreadCount(ctx context.Context, userID string) (int64, error)

	// DeleteNotification deletes a notification
	DeleteNotification(ctx context.Context, notificationID, userID string) error

	// IsUserOnline checks if a user is currently connected via WebSocket
	IsUserOnline(userID string) bool

	// GetOnlineUsers returns a list of online user IDs
	GetOnlineUsers() []string
}

// EmailService defines the interface for email sending
type EmailService interface {
	// SendEmail sends an email notification
	SendEmail(ctx context.Context, to, subject, body string) error

	// SendTemplateEmail sends an email using a template
	SendTemplateEmail(ctx context.Context, to, templateName string, data map[string]interface{}) error
}

// PushService defines the interface for push notifications
type PushService interface {
	// SendPush sends a push notification to a device
	SendPush(ctx context.Context, deviceToken, title, body string, data map[string]interface{}) error

	// SendPushToUser sends push notifications to all devices of a user
	SendPushToUser(ctx context.Context, userID, title, body string, data map[string]interface{}) error
}

// PreferenceService defines the interface for notification preferences
type PreferenceService interface {
	// GetPreferences retrieves notification preferences for a user
	GetPreferences(ctx context.Context, userID string) (*models.NotificationPreference, error)

	// UpdatePreferences updates notification preferences for a user
	UpdatePreferences(ctx context.Context, userID string, prefs *models.NotificationPreference) error

	// ShouldNotify checks if a notification should be sent based on user preferences
	ShouldNotify(ctx context.Context, userID string, channel, category string) (bool, error)
}
