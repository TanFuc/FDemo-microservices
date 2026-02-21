package repository

import (
	"context"
	"time"
)

// StorageRepository defines the interface for object storage operations.
type StorageRepository interface {
	// GeneratePresignedPutURL generates a presigned URL for uploading an object.
	GeneratePresignedPutURL(ctx context.Context, objectName string, expiry time.Duration) (string, error)

	// GetPublicURL returns the public URL for accessing an object.
	GetPublicURL(objectName string) string

	// ObjectExists checks if an object exists in the storage.
	ObjectExists(ctx context.Context, objectName string) (bool, error)

	// GetObject retrieves an object's content from the storage.
	GetObject(ctx context.Context, objectName string) ([]byte, error)

	// PutObject uploads an object to the storage.
	PutObject(ctx context.Context, objectName string, data []byte, contentType string) error

	// DeleteObject removes an object from the storage.
	DeleteObject(ctx context.Context, objectName string) error

	// EnsureBucket ensures the bucket exists and is properly configured.
	EnsureBucket(ctx context.Context) error
}

// MessageQueueRepository defines the interface for message queue operations.
type MessageQueueRepository interface {
	// Publish sends a message to the specified subject.
	Publish(ctx context.Context, subject string, data []byte) error

	// Subscribe subscribes to messages on the specified subject.
	Subscribe(ctx context.Context, subject string, handler func(data []byte) error) error

	// Close closes the connection to the message queue.
	Close() error
}
