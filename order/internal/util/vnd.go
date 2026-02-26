package util

import (
	"strconv"

	"github.com/shopspring/decimal"
)

// RoundVND rounds an amount to whole VND (no decimals).
func RoundVND(amount decimal.Decimal) decimal.Decimal {
	return amount.Round(0)
}

// FormatVNDString returns a display string like "300.000 VND".
func FormatVNDString(amount decimal.Decimal) string {
	intVal := RoundVND(amount).IntPart()
	return formatWithDots(intVal) + " VND"
}

// IsWholeNumber checks if a decimal has no fractional part (valid for VND).
func IsWholeNumber(amount decimal.Decimal) bool {
	return amount.Mod(decimal.NewFromInt(1)).IsZero()
}

// formatWithDots formats a number with dots as thousand separators (Vietnamese style).
func formatWithDots(n int64) string {
	negative := n < 0
	if negative {
		n = -n
	}

	s := strconv.FormatInt(n, 10)
	result := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(ch)
	}

	if negative {
		return "-" + result
	}
	return result
}
