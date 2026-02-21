package impl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"microservices/media/internal/config"
	"microservices/media/internal/domain"
	"microservices/media/internal/service"
)

var (
	ErrInvalidID = errors.New("invalid media ID")
)

type mediaService struct {
	storage domain.StorageClient
	queue   domain.MessageQueue
	cfg     config.MediaConfig
	subject string
}

var mimeToExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
	"video/mp4":  ".mp4",
	"video/webm": ".webm",
}

var dangerousExtensions = []string{
	".exe", ".sh", ".bat", ".cmd", ".ps1", ".vbs", ".js", ".jar", ".msi", ".dll",
}

// NewMediaService creates a new media service instance
func NewMediaService(storage domain.StorageClient, queue domain.MessageQueue, cfg config.MediaConfig, subject string) service.MediaService {
	return &mediaService{
		storage: storage,
		queue:   queue,
		cfg:     cfg,
		subject: subject,
	}
}

func (s *mediaService) GetUploadURL(ctx context.Context, req *domain.UploadRequest) (*domain.UploadURLResponse, error) {
	// Validate file type
	if !slices.Contains(s.cfg.AllowedMimeTypes, req.FileType) {
		return nil, domain.ErrInvalidFileType
	}

	// Get extension from mime type
	ext, ok := mimeToExt[req.FileType]
	if !ok {
		return nil, domain.ErrInvalidFileType
	}

	// Additional security: check for dangerous extensions
	if slices.Contains(dangerousExtensions, strings.ToLower(ext)) {
		return nil, domain.ErrInvalidFileType
	}

	// Generate unique object name based on purpose
	objectID := uuid.New().String()
	var objectName string

	switch req.Purpose {
	case "product_image":
		objectName = fmt.Sprintf("products/%s/%s%s", req.UserID, objectID, ext)
	case "avatar":
		objectName = fmt.Sprintf("avatars/%s/%s%s", req.UserID, objectID, ext)
	default:
		objectName = fmt.Sprintf("uploads/%s/%s%s", req.UserID, objectID, ext)
	}

	// Generate presigned URL
	uploadURL, err := s.storage.GeneratePresignedPutURL(ctx, objectName, s.cfg.UploadURLExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return &domain.UploadURLResponse{
		UploadURL: uploadURL,
		FileKey:   objectName,
		PublicURL: s.storage.GetPublicURL(objectName),
		ExpiresAt: time.Now().Add(s.cfg.UploadURLExpiry),
	}, nil
}

func (s *mediaService) ConfirmUpload(ctx context.Context, req *domain.ConfirmUploadRequest) error {
	// Validate file exists in storage
	exists, err := s.storage.ObjectExists(ctx, req.FileKey)
	if err != nil {
		return fmt.Errorf("failed to check file existence: %w", err)
	}
	if !exists {
		return domain.ErrFileNotFound
	}

	// Parse user ID from file key
	parts := strings.Split(req.FileKey, "/")
	var userID, purpose string
	if len(parts) >= 2 {
		purpose = parts[0]
		userID = parts[1]
	}

	// Publish event to queue
	event := domain.MediaEvent{
		Key:       req.FileKey,
		UserID:    userID,
		Purpose:   purpose,
		Timestamp: time.Now(),
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := s.queue.Publish(ctx, s.subject, eventData); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func (s *mediaService) GetMediaByID(ctx context.Context, id string) (*domain.MediaFileResponse, error) {
	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidID
	}

	// TODO: Implement database lookup when repository is added
	return nil, errors.New("not implemented: requires database repository")
}

func (s *mediaService) ListMedia(ctx context.Context, userID string, page, limit int) (*domain.ListMediaResponse, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// TODO: Implement database lookup when repository is added
	return &domain.ListMediaResponse{
		Items:      []*domain.MediaFileResponse{},
		Total:      0,
		Page:       page,
		Limit:      limit,
		TotalPages: 0,
	}, nil
}

func (s *mediaService) DeleteMedia(ctx context.Context, id string) error {
	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidID
	}

	// TODO: Implement when repository is added
	return errors.New("not implemented: requires database repository")
}

func (s *mediaService) IsImageFile(fileKey string) bool {
	ext := strings.ToLower(filepath.Ext(fileKey))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	return slices.Contains(imageExts, ext)
}
