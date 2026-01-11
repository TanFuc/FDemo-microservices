package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// OrderStatus represents the current state of an order
type OrderStatus string

const (
	StatusPending         OrderStatus = "PENDING"          // Created, waiting for payment
	StatusPaid            OrderStatus = "PAID"             // Payment success
	StatusConfirmed       OrderStatus = "CONFIRMED"        // Seller confirmed
	StatusProcessing      OrderStatus = "PROCESSING"       // Being prepared
	StatusReadyToShip     OrderStatus = "READY_TO_SHIP"    // Ready for pickup
	StatusShipped         OrderStatus = "SHIPPED"          // Handed to carrier
	StatusInTransit       OrderStatus = "IN_TRANSIT"       // On the way
	StatusOutForDelivery  OrderStatus = "OUT_FOR_DELIVERY" // Out for delivery
	StatusDelivered       OrderStatus = "DELIVERED"        // Delivered to customer
	StatusCompleted       OrderStatus = "COMPLETED"        // Order completed
	StatusCancelled       OrderStatus = "CANCELLED"        // Cancelled
	StatusRefunded        OrderStatus = "REFUNDED"         // Fully refunded
	StatusPartialRefunded OrderStatus = "PARTIAL_REFUNDED" // Partially refunded
	StatusFailed          OrderStatus = "FAILED"           // Payment failed
)

// PaymentStatus represents the payment state
type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "PENDING"
	PaymentPaid      PaymentStatus = "PAID"
	PaymentFailed    PaymentStatus = "FAILED"
	PaymentRefunded  PaymentStatus = "REFUNDED"
	PaymentCancelled PaymentStatus = "CANCELLED"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeNormal    OrderType = "NORMAL"
	OrderTypePreOrder  OrderType = "PRE_ORDER"
	OrderTypeFlashSale OrderType = "FLASH_SALE"
)

// ShippingAddress represents the delivery address snapshot
type ShippingAddress struct {
	ContactName          string  `json:"contactName"`
	Phone                string  `json:"phone"`
	Email                string  `json:"email,omitempty"`
	CountryCode          string  `json:"countryCode"`
	ProvinceCode         string  `json:"provinceCode"`
	ProvinceName         string  `json:"provinceName"`
	DistrictCode         string  `json:"districtCode"`
	DistrictName         string  `json:"districtName"`
	WardCode             string  `json:"wardCode"`
	WardName             string  `json:"wardName"`
	StreetAddress        string  `json:"streetAddress"`
	Apartment            string  `json:"apartment,omitempty"`
	PostalCode           string  `json:"postalCode,omitempty"`
	FullAddress          string  `json:"fullAddress"`
	Latitude             float64 `json:"latitude,omitempty"`
	Longitude            float64 `json:"longitude,omitempty"`
	DeliveryInstructions string  `json:"deliveryInstructions,omitempty"`
}

// Order is the aggregate root for the order domain
type Order struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID uuid.UUID `gorm:"type:uuid;index;not null"`

	// Order identification
	OrderNumber string    `gorm:"type:varchar(50);unique;index;not null"`
	OrderType   OrderType `gorm:"type:varchar(20);default:'NORMAL'"`

	// Shop information (for marketplace)
	ShopID   string `gorm:"type:varchar(100);index"`
	ShopName string `gorm:"type:varchar(255)"`

	// Money fields - using strict decimal for precision
	SubTotal       decimal.Decimal `gorm:"type:numeric(19,4);not null"`
	ShippingFee    decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	ShippingDiscount decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	TaxAmount      decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	DiscountAmount decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	VoucherDiscount decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	CoinDiscount   decimal.Decimal `gorm:"type:numeric(19,4);default:0"`
	FinalAmount    decimal.Decimal `gorm:"type:numeric(19,4);not null"`
	Currency       string          `gorm:"type:varchar(3);default:'VND'"`

	// Item counts
	TotalItems    int `gorm:"type:int;default:0"`
	TotalQuantity int `gorm:"type:int;default:0"`

	// Status
	Status        OrderStatus   `gorm:"type:varchar(20);index;not null"`
	PaymentStatus PaymentStatus `gorm:"type:varchar(20);index;default:'PENDING'"`

	// Payment information
	PaymentMethod    string     `gorm:"type:varchar(50)"`
	PaymentProvider  string     `gorm:"type:varchar(50)"`
	PaymentReference string     `gorm:"type:varchar(255)"`
	PaidAt           *time.Time `gorm:"type:timestamptz"`

	// Shipping information
	ShippingMethod   string          `gorm:"type:varchar(50)"`
	ShippingCarrier  string          `gorm:"type:varchar(50)"`
	TrackingNumber   string          `gorm:"type:varchar(100)"`
	ShippingAddress  json.RawMessage `gorm:"type:jsonb"`
	EstimatedDelivery *time.Time     `gorm:"type:timestamptz"`
	ActualDelivery   *time.Time      `gorm:"type:timestamptz"`

	// Voucher/Promotion
	VoucherCode     string `gorm:"type:varchar(50)"`
	VoucherID       string `gorm:"type:varchar(100)"`
	CampaignID      string `gorm:"type:varchar(100)"`
	PromotionIDs    string `gorm:"type:text"` // Comma-separated IDs

	// Customer notes
	CustomerNote string `gorm:"type:text"`
	SellerNote   string `gorm:"type:text"`
	InternalNote string `gorm:"type:text"`

	// Timestamps
	ConfirmedAt *time.Time `gorm:"type:timestamptz"`
	ShippedAt   *time.Time `gorm:"type:timestamptz"`
	DeliveredAt *time.Time `gorm:"type:timestamptz"`
	CompletedAt *time.Time `gorm:"type:timestamptz"`
	CancelledAt *time.Time `gorm:"type:timestamptz"`

	// Cancellation
	CancelReason    string `gorm:"type:text"`
	CancelledBy     string `gorm:"type:varchar(100)"` // user, seller, system
	CancelledByID   string `gorm:"type:varchar(100)"`

	// Risk & Fraud
	RiskScore       float64 `gorm:"type:decimal(5,2);default:0"`
	RiskFlags       string  `gorm:"type:text"` // JSON array of flags
	IsFlagged       bool    `gorm:"type:boolean;default:false"`

	// Source tracking
	Source          string `gorm:"type:varchar(50)"` // web, mobile_app, api
	DeviceType      string `gorm:"type:varchar(20)"` // mobile, desktop, tablet
	IPAddress       string `gorm:"type:varchar(45)"`
	UserAgent       string `gorm:"type:text"`

	// Metadata
	Metadata json.RawMessage `gorm:"type:jsonb"`

	// Audit
	CreatedBy string `gorm:"type:varchar(100)"`
	UpdatedBy string `gorm:"type:varchar(100)"`

	// Timestamps
	CreatedAt time.Time      `gorm:"type:timestamptz;not null"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`

	// Version for optimistic locking
	Version int `gorm:"type:int;default:1"`

	// Relations
	Items         []OrderItem         `gorm:"foreignKey:OrderID"`
	StatusHistory []OrderStatusHistory `gorm:"foreignKey:OrderID"`
	Notes         []OrderNote         `gorm:"foreignKey:OrderID"`
}

// TableName specifies the table name for Order
func (Order) TableName() string {
	return "orders"
}

// NewOrder creates a new order with generated UUID
func NewOrder(userID uuid.UUID, paymentMethod string, shippingAddr ShippingAddress) *Order {
	addrJSON, _ := json.Marshal(shippingAddr)
	return &Order{
		ID:              uuid.New(),
		UserID:          userID,
		OrderNumber:     generateOrderNumber(),
		OrderType:       OrderTypeNormal,
		Status:          StatusPending,
		PaymentStatus:   PaymentPending,
		PaymentMethod:   paymentMethod,
		ShippingAddress: addrJSON,
		SubTotal:        decimal.Zero,
		ShippingFee:     decimal.Zero,
		DiscountAmount:  decimal.Zero,
		FinalAmount:     decimal.Zero,
		Currency:        "VND",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Version:         1,
	}
}

// generateOrderNumber generates a unique order number
func generateOrderNumber() string {
	return time.Now().Format("20060102") + "-" + uuid.New().String()[:8]
}

// AddItem adds an order item to the order
func (o *Order) AddItem(item OrderItem) {
	item.OrderID = o.ID
	o.Items = append(o.Items, item)
	o.TotalItems++
	o.TotalQuantity += item.Quantity
}

// CalculateTotals calculates and sets all money fields
func (o *Order) CalculateTotals() {
	subTotal := decimal.Zero
	for _, item := range o.Items {
		subTotal = subTotal.Add(item.SubTotal)
	}

	o.SubTotal = subTotal
	totalDiscount := o.DiscountAmount.Add(o.VoucherDiscount).Add(o.CoinDiscount)
	shippingAfterDiscount := o.ShippingFee.Sub(o.ShippingDiscount)
	if shippingAfterDiscount.IsNegative() {
		shippingAfterDiscount = decimal.Zero
	}
	o.FinalAmount = subTotal.Add(shippingAfterDiscount).Add(o.TaxAmount).Sub(totalDiscount)
}

// CanCancel checks if the order can be cancelled
func (o *Order) CanCancel() bool {
	cancelableStatuses := map[OrderStatus]bool{
		StatusPending:   true,
		StatusPaid:      true,
		StatusConfirmed: true,
	}
	return cancelableStatuses[o.Status]
}

// Cancel cancels the order
func (o *Order) Cancel(reason, cancelledBy, cancelledByID string) error {
	if !o.CanCancel() {
		return ErrOrderCannotBeCancelled
	}
	now := time.Now()
	o.Status = StatusCancelled
	o.CancelReason = reason
	o.CancelledBy = cancelledBy
	o.CancelledByID = cancelledByID
	o.CancelledAt = &now
	o.UpdatedAt = now
	o.Version++
	return nil
}

// MarkAsPaid marks the order as paid
func (o *Order) MarkAsPaid(paymentRef string) error {
	if o.Status != StatusPending {
		return ErrInvalidOrderStatusTransition
	}
	now := time.Now()
	o.Status = StatusPaid
	o.PaymentStatus = PaymentPaid
	o.PaymentReference = paymentRef
	o.PaidAt = &now
	o.UpdatedAt = now
	o.Version++
	return nil
}

// Confirm confirms the order by seller
func (o *Order) Confirm() error {
	if o.Status != StatusPaid {
		return ErrInvalidOrderStatusTransition
	}
	now := time.Now()
	o.Status = StatusConfirmed
	o.ConfirmedAt = &now
	o.UpdatedAt = now
	o.Version++
	return nil
}

// MarkAsShipped marks the order as shipped
func (o *Order) MarkAsShipped(trackingNumber, carrier string) error {
	validStatuses := map[OrderStatus]bool{
		StatusConfirmed:   true,
		StatusProcessing:  true,
		StatusReadyToShip: true,
	}
	if !validStatuses[o.Status] {
		return ErrInvalidOrderStatusTransition
	}
	now := time.Now()
	o.Status = StatusShipped
	o.TrackingNumber = trackingNumber
	o.ShippingCarrier = carrier
	o.ShippedAt = &now
	o.UpdatedAt = now
	o.Version++
	return nil
}

// MarkAsDelivered marks the order as delivered
func (o *Order) MarkAsDelivered() error {
	validStatuses := map[OrderStatus]bool{
		StatusShipped:        true,
		StatusInTransit:      true,
		StatusOutForDelivery: true,
	}
	if !validStatuses[o.Status] {
		return ErrInvalidOrderStatusTransition
	}
	now := time.Now()
	o.Status = StatusDelivered
	o.DeliveredAt = &now
	o.ActualDelivery = &now
	o.UpdatedAt = now
	o.Version++
	return nil
}

// Complete marks the order as completed
func (o *Order) Complete() error {
	if o.Status != StatusDelivered {
		return ErrInvalidOrderStatusTransition
	}
	now := time.Now()
	o.Status = StatusCompleted
	o.CompletedAt = &now
	o.UpdatedAt = now
	o.Version++
	return nil
}

// GetShippingAddress parses and returns the shipping address
func (o *Order) GetShippingAddress() (*ShippingAddress, error) {
	var addr ShippingAddress
	if err := json.Unmarshal(o.ShippingAddress, &addr); err != nil {
		return nil, err
	}
	return &addr, nil
}
