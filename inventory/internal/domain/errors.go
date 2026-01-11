package domain

import "errors"

var (
	ErrInsufficientStock   = errors.New("insufficient stock available")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrAlreadyConfirmed    = errors.New("reservation already confirmed")
	ErrAlreadyCancelled    = errors.New("reservation already cancelled")
	ErrInventoryNotFound   = errors.New("inventory item not found")
)
