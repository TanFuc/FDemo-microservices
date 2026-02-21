package http

import (
	"github.com/gofiber/fiber/v2"

	"microservices/notification/internal/models"
	"microservices/notification/internal/service"
	"microservices/pkg/response"
)

type NotificationHandler struct {
	service service.NotificationService
}

func NewNotificationHandler(service service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// SendNotification sends a notification to a user
// @Summary Send notification
// @Description Send a notification to a specific user
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body models.SendNotificationRequest true "Send notification request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /notifications/send [post]
func (h *NotificationHandler) SendNotification(c *fiber.Ctx) error {
	var req models.SendNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.UserID == "" {
		return response.ValidationError(c, []response.FieldError{
			{Field: "user_id", Message: "user_id is required"},
		})
	}
	if req.Title == "" {
		return response.ValidationError(c, []response.FieldError{
			{Field: "title", Message: "title is required"},
		})
	}
	if req.Body == "" {
		return response.ValidationError(c, []response.FieldError{
			{Field: "body", Message: "body is required"},
		})
	}

	notification := &models.Notification{
		UserID:   req.UserID,
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
		Channel:  req.Channel,
		Priority: req.Priority,
	}

	if notification.Channel == "" {
		notification.Channel = "all"
	}
	if notification.Priority == "" {
		notification.Priority = "normal"
	}

	err := h.service.SendToUser(c.Context(), req.UserID, notification)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.SuccessWithMessage(c, "Notification sent successfully", fiber.Map{
		"user_online": h.service.IsUserOnline(req.UserID),
	})
}

// BroadcastNotification broadcasts a notification to all users
// @Summary Broadcast notification
// @Description Broadcast a notification to all connected users
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body models.BroadcastNotificationRequest true "Broadcast notification request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /notifications/broadcast [post]
func (h *NotificationHandler) BroadcastNotification(c *fiber.Ctx) error {
	var req models.BroadcastNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Title == "" || req.Body == "" {
		return response.BadRequest(c, "Title and body are required")
	}

	notification := &models.Notification{
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
		Channel:  "websocket",
		Priority: req.Priority,
	}

	if notification.Priority == "" {
		notification.Priority = "normal"
	}

	err := h.service.BroadcastNotification(c.Context(), notification)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	onlineUsers := h.service.GetOnlineUsers()
	return response.SuccessWithMessage(c, "Notification broadcast successfully", fiber.Map{
		"recipients": len(onlineUsers),
	})
}

// GetNotifications retrieves notifications for the authenticated user
// @Summary Get notifications
// @Description Get notifications for the authenticated user with pagination
// @Tags Notifications
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} models.NotificationListResponse
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /notifications [get]
func (h *NotificationHandler) GetNotifications(c *fiber.Ctx) error {
	userID := c.Locals("userId")
	if userID == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	result, err := h.service.GetUserNotifications(c.Context(), userID.(string), page, limit)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, result)
}

// MarkAsRead marks a notification as read
// @Summary Mark as read
// @Description Mark a specific notification as read
// @Tags Notifications
// @Accept json
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /notifications/{id}/read [post]
func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	userID := c.Locals("userId")
	if userID == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	notificationID := c.Params("id")
	if notificationID == "" {
		return response.BadRequest(c, "Notification ID is required")
	}

	err := h.service.MarkAsRead(c.Context(), notificationID, userID.(string))
	if err != nil {
		return response.NotFound(c, "Notification not found")
	}

	return response.SuccessWithMessage(c, "Notification marked as read", nil)
}

// MarkAllAsRead marks all notifications as read
// @Summary Mark all as read
// @Description Mark all notifications as read for the authenticated user
// @Tags Notifications
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /notifications/read-all [post]
func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID := c.Locals("userId")
	if userID == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	err := h.service.MarkAllAsRead(c.Context(), userID.(string))
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.SuccessWithMessage(c, "All notifications marked as read", nil)
}

// GetUnreadCount returns the count of unread notifications
// @Summary Get unread count
// @Description Get the count of unread notifications for the authenticated user
// @Tags Notifications
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(c *fiber.Ctx) error {
	userID := c.Locals("userId")
	if userID == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	count, err := h.service.GetUnreadCount(c.Context(), userID.(string))
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"unread_count": count,
	})
}

// DeleteNotification deletes a notification
// @Summary Delete notification
// @Description Delete a specific notification
// @Tags Notifications
// @Accept json
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /notifications/{id} [delete]
func (h *NotificationHandler) DeleteNotification(c *fiber.Ctx) error {
	userID := c.Locals("userId")
	if userID == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	notificationID := c.Params("id")
	if notificationID == "" {
		return response.BadRequest(c, "Notification ID is required")
	}

	err := h.service.DeleteNotification(c.Context(), notificationID, userID.(string))
	if err != nil {
		return response.NotFound(c, "Notification not found")
	}

	return response.SuccessWithMessage(c, "Notification deleted successfully", nil)
}

// GetOnlineUsers returns the list of online users
// @Summary Get online users
// @Description Get the list of currently online users (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /admin/online-users [get]
func (h *NotificationHandler) GetOnlineUsers(c *fiber.Ctx) error {
	users := h.service.GetOnlineUsers()
	return response.Success(c, fiber.Map{
		"online_users": users,
		"count":        len(users),
	})
}

// HealthCheck returns the health status
// @Summary Health check
// @Description Check if the service is healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /health [get]
func (h *NotificationHandler) HealthCheck(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status":  "healthy",
		"service": "notification-service",
	})
}
