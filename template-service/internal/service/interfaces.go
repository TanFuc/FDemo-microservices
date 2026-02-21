package service

import (
	"context"

	"microservices/template-service/internal/model"
)

// ItemService defines the interface for item business logic
type ItemService interface {
	// CreateItem creates a new item
	CreateItem(ctx context.Context, req *model.CreateItemRequest) (*model.ItemResponse, error)

	// GetItem retrieves an item by ID
	GetItem(ctx context.Context, id string) (*model.ItemResponse, error)

	// UpdateItem updates an existing item
	UpdateItem(ctx context.Context, id string, req *model.UpdateItemRequest) (*model.ItemResponse, error)

	// DeleteItem removes an item by ID
	DeleteItem(ctx context.Context, id string) error

	// ListItems retrieves items with pagination
	ListItems(ctx context.Context, page, limit int) (*model.ListItemsResponse, error)
}
