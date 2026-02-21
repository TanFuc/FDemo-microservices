package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidFileType   = errors.New("invalid or disallowed file type")
	ErrFileNotFound      = errors.New("file not found in storage")
	ErrUploadFailed      = errors.New("upload failed")
	ErrProcessingFailed  = errors.New("image processing failed")
)

type UploadRequest struct {
	UserID   string
	FileType string
	Purpose  string
}

type UploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
	PublicURL string `json:"public_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ConfirmUploadRequest struct {
	FileKey string `json:"file_key"`
}

type MediaEvent struct {
	Key       string    `json:"key"`
	UserID    string    `json:"user_id"`
	Purpose   string    `json:"purpose"`
	Timestamp time.Time `json:"timestamp"`
}

type ProcessedImage struct {
	OriginalKey  string
	ThumbnailKey string
	MediumKey    string
}

type MediaFileResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	FileName    string    `json:"file_name"`
	FileKey     string    `json:"file_key"`
	MimeType    string    `json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	PublicURL   string    `json:"public_url"`
	ThumbnailURL string   `json:"thumbnail_url,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListMediaResponse struct {
	Items      []*MediaFileResponse `json:"items"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
}

type StorageClient interface {
	GeneratePresignedPutURL(ctx context.Context, objectName string, expiry time.Duration) (string, error)
	GetPublicURL(objectName string) string
	ObjectExists(ctx context.Context, objectName string) (bool, error)
	GetObject(ctx context.Context, objectName string) ([]byte, error)
	PutObject(ctx context.Context, objectName string, data []byte, contentType string) error
	DeleteObject(ctx context.Context, objectName string) error
	EnsureBucket(ctx context.Context) error
}

type MessageQueue interface {
	Publish(ctx context.Context, subject string, data []byte) error
	Subscribe(ctx context.Context, subject string, handler func(data []byte) error) error
	Close() error
}
