package model

import (
	"time"
)

// InventoryItem represents inventory for a SKU
type InventoryItem struct {
	SkuID         string    `gorm:"primaryKey;column:sku_id" json:"sku_id"`
	TotalStock    int       `gorm:"column:total_stock;not null;check:total_stock >= reserved_stock" json:"total_stock"`
	ReservedStock int       `gorm:"column:reserved_stock;not null;default:0" json:"reserved_stock"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name
func (InventoryItem) TableName() string {
	return "inventory_items"
}

// AvailableStock calculates stock that can be reserved
func (i *InventoryItem) AvailableStock() int {
	return i.TotalStock - i.ReservedStock
}

// InventoryFilter for listing inventory items
type InventoryFilter struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// EnsureDefaults sets default values for filter
func (f *InventoryFilter) EnsureDefaults() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}
