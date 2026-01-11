package domain

import "time"

type ReviewStatus string

const (
	ReviewStatusVisible ReviewStatus = "VISIBLE"
	ReviewStatusHidden  ReviewStatus = "HIDDEN"
)

type Reply struct {
	Content   string    `json:"content" bson:"content"`
	RepliedAt time.Time `json:"repliedAt" bson:"repliedAt"`
}

type Review struct {
	ID          string       `json:"id" bson:"_id"`
	UserID      string       `json:"userId" bson:"userId"`
	UserName    string       `json:"userName" bson:"userName"`
	UserAvatar  string       `json:"userAvatar" bson:"userAvatar"`
	ProductID   string       `json:"productId" bson:"productId"`
	OrderID     string       `json:"orderId" bson:"orderId"`
	Rating      int          `json:"rating" bson:"rating"`
	Content     string       `json:"content" bson:"content"`
	Images      []string     `json:"images" bson:"images"`
	IsPurchased bool         `json:"isPurchased" bson:"isPurchased"`
	Reply       *Reply       `json:"reply,omitempty" bson:"reply,omitempty"`
	Status      ReviewStatus `json:"status" bson:"status"`
	CreatedAt   time.Time    `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt" bson:"updatedAt"`
}
