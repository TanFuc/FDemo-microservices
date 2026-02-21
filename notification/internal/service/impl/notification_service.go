package impl

import (
	"context"
	"errors"
	"time"

	"microservices/notification/internal/models"
	"microservices/notification/internal/service"
	"microservices/notification/internal/websocket"
	"microservices/pkg/logger"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidNotification  = errors.New("invalid notification")
)

type notificationService struct {
	wsManager *websocket.Manager
	// Add repository when database is connected
	// repo repository.NotificationRepository
}

// NewNotificationService creates a new notification service instance
func NewNotificationService(wsManager *websocket.Manager) service.NotificationService {
	return &notificationService{
		wsManager: wsManager,
	}
}

func (s *notificationService) SendNotification(ctx context.Context, notification *models.Notification) error {
	if notification == nil {
		return ErrInvalidNotification
	}

	// Set defaults
	if notification.Channel == "" {
		notification.Channel = "all"
	}
	if notification.Priority == "" {
		notification.Priority = "normal"
	}
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	// Send via WebSocket if user is online
	if notification.Channel == "websocket" || notification.Channel == "all" {
		if s.wsManager != nil && s.wsManager.IsUserOnline(notification.UserID) {
			err := s.wsManager.SendNotification(notification.UserID, notification.Type, map[string]interface{}{
				"id":        notification.ID,
				"title":     notification.Title,
				"body":      notification.Body,
				"type":      notification.Type,
				"data":      notification.Data,
				"priority":  notification.Priority,
				"createdAt": notification.CreatedAt,
			})
			if err != nil {
				logger.Error().Err(err).
					Str("userId", notification.UserID).
					Msg("Failed to send WebSocket notification")
			}
		}
	}

	// TODO: Implement email/push notification channels

	return nil
}

func (s *notificationService) SendToUser(ctx context.Context, userID string, notification *models.Notification) error {
	notification.UserID = userID
	return s.SendNotification(ctx, notification)
}

func (s *notificationService) SendToUsers(ctx context.Context, userIDs []string, notification *models.Notification) error {
	for _, userID := range userIDs {
		notifCopy := *notification
		notifCopy.UserID = userID
		if err := s.SendNotification(ctx, &notifCopy); err != nil {
			logger.Error().Err(err).
				Str("userId", userID).
				Msg("Failed to send notification to user")
		}
	}
	return nil
}

func (s *notificationService) BroadcastNotification(ctx context.Context, notification *models.Notification) error {
	if notification == nil {
		return ErrInvalidNotification
	}

	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	if s.wsManager == nil {
		return nil
	}

	msg := &websocket.Message{
		Type:  "notification",
		Event: notification.Type,
		Data: map[string]interface{}{
			"title":     notification.Title,
			"body":      notification.Body,
			"type":      notification.Type,
			"data":      notification.Data,
			"priority":  notification.Priority,
			"createdAt": notification.CreatedAt,
		},
		Timestamp: time.Now(),
	}

	return s.wsManager.Broadcast(msg, nil)
}

func (s *notificationService) GetUserNotifications(ctx context.Context, userID string, page, limit int) (*models.NotificationListResponse, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// TODO: Implement database query when repository is added
	return &models.NotificationListResponse{
		Items:       []*models.Notification{},
		Total:       0,
		UnreadCount: 0,
		Page:        page,
		Limit:       limit,
		TotalPages:  0,
	}, nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	// TODO: Implement database update when repository is added
	return nil
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	// TODO: Implement database update when repository is added
	return nil
}

func (s *notificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	// TODO: Implement database query when repository is added
	return 0, nil
}

func (s *notificationService) DeleteNotification(ctx context.Context, notificationID, userID string) error {
	// TODO: Implement database delete when repository is added
	return nil
}

func (s *notificationService) IsUserOnline(userID string) bool {
	if s.wsManager == nil {
		return false
	}
	return s.wsManager.IsUserOnline(userID)
}

func (s *notificationService) GetOnlineUsers() []string {
	if s.wsManager == nil {
		return []string{}
	}
	return s.wsManager.GetOnlineUsers()
}
