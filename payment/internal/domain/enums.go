package domain

// Provider represents the payment gateway provider
type Provider string

const (
	ProviderStripe  Provider = "STRIPE"
	ProviderMoMo    Provider = "MOMO"
	ProviderCOD     Provider = "COD"
	ProviderVNPay   Provider = "VNPAY"
	ProviderZaloPay Provider = "ZALOPAY"
)

// IsValid checks if the provider is valid
func (p Provider) IsValid() bool {
	switch p {
	case ProviderStripe, ProviderMoMo, ProviderCOD, ProviderVNPay, ProviderZaloPay:
		return true
	}
	return false
}

// String returns the string representation
func (p Provider) String() string {
	return string(p)
}

// Status represents the payment transaction status
type Status string

const (
	StatusPending  Status = "PENDING"
	StatusSuccess  Status = "SUCCESS"
	StatusFailed   Status = "FAILED"
	StatusRefunded Status = "REFUNDED"
)

// IsValid checks if the status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusSuccess, StatusFailed, StatusRefunded:
		return true
	}
	return false
}

// IsFinal returns true if the status is a terminal state
func (s Status) IsFinal() bool {
	switch s {
	case StatusSuccess, StatusFailed, StatusRefunded:
		return true
	}
	return false
}

// String returns the string representation
func (s Status) String() string {
	return string(s)
}

// Currency represents the currency code
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyVND Currency = "VND"
)

// IsValid checks if the currency is valid
func (c Currency) IsValid() bool {
	switch c {
	case CurrencyUSD, CurrencyVND:
		return true
	}
	return false
}

// String returns the string representation
func (c Currency) String() string {
	return string(c)
}
