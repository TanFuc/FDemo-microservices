package domain

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MediaType represents the type of media file
type MediaType string

const (
	MediaTypeImage    MediaType = "IMAGE"
	MediaTypeVideo    MediaType = "VIDEO"
	MediaTypeDocument MediaType = "DOCUMENT"
	MediaTypeAudio    MediaType = "AUDIO"
	MediaTypeOther    MediaType = "OTHER"
)

// MediaStatus represents the status of a media file
type MediaStatus string

const (
	MediaStatusPending    MediaStatus = "PENDING"
	MediaStatusProcessing MediaStatus = "PROCESSING"
	MediaStatusReady      MediaStatus = "READY"
	MediaStatusFailed     MediaStatus = "FAILED"
	MediaStatusDeleted    MediaStatus = "DELETED"
)

// MediaVisibility represents the visibility of a media file
type MediaVisibility string

const (
	MediaVisibilityPublic  MediaVisibility = "PUBLIC"
	MediaVisibilityPrivate MediaVisibility = "PRIVATE"
)

// MediaFile represents a media file in storage
type MediaFile struct {
	ID            uuid.UUID         `json:"id"`
	UserID        string            `json:"user_id"`
	FolderID      *uuid.UUID        `json:"folder_id,omitempty"`

	// File information
	OriginalName  string            `json:"original_name"`
	FileName      string            `json:"file_name"`
	FileKey       string            `json:"file_key"` // S3/MinIO key
	MimeType      string            `json:"mime_type"`
	FileSize      int64             `json:"file_size"` // in bytes
	FileHash      string            `json:"file_hash"` // MD5 or SHA256

	// Type and purpose
	Type          MediaType         `json:"type"`
	Purpose       string            `json:"purpose"` // PRODUCT, AVATAR, BANNER, REVIEW, etc.
	Visibility    MediaVisibility   `json:"visibility"`

	// URLs
	PublicURL     string            `json:"public_url"`
	CDNUrl        string            `json:"cdn_url,omitempty"`

	// Image-specific metadata
	Width         int               `json:"width,omitempty"`
	Height        int               `json:"height,omitempty"`
	AspectRatio   float64           `json:"aspect_ratio,omitempty"`

	// Video-specific metadata
	Duration      float64           `json:"duration,omitempty"` // in seconds
	FrameRate     float64           `json:"frame_rate,omitempty"`
	Codec         string            `json:"codec,omitempty"`

	// Variants (thumbnails, different sizes)
	Variants      []MediaVariant    `json:"variants,omitempty"`

	// Processing status
	Status        MediaStatus       `json:"status"`
	ProcessingError string          `json:"processing_error,omitempty"`

	// Tags and metadata
	Tags          []string          `json:"tags,omitempty"`
	AltText       string            `json:"alt_text,omitempty"`
	Caption       string            `json:"caption,omitempty"`
	Metadata      json.RawMessage   `json:"metadata,omitempty"`

	// Usage tracking
	UsageCount    int64             `json:"usage_count"`
	LastUsedAt    *time.Time        `json:"last_used_at,omitempty"`

	// Timestamps
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	DeletedAt     *time.Time        `json:"deleted_at,omitempty"`
}

// MediaVariant represents a variant of the original media (thumbnail, resized, etc.)
type MediaVariant struct {
	Name      string `json:"name"`      // thumbnail, small, medium, large
	FileKey   string `json:"file_key"`
	PublicURL string `json:"public_url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	FileSize  int64  `json:"file_size"`
	MimeType  string `json:"mime_type"`
}

// NewMediaFile creates a new media file
func NewMediaFile(userID, originalName, fileKey, mimeType string, fileSize int64) *MediaFile {
	now := time.Now()
	id := uuid.New()
	fileName := id.String() + filepath.Ext(originalName)

	return &MediaFile{
		ID:           id,
		UserID:       userID,
		OriginalName: originalName,
		FileName:     fileName,
		FileKey:      fileKey,
		MimeType:     mimeType,
		FileSize:     fileSize,
		Type:         DetectMediaType(mimeType),
		Visibility:   MediaVisibilityPublic,
		Status:       MediaStatusPending,
		Variants:     make([]MediaVariant, 0),
		Tags:         make([]string, 0),
		Metadata:     json.RawMessage("{}"),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// DetectMediaType detects the media type from MIME type
func DetectMediaType(mimeType string) MediaType {
	mimeType = strings.ToLower(mimeType)
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return MediaTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return MediaTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return MediaTypeAudio
	case strings.HasPrefix(mimeType, "application/pdf"),
		strings.HasPrefix(mimeType, "application/msword"),
		strings.HasPrefix(mimeType, "application/vnd."),
		strings.HasPrefix(mimeType, "text/"):
		return MediaTypeDocument
	default:
		return MediaTypeOther
	}
}

// SetImageDimensions sets image dimensions
func (m *MediaFile) SetImageDimensions(width, height int) {
	m.Width = width
	m.Height = height
	if height > 0 {
		m.AspectRatio = float64(width) / float64(height)
	}
	m.UpdatedAt = time.Now()
}

// SetVideoMetadata sets video metadata
func (m *MediaFile) SetVideoMetadata(duration, frameRate float64, codec string) {
	m.Duration = duration
	m.FrameRate = frameRate
	m.Codec = codec
	m.UpdatedAt = time.Now()
}

// AddVariant adds a variant to the media file
func (m *MediaFile) AddVariant(variant MediaVariant) {
	m.Variants = append(m.Variants, variant)
	m.UpdatedAt = time.Now()
}

// GetVariant gets a variant by name
func (m *MediaFile) GetVariant(name string) *MediaVariant {
	for _, v := range m.Variants {
		if v.Name == name {
			return &v
		}
	}
	return nil
}

// SetPublicURL sets the public URL
func (m *MediaFile) SetPublicURL(url string) {
	m.PublicURL = url
	m.UpdatedAt = time.Now()
}

// SetCDNUrl sets the CDN URL
func (m *MediaFile) SetCDNUrl(url string) {
	m.CDNUrl = url
	m.UpdatedAt = time.Now()
}

// MarkReady marks the file as ready
func (m *MediaFile) MarkReady() {
	m.Status = MediaStatusReady
	m.UpdatedAt = time.Now()
}

// MarkProcessing marks the file as processing
func (m *MediaFile) MarkProcessing() {
	m.Status = MediaStatusProcessing
	m.UpdatedAt = time.Now()
}

// MarkFailed marks the file as failed
func (m *MediaFile) MarkFailed(err string) {
	m.Status = MediaStatusFailed
	m.ProcessingError = err
	m.UpdatedAt = time.Now()
}

// SoftDelete performs a soft delete
func (m *MediaFile) SoftDelete() {
	now := time.Now()
	m.DeletedAt = &now
	m.Status = MediaStatusDeleted
	m.UpdatedAt = now
}

// RecordUsage records a usage of the file
func (m *MediaFile) RecordUsage() {
	now := time.Now()
	m.UsageCount++
	m.LastUsedAt = &now
	m.UpdatedAt = now
}

// SetTags sets the tags
func (m *MediaFile) SetTags(tags []string) {
	m.Tags = tags
	m.UpdatedAt = time.Now()
}

// AddTag adds a tag
func (m *MediaFile) AddTag(tag string) {
	for _, t := range m.Tags {
		if t == tag {
			return
		}
	}
	m.Tags = append(m.Tags, tag)
	m.UpdatedAt = time.Now()
}

// SetAltText sets the alt text
func (m *MediaFile) SetAltText(altText string) {
	m.AltText = altText
	m.UpdatedAt = time.Now()
}

// SetCaption sets the caption
func (m *MediaFile) SetCaption(caption string) {
	m.Caption = caption
	m.UpdatedAt = time.Now()
}

// SetVisibility sets the visibility
func (m *MediaFile) SetVisibility(visibility MediaVisibility) {
	m.Visibility = visibility
	m.UpdatedAt = time.Now()
}

// SetFolder sets the folder
func (m *MediaFile) SetFolder(folderID *uuid.UUID) {
	m.FolderID = folderID
	m.UpdatedAt = time.Now()
}

// IsImage returns true if the file is an image
func (m *MediaFile) IsImage() bool {
	return m.Type == MediaTypeImage
}

// IsVideo returns true if the file is a video
func (m *MediaFile) IsVideo() bool {
	return m.Type == MediaTypeVideo
}

// IsReady returns true if the file is ready
func (m *MediaFile) IsReady() bool {
	return m.Status == MediaStatusReady && m.DeletedAt == nil
}

// GetBestURL returns the best available URL (CDN preferred)
func (m *MediaFile) GetBestURL() string {
	if m.CDNUrl != "" {
		return m.CDNUrl
	}
	return m.PublicURL
}

// FileSizeInMB returns the file size in MB
func (m *MediaFile) FileSizeInMB() float64 {
	return float64(m.FileSize) / (1024 * 1024)
}
