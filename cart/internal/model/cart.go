package model

import "time"

// CartItem represents a single item in the shopping cart.
// Note: Price is stored for display purposes only.
// Final price must be re-validated with Catalog Service during Checkout/Order creation.
type CartItem struct {
	SkuID     string  `bson:"skuId" json:"skuId"`
	Name      string  `bson:"name" json:"name"`
	Price     float64 `bson:"price" json:"price"`
	Quantity  int     `bson:"quantity" json:"quantity"`
	Thumbnail string  `bson:"thumbnail" json:"thumbnail"`
	Selected  bool    `bson:"selected" json:"selected"`
	AddedAt   int64   `bson:"addedAt" json:"addedAt"`
}

// Cart represents a user's shopping cart.
type Cart struct {
	UserID    string     `bson:"_id" json:"userId"`
	Items     []CartItem `bson:"items" json:"items"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`

	// Applied voucher state (optional)
	AppliedVoucherCode   string  `bson:"appliedVoucherCode,omitempty" json:"appliedVoucherCode,omitempty"`
	DiscountAmount       float64 `bson:"discountAmount,omitempty" json:"discountAmount,omitempty"`
	DiscountType         string  `bson:"discountType,omitempty" json:"discountType,omitempty"`         // "PERCENTAGE" or "FIXED_AMOUNT"
	VoucherDiscountValue float64 `bson:"voucherDiscountValue,omitempty" json:"voucherDiscountValue,omitempty"` // actual deducted amount
	VoucherID            string  `bson:"voucherId,omitempty" json:"voucherId,omitempty"`
	CampaignID           string  `bson:"campaignId,omitempty" json:"campaignId,omitempty"`
}

// MaxCartItems is the maximum number of SKUs allowed in a cart to prevent Redis memory abuse.
const MaxCartItems = 100

// CartTTLDays is the TTL for cart data in Redis (30 days).
const CartTTLDays = 30
