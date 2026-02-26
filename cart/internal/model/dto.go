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
	UserID             string     `json:"userId"`
	Items              []CartItem `json:"items"`
	TotalItems         int        `json:"totalItems"`
	TotalPrice         float64    `json:"totalPrice"`         // Deprecated: use OriginalPrice
	OriginalPrice      float64    `json:"originalPrice"`      // before discount
	DiscountAmount     float64    `json:"discountAmount"`     // amount deducted
	FinalPrice         float64    `json:"finalPrice"`         // after discount
	AppliedVoucherCode string     `json:"appliedVoucherCode,omitempty"`
	DiscountType       string     `json:"discountType,omitempty"`
	SelectedOnly       bool       `json:"selectedOnly"`
}

// ToResponse converts Cart entity to CartResponse DTO.
func (c *Cart) ToResponse() *CartResponse {
	return &CartResponse{
		UserID:     c.UserID,
		Items:      c.Items,
		TotalItems: len(c.Items),
	}
}

// ApplyVoucherRequest represents the request to apply a voucher to cart.
type ApplyVoucherRequest struct {
	VoucherCode string `json:"voucherCode" validate:"required"`
}

// ApplyVoucherResponse represents the result of applying a voucher.
type ApplyVoucherResponse struct {
	Success        bool    `json:"success"`
	VoucherCode    string  `json:"voucherCode"`
	OriginalPrice  float64 `json:"originalPrice"`
	DiscountAmount float64 `json:"discountAmount"`
	FinalPrice     float64 `json:"finalPrice"`
	DiscountType   string  `json:"discountType"`
	Message        string  `json:"message,omitempty"`
	ErrorCode      string  `json:"errorCode,omitempty"`
}

// CheckoutRequest represents the request to checkout cart and create draft order.
type CheckoutRequest struct {
	ShippingAddress CheckoutShippingAddress `json:"shippingAddress" validate:"required"`
	PaymentMethod   string                  `json:"paymentMethod" validate:"required,oneof=MOMO COD STRIPE VNPAY"`
	CustomerNote    string                  `json:"customerNote"`
}

// CheckoutShippingAddress represents shipping address for checkout.
type CheckoutShippingAddress struct {
	FullName   string `json:"fullName" validate:"required"`
	Phone      string `json:"phone" validate:"required"`
	Address    string `json:"address" validate:"required"`
	Ward       string `json:"ward"`
	District   string `json:"district" validate:"required"`
	City       string `json:"city" validate:"required"`
	Country    string `json:"country" validate:"required,len=2"` // e.g. VN, US
	PostalCode string `json:"postalCode"`
}

// CheckoutResponse represents the response after checkout (draft order created).
type CheckoutResponse struct {
	DraftOrderID   string                 `json:"draftOrderId"`
	OrderNumber    string                 `json:"orderNumber"`
	OriginalPrice  float64                `json:"originalPrice"`
	DiscountAmount float64                `json:"discountAmount"`
	ShippingFee    float64                `json:"shippingFee"`
	FinalAmount    float64                `json:"finalAmount"`
	VoucherCode    string                 `json:"voucherCode,omitempty"`
	Status         string                 `json:"status"` // "DRAFT"
	Items          []CheckoutItemResponse `json:"items"`
	CreatedAt      string                 `json:"createdAt"`
}

// CheckoutItemResponse represents an item in the checkout response.
type CheckoutItemResponse struct {
	SkuID     string  `json:"skuId"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	Thumbnail string  `json:"thumbnail"`
	SubTotal  float64 `json:"subTotal"`
}
