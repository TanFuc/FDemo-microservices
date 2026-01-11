package usecase

import (
	"context"
	"time"

	"inventory-service/internal/domain"
	"inventory-service/internal/repository"

	"gorm.io/gorm"
)

type InventoryUseCase interface {
	ReserveStock(ctx context.Context, orderID string, items []domain.ReservationItem) error
	ConfirmStock(ctx context.Context, orderID string) error
	ReleaseStock(ctx context.Context, orderID string) error
	SyncRedisFromDB(ctx context.Context, skuID string) error
	CreateInventory(ctx context.Context, item *domain.InventoryItem) error
	GetInventory(ctx context.Context, skuID string) (*domain.InventoryItem, error)
}

type inventoryUseCase struct {
	db              *gorm.DB
	inventoryRepo   repository.InventoryRepository
	reservationRepo repository.ReservationRepository
	cacheRepo       repository.CacheRepository
	reservationTTL  time.Duration
}

func NewInventoryUseCase(
	db *gorm.DB,
	inventoryRepo repository.InventoryRepository,
	reservationRepo repository.ReservationRepository,
	cacheRepo repository.CacheRepository,
	reservationTTL time.Duration,
) InventoryUseCase {
	return &inventoryUseCase{
		db:              db,
		inventoryRepo:   inventoryRepo,
		reservationRepo: reservationRepo,
		cacheRepo:       cacheRepo,
		reservationTTL:  reservationTTL,
	}
}

func (uc *inventoryUseCase) ReserveStock(ctx context.Context, orderID string, items []domain.ReservationItem) error {
	reservedItems := make([]domain.ReservationItem, 0, len(items))

	for _, item := range items {
		success, err := uc.cacheRepo.Reserve(ctx, item.SkuID, item.Quantity)
		if err != nil {
			uc.rollbackRedisReservations(ctx, reservedItems)
			return err
		}

		if !success {
			uc.rollbackRedisReservations(ctx, reservedItems)
			return domain.ErrInsufficientStock
		}

		reservedItems = append(reservedItems, item)
	}

	tx := uc.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		uc.rollbackRedisReservations(ctx, reservedItems)
		return tx.Error
	}

	expiresAt := time.Now().Add(uc.reservationTTL)

	for _, item := range items {
		reservation := domain.NewStockReservation(orderID, item.SkuID, item.Quantity, expiresAt)
		if err := uc.reservationRepo.Create(ctx, tx, reservation); err != nil {
			tx.Rollback()
			uc.rollbackRedisReservations(ctx, reservedItems)
			return err
		}

		if err := uc.inventoryRepo.UpdateReservedStock(ctx, tx, item.SkuID, item.Quantity); err != nil {
			tx.Rollback()
			uc.rollbackRedisReservations(ctx, reservedItems)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		uc.rollbackRedisReservations(ctx, reservedItems)
		return err
	}

	return nil
}

func (uc *inventoryUseCase) rollbackRedisReservations(ctx context.Context, items []domain.ReservationItem) {
	for _, item := range items {
		_ = uc.cacheRepo.Release(ctx, item.SkuID, item.Quantity)
	}
}

func (uc *inventoryUseCase) ConfirmStock(ctx context.Context, orderID string) error {
	reservations, err := uc.reservationRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	if len(reservations) == 0 {
		return domain.ErrReservationNotFound
	}

	if reservations[0].Status == domain.StatusConfirmed {
		return nil
	}

	if reservations[0].Status == domain.StatusCancelled {
		return domain.ErrAlreadyCancelled
	}

	tx := uc.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	for _, reservation := range reservations {
		if reservation.Status != domain.StatusPending {
			continue
		}

		if err := uc.inventoryRepo.DeductStock(ctx, tx, reservation.SkuID, reservation.Quantity); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := uc.reservationRepo.UpdateStatus(ctx, tx, orderID, domain.StatusPending, domain.StatusConfirmed); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	for _, reservation := range reservations {
		if reservation.Status == domain.StatusPending {
			_ = uc.inventoryRepo.SyncFromDB(ctx, reservation.SkuID)
		}
	}

	return nil
}

func (uc *inventoryUseCase) ReleaseStock(ctx context.Context, orderID string) error {
	reservations, err := uc.reservationRepo.GetPendingByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	if len(reservations) == 0 {
		return domain.ErrReservationNotFound
	}

	tx := uc.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	for _, reservation := range reservations {
		if err := uc.inventoryRepo.UpdateReservedStock(ctx, tx, reservation.SkuID, -reservation.Quantity); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := uc.reservationRepo.UpdateStatus(ctx, tx, orderID, domain.StatusPending, domain.StatusCancelled); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	for _, reservation := range reservations {
		_ = uc.cacheRepo.Release(ctx, reservation.SkuID, reservation.Quantity)
	}

	return nil
}

func (uc *inventoryUseCase) SyncRedisFromDB(ctx context.Context, skuID string) error {
	return uc.inventoryRepo.SyncFromDB(ctx, skuID)
}

func (uc *inventoryUseCase) CreateInventory(ctx context.Context, item *domain.InventoryItem) error {
	return uc.inventoryRepo.Create(ctx, item)
}

func (uc *inventoryUseCase) GetInventory(ctx context.Context, skuID string) (*domain.InventoryItem, error) {
	return uc.inventoryRepo.GetBySkuID(ctx, skuID)
}
