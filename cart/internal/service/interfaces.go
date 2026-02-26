package service

import (
	"context"

	"microservices/cart/internal/model"
)

// CartService defines the interface for cart business logic.
type CartService interface {
	// GetCart retrieves the user's cart.
	GetCart(ctx context.Context, userID string) (*model.CartResponse, error)

	// AddToCart adds an item to the user's cart.
	AddToCart(ctx context.Context, userID string, req *model.AddItemRequest) error

	// RemoveItem removes an item from the user's cart.
	RemoveItem(ctx context.Context, userID string, skuID string) error

	// UpdateQuantity updates the quantity of an item in the cart.
	UpdateQuantity(ctx context.Context, userID string, skuID string, quantity int) error

	// UpdateSelection updates the selection status of an item.
	UpdateSelection(ctx context.Context, userID string, skuID string, selected bool) error

	// ClearCart removes all items from the user's cart.
	ClearCart(ctx context.Context, userID string) error

	// GetCartSummary retrieves cart summary with total price.
	GetCartSummary(ctx context.Context, userID string, selectedOnly bool) (*model.CartSummary, error)

	// ApplyVoucher validates and applies a voucher to the cart.
	ApplyVoucher(ctx context.Context, userID, voucherCode string) (*model.ApplyVoucherResponse, error)

	// RemoveVoucher clears the applied voucher from the cart.
	RemoveVoucher(ctx context.Context, userID string) error

	// GetCartSummaryWithDiscount returns cart summary including applied discount.
	GetCartSummaryWithDiscount(ctx context.Context, userID string, selectedOnly bool) (*model.CartSummary, error)

	// Checkout confirms the cart and creates a draft order in Order Service.
	Checkout(ctx context.Context, userID string, req *model.CheckoutRequest) (*model.CheckoutResponse, error)
}
