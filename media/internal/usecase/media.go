package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"microservices/media/internal/config"
	"microservices/media/internal/domain"
)

type MediaUseCase struct {
	storage domain.StorageClient
	queue   domain.MessageQueue
	cfg     config.MediaConfig
	subject string
}

func NewMediaUseCase(storage domain.StorageClient, queue domain.MessageQueue, cfg config.MediaConfig, subject string) *MediaUseCase {
	return &MediaUseCase{
		storage: storage,
		queue:   queue,
		cfg:     cfg,
		subject: subject,
	}
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

func (m *MediaUseCase) GetUploadURL(ctx context.Context, req domain.UploadRequest) (*domain.UploadURLResponse, error) {
	// Validate file type
	if !slices.Contains(m.cfg.AllowedMimeTypes, req.FileType) {
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
	uploadURL, err := m.storage.GeneratePresignedPutURL(ctx, objectName, m.cfg.UploadURLExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return &domain.UploadURLResponse{
		UploadURL: uploadURL,
		FileKey:   objectName,
		PublicURL: m.storage.GetPublicURL(objectName),
		ExpiresAt: time.Now().Add(m.cfg.UploadURLExpiry),
	}, nil
}

func (m *MediaUseCase) ConfirmUpload(ctx context.Context, req domain.ConfirmUploadRequest) error {
	// Validate file exists in storage
	exists, err := m.storage.ObjectExists(ctx, req.FileKey)
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

	if err := m.queue.Publish(ctx, m.subject, eventData); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func (m *MediaUseCase) IsImageFile(fileKey string) bool {
	ext := strings.ToLower(filepath.Ext(fileKey))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	return slices.Contains(imageExts, ext)
}
