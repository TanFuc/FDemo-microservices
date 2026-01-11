package domain

import (
	"time"

	"github.com/google/uuid"
)

type CampaignStatus string

const (
	CampaignStatusActive   CampaignStatus = "ACTIVE"
	CampaignStatusInactive CampaignStatus = "INACTIVE"
)

type Campaign struct {
	ID        uuid.UUID
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Status    CampaignStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCampaign(name string, startTime, endTime time.Time) *Campaign {
	return &Campaign{
		ID:        uuid.New(),
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
		Status:    CampaignStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (c *Campaign) IsActive() bool {
	now := time.Now()
	return c.Status == CampaignStatusActive &&
		now.After(c.StartTime) &&
		now.Before(c.EndTime)
}
