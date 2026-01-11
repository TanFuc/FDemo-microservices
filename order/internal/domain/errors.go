package domain

import "errors"

var (
	// Order errors
	ErrOrderNotFound                = errors.New("order not found")
	ErrOrderCannotBeCancelled       = errors.New("order cannot be cancelled in current status")
	ErrInvalidOrderStatusTransition = errors.New("invalid order status transition")
	ErrEmptyOrderItems              = errors.New("order must have at least one item")
	ErrInvalidQuantity              = errors.New("quantity must be greater than zero")
	ErrInvalidPrice                 = errors.New("price must be greater than zero")
	ErrOrderAlreadyPaid             = errors.New("order is already paid")
	ErrOrderNotPaid                 = errors.New("order is not paid yet")

	// Refund errors
	ErrRefundNotFound        = errors.New("refund not found")
	ErrInvalidRefundStatus   = errors.New("invalid refund status")
	ErrInvalidRefundQuantity = errors.New("invalid refund quantity")
	ErrRefundExceedsAmount   = errors.New("refund amount exceeds order amount")
	ErrRefundAlreadyProcessed = errors.New("refund already processed")

	// Return errors
	ErrReturnNotFound       = errors.New("return request not found")
	ErrInvalidReturnStatus  = errors.New("invalid return status")
	ErrReturnWindowExpired  = errors.New("return window has expired")
	ErrItemNotReturnable    = errors.New("item is not returnable")

	// Stock errors
	ErrOutOfStock             = errors.New("insufficient stock available")
	ErrStockReservationFailed = errors.New("failed to reserve stock")
	ErrStockReleaseFailed     = errors.New("failed to release stock")

	// Validation errors
	ErrInvalidUserID        = errors.New("invalid user ID")
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
	ErrInvalidAddress       = errors.New("invalid shipping address")
	ErrInvalidOrderID       = errors.New("invalid order ID")

	// Infrastructure errors
	ErrDatabaseConnection          = errors.New("database connection failed")
	ErrInventoryServiceUnavailable = errors.New("inventory service unavailable")
	ErrEventPublishFailed          = errors.New("failed to publish event")
	ErrPaymentServiceUnavailable   = errors.New("payment service unavailable")
)
