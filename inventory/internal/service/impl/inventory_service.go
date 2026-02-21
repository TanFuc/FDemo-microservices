package impl

import (
	"context"

	"microservices/inventory/internal/model"
	"microservices/inventory/internal/repository"
	"microservices/inventory/internal/service"
)

var _ service.InventoryService = (*inventoryService)(nil)

type inventoryService struct {
	inventoryRepo repository.InventoryRepository
	cacheRepo     repository.CacheRepository
}

func NewInventoryService(
	inventoryRepo repository.InventoryRepository,
	cacheRepo repository.CacheRepository,
) service.InventoryService {
	return &inventoryService{
		inventoryRepo: inventoryRepo,
		cacheRepo:     cacheRepo,
	}
}

func (s *inventoryService) CreateInventory(ctx context.Context, req *model.CreateInventoryRequest) (*model.InventoryItem, error) {
	if req.SkuID == "" {
		return nil, model.ErrInvalidSkuID
	}
	if req.TotalStock < 0 {
		return nil, model.ErrInvalidQuantity
	}

	// Check if already exists
	exists, err := s.inventoryRepo.ExistsBySkuID(ctx, req.SkuID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, model.ErrInventoryAlreadyExists
	}

	item := &model.InventoryItem{
		SkuID:         req.SkuID,
		TotalStock:    req.TotalStock,
		ReservedStock: 0,
	}

	if err := s.inventoryRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *inventoryService) GetInventory(ctx context.Context, skuID string) (*model.InventoryItem, error) {
	if skuID == "" {
		return nil, model.ErrInvalidSkuID
	}

	item, err := s.inventoryRepo.GetBySkuID(ctx, skuID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, model.ErrInventoryNotFound
	}

	return item, nil
}

func (s *inventoryService) UpdateInventory(ctx context.Context, skuID string, req *model.UpdateInventoryRequest) (*model.InventoryItem, error) {
	if skuID == "" {
		return nil, model.ErrInvalidSkuID
	}
	if req.TotalStock < 0 {
		return nil, model.ErrInvalidQuantity
	}

	item, err := s.inventoryRepo.GetBySkuID(ctx, skuID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, model.ErrInventoryNotFound
	}

	// Ensure total stock >= reserved stock
	if req.TotalStock < item.ReservedStock {
		return nil, model.ErrInsufficientStock
	}

	item.TotalStock = req.TotalStock

	if err := s.inventoryRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *inventoryService) DeleteInventory(ctx context.Context, skuID string) error {
	if skuID == "" {
		return model.ErrInvalidSkuID
	}

	item, err := s.inventoryRepo.GetBySkuID(ctx, skuID)
	if err != nil {
		return err
	}
	if item == nil {
		return model.ErrInventoryNotFound
	}

	// Don't allow deleting if there are reserved items
	if item.ReservedStock > 0 {
		return model.ErrInsufficientStock
	}

	return s.inventoryRepo.Delete(ctx, skuID)
}

func (s *inventoryService) ListInventory(ctx context.Context, filter *model.InventoryFilter) ([]model.InventoryItem, int64, error) {
	if filter == nil {
		filter = &model.InventoryFilter{}
	}
	filter.EnsureDefaults()

	return s.inventoryRepo.List(ctx, filter)
}

func (s *inventoryService) SyncInventory(ctx context.Context, skuID string) (*model.InventoryItem, error) {
	if skuID == "" {
		return nil, model.ErrInvalidSkuID
	}

	if err := s.inventoryRepo.SyncFromDB(ctx, skuID); err != nil {
		return nil, err
	}

	return s.inventoryRepo.GetBySkuID(ctx, skuID)
}
