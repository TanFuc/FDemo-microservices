package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RateType represents the type of shipping rate calculation
type RateType string

const (
	RateTypeFlat      RateType = "FLAT"       // Fixed rate
	RateTypeWeight    RateType = "WEIGHT"     // Based on weight
	RateTypeDistance  RateType = "DISTANCE"   // Based on distance
	RateTypeTiered    RateType = "TIERED"     // Tiered pricing
	RateTypeProvider  RateType = "PROVIDER"   // Use carrier's rate API
)

// ShippingRate represents a shipping rate configuration
type ShippingRate struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	DisplayName     string          `json:"display_name" db:"display_name"`
	Description     string          `json:"description" db:"description"`

	// Zone and carrier association
	OriginZoneID    uuid.UUID       `json:"origin_zone_id" db:"origin_zone_id"`
	DestZoneID      uuid.UUID       `json:"dest_zone_id" db:"dest_zone_id"`
	CarrierID       *uuid.UUID      `json:"carrier_id,omitempty" db:"carrier_id"` // If null, applies to all

	// Service type
	ServiceType     string          `json:"service_type" db:"service_type"` // STANDARD, EXPRESS, ECONOMY

	// Rate calculation
	RateType        RateType        `json:"rate_type" db:"rate_type"`
	BaseFee         float64         `json:"base_fee" db:"base_fee"`
	PerKgFee        float64         `json:"per_kg_fee" db:"per_kg_fee"`
	PerKmFee        float64         `json:"per_km_fee" db:"per_km_fee"`

	// Weight-based tiers (for TIERED rate type)
	WeightTiers     json.RawMessage `json:"weight_tiers,omitempty" db:"weight_tiers"`

	// Minimum and maximum
	MinFee          float64         `json:"min_fee" db:"min_fee"`
	MaxFee          float64         `json:"max_fee" db:"max_fee"` // 0 = no max
	MinWeight       float64         `json:"min_weight" db:"min_weight"` // in kg
	MaxWeight       float64         `json:"max_weight" db:"max_weight"` // in kg, 0 = no max

	// Estimated delivery
	EstDeliveryMin  int             `json:"est_delivery_min" db:"est_delivery_min"` // Min days
	EstDeliveryMax  int             `json:"est_delivery_max" db:"est_delivery_max"` // Max days

	// Surcharges
	Surcharges      json.RawMessage `json:"surcharges,omitempty" db:"surcharges"`

	// Free shipping threshold
	FreeShippingMin float64         `json:"free_shipping_min" db:"free_shipping_min"` // 0 = no free shipping

	// Validity
	IsActive        bool            `json:"is_active" db:"is_active"`
	ValidFrom       *time.Time      `json:"valid_from,omitempty" db:"valid_from"`
	ValidUntil      *time.Time      `json:"valid_until,omitempty" db:"valid_until"`

	// Priority (higher = preferred)
	Priority        int             `json:"priority" db:"priority"`

	// Currency
	Currency        string          `json:"currency" db:"currency"`

	// Metadata
	Metadata        json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}

// WeightTier represents a weight-based pricing tier
type WeightTier struct {
	MinWeight float64 `json:"min_weight"` // in kg
	MaxWeight float64 `json:"max_weight"` // in kg
	Fee       float64 `json:"fee"`        // Fixed fee for this tier
	PerKgFee  float64 `json:"per_kg_fee"` // Per kg fee above min
}

// Surcharge represents an additional surcharge
type Surcharge struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Type        string  `json:"type"` // FLAT, PERCENT
	Value       float64 `json:"value"`
	Condition   string  `json:"condition,omitempty"` // e.g., "weight > 30", "is_remote"
	IsOptional  bool    `json:"is_optional"`
}

// NewShippingRate creates a new shipping rate
func NewShippingRate(name string, originZoneID, destZoneID uuid.UUID, rateType RateType) *ShippingRate {
	now := time.Now()
	return &ShippingRate{
		ID:             uuid.New(),
		Name:           name,
		OriginZoneID:   originZoneID,
		DestZoneID:     destZoneID,
		RateType:       rateType,
		ServiceType:    "STANDARD",
		BaseFee:        0,
		MinFee:         0,
		MaxFee:         0,
		IsActive:       true,
		Priority:       0,
		Currency:       "VND",
		WeightTiers:    json.RawMessage("[]"),
		Surcharges:     json.RawMessage("[]"),
		Metadata:       json.RawMessage("{}"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// SetWeightTiers sets the weight-based pricing tiers
func (sr *ShippingRate) SetWeightTiers(tiers []WeightTier) error {
	bytes, err := json.Marshal(tiers)
	if err != nil {
		return err
	}
	sr.WeightTiers = bytes
	sr.UpdatedAt = time.Now()
	return nil
}

// GetWeightTiers returns the weight tiers
func (sr *ShippingRate) GetWeightTiers() ([]WeightTier, error) {
	var tiers []WeightTier
	if err := json.Unmarshal(sr.WeightTiers, &tiers); err != nil {
		return nil, err
	}
	return tiers, nil
}

// SetSurcharges sets the surcharges
func (sr *ShippingRate) SetSurcharges(surcharges []Surcharge) error {
	bytes, err := json.Marshal(surcharges)
	if err != nil {
		return err
	}
	sr.Surcharges = bytes
	sr.UpdatedAt = time.Now()
	return nil
}

// GetSurcharges returns the surcharges
func (sr *ShippingRate) GetSurcharges() ([]Surcharge, error) {
	var surcharges []Surcharge
	if err := json.Unmarshal(sr.Surcharges, &surcharges); err != nil {
		return nil, err
	}
	return surcharges, nil
}

// SetEstimatedDelivery sets the estimated delivery window
func (sr *ShippingRate) SetEstimatedDelivery(min, max int) {
	sr.EstDeliveryMin = min
	sr.EstDeliveryMax = max
	sr.UpdatedAt = time.Now()
}

// SetValidity sets the validity period
func (sr *ShippingRate) SetValidity(from, until *time.Time) {
	sr.ValidFrom = from
	sr.ValidUntil = until
	sr.UpdatedAt = time.Now()
}

// IsValid checks if the rate is currently valid
func (sr *ShippingRate) IsValid() bool {
	if !sr.IsActive || sr.DeletedAt != nil {
		return false
	}
	now := time.Now()
	if sr.ValidFrom != nil && now.Before(*sr.ValidFrom) {
		return false
	}
	if sr.ValidUntil != nil && now.After(*sr.ValidUntil) {
		return false
	}
	return true
}

// CalculateRate calculates the shipping fee for given parameters
func (sr *ShippingRate) CalculateRate(weight float64, distance float64, orderTotal float64) float64 {
	// Check for free shipping
	if sr.FreeShippingMin > 0 && orderTotal >= sr.FreeShippingMin {
		return 0
	}

	var fee float64

	switch sr.RateType {
	case RateTypeFlat:
		fee = sr.BaseFee

	case RateTypeWeight:
		fee = sr.BaseFee + (weight * sr.PerKgFee)

	case RateTypeDistance:
		fee = sr.BaseFee + (distance * sr.PerKmFee)

	case RateTypeTiered:
		fee = sr.calculateTieredRate(weight)

	default:
		fee = sr.BaseFee
	}

	// Apply min/max constraints
	if sr.MinFee > 0 && fee < sr.MinFee {
		fee = sr.MinFee
	}
	if sr.MaxFee > 0 && fee > sr.MaxFee {
		fee = sr.MaxFee
	}

	return fee
}

// calculateTieredRate calculates rate based on weight tiers
func (sr *ShippingRate) calculateTieredRate(weight float64) float64 {
	tiers, err := sr.GetWeightTiers()
	if err != nil || len(tiers) == 0 {
		return sr.BaseFee
	}

	for _, tier := range tiers {
		if weight >= tier.MinWeight && (tier.MaxWeight == 0 || weight <= tier.MaxWeight) {
			if tier.PerKgFee > 0 {
				excessWeight := weight - tier.MinWeight
				return tier.Fee + (excessWeight * tier.PerKgFee)
			}
			return tier.Fee
		}
	}

	// If no tier matches, use the last tier
	lastTier := tiers[len(tiers)-1]
	return lastTier.Fee
}

// CanHandleWeight checks if this rate can handle the given weight
func (sr *ShippingRate) CanHandleWeight(weight float64) bool {
	if weight < sr.MinWeight {
		return false
	}
	if sr.MaxWeight > 0 && weight > sr.MaxWeight {
		return false
	}
	return true
}

// Activate activates the rate
func (sr *ShippingRate) Activate() {
	sr.IsActive = true
	sr.UpdatedAt = time.Now()
}

// Deactivate deactivates the rate
func (sr *ShippingRate) Deactivate() {
	sr.IsActive = false
	sr.UpdatedAt = time.Now()
}

// SoftDelete performs a soft delete
func (sr *ShippingRate) SoftDelete() {
	now := time.Now()
	sr.DeletedAt = &now
	sr.IsActive = false
	sr.UpdatedAt = now
}

// RateQuote represents a shipping rate quote
type RateQuote struct {
	RateID          uuid.UUID    `json:"rate_id"`
	CarrierID       *uuid.UUID   `json:"carrier_id,omitempty"`
	CarrierName     string       `json:"carrier_name"`
	ServiceType     string       `json:"service_type"`
	BaseFee         float64      `json:"base_fee"`
	Surcharges      []Surcharge  `json:"surcharges,omitempty"`
	TotalFee        float64      `json:"total_fee"`
	Currency        string       `json:"currency"`
	EstDeliveryMin  int          `json:"est_delivery_min"`
	EstDeliveryMax  int          `json:"est_delivery_max"`
	IsFreeShipping  bool         `json:"is_free_shipping"`
}
