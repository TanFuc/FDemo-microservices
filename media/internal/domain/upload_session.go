package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UploadSessionStatus represents the status of an upload session
type UploadSessionStatus string

const (
	UploadSessionStatusInitiated  UploadSessionStatus = "INITIATED"
	UploadSessionStatusUploading  UploadSessionStatus = "UPLOADING"
	UploadSessionStatusCompleted  UploadSessionStatus = "COMPLETED"
	UploadSessionStatusFailed     UploadSessionStatus = "FAILED"
	UploadSessionStatusExpired    UploadSessionStatus = "EXPIRED"
	UploadSessionStatusCancelled  UploadSessionStatus = "CANCELLED"
)

// UploadSession represents a multipart upload session
type UploadSession struct {
	ID              uuid.UUID             `json:"id"`
	UserID          string                `json:"user_id"`
	FolderID        *uuid.UUID            `json:"folder_id,omitempty"`

	// File information
	FileName        string                `json:"file_name"`
	FileSize        int64                 `json:"file_size"` // Expected total size
	MimeType        string                `json:"mime_type"`
	Purpose         string                `json:"purpose"` // PRODUCT, AVATAR, etc.

	// S3/MinIO multipart upload
	UploadID        string                `json:"upload_id"` // S3 multipart upload ID
	FileKey         string                `json:"file_key"` // Target file key
	PartSize        int64                 `json:"part_size"` // Size of each part (default 5MB)
	TotalParts      int                   `json:"total_parts"`

	// Progress tracking
	UploadedParts   []UploadedPart        `json:"uploaded_parts,omitempty"`
	UploadedSize    int64                 `json:"uploaded_size"`
	UploadedCount   int                   `json:"uploaded_count"`

	// Presigned URLs for parts
	PartURLs        []PartUploadURL       `json:"part_urls,omitempty"`

	// Status
	Status          UploadSessionStatus   `json:"status"`
	ErrorMessage    string                `json:"error_message,omitempty"`

	// Result
	MediaFileID     *uuid.UUID            `json:"media_file_id,omitempty"`
	PublicURL       string                `json:"public_url,omitempty"`

	// Metadata
	Metadata        json.RawMessage       `json:"metadata,omitempty"`

	// Client information
	ClientIP        string                `json:"client_ip,omitempty"`
	UserAgent       string                `json:"user_agent,omitempty"`

	// Timestamps
	ExpiresAt       time.Time             `json:"expires_at"`
	StartedAt       *time.Time            `json:"started_at,omitempty"`
	CompletedAt     *time.Time            `json:"completed_at,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// UploadedPart represents a completed upload part
type UploadedPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// PartUploadURL represents a presigned URL for a part upload
type PartUploadURL struct {
	PartNumber int       `json:"part_number"`
	URL        string    `json:"url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// NewUploadSession creates a new upload session
func NewUploadSession(userID, fileName, mimeType, purpose string, fileSize int64, partSize int64) *UploadSession {
	now := time.Now()
	id := uuid.New()

	// Default part size is 5MB
	if partSize == 0 {
		partSize = 5 * 1024 * 1024
	}

	// Calculate total parts
	totalParts := int(fileSize / partSize)
	if fileSize%partSize != 0 {
		totalParts++
	}

	return &UploadSession{
		ID:            id,
		UserID:        userID,
		FileName:      fileName,
		FileSize:      fileSize,
		MimeType:      mimeType,
		Purpose:       purpose,
		PartSize:      partSize,
		TotalParts:    totalParts,
		UploadedParts: make([]UploadedPart, 0),
		PartURLs:      make([]PartUploadURL, 0),
		Status:        UploadSessionStatusInitiated,
		Metadata:      json.RawMessage("{}"),
		ExpiresAt:     now.Add(24 * time.Hour), // 24 hour expiry
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// SetUploadID sets the S3 multipart upload ID
func (us *UploadSession) SetUploadID(uploadID, fileKey string) {
	us.UploadID = uploadID
	us.FileKey = fileKey
	us.UpdatedAt = time.Now()
}

// SetFolder sets the target folder
func (us *UploadSession) SetFolder(folderID uuid.UUID) {
	us.FolderID = &folderID
	us.UpdatedAt = time.Now()
}

// AddPartURL adds a presigned URL for a part
func (us *UploadSession) AddPartURL(partNumber int, url string, expiresAt time.Time) {
	us.PartURLs = append(us.PartURLs, PartUploadURL{
		PartNumber: partNumber,
		URL:        url,
		ExpiresAt:  expiresAt,
	})
	us.UpdatedAt = time.Now()
}

// RecordPartUpload records a successful part upload
func (us *UploadSession) RecordPartUpload(partNumber int, etag string, size int64) {
	// Check if part already exists
	for i, p := range us.UploadedParts {
		if p.PartNumber == partNumber {
			us.UploadedParts[i] = UploadedPart{
				PartNumber: partNumber,
				ETag:       etag,
				Size:       size,
				UploadedAt: time.Now(),
			}
			us.UpdatedAt = time.Now()
			return
		}
	}

	us.UploadedParts = append(us.UploadedParts, UploadedPart{
		PartNumber: partNumber,
		ETag:       etag,
		Size:       size,
		UploadedAt: time.Now(),
	})
	us.UploadedCount++
	us.UploadedSize += size
	us.UpdatedAt = time.Now()
}

// StartUpload marks the upload as started
func (us *UploadSession) StartUpload() {
	now := time.Now()
	us.Status = UploadSessionStatusUploading
	us.StartedAt = &now
	us.UpdatedAt = now
}

// Complete marks the upload as completed
func (us *UploadSession) Complete(mediaFileID uuid.UUID, publicURL string) {
	now := time.Now()
	us.Status = UploadSessionStatusCompleted
	us.MediaFileID = &mediaFileID
	us.PublicURL = publicURL
	us.CompletedAt = &now
	us.UpdatedAt = now
}

// Fail marks the upload as failed
func (us *UploadSession) Fail(errorMessage string) {
	us.Status = UploadSessionStatusFailed
	us.ErrorMessage = errorMessage
	us.UpdatedAt = time.Now()
}

// Cancel cancels the upload
func (us *UploadSession) Cancel() {
	us.Status = UploadSessionStatusCancelled
	us.UpdatedAt = time.Now()
}

// Expire marks the upload as expired
func (us *UploadSession) Expire() {
	us.Status = UploadSessionStatusExpired
	us.UpdatedAt = time.Now()
}

// IsExpired checks if the session is expired
func (us *UploadSession) IsExpired() bool {
	return time.Now().After(us.ExpiresAt)
}

// IsActive checks if the session is still active
func (us *UploadSession) IsActive() bool {
	return us.Status == UploadSessionStatusInitiated || us.Status == UploadSessionStatusUploading
}

// IsCompleted checks if all parts are uploaded
func (us *UploadSession) IsAllPartsUploaded() bool {
	return us.UploadedCount >= us.TotalParts
}

// GetProgress returns upload progress as a percentage
func (us *UploadSession) GetProgress() float64 {
	if us.FileSize == 0 {
		return 0
	}
	return float64(us.UploadedSize) / float64(us.FileSize) * 100
}

// GetMissingParts returns the part numbers that haven't been uploaded
func (us *UploadSession) GetMissingParts() []int {
	uploadedSet := make(map[int]bool)
	for _, p := range us.UploadedParts {
		uploadedSet[p.PartNumber] = true
	}

	var missing []int
	for i := 1; i <= us.TotalParts; i++ {
		if !uploadedSet[i] {
			missing = append(missing, i)
		}
	}
	return missing
}

// ExtendExpiry extends the expiry time
func (us *UploadSession) ExtendExpiry(duration time.Duration) {
	us.ExpiresAt = time.Now().Add(duration)
	us.UpdatedAt = time.Now()
}

// SetClientInfo sets client information
func (us *UploadSession) SetClientInfo(ip, userAgent string) {
	us.ClientIP = ip
	us.UserAgent = userAgent
	us.UpdatedAt = time.Now()
}

// GetSortedParts returns uploaded parts sorted by part number
func (us *UploadSession) GetSortedParts() []UploadedPart {
	parts := make([]UploadedPart, len(us.UploadedParts))
	copy(parts, us.UploadedParts)

	// Simple insertion sort (parts are usually ordered anyway)
	for i := 1; i < len(parts); i++ {
		key := parts[i]
		j := i - 1
		for j >= 0 && parts[j].PartNumber > key.PartNumber {
			parts[j+1] = parts[j]
			j--
		}
		parts[j+1] = key
	}
	return parts
}
