package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NoteType represents the type of note
type NoteType string

const (
	NoteTypeInternal NoteType = "INTERNAL" // Visible only to staff
	NoteTypePublic   NoteType = "PUBLIC"   // Visible to customer
	NoteTypeSeller   NoteType = "SELLER"   // From seller
	NoteTypeSystem   NoteType = "SYSTEM"   // Auto-generated
)

// OrderNote represents notes/comments on an order
type OrderNote struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key"`
	OrderID uuid.UUID `gorm:"type:uuid;index;not null"`

	// Note content
	Type    NoteType `gorm:"type:varchar(20);not null"`
	Content string   `gorm:"type:text;not null"`

	// Author
	AuthorID   string `gorm:"type:varchar(100)"`
	AuthorName string `gorm:"type:varchar(255)"`
	AuthorType string `gorm:"type:varchar(20)"` // customer, seller, admin, system

	// Visibility
	IsPrivate bool `gorm:"type:boolean;default:false"`

	// For attachments
	AttachmentURLs string `gorm:"type:text"` // JSON array of URLs

	// Timestamps
	CreatedAt time.Time      `gorm:"type:timestamptz;not null"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

// TableName specifies the table name for OrderNote
func (OrderNote) TableName() string {
	return "order_notes"
}

// NewOrderNote creates a new order note
func NewOrderNote(
	orderID uuid.UUID,
	noteType NoteType,
	content string,
	authorID, authorName, authorType string,
	isPrivate bool,
) *OrderNote {
	return &OrderNote{
		ID:         uuid.New(),
		OrderID:    orderID,
		Type:       noteType,
		Content:    content,
		AuthorID:   authorID,
		AuthorName: authorName,
		AuthorType: authorType,
		IsPrivate:  isPrivate,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}
