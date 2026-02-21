package model

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AddressType string

const (
	AddressTypeHome   AddressType = "HOME"
	AddressTypeOffice AddressType = "OFFICE"
	AddressTypeOther  AddressType = "OTHER"
)

type Address struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID               string             `bson:"userId" json:"userId"`
	ContactName          string             `bson:"contactName" json:"contactName"`
	Phone                string             `bson:"phone" json:"phone"`
	Email                string             `bson:"email,omitempty" json:"email,omitempty"`
	CountryCode          string             `bson:"countryCode" json:"countryCode"`
	ProvinceCode         string             `bson:"provinceCode" json:"provinceCode"`
	ProvinceName         string             `bson:"provinceName,omitempty" json:"provinceName,omitempty"`
	DistrictCode         string             `bson:"districtCode" json:"districtCode"`
	DistrictName         string             `bson:"districtName,omitempty" json:"districtName,omitempty"`
	WardCode             string             `bson:"wardCode" json:"wardCode"`
	WardName             string             `bson:"wardName,omitempty" json:"wardName,omitempty"`
	StreetAddress        string             `bson:"streetAddress" json:"streetAddress"`
	Apartment            string             `bson:"apartment,omitempty" json:"apartment,omitempty"`
	PostalCode           string             `bson:"postalCode,omitempty" json:"postalCode,omitempty"`
	FullAddress          string             `bson:"fullAddress,omitempty" json:"fullAddress,omitempty"`
	Coordinates          *Coordinates       `bson:"coordinates,omitempty" json:"coordinates,omitempty"`
	Type                 AddressType        `bson:"type" json:"type"`
	Label                string             `bson:"label,omitempty" json:"label,omitempty"`
	IsDefault            bool               `bson:"isDefault" json:"isDefault"`
	IsDefaultBilling     bool               `bson:"isDefaultBilling" json:"isDefaultBilling"`
	IsDefaultPickup      bool               `bson:"isDefaultPickup" json:"isDefaultPickup"`
	DeliveryInstructions string             `bson:"deliveryInstructions,omitempty" json:"deliveryInstructions,omitempty"`
	IsVerified           bool               `bson:"isVerified" json:"isVerified"`
	VerifiedAt           *time.Time         `bson:"verifiedAt,omitempty" json:"verifiedAt,omitempty"`
	IsDeleted            bool               `bson:"isDeleted" json:"isDeleted"`
	DeletedAt            *time.Time         `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	CreatedAt            time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type Coordinates struct {
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
}

func NewAddress(userID string) *Address {
	now := time.Now()
	return &Address{
		UserID:    userID,
		Type:      AddressTypeHome,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (a *Address) ComputeFullAddress() {
	address := ""
	if a.Apartment != "" {
		address = a.Apartment + ", "
	}
	address += a.StreetAddress
	if a.WardName != "" {
		address += ", " + a.WardName
	}
	if a.DistrictName != "" {
		address += ", " + a.DistrictName
	}
	if a.ProvinceName != "" {
		address += ", " + a.ProvinceName
	}
	a.FullAddress = address
}

func (a *Address) String() string {
	return fmt.Sprintf("%s - %s", a.ContactName, a.FullAddress)
}
