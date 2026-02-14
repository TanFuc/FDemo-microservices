package impl

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"microservices/template-service/internal/model"
	"microservices/template-service/internal/repository"
	"microservices/template-service/internal/service"
)

var (
	ErrItemNotFound = errors.New("item not found")
	ErrInvalidID    = errors.New("invalid item ID")
)

type itemService struct {
	itemRepo repository.ItemRepository
}

// NewItemService creates a new item service
func NewItemService(itemRepo repository.ItemRepository) service.ItemService {
	return &itemService{
		itemRepo: itemRepo,
	}
}

func (s *itemService) CreateItem(ctx context.Context, req *model.CreateItemRequest) (*model.ItemResponse, error) {
	item := &model.Item{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.itemRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item.ToResponse(), nil
}

func (s *itemService) GetItem(ctx context.Context, id string) (*model.ItemResponse, error) {
	itemID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrItemNotFound
	}

	return item.ToResponse(), nil
}

func (s *itemService) UpdateItem(ctx context.Context, id string, req *model.UpdateItemRequest) (*model.ItemResponse, error) {
	itemID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrItemNotFound
	}

	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Description != "" {
		item.Description = req.Description
	}

	if err := s.itemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item.ToResponse(), nil
}

func (s *itemService) DeleteItem(ctx context.Context, id string) error {
	itemID, err := uuid.Parse(id)
	if err != nil {
		return ErrInvalidID
	}

	exists, err := s.itemRepo.ExistsByID(ctx, itemID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrItemNotFound
	}

	return s.itemRepo.Delete(ctx, itemID)
}

func (s *itemService) ListItems(ctx context.Context, page, limit int) (*model.ListItemsResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	items, total, err := s.itemRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]model.ItemResponse, len(items))
	for i, item := range items {
		responses[i] = *item.ToResponse()
	}

	return &model.ListItemsResponse{
		Items: responses,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
