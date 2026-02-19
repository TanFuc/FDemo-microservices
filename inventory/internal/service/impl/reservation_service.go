package impl

import (
	"context"
	"time"

	"microservices/inventory/internal/model"
	"microservices/inventory/internal/repository"
	"microservices/inventory/internal/service"
)

var _ service.ReservationService = (*reservationService)(nil)

type reservationService struct {
	inventoryRepo   repository.InventoryRepository
	reservationRepo repository.ReservationRepository
	cacheRepo       repository.CacheRepository
	reservationTTL  time.Duration
}

func NewReservationService(
	inventoryRepo repository.InventoryRepository,
	reservationRepo repository.ReservationRepository,
	cacheRepo repository.CacheRepository,
	reservationTTL time.Duration,
) service.ReservationService {
	return &reservationService{
		inventoryRepo:   inventoryRepo,
		reservationRepo: reservationRepo,
		cacheRepo:       cacheRepo,
		reservationTTL:  reservationTTL,
	}
}

func (s *reservationService) ReserveStock(ctx context.Context, req *model.ReserveStockRequest) ([]model.StockReservation, error) {
	if req.OrderID == "" {
		return nil, model.ErrInvalidOrderID
	}
	if len(req.Items) == 0 {
		return nil, model.ErrInvalidQuantity
	}

	// Validate items
	for _, item := range req.Items {
		if item.SkuID == "" {
			return nil, model.ErrInvalidSkuID
		}
		if item.Quantity <= 0 {
			return nil, model.ErrInvalidQuantity
		}
	}

	// Reserve in cache first (atomic)
	reservedItems := make([]model.ReservationItem, 0, len(req.Items))

	for _, item := range req.Items {
		success, err := s.cacheRepo.Reserve(ctx, item.SkuID, item.Quantity)
		if err != nil {
			s.rollbackCacheReservations(ctx, reservedItems)
			return nil, err
		}

		if !success {
			s.rollbackCacheReservations(ctx, reservedItems)
			return nil, model.ErrInsufficientStock
		}

		reservedItems = append(reservedItems, item)
	}

	// Start database transaction
	db := s.inventoryRepo.GetDB()
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		s.rollbackCacheReservations(ctx, reservedItems)
		return nil, tx.Error
	}

	expiresAt := time.Now().Add(s.reservationTTL)
	reservations := make([]model.StockReservation, 0, len(req.Items))

	for _, item := range req.Items {
		reservation := model.NewStockReservation(req.OrderID, item.SkuID, item.Quantity, expiresAt)

		if err := s.reservationRepo.Create(ctx, tx, reservation); err != nil {
			tx.Rollback()
			s.rollbackCacheReservations(ctx, reservedItems)
			return nil, err
		}

		if err := s.inventoryRepo.UpdateReservedStock(ctx, tx, item.SkuID, item.Quantity); err != nil {
			tx.Rollback()
			s.rollbackCacheReservations(ctx, reservedItems)
			return nil, err
		}

		reservations = append(reservations, *reservation)
	}

	if err := tx.Commit().Error; err != nil {
		s.rollbackCacheReservations(ctx, reservedItems)
		return nil, err
	}

	return reservations, nil
}

func (s *reservationService) rollbackCacheReservations(ctx context.Context, items []model.ReservationItem) {
	for _, item := range items {
		_ = s.cacheRepo.Release(ctx, item.SkuID, item.Quantity)
	}
}

func (s *reservationService) ConfirmStock(ctx context.Context, orderID string) error {
	if orderID == "" {
		return model.ErrInvalidOrderID
	}

	reservations, err := s.reservationRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	if len(reservations) == 0 {
		return model.ErrReservationNotFound
	}

	// Check status of first reservation
	if reservations[0].Status == model.StatusConfirmed {
		return nil // Already confirmed
	}

	if reservations[0].Status == model.StatusCancelled {
		return model.ErrAlreadyCancelled
	}

	// Start transaction
	db := s.inventoryRepo.GetDB()
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Deduct stock for each pending reservation
	for _, reservation := range reservations {
		if reservation.Status != model.StatusPending {
			continue
		}

		if err := s.inventoryRepo.DeductStock(ctx, tx, reservation.SkuID, reservation.Quantity); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update reservation status
	if err := s.reservationRepo.UpdateStatus(ctx, tx, orderID, model.StatusPending, model.StatusConfirmed); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Sync cache for affected SKUs
	for _, reservation := range reservations {
		if reservation.Status == model.StatusPending {
			_ = s.inventoryRepo.SyncFromDB(ctx, reservation.SkuID)
		}
	}

	return nil
}

func (s *reservationService) ReleaseStock(ctx context.Context, orderID string) error {
	if orderID == "" {
		return model.ErrInvalidOrderID
	}

	reservations, err := s.reservationRepo.GetPendingByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	if len(reservations) == 0 {
		return model.ErrReservationNotFound
	}

	// Start transaction
	db := s.inventoryRepo.GetDB()
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Release reserved stock
	for _, reservation := range reservations {
		if err := s.inventoryRepo.UpdateReservedStock(ctx, tx, reservation.SkuID, -reservation.Quantity); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update reservation status
	if err := s.reservationRepo.UpdateStatus(ctx, tx, orderID, model.StatusPending, model.StatusCancelled); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Release from cache
	for _, reservation := range reservations {
		_ = s.cacheRepo.Release(ctx, reservation.SkuID, reservation.Quantity)
	}

	return nil
}

func (s *reservationService) GetReservation(ctx context.Context, orderID string) ([]model.StockReservation, error) {
	if orderID == "" {
		return nil, model.ErrInvalidOrderID
	}

	reservations, err := s.reservationRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if len(reservations) == 0 {
		return nil, model.ErrReservationNotFound
	}

	return reservations, nil
}

func (s *reservationService) ListReservations(ctx context.Context, filter *model.ReservationFilter) ([]model.StockReservation, int64, error) {
	if filter == nil {
		filter = &model.ReservationFilter{}
	}
	filter.EnsureDefaults()

	return s.reservationRepo.List(ctx, filter)
}
