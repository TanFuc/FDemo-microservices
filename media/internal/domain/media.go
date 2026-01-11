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
