package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FolderType represents the type of folder
type FolderType string

const (
	FolderTypeUser    FolderType = "USER"    // User's personal folder
	FolderTypeShop    FolderType = "SHOP"    // Shop's folder
	FolderTypeSystem  FolderType = "SYSTEM"  // System folders (avatars, banners, etc.)
	FolderTypeShared  FolderType = "SHARED"  // Shared folders
)

// MediaFolder represents a folder for organizing media files
type MediaFolder struct {
	ID            uuid.UUID       `json:"id"`
	UserID        string          `json:"user_id"`
	ShopID        *string         `json:"shop_id,omitempty"`

	// Folder information
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	Description   string          `json:"description,omitempty"`
	Type          FolderType      `json:"type"`

	// Hierarchy
	ParentID      *uuid.UUID      `json:"parent_id,omitempty"`
	Path          string          `json:"path"` // Full path like /products/electronics/phones
	Level         int             `json:"level"` // Depth level (0 = root)

	// Cover image
	CoverImageID  *uuid.UUID      `json:"cover_image_id,omitempty"`
	CoverImageURL string          `json:"cover_image_url,omitempty"`

	// Stats (denormalized)
	FileCount     int64           `json:"file_count"`
	TotalSize     int64           `json:"total_size"` // in bytes
	SubfolderCount int64          `json:"subfolder_count"`

	// Permissions
	IsPublic      bool            `json:"is_public"`
	AllowedUsers  []string        `json:"allowed_users,omitempty"`

	// Sorting
	SortOrder     int             `json:"sort_order"`

	// Metadata
	Metadata      json.RawMessage `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty"`
}

// NewMediaFolder creates a new media folder
func NewMediaFolder(userID, name string, folderType FolderType) *MediaFolder {
	now := time.Now()
	id := uuid.New()
	return &MediaFolder{
		ID:             id,
		UserID:         userID,
		Name:           name,
		Slug:           GenerateSlug(name),
		Type:           folderType,
		Path:           "/" + GenerateSlug(name),
		Level:          0,
		FileCount:      0,
		TotalSize:      0,
		SubfolderCount: 0,
		IsPublic:       false,
		AllowedUsers:   make([]string, 0),
		SortOrder:      0,
		Metadata:       json.RawMessage("{}"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// GenerateSlug generates a URL-safe slug from a name
func GenerateSlug(name string) string {
	// Simple slug generation - in production, use a proper slugify library
	slug := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			slug += string(r)
		} else if r >= 'A' && r <= 'Z' {
			slug += string(r + 32) // lowercase
		} else if r == ' ' || r == '-' || r == '_' {
			slug += "-"
		}
	}
	return slug
}

// SetParent sets the parent folder
func (f *MediaFolder) SetParent(parent *MediaFolder) {
	f.ParentID = &parent.ID
	f.Path = parent.Path + "/" + f.Slug
	f.Level = parent.Level + 1
	f.UpdatedAt = time.Now()
}

// SetRootFolder marks this as a root folder
func (f *MediaFolder) SetRootFolder() {
	f.ParentID = nil
	f.Path = "/" + f.Slug
	f.Level = 0
	f.UpdatedAt = time.Now()
}

// SetCoverImage sets the cover image
func (f *MediaFolder) SetCoverImage(imageID uuid.UUID, imageURL string) {
	f.CoverImageID = &imageID
	f.CoverImageURL = imageURL
	f.UpdatedAt = time.Now()
}

// RemoveCoverImage removes the cover image
func (f *MediaFolder) RemoveCoverImage() {
	f.CoverImageID = nil
	f.CoverImageURL = ""
	f.UpdatedAt = time.Now()
}

// Rename renames the folder
func (f *MediaFolder) Rename(newName string) {
	f.Name = newName
	f.Slug = GenerateSlug(newName)
	// Note: Path should be updated along with all child folders
	f.UpdatedAt = time.Now()
}

// SetPublic makes the folder public
func (f *MediaFolder) SetPublic() {
	f.IsPublic = true
	f.UpdatedAt = time.Now()
}

// SetPrivate makes the folder private
func (f *MediaFolder) SetPrivate() {
	f.IsPublic = false
	f.UpdatedAt = time.Now()
}

// AddAllowedUser adds a user to allowed users
func (f *MediaFolder) AddAllowedUser(userID string) {
	for _, id := range f.AllowedUsers {
		if id == userID {
			return
		}
	}
	f.AllowedUsers = append(f.AllowedUsers, userID)
	f.UpdatedAt = time.Now()
}

// RemoveAllowedUser removes a user from allowed users
func (f *MediaFolder) RemoveAllowedUser(userID string) {
	for i, id := range f.AllowedUsers {
		if id == userID {
			f.AllowedUsers = append(f.AllowedUsers[:i], f.AllowedUsers[i+1:]...)
			f.UpdatedAt = time.Now()
			return
		}
	}
}

// CanAccess checks if a user can access this folder
func (f *MediaFolder) CanAccess(userID string) bool {
	if f.IsPublic {
		return true
	}
	if f.UserID == userID {
		return true
	}
	for _, id := range f.AllowedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

// IncrementFileCount increments the file count
func (f *MediaFolder) IncrementFileCount(fileSize int64) {
	f.FileCount++
	f.TotalSize += fileSize
	f.UpdatedAt = time.Now()
}

// DecrementFileCount decrements the file count
func (f *MediaFolder) DecrementFileCount(fileSize int64) {
	if f.FileCount > 0 {
		f.FileCount--
	}
	if f.TotalSize >= fileSize {
		f.TotalSize -= fileSize
	}
	f.UpdatedAt = time.Now()
}

// IncrementSubfolderCount increments the subfolder count
func (f *MediaFolder) IncrementSubfolderCount() {
	f.SubfolderCount++
	f.UpdatedAt = time.Now()
}

// DecrementSubfolderCount decrements the subfolder count
func (f *MediaFolder) DecrementSubfolderCount() {
	if f.SubfolderCount > 0 {
		f.SubfolderCount--
	}
	f.UpdatedAt = time.Now()
}

// SoftDelete performs a soft delete
func (f *MediaFolder) SoftDelete() {
	now := time.Now()
	f.DeletedAt = &now
	f.UpdatedAt = now
}

// IsDeleted checks if the folder is deleted
func (f *MediaFolder) IsDeleted() bool {
	return f.DeletedAt != nil
}

// IsRoot checks if this is a root folder
func (f *MediaFolder) IsRoot() bool {
	return f.ParentID == nil
}

// TotalSizeInMB returns the total size in MB
func (f *MediaFolder) TotalSizeInMB() float64 {
	return float64(f.TotalSize) / (1024 * 1024)
}

// FolderTree represents a folder with its children for tree display
type FolderTree struct {
	Folder   *MediaFolder   `json:"folder"`
	Children []*FolderTree  `json:"children,omitempty"`
}

// FolderStats represents folder statistics
type FolderStats struct {
	TotalFolders   int64 `json:"total_folders"`
	TotalFiles     int64 `json:"total_files"`
	TotalSize      int64 `json:"total_size"`
	ImageCount     int64 `json:"image_count"`
	VideoCount     int64 `json:"video_count"`
	DocumentCount  int64 `json:"document_count"`
	OtherCount     int64 `json:"other_count"`
}
