package model

import "time"

// CreateItemRequest represents the request to create an item
type CreateItemRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1000"`
}

// UpdateItemRequest represents the request to update an item
type UpdateItemRequest struct {
	Name        string `json:"name" validate:"omitempty,min=1,max=255"`
	Description string `json:"description" validate:"omitempty,max=1000"`
}

// ItemResponse represents the response for an item
type ItemResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListItemsRequest represents pagination request
type ListItemsRequest struct {
	Page  int `json:"page" validate:"min=1"`
	Limit int `json:"limit" validate:"min=1,max=100"`
}

// ListItemsResponse represents paginated response
type ListItemsResponse struct {
	Items []ItemResponse `json:"items"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// ToResponse converts Item entity to ItemResponse DTO
func (i *Item) ToResponse() *ItemResponse {
	return &ItemResponse{
		ID:          i.ID.String(),
		Name:        i.Name,
		Description: i.Description,
		CreatedAt:   i.CreatedAt,
		UpdatedAt:   i.UpdatedAt,
	}
}
