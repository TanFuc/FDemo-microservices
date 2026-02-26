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

// ===================== Return Request DTOs =====================

// CreateReturnRequest represents the input for creating a return request
type CreateReturnRequest struct {
	OrderID        uuid.UUID           `json:"order_id" validate:"required"`
	UserID         uuid.UUID           `json:"user_id" validate:"required"`
	ReturnType     string              `json:"return_type" validate:"required,oneof=REFUND EXCHANGE"`
	Reason         string              `json:"reason" validate:"required"`
	Items          []CreateReturnItem  `json:"items" validate:"required,min=1,dive"`
	CustomerReason string              `json:"customer_reason"`
	CustomerNote   string              `json:"customer_note"`
	EvidenceURLs   []string            `json:"evidence_urls"`
}

// CreateReturnItem represents an item in the return request
type CreateReturnItem struct {
	OrderItemID uuid.UUID `json:"order_item_id" validate:"required"`
	Quantity    int       `json:"quantity" validate:"required,min=1"`
	Reason      string    `json:"reason"`
}

// ApproveReturnRequest represents the input for approving a return
type ApproveReturnRequest struct {
	ReturnID     uuid.UUID `json:"return_id" validate:"required"`
	ApprovedBy   string    `json:"approved_by" validate:"required"`
	ApprovedByID string    `json:"approved_by_id" validate:"required"`
	Note         string    `json:"note"`
}

// RejectReturnRequest represents the input for rejecting a return
type RejectReturnRequest struct {
	ReturnID   uuid.UUID `json:"return_id" validate:"required"`
	RejectedBy string    `json:"rejected_by" validate:"required"`
	Reason     string    `json:"reason" validate:"required"`
}

// CompleteReturnRequest represents the input for completing a return
type CompleteReturnRequest struct {
	ReturnID uuid.UUID `json:"return_id" validate:"required"`
}

// ReturnResponse represents the output for a return request
type ReturnResponse struct {
	ID                   uuid.UUID           `json:"id"`
	OrderID              uuid.UUID           `json:"order_id"`
	UserID               uuid.UUID           `json:"user_id"`
	ReturnNumber         string              `json:"return_number"`
	Type                 string              `json:"type"`
	Status               string              `json:"status"`
	Reason               string              `json:"reason"`
	Items                []ReturnItemResponse `json:"items"`
	CustomerReason       string              `json:"customer_reason,omitempty"`
	CustomerNote         string              `json:"customer_note,omitempty"`
	EvidenceURLs         string              `json:"evidence_urls,omitempty"`
	SellerNote           string              `json:"seller_note,omitempty"`
	RejectionReason      string              `json:"rejection_reason,omitempty"`
	StockRestored        bool                `json:"stock_restored"`
	RequestedAt          string              `json:"requested_at"`
	ApprovedAt           string              `json:"approved_at,omitempty"`
	RejectedAt           string              `json:"rejected_at,omitempty"`
	CompletedAt          string              `json:"completed_at,omitempty"`
	CreatedAt            string              `json:"created_at"`
	UpdatedAt            string              `json:"updated_at"`
}

// ReturnItemResponse represents an item in the return response
type ReturnItemResponse struct {
	OrderItemID uuid.UUID `json:"order_item_id"`
	Quantity    int       `json:"quantity"`
	Reason      string    `json:"reason,omitempty"`
	Condition   string    `json:"condition,omitempty"`
}

// ListReturnsRequest represents the input for listing returns
type ListReturnsRequest struct {
	UserID  uuid.UUID `json:"user_id"`
	OrderID uuid.UUID `json:"order_id"`
	Status  string    `json:"status"`
	Page    int       `json:"page"`
	Limit   int       `json:"limit"`
}

// ListReturnsResponse represents the output for listing returns
type ListReturnsResponse struct {
	Returns    []ReturnResponse `json:"returns"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}
