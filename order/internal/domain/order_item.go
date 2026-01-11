package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderItem represents a snapshot of a product at the time of purchase
type OrderItem struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key"`
	OrderID uuid.UUID `gorm:"type:uuid;index;not null"`

	// Product Snapshot - immutable after creation
	ProductID   string `gorm:"type:varchar(100);index"` // MongoID string from Catalog
	SkuID       string `gorm:"type:varchar(100);index"` // Inventory SkuID
	ProductName string `gorm:"type:varchar(255)"`       // Snapshot name at time of purchase
	SkuCode     string `gorm:"type:varchar(100)"`       // Snapshot code
	Thumbnail   string `gorm:"type:varchar(500)"`

	// Variant attributes snapshot
	VariantAttributes json.RawMessage `gorm:"type:jsonb"` // {"color": "Red", "size": "M"}

	// Pricing at time of purchase
	OriginalPrice decimal.Decimal `gorm:"type:numeric(19,4)"` // Original price before any discount
	UnitPrice     decimal.Decimal `gorm:"type:numeric(19,4)"` // Price after item discount
	Quantity      int             `gorm:"type:int;not null"`
	SubTotal      decimal.Decimal `gorm:"type:numeric(19,4)"` // UnitPrice * Quantity

	// Discounts applied to this item
	DiscountAmount decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	DiscountType   string          `gorm:"type:varchar(20)"` // PERCENTAGE, FIXED
	DiscountReason string          `gorm:"type:varchar(255)"`

	// Weight for shipping calculation
	Weight float64 `gorm:"type:decimal(10,2)"` // grams

	// Reservation ID from inventory service
	ReservationID string `gorm:"type:varchar(100)"`

	// For refunds/returns
	RefundedQuantity int             `gorm:"type:int;default:0"`
	RefundedAmount   decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	ReturnedQuantity int             `gorm:"type:int;default:0"`

	// Status
	Status string `gorm:"type:varchar(20);default:'ACTIVE'"` // ACTIVE, REFUNDED, RETURNED

	// Timestamps
	CreatedAt time.Time `gorm:"type:timestamptz"`
	UpdatedAt time.Time `gorm:"type:timestamptz"`
}

// TableName specifies the table name for OrderItem
func (OrderItem) TableName() string {
	return "order_items"
}

// NewOrderItem creates a new order item with calculated subtotal
func NewOrderItem(
	productID string,
	skuID string,
	productName string,
	skuCode string,
	thumbnail string,
	variantAttrs map[string]string,
	quantity int,
	originalPrice decimal.Decimal,
	unitPrice decimal.Decimal,
	weight float64,
) *OrderItem {
	variantJSON, _ := json.Marshal(variantAttrs)
	return &OrderItem{
		ID:                uuid.New(),
		ProductID:         productID,
		SkuID:             skuID,
		ProductName:       productName,
		SkuCode:           skuCode,
		Thumbnail:         thumbnail,
		VariantAttributes: variantJSON,
		Quantity:          quantity,
		OriginalPrice:     originalPrice,
		UnitPrice:         unitPrice,
		SubTotal:          unitPrice.Mul(decimal.NewFromInt(int64(quantity))),
		Weight:            weight,
		Status:            "ACTIVE",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// SetReservationID sets the reservation ID from inventory service
func (oi *OrderItem) SetReservationID(reservationID string) {
	oi.ReservationID = reservationID
}

// ApplyDiscount applies a discount to the item
func (oi *OrderItem) ApplyDiscount(amount decimal.Decimal, discountType, reason string) {
	oi.DiscountAmount = amount
	oi.DiscountType = discountType
	oi.DiscountReason = reason
	oi.UnitPrice = oi.OriginalPrice.Sub(amount.Div(decimal.NewFromInt(int64(oi.Quantity))))
	oi.SubTotal = oi.UnitPrice.Mul(decimal.NewFromInt(int64(oi.Quantity)))
}

// CanRefund checks if the item can be refunded
func (oi *OrderItem) CanRefund(quantity int) bool {
	return oi.Status == "ACTIVE" && quantity <= (oi.Quantity-oi.RefundedQuantity)
}

// Refund processes a refund for the item
func (oi *OrderItem) Refund(quantity int, amount decimal.Decimal) error {
	if !oi.CanRefund(quantity) {
		return ErrInvalidRefundQuantity
	}
	oi.RefundedQuantity += quantity
	oi.RefundedAmount = oi.RefundedAmount.Add(amount)
	if oi.RefundedQuantity == oi.Quantity {
		oi.Status = "REFUNDED"
	}
	oi.UpdatedAt = time.Now()
	return nil
}

// GetVariantAttributes returns the parsed variant attributes
func (oi *OrderItem) GetVariantAttributes() (map[string]string, error) {
	var attrs map[string]string
	if err := json.Unmarshal(oi.VariantAttributes, &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}
