package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PaymentMethodType represents the type of payment method
type PaymentMethodType string

const (
	PaymentMethodTypeCard         PaymentMethodType = "CARD"
	PaymentMethodTypeBankAccount  PaymentMethodType = "BANK_ACCOUNT"
	PaymentMethodTypeEWallet      PaymentMethodType = "E_WALLET"
)

// PaymentMethodStatus represents the status of a payment method
type PaymentMethodStatus string

const (
	PaymentMethodStatusActive   PaymentMethodStatus = "ACTIVE"
	PaymentMethodStatusInactive PaymentMethodStatus = "INACTIVE"
	PaymentMethodStatusExpired  PaymentMethodStatus = "EXPIRED"
)

// PaymentMethod represents a saved payment method for a user
type PaymentMethod struct {
	ID              uuid.UUID           `json:"id"`
	UserID          uuid.UUID           `json:"user_id"`
	Type            PaymentMethodType   `json:"type"`
	Provider        Provider            `json:"provider"`
	ProviderTokenID string              `json:"provider_token_id"`

	// Card details (masked for security)
	CardBrand       string              `json:"card_brand,omitempty"`
	CardLast4       string              `json:"card_last4,omitempty"`
	CardExpMonth    int                 `json:"card_exp_month,omitempty"`
	CardExpYear     int                 `json:"card_exp_year,omitempty"`
	CardholderName  string              `json:"cardholder_name,omitempty"`

	// Bank account details (masked for security)
	BankName        string              `json:"bank_name,omitempty"`
	BankAccountLast4 string             `json:"bank_account_last4,omitempty"`
	AccountHolder   string              `json:"account_holder,omitempty"`

	// E-wallet details
	WalletType      string              `json:"wallet_type,omitempty"`
	WalletID        string              `json:"wallet_id,omitempty"`

	// Preferences
	IsDefault       bool                `json:"is_default"`
	Label           string              `json:"label,omitempty"`

	// Status
	Status          PaymentMethodStatus `json:"status"`
	VerifiedAt      *time.Time          `json:"verified_at,omitempty"`

	// Billing address
	BillingAddress  json.RawMessage     `json:"billing_address,omitempty"`

	// Metadata
	Metadata        json.RawMessage     `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	DeletedAt       *time.Time          `json:"deleted_at,omitempty"`
}

// NewPaymentMethod creates a new payment method
func NewPaymentMethod(userID uuid.UUID, methodType PaymentMethodType, provider Provider) *PaymentMethod {
	now := time.Now()
	return &PaymentMethod{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      methodType,
		Provider:  provider,
		Status:    PaymentMethodStatusActive,
		IsDefault: false,
		Metadata:  json.RawMessage("{}"),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetCardDetails sets card details for a card payment method
func (pm *PaymentMethod) SetCardDetails(brand, last4, cardholderName string, expMonth, expYear int) {
	pm.CardBrand = brand
	pm.CardLast4 = last4
	pm.CardholderName = cardholderName
	pm.CardExpMonth = expMonth
	pm.CardExpYear = expYear
	pm.UpdatedAt = time.Now()
}

// SetBankDetails sets bank account details
func (pm *PaymentMethod) SetBankDetails(bankName, accountLast4, accountHolder string) {
	pm.BankName = bankName
	pm.BankAccountLast4 = accountLast4
	pm.AccountHolder = accountHolder
	pm.UpdatedAt = time.Now()
}

// SetEWalletDetails sets e-wallet details
func (pm *PaymentMethod) SetEWalletDetails(walletType, walletID string) {
	pm.WalletType = walletType
	pm.WalletID = walletID
	pm.UpdatedAt = time.Now()
}

// SetAsDefault marks this payment method as default
func (pm *PaymentMethod) SetAsDefault() {
	pm.IsDefault = true
	pm.UpdatedAt = time.Now()
}

// RemoveDefault removes the default status
func (pm *PaymentMethod) RemoveDefault() {
	pm.IsDefault = false
	pm.UpdatedAt = time.Now()
}

// Verify marks the payment method as verified
func (pm *PaymentMethod) Verify() {
	now := time.Now()
	pm.VerifiedAt = &now
	pm.UpdatedAt = now
}

// Deactivate deactivates the payment method
func (pm *PaymentMethod) Deactivate() {
	pm.Status = PaymentMethodStatusInactive
	pm.UpdatedAt = time.Now()
}

// MarkExpired marks the payment method as expired
func (pm *PaymentMethod) MarkExpired() {
	pm.Status = PaymentMethodStatusExpired
	pm.UpdatedAt = time.Now()
}

// SoftDelete performs a soft delete
func (pm *PaymentMethod) SoftDelete() {
	now := time.Now()
	pm.DeletedAt = &now
	pm.Status = PaymentMethodStatusInactive
	pm.UpdatedAt = now
}

// IsActive returns true if the payment method is active and not deleted
func (pm *PaymentMethod) IsActive() bool {
	return pm.Status == PaymentMethodStatusActive && pm.DeletedAt == nil
}

// IsExpiredCard checks if the card has expired
func (pm *PaymentMethod) IsExpiredCard() bool {
	if pm.Type != PaymentMethodTypeCard {
		return false
	}
	now := time.Now()
	// Card expires at the end of the expiration month
	expDate := time.Date(pm.CardExpYear, time.Month(pm.CardExpMonth+1), 1, 0, 0, 0, 0, time.UTC)
	return now.After(expDate)
}

// GetDisplayName returns a human-readable display name for the payment method
func (pm *PaymentMethod) GetDisplayName() string {
	switch pm.Type {
	case PaymentMethodTypeCard:
		if pm.CardBrand != "" && pm.CardLast4 != "" {
			return pm.CardBrand + " ending in " + pm.CardLast4
		}
	case PaymentMethodTypeBankAccount:
		if pm.BankName != "" && pm.BankAccountLast4 != "" {
			return pm.BankName + " ending in " + pm.BankAccountLast4
		}
	case PaymentMethodTypeEWallet:
		if pm.WalletType != "" {
			return pm.WalletType + " Wallet"
		}
	}
	if pm.Label != "" {
		return pm.Label
	}
	return string(pm.Type)
}
