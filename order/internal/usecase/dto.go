package usecase

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateOrderRequest represents the input for creating an order
type CreateOrderRequest struct {
	UserID          uuid.UUID              `json:"user_id" validate:"required"`
	Items           []CreateOrderItemDTO   `json:"items" validate:"required,min=1,dive"`
	ShippingAddress ShippingAddressDTO     `json:"shipping_address" validate:"required"`
	PaymentMethod   string                 `json:"payment_method" validate:"required,oneof=MOMO COD STRIPE"`
	ShippingFee     decimal.Decimal        `json:"shipping_fee"`
	DiscountAmount  decimal.Decimal        `json:"discount_amount"`
}

// CreateOrderItemDTO represents an item in the create order request
type CreateOrderItemDTO struct {
	ProductID   string          `json:"product_id" validate:"required"`
	SkuID       string          `json:"sku_id" validate:"required"`
	ProductName string          `json:"product_name" validate:"required"`
	SkuCode     string          `json:"sku_code" validate:"required"`
	Thumbnail   string          `json:"thumbnail"`
	Quantity    int             `json:"quantity" validate:"required,min=1"`
	UnitPrice   decimal.Decimal `json:"unit_price" validate:"required"`
}

// ShippingAddressDTO represents the shipping address
type ShippingAddressDTO struct {
	FullName    string `json:"full_name" validate:"required"`
	Phone       string `json:"phone" validate:"required"`
	Address     string `json:"address" validate:"required"`
	Ward        string `json:"ward"`
	District    string `json:"district" validate:"required"`
	City        string `json:"city" validate:"required"`
	Country     string `json:"country" validate:"required"`
	PostalCode  string `json:"postal_code"`
}

// ToJSON converts ShippingAddressDTO to JSON
func (s *ShippingAddressDTO) ToJSON() (json.RawMessage, error) {
	return json.Marshal(s)
}

// OrderResponse represents the output for an order
type OrderResponse struct {
	ID              uuid.UUID          `json:"id"`
	UserID          uuid.UUID          `json:"user_id"`
	TotalAmount     decimal.Decimal    `json:"total_amount"`
	ShippingFee     decimal.Decimal    `json:"shipping_fee"`
	DiscountAmount  decimal.Decimal    `json:"discount_amount"`
	FinalAmount     decimal.Decimal    `json:"final_amount"`
	Status          string             `json:"status"`
	PaymentMethod   string             `json:"payment_method"`
	ShippingAddress json.RawMessage    `json:"shipping_address"`
	Items           []OrderItemResponse `json:"items"`
	CreatedAt       string             `json:"created_at"`
	UpdatedAt       string             `json:"updated_at"`
}

// OrderItemResponse represents an order item in the response
type OrderItemResponse struct {
	ID          uuid.UUID       `json:"id"`
	ProductID   string          `json:"product_id"`
	SkuID       string          `json:"sku_id"`
	ProductName string          `json:"product_name"`
	SkuCode     string          `json:"sku_code"`
	Thumbnail   string          `json:"thumbnail"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	SubTotal    decimal.Decimal `json:"sub_total"`
}

// CancelOrderRequest represents the input for cancelling an order
type CancelOrderRequest struct {
	OrderID uuid.UUID `json:"order_id" validate:"required"`
}

// GetOrderRequest represents the input for getting an order
type GetOrderRequest struct {
	OrderID uuid.UUID `json:"order_id" validate:"required"`
}

// ListOrdersRequest represents the input for listing orders
type ListOrdersRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}
