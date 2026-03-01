package dto

import (
	"github.com/google/uuid"

	"microservices/logistic/internal/core/domain"
)

// CalculateFeeRequest represents the request body for fee calculation
type CalculateFeeRequest struct {
	Provider       string `json:"provider" binding:"required"`
	FromDistrictID int    `json:"from_district_id" binding:"required"`
	ToDistrictID   int    `json:"to_district_id" binding:"required"`
	WeightGram     int    `json:"weight_gram" binding:"required,min=1"`
	InsuranceValue int    `json:"insurance_value"`
}

// CreateShipmentRequest represents the request body for creating a shipment
type CreateShipmentRequest struct {
	InternalOrderID string             `json:"internal_order_id" binding:"required,uuid"`
	Provider        string             `json:"provider" binding:"required"`
	Sender          ContactInfo        `json:"sender" binding:"required"`
	Receiver        ContactInfo        `json:"receiver" binding:"required"`
	Parcels         []Parcel           `json:"parcels" binding:"required,min=1,dive"`
	IsCOD           bool               `json:"is_cod"`
	CODAmount       float64            `json:"cod_amount"`
	Note            string             `json:"note"`
}

// ContactInfo represents contact details in request
type ContactInfo struct {
	Name       string `json:"name" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Address    string `json:"address" binding:"required"`
	WardCode   string `json:"ward_code" binding:"required"`
	DistrictID int    `json:"district_id" binding:"required"`
	ProvinceID int    `json:"province_id" binding:"required"`
}

// Parcel represents a parcel in request
type Parcel struct {
	Name       string  `json:"name" binding:"required"`
	Quantity   int     `json:"quantity" binding:"required,min=1"`
	WeightGram int     `json:"weight_gram" binding:"required,min=1"`
	Value      float64 `json:"value"`
}

// ToDomain converts ContactInfo DTO to domain model
func (c *ContactInfo) ToDomain() domain.ContactInfo {
	return domain.ContactInfo{
		Name:       c.Name,
		Phone:      c.Phone,
		Address:    c.Address,
		WardCode:   c.WardCode,
		DistrictID: c.DistrictID,
		ProvinceID: c.ProvinceID,
	}
}

// ToDomain converts Parcel DTO to domain model
func (p *Parcel) ToDomain() domain.Parcel {
	return domain.Parcel{
		Name:       p.Name,
		Quantity:   p.Quantity,
		WeightGram: p.WeightGram,
		Value:      p.Value,
	}
}

// ToDomainParcels converts slice of Parcel DTOs to domain models
func ToDomainParcels(parcels []Parcel) []domain.Parcel {
	result := make([]domain.Parcel, len(parcels))
	for i, p := range parcels {
		result[i] = p.ToDomain()
	}
	return result
}

// ParseInternalOrderID parses the internal order ID string to UUID
func (r *CreateShipmentRequest) ParseInternalOrderID() (uuid.UUID, error) {
	return uuid.Parse(r.InternalOrderID)
}

// CompareFeesRequest represents the request body for comparing fees
type CompareFeesRequest struct {
	FromDistrictID int      `json:"from_district_id" binding:"required"`
	ToDistrictID   int      `json:"to_district_id" binding:"required"`
	WeightGram     int      `json:"weight_gram" binding:"required,min=1"`
	InsuranceValue int      `json:"insurance_value"`
	Providers      []string `json:"providers,omitempty"`
}
