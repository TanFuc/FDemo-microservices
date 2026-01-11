package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DeliverySlotStatus represents the status of a delivery slot
type DeliverySlotStatus string

const (
	DeliverySlotStatusAvailable DeliverySlotStatus = "AVAILABLE"
	DeliverySlotStatusFull      DeliverySlotStatus = "FULL"
	DeliverySlotStatusClosed    DeliverySlotStatus = "CLOSED"
)

// DeliverySlotTemplate represents a recurring delivery slot template
type DeliverySlotTemplate struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	DisplayName     string          `json:"display_name" db:"display_name"`

	// Zone association
	ZoneID          *uuid.UUID      `json:"zone_id,omitempty" db:"zone_id"` // null = all zones
	CarrierID       *uuid.UUID      `json:"carrier_id,omitempty" db:"carrier_id"` // null = all carriers

	// Time configuration
	StartTime       string          `json:"start_time" db:"start_time"` // HH:MM format
	EndTime         string          `json:"end_time" db:"end_time"`     // HH:MM format
	Duration        int             `json:"duration" db:"duration"`     // Duration in minutes

	// Capacity
	MaxOrders       int             `json:"max_orders" db:"max_orders"`

	// Days of week (bitmask: 1=Mon, 2=Tue, 4=Wed, 8=Thu, 16=Fri, 32=Sat, 64=Sun)
	DaysOfWeek      int             `json:"days_of_week" db:"days_of_week"`

	// Lead time requirements
	MinLeadHours    int             `json:"min_lead_hours" db:"min_lead_hours"` // Min hours before slot
	MaxLeadDays     int             `json:"max_lead_days" db:"max_lead_days"`   // Max days in advance

	// Cutoff time for same-day slots
	CutoffTime      string          `json:"cutoff_time" db:"cutoff_time"` // HH:MM format

	// Pricing
	SlotFee         float64         `json:"slot_fee" db:"slot_fee"` // Extra fee for time slot
	IsPremium       bool            `json:"is_premium" db:"is_premium"`

	// Status
	IsActive        bool            `json:"is_active" db:"is_active"`

	// Metadata
	Metadata        json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// DeliverySlot represents an actual delivery slot instance for a specific date
type DeliverySlot struct {
	ID              uuid.UUID          `json:"id" db:"id"`
	TemplateID      uuid.UUID          `json:"template_id" db:"template_id"`

	// Date and time
	Date            time.Time          `json:"date" db:"date"` // The delivery date
	StartTime       time.Time          `json:"start_time" db:"start_time"`
	EndTime         time.Time          `json:"end_time" db:"end_time"`

	// Zone and carrier
	ZoneID          *uuid.UUID         `json:"zone_id,omitempty" db:"zone_id"`
	CarrierID       *uuid.UUID         `json:"carrier_id,omitempty" db:"carrier_id"`

	// Capacity tracking
	MaxOrders       int                `json:"max_orders" db:"max_orders"`
	BookedOrders    int                `json:"booked_orders" db:"booked_orders"`
	RemainingSlots  int                `json:"remaining_slots" db:"remaining_slots"`

	// Status
	Status          DeliverySlotStatus `json:"status" db:"status"`

	// Pricing
	SlotFee         float64            `json:"slot_fee" db:"slot_fee"`
	IsPremium       bool               `json:"is_premium" db:"is_premium"`

	// Timestamps
	CreatedAt       time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" db:"updated_at"`
}

// DeliverySlotBooking represents a booking for a delivery slot
type DeliverySlotBooking struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	SlotID          uuid.UUID  `json:"slot_id" db:"slot_id"`
	OrderID         uuid.UUID  `json:"order_id" db:"order_id"`
	ShippingOrderID *uuid.UUID `json:"shipping_order_id,omitempty" db:"shipping_order_id"`

	// Booking status
	Status          string     `json:"status" db:"status"` // BOOKED, CONFIRMED, CANCELLED, COMPLETED

	// Customer preferences
	ContactPhone    string     `json:"contact_phone" db:"contact_phone"`
	Instructions    string     `json:"instructions,omitempty" db:"instructions"`

	// Timestamps
	BookedAt        time.Time  `json:"booked_at" db:"booked_at"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty" db:"confirmed_at"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`

	// Cancellation
	CancelledBy     *uuid.UUID `json:"cancelled_by,omitempty" db:"cancelled_by"`
	CancelReason    string     `json:"cancel_reason,omitempty" db:"cancel_reason"`

	// Timestamps
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// NewDeliverySlotTemplate creates a new delivery slot template
func NewDeliverySlotTemplate(name, startTime, endTime string, maxOrders int) *DeliverySlotTemplate {
	now := time.Now()
	return &DeliverySlotTemplate{
		ID:           uuid.New(),
		Name:         name,
		DisplayName:  name,
		StartTime:    startTime,
		EndTime:      endTime,
		MaxOrders:    maxOrders,
		DaysOfWeek:   127, // All days by default (1+2+4+8+16+32+64)
		MinLeadHours: 2,
		MaxLeadDays:  7,
		IsActive:     true,
		Metadata:     json.RawMessage("{}"),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetDays sets which days of the week this template applies to
func (dst *DeliverySlotTemplate) SetDays(mon, tue, wed, thu, fri, sat, sun bool) {
	dst.DaysOfWeek = 0
	if mon { dst.DaysOfWeek |= 1 }
	if tue { dst.DaysOfWeek |= 2 }
	if wed { dst.DaysOfWeek |= 4 }
	if thu { dst.DaysOfWeek |= 8 }
	if fri { dst.DaysOfWeek |= 16 }
	if sat { dst.DaysOfWeek |= 32 }
	if sun { dst.DaysOfWeek |= 64 }
	dst.UpdatedAt = time.Now()
}

// AppliesToDay checks if this template applies to a given day of week
func (dst *DeliverySlotTemplate) AppliesToDay(day time.Weekday) bool {
	var dayBit int
	switch day {
	case time.Monday:
		dayBit = 1
	case time.Tuesday:
		dayBit = 2
	case time.Wednesday:
		dayBit = 4
	case time.Thursday:
		dayBit = 8
	case time.Friday:
		dayBit = 16
	case time.Saturday:
		dayBit = 32
	case time.Sunday:
		dayBit = 64
	}
	return dst.DaysOfWeek&dayBit != 0
}

// Activate activates the template
func (dst *DeliverySlotTemplate) Activate() {
	dst.IsActive = true
	dst.UpdatedAt = time.Now()
}

// Deactivate deactivates the template
func (dst *DeliverySlotTemplate) Deactivate() {
	dst.IsActive = false
	dst.UpdatedAt = time.Now()
}

// NewDeliverySlot creates a new delivery slot from a template for a specific date
func NewDeliverySlot(template *DeliverySlotTemplate, date time.Time) *DeliverySlot {
	now := time.Now()

	// Parse start and end times
	startHour, startMin := parseTimeString(template.StartTime)
	endHour, endMin := parseTimeString(template.EndTime)

	startTime := time.Date(date.Year(), date.Month(), date.Day(), startHour, startMin, 0, 0, date.Location())
	endTime := time.Date(date.Year(), date.Month(), date.Day(), endHour, endMin, 0, 0, date.Location())

	return &DeliverySlot{
		ID:             uuid.New(),
		TemplateID:     template.ID,
		Date:           date,
		StartTime:      startTime,
		EndTime:        endTime,
		ZoneID:         template.ZoneID,
		CarrierID:      template.CarrierID,
		MaxOrders:      template.MaxOrders,
		BookedOrders:   0,
		RemainingSlots: template.MaxOrders,
		Status:         DeliverySlotStatusAvailable,
		SlotFee:        template.SlotFee,
		IsPremium:      template.IsPremium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// parseTimeString parses a HH:MM string into hours and minutes
func parseTimeString(timeStr string) (int, int) {
	var hour, min int
	// Simple parsing, assumes format HH:MM
	if len(timeStr) >= 5 {
		hour = int(timeStr[0]-'0')*10 + int(timeStr[1]-'0')
		min = int(timeStr[3]-'0')*10 + int(timeStr[4]-'0')
	}
	return hour, min
}

// Book books a slot, incrementing the booked count
func (ds *DeliverySlot) Book() bool {
	if ds.Status != DeliverySlotStatusAvailable || ds.RemainingSlots <= 0 {
		return false
	}
	ds.BookedOrders++
	ds.RemainingSlots = ds.MaxOrders - ds.BookedOrders
	if ds.RemainingSlots <= 0 {
		ds.Status = DeliverySlotStatusFull
	}
	ds.UpdatedAt = time.Now()
	return true
}

// Release releases a booking, decrementing the booked count
func (ds *DeliverySlot) Release() {
	if ds.BookedOrders > 0 {
		ds.BookedOrders--
		ds.RemainingSlots = ds.MaxOrders - ds.BookedOrders
		if ds.Status == DeliverySlotStatusFull && ds.RemainingSlots > 0 {
			ds.Status = DeliverySlotStatusAvailable
		}
		ds.UpdatedAt = time.Now()
	}
}

// Close closes the slot for new bookings
func (ds *DeliverySlot) Close() {
	ds.Status = DeliverySlotStatusClosed
	ds.UpdatedAt = time.Now()
}

// IsBookable checks if the slot can accept new bookings
func (ds *DeliverySlot) IsBookable() bool {
	return ds.Status == DeliverySlotStatusAvailable && ds.RemainingSlots > 0
}

// NewDeliverySlotBooking creates a new slot booking
func NewDeliverySlotBooking(slotID, orderID uuid.UUID, contactPhone string) *DeliverySlotBooking {
	now := time.Now()
	return &DeliverySlotBooking{
		ID:           uuid.New(),
		SlotID:       slotID,
		OrderID:      orderID,
		ContactPhone: contactPhone,
		Status:       "BOOKED",
		BookedAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Confirm confirms the booking
func (dsb *DeliverySlotBooking) Confirm() {
	now := time.Now()
	dsb.Status = "CONFIRMED"
	dsb.ConfirmedAt = &now
	dsb.UpdatedAt = now
}

// Cancel cancels the booking
func (dsb *DeliverySlotBooking) Cancel(cancelledBy uuid.UUID, reason string) {
	now := time.Now()
	dsb.Status = "CANCELLED"
	dsb.CancelledBy = &cancelledBy
	dsb.CancelReason = reason
	dsb.CancelledAt = &now
	dsb.UpdatedAt = now
}

// Complete marks the booking as completed
func (dsb *DeliverySlotBooking) Complete() {
	now := time.Now()
	dsb.Status = "COMPLETED"
	dsb.CompletedAt = &now
	dsb.UpdatedAt = now
}
