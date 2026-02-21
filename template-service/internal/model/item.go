package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Item represents the core entity
type Item struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName specifies the database table name
func (Item) TableName() string {
	return "items"
}

// BeforeCreate hook to generate UUID
func (i *Item) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}
