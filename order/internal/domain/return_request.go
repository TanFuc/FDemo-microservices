package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ReturnStatus represents the status of a return request
type ReturnStatus string

const (
	ReturnStatusPending          ReturnStatus = "PENDING"
	ReturnStatusApproved         ReturnStatus = "APPROVED"
	ReturnStatusRejected         ReturnStatus = "REJECTED"
	ReturnStatusAwaitingPickup   ReturnStatus = "AWAITING_PICKUP"
	ReturnStatusInTransit        ReturnStatus = "IN_TRANSIT"
	ReturnStatusReceived         ReturnStatus = "RECEIVED"
	ReturnStatusInspecting       ReturnStatus = "INSPECTING"
	ReturnStatusCompleted        ReturnStatus = "COMPLETED"
	ReturnStatusCancelled        ReturnStatus = "CANCELLED"
)

// ReturnType represents the type of return
type ReturnType string

const (
	ReturnTypeRefund   ReturnType = "REFUND"
	ReturnTypeExchange ReturnType = "EXCHANGE"
)

// ReturnReason represents predefined return reasons
type ReturnReason string

const (
	ReturnReasonDefective      ReturnReason = "DEFECTIVE"
	ReturnReasonWrongItem      ReturnReason = "WRONG_ITEM"
	ReturnReasonNotAsDescribed ReturnReason = "NOT_AS_DESCRIBED"
	ReturnReasonChangedMind    ReturnReason = "CHANGED_MIND"
	ReturnReasonSizeFit        ReturnReason = "SIZE_FIT"
	ReturnReasonQualityIssue   ReturnReason = "QUALITY_ISSUE"
	ReturnReasonDamagedPackage ReturnReason = "DAMAGED_PACKAGE"
	ReturnReasonOther          ReturnReason = "OTHER"
)

// ReturnRequest represents a return/exchange request for an order
type ReturnRequest struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key"`
	OrderID uuid.UUID `gorm:"type:uuid;index;not null"`
	UserID  uuid.UUID `gorm:"type:uuid;index;not null"`

	// Return identification
	ReturnNumber string `gorm:"type:varchar(50);unique;index;not null"`

	// Return details
	Type   ReturnType   `gorm:"type:varchar(20);not null"`
	Status ReturnStatus `gorm:"type:varchar(20);index;not null"`
	Reason ReturnReason `gorm:"type:varchar(50)"`

	// Items being returned
	ReturnItems json.RawMessage `gorm:"type:jsonb;not null"` // Array of {itemId, quantity, reason}

	// Customer input
	CustomerReason string `gorm:"type:text"`
	CustomerNote   string `gorm:"type:text"`
	EvidenceURLs   string `gorm:"type:text"` // JSON array of image/video URLs

	// Seller/Admin response
	SellerNote       string `gorm:"type:text"`
	AdminNote        string `gorm:"type:text"`
	RejectionReason  string `gorm:"type:text"`

	// Return shipping
	ReturnShippingMethod  string `gorm:"type:varchar(50)"`
	ReturnTrackingNumber  string `gorm:"type:varchar(100)"`
	ReturnCarrier         string `gorm:"type:varchar(50)"`
	ReturnShippingFee     decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	ReturnShippingPaidBy  string `gorm:"type:varchar(20)"` // customer, seller, platform

	// Pickup address (if different from order address)
	PickupAddress json.RawMessage `gorm:"type:jsonb"`

	// Warehouse receiving
	WarehouseID       string     `gorm:"type:varchar(100)"`
	ReceivedAt        *time.Time `gorm:"type:timestamptz"`
	ReceivedBy        string     `gorm:"type:varchar(100)"`
	ReceivedCondition string     `gorm:"type:varchar(50)"` // GOOD, DAMAGED, MISSING_PARTS

	// Inspection
	InspectionNote   string     `gorm:"type:text"`
	InspectedAt      *time.Time `gorm:"type:timestamptz"`
	InspectedBy      string     `gorm:"type:varchar(100)"`

	// Exchange details (if ReturnType is EXCHANGE)
	ExchangeOrderID    *uuid.UUID `gorm:"type:uuid"`
	ExchangeProductIDs string     `gorm:"type:text"` // JSON array

	// Refund reference (if ReturnType is REFUND)
	RefundID *uuid.UUID `gorm:"type:uuid"`

	// Processing
	ApprovedBy    string `gorm:"type:varchar(100)"`
	ApprovedByID  string `gorm:"type:varchar(100)"`

	// Stock restore tracking
	StockRestored   bool       `gorm:"type:boolean;default:false"`
	StockRestoredAt *time.Time `gorm:"type:timestamptz"`

	// Timestamps
	RequestedAt   time.Time      `gorm:"type:timestamptz;not null"`
	ApprovedAt    *time.Time     `gorm:"type:timestamptz"`
	RejectedAt    *time.Time     `gorm:"type:timestamptz"`
	ShippedAt     *time.Time     `gorm:"type:timestamptz"`
	CompletedAt   *time.Time     `gorm:"type:timestamptz"`
	CancelledAt   *time.Time     `gorm:"type:timestamptz"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;not null"`
	UpdatedAt     time.Time      `gorm:"type:timestamptz;not null"`
	DeletedAt     gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

// TableName specifies the table name for ReturnRequest
func (ReturnRequest) TableName() string {
	return "return_requests"
}

// ReturnItem represents an item in a return request
type ReturnItem struct {
	OrderItemID uuid.UUID `json:"orderItemId"`
	Quantity    int       `json:"quantity"`
	Reason      string    `json:"reason,omitempty"`
	Condition   string    `json:"condition,omitempty"` // Condition when received
}

// NewReturnRequest creates a new return request
func NewReturnRequest(
	orderID, userID uuid.UUID,
	returnType ReturnType,
	reason ReturnReason,
	items []ReturnItem,
	customerReason string,
) (*ReturnRequest, error) {
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &ReturnRequest{
		ID:             uuid.New(),
		OrderID:        orderID,
		UserID:         userID,
		ReturnNumber:   generateReturnNumber(),
		Type:           returnType,
		Status:         ReturnStatusPending,
		Reason:         reason,
		ReturnItems:    itemsJSON,
		CustomerReason: customerReason,
		RequestedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// generateReturnNumber generates a unique return number
func generateReturnNumber() string {
	return "RT" + time.Now().Format("20060102") + "-" + uuid.New().String()[:8]
}

// Approve approves the return request
func (r *ReturnRequest) Approve(approvedBy, approvedByID, note string) error {
	if r.Status != ReturnStatusPending {
		return ErrInvalidReturnStatus
	}
	now := time.Now()
	r.Status = ReturnStatusApproved
	r.ApprovedBy = approvedBy
	r.ApprovedByID = approvedByID
	r.SellerNote = note
	r.ApprovedAt = &now
	r.UpdatedAt = now
	return nil
}

// Reject rejects the return request
func (r *ReturnRequest) Reject(reason string, rejectedBy string) error {
	if r.Status != ReturnStatusPending {
		return ErrInvalidReturnStatus
	}
	now := time.Now()
	r.Status = ReturnStatusRejected
	r.RejectionReason = reason
	r.ApprovedBy = rejectedBy
	r.RejectedAt = &now
	r.UpdatedAt = now
	return nil
}

// MarkAsShipped marks the return as shipped by customer
func (r *ReturnRequest) MarkAsShipped(trackingNumber, carrier string) error {
	if r.Status != ReturnStatusApproved && r.Status != ReturnStatusAwaitingPickup {
		return ErrInvalidReturnStatus
	}
	now := time.Now()
	r.Status = ReturnStatusInTransit
	r.ReturnTrackingNumber = trackingNumber
	r.ReturnCarrier = carrier
	r.ShippedAt = &now
	r.UpdatedAt = now
	return nil
}

// MarkAsReceived marks the return as received at warehouse
func (r *ReturnRequest) MarkAsReceived(receivedBy, condition string) error {
	if r.Status != ReturnStatusInTransit {
		return ErrInvalidReturnStatus
	}
	now := time.Now()
	r.Status = ReturnStatusReceived
	r.ReceivedBy = receivedBy
	r.ReceivedCondition = condition
	r.ReceivedAt = &now
	r.UpdatedAt = now
	return nil
}

// Complete marks the return as completed
func (r *ReturnRequest) Complete() error {
	if r.Status != ReturnStatusReceived && r.Status != ReturnStatusInspecting {
		return ErrInvalidReturnStatus
	}
	now := time.Now()
	r.Status = ReturnStatusCompleted
	r.CompletedAt = &now
	r.UpdatedAt = now
	return nil
}

// Cancel cancels the return request
func (r *ReturnRequest) Cancel() error {
	cancelableStatuses := map[ReturnStatus]bool{
		ReturnStatusPending:        true,
		ReturnStatusApproved:       true,
		ReturnStatusAwaitingPickup: true,
	}
	if !cancelableStatuses[r.Status] {
		return ErrInvalidReturnStatus
	}
	now := time.Now()
	r.Status = ReturnStatusCancelled
	r.CancelledAt = &now
	r.UpdatedAt = now
	return nil
}

// GetReturnItems returns the parsed return items
func (r *ReturnRequest) GetReturnItems() ([]ReturnItem, error) {
	var items []ReturnItem
	if err := json.Unmarshal(r.ReturnItems, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// SetRefund links the return to a refund
func (r *ReturnRequest) SetRefund(refundID uuid.UUID) {
	r.RefundID = &refundID
	r.UpdatedAt = time.Now()
}

// SetExchangeOrder links the return to an exchange order
func (r *ReturnRequest) SetExchangeOrder(exchangeOrderID uuid.UUID) {
	r.ExchangeOrderID = &exchangeOrderID
	r.UpdatedAt = time.Now()
}
