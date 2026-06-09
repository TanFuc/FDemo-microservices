package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	"microservices/media/internal/config"
	"microservices/media/internal/repository"
	"microservices/pkg/logger"
)

// StorageRepo implements the StorageRepository interface using MinIO.
type StorageRepo struct {
	client     *minio.Client
	bucketName string
	endpoint   string
	useSSL     bool
}

// Ensure StorageRepo implements StorageRepository.
var _ repository.StorageRepository = (*StorageRepo)(nil)

// NewStorageRepo creates a new MinIO storage repository.
func NewStorageRepo(cfg config.MinIOConfig) (*StorageRepo, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	logger.Info().
		Str("endpoint", cfg.Endpoint).
		Str("bucket", cfg.BucketName).
		Msg("MinIO client initialized")

	return &StorageRepo{
		client:     client,
		bucketName: cfg.BucketName,
		endpoint:   cfg.Endpoint,
		useSSL:     cfg.UseSSL,
	}, nil
}

// EnsureBucket ensures the bucket exists and is properly configured.
func (s *StorageRepo) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		logger.Info().
			Str("bucket", s.bucketName).
			Msg("Bucket created")

		// Set bucket policy for public read access
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::%s/*"]
				}
			]
		}`, s.bucketName)

		if err := s.client.SetBucketPolicy(ctx, s.bucketName, policy); err != nil {
			return fmt.Errorf("failed to set bucket policy: %w", err)
		}

		// Set lifecycle rule to delete temp files after 24 hours
		lifecycleConfig := lifecycle.NewConfiguration()
		lifecycleConfig.Rules = []lifecycle.Rule{{
			ID:         "DeleteTempFiles",
			Status:     "Enabled",
			RuleFilter: lifecycle.Filter{Prefix: "temp/"},
			Expiration: lifecycle.Expiration{Days: 1},
		}}

		if err := s.client.SetBucketLifecycle(ctx, s.bucketName, lifecycleConfig); err != nil {
			// Log but don't fail - lifecycle might not be supported
			logger.Warn().
				Err(err).
				Msg("Failed to set lifecycle policy")
		}
	}

	return nil
}

// GeneratePresignedPutURL generates a presigned URL for uploading an object.
func (s *StorageRepo) GeneratePresignedPutURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucketName, objectName, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedURL.String(), nil
}

// GetPublicURL returns the public URL for accessing an object.
func (s *StorageRepo) GetPublicURL(objectName string) string {
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.endpoint, s.bucketName, url.PathEscape(objectName))
}

// ObjectExists checks if an object exists in the storage.
func (s *StorageRepo) ObjectExists(ctx context.Context, objectName string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat object: %w", err)
	}
	return true, nil
}

// GetObject retrieves an object's content from the storage.
func (s *StorageRepo) GetObject(ctx context.Context, objectName string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// PutObject uploads an object to the storage.
func (s *StorageRepo) PutObject(ctx context.Context, objectName string, data []byte, contentType string) error {
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

// DeleteObject removes an object from the storage.
func (s *StorageRepo) DeleteObject(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
