package model

// AddItemRequest represents the request payload for adding an item to cart.
type AddItemRequest struct {
	SkuID     string  `json:"skuId" validate:"required"`
	Name      string  `json:"name" validate:"required"`
	Price     float64 `json:"price" validate:"required,gt=0"`
	Quantity  int     `json:"quantity" validate:"required,gt=0"`
	Thumbnail string  `json:"thumbnail"`
	Selected  bool    `json:"selected"`
}

// UpdateQuantityRequest represents the request payload for updating item quantity.
type UpdateQuantityRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}

// UpdateSelectionRequest represents the request payload for updating item selection.
type UpdateSelectionRequest struct {
	Selected bool `json:"selected"`
}

// CartResponse represents the response for cart operations.
type CartResponse struct {
	UserID     string     `json:"userId"`
	Items      []CartItem `json:"items"`
	TotalItems int        `json:"totalItems"`
}

// CartSummary represents a summary of the cart for checkout.
type CartSummary struct {
	UserID       string     `json:"userId"`
	Items        []CartItem `json:"items"`
	TotalItems   int        `json:"totalItems"`
	TotalPrice   float64    `json:"totalPrice"`
	SelectedOnly bool       `json:"selectedOnly"`
}

// ToResponse converts Cart entity to CartResponse DTO.
func (c *Cart) ToResponse() *CartResponse {
	return &CartResponse{
		UserID:     c.UserID,
		Items:      c.Items,
		TotalItems: len(c.Items),
	}
}
