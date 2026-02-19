package model

import "errors"

var (
	// ErrInsufficientStock is returned when there is not enough stock available
	ErrInsufficientStock = errors.New("insufficient stock available")

	// ErrReservationNotFound is returned when a reservation is not found
	ErrReservationNotFound = errors.New("reservation not found")

	// ErrAlreadyConfirmed is returned when trying to modify a confirmed reservation
	ErrAlreadyConfirmed = errors.New("reservation already confirmed")

	// ErrAlreadyCancelled is returned when trying to modify a cancelled reservation
	ErrAlreadyCancelled = errors.New("reservation already cancelled")

	// ErrInventoryNotFound is returned when an inventory item is not found
	ErrInventoryNotFound = errors.New("inventory item not found")

	// ErrInvalidQuantity is returned when quantity is invalid
	ErrInvalidQuantity = errors.New("quantity must be greater than 0")

	// ErrInvalidSkuID is returned when SKU ID is empty
	ErrInvalidSkuID = errors.New("sku_id is required")

	// ErrInvalidOrderID is returned when order ID is empty
	ErrInvalidOrderID = errors.New("order_id is required")

	// ErrInventoryAlreadyExists is returned when trying to create duplicate inventory
	ErrInventoryAlreadyExists = errors.New("inventory item already exists")
)
