package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ZoneType represents the type of shipping zone
type ZoneType string

const (
	ZoneTypeCountry  ZoneType = "COUNTRY"
	ZoneTypeProvince ZoneType = "PROVINCE"
	ZoneTypeDistrict ZoneType = "DISTRICT"
	ZoneTypePostal   ZoneType = "POSTAL"
	ZoneTypeCustom   ZoneType = "CUSTOM"
)

// ShippingZone represents a geographic shipping zone
type ShippingZone struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	Name         string          `json:"name" db:"name"`
	DisplayName  string          `json:"display_name" db:"display_name"`
	Description  string          `json:"description" db:"description"`
	Type         ZoneType        `json:"type" db:"type"`

	// Geographic identification
	CountryCode  string          `json:"country_code" db:"country_code"`
	ProvinceCode string          `json:"province_code,omitempty" db:"province_code"`
	DistrictCode string          `json:"district_code,omitempty" db:"district_code"`
	PostalCodes  json.RawMessage `json:"postal_codes,omitempty" db:"postal_codes"` // Array of postal codes

	// Hierarchy
	ParentZoneID *uuid.UUID      `json:"parent_zone_id,omitempty" db:"parent_zone_id"`

	// Configuration
	IsActive     bool            `json:"is_active" db:"is_active"`
	Priority     int             `json:"priority" db:"priority"` // For matching priority

	// Supported carriers in this zone
	CarrierIDs   json.RawMessage `json:"carrier_ids" db:"carrier_ids"` // Array of carrier UUIDs

	// Zone-specific settings
	Settings     json.RawMessage `json:"settings" db:"settings"`

	// Metadata
	Metadata     json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewShippingZone creates a new shipping zone
func NewShippingZone(name, displayName string, zoneType ZoneType, countryCode string) *ShippingZone {
	now := time.Now()
	return &ShippingZone{
		ID:          uuid.New(),
		Name:        name,
		DisplayName: displayName,
		Type:        zoneType,
		CountryCode: countryCode,
		IsActive:    true,
		Priority:    0,
		PostalCodes: json.RawMessage("[]"),
		CarrierIDs:  json.RawMessage("[]"),
		Settings:    json.RawMessage("{}"),
		Metadata:    json.RawMessage("{}"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetProvinceCode sets the province code
func (sz *ShippingZone) SetProvinceCode(code string) {
	sz.ProvinceCode = code
	sz.UpdatedAt = time.Now()
}

// SetDistrictCode sets the district code
func (sz *ShippingZone) SetDistrictCode(code string) {
	sz.DistrictCode = code
	sz.UpdatedAt = time.Now()
}

// SetPostalCodes sets the postal codes for the zone
func (sz *ShippingZone) SetPostalCodes(codes []string) error {
	bytes, err := json.Marshal(codes)
	if err != nil {
		return err
	}
	sz.PostalCodes = bytes
	sz.UpdatedAt = time.Now()
	return nil
}

// GetPostalCodes returns the postal codes
func (sz *ShippingZone) GetPostalCodes() ([]string, error) {
	var codes []string
	if err := json.Unmarshal(sz.PostalCodes, &codes); err != nil {
		return nil, err
	}
	return codes, nil
}

// SetCarrierIDs sets the supported carrier IDs
func (sz *ShippingZone) SetCarrierIDs(ids []uuid.UUID) error {
	bytes, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	sz.CarrierIDs = bytes
	sz.UpdatedAt = time.Now()
	return nil
}

// GetCarrierIDs returns the carrier IDs
func (sz *ShippingZone) GetCarrierIDs() ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := json.Unmarshal(sz.CarrierIDs, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// Activate activates the zone
func (sz *ShippingZone) Activate() {
	sz.IsActive = true
	sz.UpdatedAt = time.Now()
}

// Deactivate deactivates the zone
func (sz *ShippingZone) Deactivate() {
	sz.IsActive = false
	sz.UpdatedAt = time.Now()
}

// SoftDelete performs a soft delete
func (sz *ShippingZone) SoftDelete() {
	now := time.Now()
	sz.DeletedAt = &now
	sz.IsActive = false
	sz.UpdatedAt = now
}

// ShippingZoneMatch represents a zone match result
type ShippingZoneMatch struct {
	Zone     *ShippingZone
	Priority int
	MatchType string // EXACT, POSTAL, DISTRICT, PROVINCE, COUNTRY
}
