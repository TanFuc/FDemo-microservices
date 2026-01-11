package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// RefundStatus represents the status of a refund
type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "PENDING"
	RefundStatusApproved  RefundStatus = "APPROVED"
	RefundStatusRejected  RefundStatus = "REJECTED"
	RefundStatusProcessed RefundStatus = "PROCESSED"
	RefundStatusFailed    RefundStatus = "FAILED"
)

// RefundType represents the type of refund
type RefundType string

const (
	RefundTypeFull    RefundType = "FULL"
	RefundTypePartial RefundType = "PARTIAL"
)

// RefundReason represents predefined refund reasons
type RefundReason string

const (
	RefundReasonCustomerRequest  RefundReason = "CUSTOMER_REQUEST"
	RefundReasonItemNotReceived  RefundReason = "ITEM_NOT_RECEIVED"
	RefundReasonItemDamaged      RefundReason = "ITEM_DAMAGED"
	RefundReasonWrongItem        RefundReason = "WRONG_ITEM"
	RefundReasonItemNotAsDesc    RefundReason = "ITEM_NOT_AS_DESCRIBED"
	RefundReasonDuplicateOrder   RefundReason = "DUPLICATE_ORDER"
	RefundReasonFraudulent       RefundReason = "FRAUDULENT"
	RefundReasonSellerCancelled  RefundReason = "SELLER_CANCELLED"
	RefundReasonOutOfStock       RefundReason = "OUT_OF_STOCK"
	RefundReasonOther            RefundReason = "OTHER"
)

// OrderRefund represents a refund request for an order
type OrderRefund struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key"`
	OrderID uuid.UUID `gorm:"type:uuid;index;not null"`

	// Refund identification
	RefundNumber string `gorm:"type:varchar(50);unique;index;not null"`

	// Refund details
	Type   RefundType   `gorm:"type:varchar(20);not null"`
	Status RefundStatus `gorm:"type:varchar(20);index;not null"`
	Reason RefundReason `gorm:"type:varchar(50)"`

	// Amount
	RequestedAmount decimal.Decimal `gorm:"type:numeric(19,4);not null"`
	ApprovedAmount  decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	RefundedAmount  decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	Currency        string          `gorm:"type:varchar(3);default:'VND'"`

	// Items being refunded (for partial refunds)
	RefundItems json.RawMessage `gorm:"type:jsonb"` // Array of {itemId, quantity, amount}

	// Customer input
	CustomerReason string `gorm:"type:text"`
	CustomerNote   string `gorm:"type:text"`
	EvidenceURLs   string `gorm:"type:text"` // JSON array of image/video URLs

	// Admin/Seller response
	AdminNote        string `gorm:"type:text"`
	RejectionReason  string `gorm:"type:text"`

	// Payment refund tracking
	PaymentProvider   string `gorm:"type:varchar(50)"`
	PaymentRefundID   string `gorm:"type:varchar(255)"`
	RefundMethod      string `gorm:"type:varchar(50)"` // original_payment, wallet, bank_transfer

	// Processing
	RequestedBy   string     `gorm:"type:varchar(100)"` // customer, seller, admin
	RequestedByID string     `gorm:"type:varchar(100)"`
	ApprovedBy    string     `gorm:"type:varchar(100)"`
	ApprovedByID  string     `gorm:"type:varchar(100)"`
	ProcessedBy   string     `gorm:"type:varchar(100)"`
	ProcessedByID string     `gorm:"type:varchar(100)"`

	// Timestamps
	RequestedAt *time.Time     `gorm:"type:timestamptz"`
	ApprovedAt  *time.Time     `gorm:"type:timestamptz"`
	RejectedAt  *time.Time     `gorm:"type:timestamptz"`
	ProcessedAt *time.Time     `gorm:"type:timestamptz"`
	CreatedAt   time.Time      `gorm:"type:timestamptz;not null"`
	UpdatedAt   time.Time      `gorm:"type:timestamptz;not null"`
	DeletedAt   gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

// TableName specifies the table name for OrderRefund
func (OrderRefund) TableName() string {
	return "order_refunds"
}

// RefundItem represents an item in a partial refund
type RefundItem struct {
	OrderItemID uuid.UUID       `json:"orderItemId"`
	Quantity    int             `json:"quantity"`
	Amount      decimal.Decimal `json:"amount"`
	Reason      string          `json:"reason,omitempty"`
}

// NewOrderRefund creates a new refund request
func NewOrderRefund(
	orderID uuid.UUID,
	refundType RefundType,
	reason RefundReason,
	requestedAmount decimal.Decimal,
	customerReason string,
	requestedBy, requestedByID string,
) *OrderRefund {
	now := time.Now()
	return &OrderRefund{
		ID:              uuid.New(),
		OrderID:         orderID,
		RefundNumber:    generateRefundNumber(),
		Type:            refundType,
		Status:          RefundStatusPending,
		Reason:          reason,
		RequestedAmount: requestedAmount,
		Currency:        "VND",
		CustomerReason:  customerReason,
		RequestedBy:     requestedBy,
		RequestedByID:   requestedByID,
		RequestedAt:     &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// generateRefundNumber generates a unique refund number
func generateRefundNumber() string {
	return "RF" + time.Now().Format("20060102") + "-" + uuid.New().String()[:8]
}

// Approve approves the refund request
func (r *OrderRefund) Approve(amount decimal.Decimal, approvedBy, approvedByID, note string) error {
	if r.Status != RefundStatusPending {
		return ErrInvalidRefundStatus
	}
	now := time.Now()
	r.Status = RefundStatusApproved
	r.ApprovedAmount = amount
	r.ApprovedBy = approvedBy
	r.ApprovedByID = approvedByID
	r.AdminNote = note
	r.ApprovedAt = &now
	r.UpdatedAt = now
	return nil
}

// Reject rejects the refund request
func (r *OrderRefund) Reject(reason string, rejectedBy, rejectedByID string) error {
	if r.Status != RefundStatusPending {
		return ErrInvalidRefundStatus
	}
	now := time.Now()
	r.Status = RefundStatusRejected
	r.RejectionReason = reason
	r.ApprovedBy = rejectedBy
	r.ApprovedByID = rejectedByID
	r.RejectedAt = &now
	r.UpdatedAt = now
	return nil
}

// Process marks the refund as processed
func (r *OrderRefund) Process(paymentRefundID string, processedBy, processedByID string) error {
	if r.Status != RefundStatusApproved {
		return ErrInvalidRefundStatus
	}
	now := time.Now()
	r.Status = RefundStatusProcessed
	r.RefundedAmount = r.ApprovedAmount
	r.PaymentRefundID = paymentRefundID
	r.ProcessedBy = processedBy
	r.ProcessedByID = processedByID
	r.ProcessedAt = &now
	r.UpdatedAt = now
	return nil
}

// SetRefundItems sets the items for a partial refund
func (r *OrderRefund) SetRefundItems(items []RefundItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	r.RefundItems = data
	return nil
}

// GetRefundItems returns the refund items
func (r *OrderRefund) GetRefundItems() ([]RefundItem, error) {
	if r.RefundItems == nil {
		return nil, nil
	}
	var items []RefundItem
	if err := json.Unmarshal(r.RefundItems, &items); err != nil {
		return nil, err
	}
	return items, nil
}
