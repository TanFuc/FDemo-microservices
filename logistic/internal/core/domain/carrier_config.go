package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CarrierStatus represents the status of a carrier configuration
type CarrierStatus string

const (
	CarrierStatusActive   CarrierStatus = "ACTIVE"
	CarrierStatusInactive CarrierStatus = "INACTIVE"
	CarrierStatusTesting  CarrierStatus = "TESTING"
)

// CarrierConfig represents a shipping carrier configuration
type CarrierConfig struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	Provider        ProviderName    `json:"provider" db:"provider"`
	Name            string          `json:"name" db:"name"`
	DisplayName     string          `json:"display_name" db:"display_name"`
	Description     string          `json:"description" db:"description"`
	LogoURL         string          `json:"logo_url" db:"logo_url"`

	// API Configuration
	APIEndpoint     string          `json:"api_endpoint" db:"api_endpoint"`
	APIKey          string          `json:"-" db:"api_key"` // Hidden in JSON
	APISecret       string          `json:"-" db:"api_secret"` // Hidden in JSON
	APIVersion      string          `json:"api_version" db:"api_version"`
	WebhookSecret   string          `json:"-" db:"webhook_secret"` // Hidden in JSON

	// Service Configuration
	SupportsCOD     bool            `json:"supports_cod" db:"supports_cod"`
	SupportsReturn  bool            `json:"supports_return" db:"supports_return"`
	SupportsExpress bool            `json:"supports_express" db:"supports_express"`
	SupportsIntl    bool            `json:"supports_intl" db:"supports_intl"`

	// Limits and constraints
	MaxWeight       float64         `json:"max_weight" db:"max_weight"` // in kg
	MaxDimensions   *Dimensions     `json:"max_dimensions,omitempty"`
	MaxCODAmount    float64         `json:"max_cod_amount" db:"max_cod_amount"`

	// Pricing
	BaseFee         float64         `json:"base_fee" db:"base_fee"`
	CODFeePercent   float64         `json:"cod_fee_percent" db:"cod_fee_percent"`
	InsuranceFee    float64         `json:"insurance_fee" db:"insurance_fee"`

	// Service level configuration
	ServiceTypes    json.RawMessage `json:"service_types" db:"service_types"` // Available service types
	CutoffTime      string          `json:"cutoff_time" db:"cutoff_time"` // Daily cutoff for same-day pickup

	// Status
	Status          CarrierStatus   `json:"status" db:"status"`
	Priority        int             `json:"priority" db:"priority"` // For sorting/selection

	// Rate limiting
	RateLimitRPS    int             `json:"rate_limit_rps" db:"rate_limit_rps"` // Requests per second

	// Metadata
	Metadata        json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// Dimensions represents package dimensions
type Dimensions struct {
	Length float64 `json:"length" db:"length"` // in cm
	Width  float64 `json:"width" db:"width"`   // in cm
	Height float64 `json:"height" db:"height"` // in cm
}

// NewCarrierConfig creates a new carrier configuration
func NewCarrierConfig(provider ProviderName, name, displayName string) *CarrierConfig {
	now := time.Now()
	return &CarrierConfig{
		ID:            uuid.New(),
		Provider:      provider,
		Name:          name,
		DisplayName:   displayName,
		Status:        CarrierStatusTesting,
		Priority:      0,
		SupportsCOD:   false,
		SupportsReturn: false,
		SupportsExpress: false,
		SupportsIntl:  false,
		ServiceTypes:  json.RawMessage("[]"),
		Metadata:      json.RawMessage("{}"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// SetAPICredentials sets the API credentials
func (cc *CarrierConfig) SetAPICredentials(endpoint, key, secret, version string) {
	cc.APIEndpoint = endpoint
	cc.APIKey = key
	cc.APISecret = secret
	cc.APIVersion = version
	cc.UpdatedAt = time.Now()
}

// SetWebhookSecret sets the webhook secret for verification
func (cc *CarrierConfig) SetWebhookSecret(secret string) {
	cc.WebhookSecret = secret
	cc.UpdatedAt = time.Now()
}

// Activate activates the carrier
func (cc *CarrierConfig) Activate() {
	cc.Status = CarrierStatusActive
	cc.UpdatedAt = time.Now()
}

// Deactivate deactivates the carrier
func (cc *CarrierConfig) Deactivate() {
	cc.Status = CarrierStatusInactive
	cc.UpdatedAt = time.Now()
}

// IsActive returns true if the carrier is active
func (cc *CarrierConfig) IsActive() bool {
	return cc.Status == CarrierStatusActive
}

// CanHandleWeight checks if the carrier can handle the given weight
func (cc *CarrierConfig) CanHandleWeight(weight float64) bool {
	if cc.MaxWeight <= 0 {
		return true
	}
	return weight <= cc.MaxWeight
}

// CanHandleCOD checks if the carrier supports COD for the given amount
func (cc *CarrierConfig) CanHandleCOD(amount float64) bool {
	if !cc.SupportsCOD {
		return false
	}
	if cc.MaxCODAmount <= 0 {
		return true
	}
	return amount <= cc.MaxCODAmount
}

// CalculateCODFee calculates the COD fee for a given amount
func (cc *CarrierConfig) CalculateCODFee(amount float64) float64 {
	if cc.CODFeePercent <= 0 {
		return 0
	}
	return amount * cc.CODFeePercent / 100
}

// CarrierServiceType represents a service type offered by a carrier
type CarrierServiceType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	EstDays     int    `json:"est_days"` // Estimated delivery days
	IsExpress   bool   `json:"is_express"`
}
