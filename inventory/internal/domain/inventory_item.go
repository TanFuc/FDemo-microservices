package domain

import (
	"time"
)

type InventoryItem struct {
	SkuID         string    `gorm:"primaryKey;column:sku_id" json:"sku_id"`
	TotalStock    int       `gorm:"column:total_stock;not null;check:total_stock >= reserved_stock" json:"total_stock"`
	ReservedStock int       `gorm:"column:reserved_stock;not null;default:0" json:"reserved_stock"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (InventoryItem) TableName() string {
	return "inventory_items"
}

func (i *InventoryItem) AvailableStock() int {
	return i.TotalStock - i.ReservedStock
}
