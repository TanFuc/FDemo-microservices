package domain

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

// CartResponse represents the response for cart operations.
type CartResponse struct {
	UserID     string     `json:"userId"`
	Items      []CartItem `json:"items"`
	TotalItems int        `json:"totalItems"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
