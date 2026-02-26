package models

import "time"

// ReviewCreatedEvent received from NATS after a review is posted
type ReviewCreatedEvent struct {
	EventID    string    `json:"eventId"`
	ReviewID   string    `json:"reviewId"`
	ProductID  string    `json:"productId"`
	OrderID    string    `json:"orderId"`
	UserID     string    `json:"userId"`
	UserName   string    `json:"userName"`
	Rating     int       `json:"rating"`
	Content    string    `json:"content"`
	Images     []string  `json:"images"`
	IsEdited   bool      `json:"isEdited"`
	EventType  string    `json:"eventType"`
	OccurredAt time.Time `json:"occurredAt"`
}

// RatingUpdatedEvent received when ProductRating is recalculated
type RatingUpdatedEvent struct {
	EventID       string    `json:"eventId"`
	ProductID     string    `json:"productId"`
	AverageRating float64   `json:"averageRating"`
	TotalReviews  int       `json:"totalReviews"`
	OccurredAt    time.Time `json:"occurredAt"`
}
