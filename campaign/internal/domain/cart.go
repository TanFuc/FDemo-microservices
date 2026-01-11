package domain

import "github.com/shopspring/decimal"

// CartItem represents an item in the shopping cart for discount calculation
type CartItem struct {
	SKU      string
	Name     string
	Price    decimal.Decimal
	Quantity int
	Category string
}

func (ci *CartItem) TotalPrice() decimal.Decimal {
	return ci.Price.Mul(decimal.NewFromInt(int64(ci.Quantity)))
}

// CalculateCartResult holds the result of cart calculation with voucher applied
type CalculateCartResult struct {
	OriginalTotal  decimal.Decimal
	DiscountAmount decimal.Decimal
	FinalPrice     decimal.Decimal
	VoucherApplied bool
	VoucherCode    string
	ErrorMessage   string
}

func NewCalculateCartResult(originalTotal, discountAmount decimal.Decimal, voucherCode string) *CalculateCartResult {
	return &CalculateCartResult{
		OriginalTotal:  originalTotal,
		DiscountAmount: discountAmount,
		FinalPrice:     originalTotal.Sub(discountAmount),
		VoucherApplied: true,
		VoucherCode:    voucherCode,
	}
}

func NewCalculateCartResultNoDiscount(originalTotal decimal.Decimal) *CalculateCartResult {
	return &CalculateCartResult{
		OriginalTotal:  originalTotal,
		DiscountAmount: decimal.Zero,
		FinalPrice:     originalTotal,
		VoucherApplied: false,
	}
}

func NewCalculateCartResultError(originalTotal decimal.Decimal, errMsg string) *CalculateCartResult {
	return &CalculateCartResult{
		OriginalTotal:  originalTotal,
		DiscountAmount: decimal.Zero,
		FinalPrice:     originalTotal,
		VoucherApplied: false,
		ErrorMessage:   errMsg,
	}
}
