package domain

import "time"

type StarCounts struct {
	One   int `json:"1" bson:"1"`
	Two   int `json:"2" bson:"2"`
	Three int `json:"3" bson:"3"`
	Four  int `json:"4" bson:"4"`
	Five  int `json:"5" bson:"5"`
}

type ProductRating struct {
	ProductID     string     `json:"productId" bson:"_id"`
	AverageRating float64    `json:"averageRating" bson:"averageRating"`
	TotalReviews  int        `json:"totalReviews" bson:"totalReviews"`
	StarCounts    StarCounts `json:"starCounts" bson:"starCounts"`
	UpdatedAt     time.Time  `json:"updatedAt" bson:"updatedAt"`
}

type RatingSummary struct {
	Average   float64            `json:"average"`
	Total     int                `json:"total"`
	Breakdown map[string]int     `json:"breakdown"`
}

func (pr *ProductRating) ToSummary() *RatingSummary {
	return &RatingSummary{
		Average: pr.AverageRating,
		Total:   pr.TotalReviews,
		Breakdown: map[string]int{
			"1": pr.StarCounts.One,
			"2": pr.StarCounts.Two,
			"3": pr.StarCounts.Three,
			"4": pr.StarCounts.Four,
			"5": pr.StarCounts.Five,
		},
	}
}
