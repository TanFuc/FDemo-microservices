package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransformationType represents the type of image transformation
type TransformationType string

const (
	TransformationTypeResize   TransformationType = "RESIZE"
	TransformationTypeCrop     TransformationType = "CROP"
	TransformationTypeThumbnail TransformationType = "THUMBNAIL"
	TransformationTypeWatermark TransformationType = "WATERMARK"
	TransformationTypeConvert  TransformationType = "CONVERT"
	TransformationTypeOptimize TransformationType = "OPTIMIZE"
	TransformationTypeRotate   TransformationType = "ROTATE"
	TransformationTypeFlip     TransformationType = "FLIP"
	TransformationTypeBlur     TransformationType = "BLUR"
	TransformationTypeGrayscale TransformationType = "GRAYSCALE"
)

// TransformationStatus represents the status of a transformation
type TransformationStatus string

const (
	TransformationStatusPending    TransformationStatus = "PENDING"
	TransformationStatusProcessing TransformationStatus = "PROCESSING"
	TransformationStatusCompleted  TransformationStatus = "COMPLETED"
	TransformationStatusFailed     TransformationStatus = "FAILED"
	TransformationStatusCached     TransformationStatus = "CACHED"
)

// ResizeMode represents how images should be resized
type ResizeMode string

const (
	ResizeModeFit     ResizeMode = "FIT"     // Fit within bounds, maintain aspect ratio
	ResizeModeFill    ResizeMode = "FILL"    // Fill bounds, crop if needed
	ResizeModeExact   ResizeMode = "EXACT"   // Exact dimensions, may distort
	ResizeModeWidth   ResizeMode = "WIDTH"   // Fixed width, auto height
	ResizeModeHeight  ResizeMode = "HEIGHT"  // Fixed height, auto width
)

// CropPosition represents anchor position for cropping
type CropPosition string

const (
	CropPositionCenter       CropPosition = "CENTER"
	CropPositionTopLeft      CropPosition = "TOP_LEFT"
	CropPositionTopCenter    CropPosition = "TOP_CENTER"
	CropPositionTopRight     CropPosition = "TOP_RIGHT"
	CropPositionMiddleLeft   CropPosition = "MIDDLE_LEFT"
	CropPositionMiddleRight  CropPosition = "MIDDLE_RIGHT"
	CropPositionBottomLeft   CropPosition = "BOTTOM_LEFT"
	CropPositionBottomCenter CropPosition = "BOTTOM_CENTER"
	CropPositionBottomRight  CropPosition = "BOTTOM_RIGHT"
	CropPositionEntropy      CropPosition = "ENTROPY"   // Smart crop based on image content
	CropPositionAttention    CropPosition = "ATTENTION" // Smart crop focusing on interesting areas
)

// TransformationParams represents parameters for image transformation
type TransformationParams struct {
	// Resize parameters
	Width       int        `json:"width,omitempty"`
	Height      int        `json:"height,omitempty"`
	Mode        ResizeMode `json:"mode,omitempty"`

	// Crop parameters
	CropX       int          `json:"crop_x,omitempty"`
	CropY       int          `json:"crop_y,omitempty"`
	CropWidth   int          `json:"crop_width,omitempty"`
	CropHeight  int          `json:"crop_height,omitempty"`
	CropPosition CropPosition `json:"crop_position,omitempty"`

	// Format conversion
	Format      string `json:"format,omitempty"` // jpeg, png, webp, avif

	// Quality (1-100)
	Quality     int    `json:"quality,omitempty"`

	// Rotation (degrees)
	Rotate      int    `json:"rotate,omitempty"`

	// Flip
	FlipH       bool   `json:"flip_h,omitempty"`
	FlipV       bool   `json:"flip_v,omitempty"`

	// Blur (sigma value)
	Blur        float64 `json:"blur,omitempty"`

	// Grayscale
	Grayscale   bool   `json:"grayscale,omitempty"`

	// Watermark parameters
	WatermarkText    string  `json:"watermark_text,omitempty"`
	WatermarkImage   string  `json:"watermark_image,omitempty"`
	WatermarkOpacity float64 `json:"watermark_opacity,omitempty"`
	WatermarkPosition CropPosition `json:"watermark_position,omitempty"`

	// Optimization
	Optimize    bool `json:"optimize,omitempty"`
	Progressive bool `json:"progressive,omitempty"`
	StripMetadata bool `json:"strip_metadata,omitempty"`
}

// ImageTransformation represents an image transformation request/result
type ImageTransformation struct {
	ID             uuid.UUID            `json:"id"`
	SourceFileID   uuid.UUID            `json:"source_file_id"`
	SourceFileKey  string               `json:"source_file_key"`

	// Transformation details
	Type           TransformationType   `json:"type"`
	Params         TransformationParams `json:"params"`
	ParamsHash     string               `json:"params_hash"` // For caching

	// Result
	ResultFileKey  string               `json:"result_file_key,omitempty"`
	ResultURL      string               `json:"result_url,omitempty"`
	ResultWidth    int                  `json:"result_width,omitempty"`
	ResultHeight   int                  `json:"result_height,omitempty"`
	ResultSize     int64                `json:"result_size,omitempty"`
	ResultMimeType string               `json:"result_mime_type,omitempty"`

	// Status
	Status         TransformationStatus `json:"status"`
	ErrorMessage   string               `json:"error_message,omitempty"`

	// Performance metrics
	ProcessingTime int64                `json:"processing_time_ms,omitempty"` // in milliseconds

	// Cache info
	CacheKey       string               `json:"cache_key,omitempty"`
	CacheHit       bool                 `json:"cache_hit"`
	CachedAt       *time.Time           `json:"cached_at,omitempty"`
	ExpiresAt      *time.Time           `json:"expires_at,omitempty"`

	// Usage tracking
	UsageCount     int64                `json:"usage_count"`
	LastUsedAt     *time.Time           `json:"last_used_at,omitempty"`

	// Metadata
	Metadata       json.RawMessage      `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// NewImageTransformation creates a new image transformation
func NewImageTransformation(sourceFileID uuid.UUID, sourceFileKey string, transformType TransformationType, params TransformationParams) *ImageTransformation {
	now := time.Now()
	id := uuid.New()

	it := &ImageTransformation{
		ID:            id,
		SourceFileID:  sourceFileID,
		SourceFileKey: sourceFileKey,
		Type:          transformType,
		Params:        params,
		Status:        TransformationStatusPending,
		Metadata:      json.RawMessage("{}"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Generate params hash for caching
	it.ParamsHash = it.generateParamsHash()
	it.CacheKey = it.generateCacheKey()

	return it
}

// generateParamsHash generates a hash of the transformation parameters
func (it *ImageTransformation) generateParamsHash() string {
	paramsJSON, _ := json.Marshal(it.Params)
	hash := sha256.Sum256(paramsJSON)
	return hex.EncodeToString(hash[:8]) // First 8 bytes for shorter hash
}

// generateCacheKey generates a unique cache key for this transformation
func (it *ImageTransformation) generateCacheKey() string {
	return fmt.Sprintf("%s_%s_%s", it.SourceFileKey, it.Type, it.ParamsHash)
}

// MarkProcessing marks the transformation as processing
func (it *ImageTransformation) MarkProcessing() {
	it.Status = TransformationStatusProcessing
	it.UpdatedAt = time.Now()
}

// Complete marks the transformation as completed
func (it *ImageTransformation) Complete(resultKey, resultURL, mimeType string, width, height int, size int64, processingTimeMs int64) {
	now := time.Now()
	it.Status = TransformationStatusCompleted
	it.ResultFileKey = resultKey
	it.ResultURL = resultURL
	it.ResultMimeType = mimeType
	it.ResultWidth = width
	it.ResultHeight = height
	it.ResultSize = size
	it.ProcessingTime = processingTimeMs
	it.UpdatedAt = now
}

// Fail marks the transformation as failed
func (it *ImageTransformation) Fail(errorMessage string) {
	it.Status = TransformationStatusFailed
	it.ErrorMessage = errorMessage
	it.UpdatedAt = time.Now()
}

// MarkCached marks the transformation as served from cache
func (it *ImageTransformation) MarkCached() {
	now := time.Now()
	it.Status = TransformationStatusCached
	it.CacheHit = true
	it.CachedAt = &now
	it.UpdatedAt = now
}

// SetCacheExpiry sets when the cached result expires
func (it *ImageTransformation) SetCacheExpiry(duration time.Duration) {
	expiresAt := time.Now().Add(duration)
	it.ExpiresAt = &expiresAt
	it.UpdatedAt = time.Now()
}

// RecordUsage records a usage of this transformation
func (it *ImageTransformation) RecordUsage() {
	now := time.Now()
	it.UsageCount++
	it.LastUsedAt = &now
	it.UpdatedAt = now
}

// IsCacheExpired checks if the cache has expired
func (it *ImageTransformation) IsCacheExpired() bool {
	if it.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*it.ExpiresAt)
}

// IsCompleted checks if the transformation is completed
func (it *ImageTransformation) IsCompleted() bool {
	return it.Status == TransformationStatusCompleted || it.Status == TransformationStatusCached
}

// GetResultURL returns the result URL (empty if not completed)
func (it *ImageTransformation) GetResultURL() string {
	if !it.IsCompleted() {
		return ""
	}
	return it.ResultURL
}

// TransformationPreset represents a predefined transformation preset
type TransformationPreset struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Type        TransformationType   `json:"type"`
	Params      TransformationParams `json:"params"`
}

// Common presets
var (
	PresetThumbnail = TransformationPreset{
		Name:        "thumbnail",
		Description: "200x200 thumbnail",
		Type:        TransformationTypeThumbnail,
		Params: TransformationParams{
			Width:   200,
			Height:  200,
			Mode:    ResizeModeFill,
			Quality: 80,
		},
	}

	PresetSmall = TransformationPreset{
		Name:        "small",
		Description: "400x400 small image",
		Type:        TransformationTypeResize,
		Params: TransformationParams{
			Width:   400,
			Height:  400,
			Mode:    ResizeModeFit,
			Quality: 85,
		},
	}

	PresetMedium = TransformationPreset{
		Name:        "medium",
		Description: "800x800 medium image",
		Type:        TransformationTypeResize,
		Params: TransformationParams{
			Width:   800,
			Height:  800,
			Mode:    ResizeModeFit,
			Quality: 85,
		},
	}

	PresetLarge = TransformationPreset{
		Name:        "large",
		Description: "1200x1200 large image",
		Type:        TransformationTypeResize,
		Params: TransformationParams{
			Width:   1200,
			Height:  1200,
			Mode:    ResizeModeFit,
			Quality: 90,
		},
	}

	PresetWebP = TransformationPreset{
		Name:        "webp",
		Description: "Convert to WebP format",
		Type:        TransformationTypeConvert,
		Params: TransformationParams{
			Format:  "webp",
			Quality: 85,
		},
	}

	PresetOptimized = TransformationPreset{
		Name:        "optimized",
		Description: "Optimized for web",
		Type:        TransformationTypeOptimize,
		Params: TransformationParams{
			Optimize:      true,
			Progressive:   true,
			StripMetadata: true,
			Quality:       85,
		},
	}
)

// GetPreset returns a transformation preset by name
func GetPreset(name string) *TransformationPreset {
	switch name {
	case "thumbnail":
		return &PresetThumbnail
	case "small":
		return &PresetSmall
	case "medium":
		return &PresetMedium
	case "large":
		return &PresetLarge
	case "webp":
		return &PresetWebP
	case "optimized":
		return &PresetOptimized
	default:
		return nil
	}
}

// TransformationBatch represents a batch of transformations to apply
type TransformationBatch struct {
	SourceFileID    uuid.UUID              `json:"source_file_id"`
	Transformations []TransformationType   `json:"transformations"`
	Presets         []string               `json:"presets,omitempty"`
	Results         []*ImageTransformation `json:"results,omitempty"`
}
