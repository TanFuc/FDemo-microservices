package service

import (
	"context"

	"microservices/media/internal/domain"
)

// MediaService defines the interface for media business logic
type MediaService interface {
	// GetUploadURL generates a presigned URL for file upload
	GetUploadURL(ctx context.Context, req *domain.UploadRequest) (*domain.UploadURLResponse, error)

	// ConfirmUpload confirms that a file has been uploaded and triggers processing
	ConfirmUpload(ctx context.Context, req *domain.ConfirmUploadRequest) error

	// GetMediaByID retrieves media metadata by ID
	GetMediaByID(ctx context.Context, id string) (*domain.MediaFileResponse, error)

	// ListMedia retrieves media files with pagination
	ListMedia(ctx context.Context, userID string, page, limit int) (*domain.ListMediaResponse, error)

	// DeleteMedia deletes a media file
	DeleteMedia(ctx context.Context, id string) error

	// IsImageFile checks if a file is an image based on extension
	IsImageFile(fileKey string) bool
}
