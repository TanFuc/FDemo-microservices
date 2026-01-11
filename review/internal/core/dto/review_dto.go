package dto

type CreateReviewRequest struct {
	UserID     string   `json:"userId" validate:"required,uuid"`
	UserName   string   `json:"userName" validate:"required,min=1,max=100"`
	UserAvatar string   `json:"userAvatar" validate:"omitempty,url"`
	ProductID  string   `json:"productId" validate:"required,uuid"`
	OrderID    string   `json:"orderId" validate:"required,uuid"`
	Rating     int      `json:"rating" validate:"required,min=1,max=5"`
	Content    string   `json:"content" validate:"required,min=1,max=5000"`
	Images     []string `json:"images" validate:"omitempty,dive,url"`
}

type ReplyReviewRequest struct {
	ReviewID string `json:"reviewId" validate:"required,uuid"`
	Content  string `json:"content" validate:"required,min=1,max=2000"`
}

type GetReviewsRequest struct {
	ProductID string `json:"productId" validate:"required,uuid"`
	Page      int    `json:"page" validate:"min=1"`
	Limit     int    `json:"limit" validate:"min=1,max=100"`
	SortBy    string `json:"sortBy" validate:"omitempty,oneof=createdAt rating"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalItems int64       `json:"totalItems"`
	TotalPages int64       `json:"totalPages"`
}
