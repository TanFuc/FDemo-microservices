package http

import (
	"github.com/gofiber/fiber/v2"

	"microservices/media/internal/domain"
	"microservices/media/internal/service"
	"microservices/pkg/response"
)

type MediaHandler struct {
	service service.MediaService
}

func NewMediaHandler(service service.MediaService) *MediaHandler {
	return &MediaHandler{service: service}
}

type GetUploadURLRequest struct {
	UserID   string `json:"user_id" validate:"required"`
	FileType string `json:"file_type" validate:"required"`
	Purpose  string `json:"purpose"`
}

// GetUploadURL generates a presigned URL for file upload
// @Summary Get upload URL
// @Description Generate a presigned URL for uploading a file
// @Tags Media
// @Accept json
// @Produce json
// @Param request body GetUploadURLRequest true "Upload URL request"
// @Success 200 {object} domain.UploadURLResponse
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /media/upload-url [post]
func (h *MediaHandler) GetUploadURL(c *fiber.Ctx) error {
	var req GetUploadURLRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.UserID == "" {
		return response.ValidationError(c, []response.FieldError{
			{Field: "user_id", Message: "user_id is required"},
		})
	}
	if req.FileType == "" {
		return response.ValidationError(c, []response.FieldError{
			{Field: "file_type", Message: "file_type is required"},
		})
	}

	result, err := h.service.GetUploadURL(c.Context(), &domain.UploadRequest{
		UserID:   req.UserID,
		FileType: req.FileType,
		Purpose:  req.Purpose,
	})
	if err != nil {
		if err == domain.ErrInvalidFileType {
			return response.BadRequest(c, "File type not allowed")
		}
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, result)
}

type ConfirmUploadRequest struct {
	FileKey string `json:"file_key" validate:"required"`
}

// ConfirmUpload confirms that a file has been uploaded
// @Summary Confirm upload
// @Description Confirm that a file has been uploaded and trigger processing
// @Tags Media
// @Accept json
// @Produce json
// @Param request body ConfirmUploadRequest true "Confirm upload request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /media/confirm [post]
func (h *MediaHandler) ConfirmUpload(c *fiber.Ctx) error {
	var req ConfirmUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.FileKey == "" {
		return response.ValidationError(c, []response.FieldError{
			{Field: "file_key", Message: "file_key is required"},
		})
	}

	err := h.service.ConfirmUpload(c.Context(), &domain.ConfirmUploadRequest{
		FileKey: req.FileKey,
	})
	if err != nil {
		if err == domain.ErrFileNotFound {
			return response.NotFound(c, "File not found in storage")
		}
		return response.InternalError(c, err.Error())
	}

	return response.SuccessWithMessage(c, "Upload confirmed", map[string]string{"status": "confirmed"})
}

// GetMedia retrieves media metadata by ID
// @Summary Get media
// @Description Get media file metadata by ID
// @Tags Media
// @Accept json
// @Produce json
// @Param id path string true "Media ID"
// @Success 200 {object} domain.MediaFileResponse
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /media/{id} [get]
func (h *MediaHandler) GetMedia(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "Media ID is required")
	}

	media, err := h.service.GetMediaByID(c.Context(), id)
	if err != nil {
		return response.NotFound(c, "Media not found")
	}

	return response.Success(c, media)
}

// ListMedia retrieves media files with pagination
// @Summary List media
// @Description List media files for a user with pagination
// @Tags Media
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} domain.ListMediaResponse
// @Failure 400 {object} response.Response
// @Router /media [get]
func (h *MediaHandler) ListMedia(c *fiber.Ctx) error {
	userID := c.Query("user_id")
	if userID == "" {
		return response.BadRequest(c, "user_id is required")
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	result, err := h.service.ListMedia(c.Context(), userID, page, limit)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, result)
}

// DeleteMedia deletes a media file
// @Summary Delete media
// @Description Delete a media file by ID
// @Tags Media
// @Accept json
// @Produce json
// @Param id path string true "Media ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /media/{id} [delete]
func (h *MediaHandler) DeleteMedia(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "Media ID is required")
	}

	err := h.service.DeleteMedia(c.Context(), id)
	if err != nil {
		return response.NotFound(c, "Media not found")
	}

	return response.SuccessWithMessage(c, "Media deleted successfully", nil)
}

// HealthCheck returns the health status
// @Summary Health check
// @Description Check if the service is healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /health [get]
func (h *MediaHandler) HealthCheck(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status":  "healthy",
		"service": "media-service",
	})
}
