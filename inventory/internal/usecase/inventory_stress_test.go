package usecase

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"microservices/inventory/internal/domain"
	"microservices/inventory/internal/infrastructure"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=inventory_test sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := infrastructure.AutoMigrate(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	db.Exec("DELETE FROM stock_reservations")
	db.Exec("DELETE FROM inventory_items")

	return db
}

func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	client.FlushDB(ctx)

	return client
}

func TestConcurrencyStressTest(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	ctx := context.Background()

	cacheRepo := infrastructure.NewCacheRepository(redisClient)
	inventoryRepo := infrastructure.NewInventoryRepository(db, cacheRepo)
	reservationRepo := infrastructure.NewReservationRepository(db)

	useCase := NewInventoryUseCase(
		db,
		inventoryRepo,
		reservationRepo,
		cacheRepo,
		15*time.Minute,
	)

	skuID := "SKU-A"
	initialStock := 100
	numGoroutines := 150
	quantityPerReservation := 1

	item := &domain.InventoryItem{
		SkuID:         skuID,
		TotalStock:    initialStock,
		ReservedStock: 0,
	}
	if err := useCase.CreateInventory(ctx, item); err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	var successCount int64
	var failCount int64
	var wg sync.WaitGroup

	startChan := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			<-startChan

			orderID := fmt.Sprintf("order-%d-%d", workerID, time.Now().UnixNano())
			items := []domain.ReservationItem{
				{SkuID: skuID, Quantity: quantityPerReservation},
			}

			err := useCase.ReserveStock(ctx, orderID, items)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
		}(i)
	}

	close(startChan)

	wg.Wait()

	t.Logf("Success count: %d", successCount)
	t.Logf("Fail count: %d", failCount)

	if successCount != int64(initialStock) {
		t.Errorf("Expected exactly %d successful reservations, got %d", initialStock, successCount)
	}

	expectedFails := int64(numGoroutines - initialStock)
	if failCount != expectedFails {
		t.Errorf("Expected exactly %d failed reservations, got %d", expectedFails, failCount)
	}

	total, reserved, err := cacheRepo.GetInventory(ctx, skuID)
	if err != nil {
		t.Fatalf("Failed to get Redis inventory: %v", err)
	}

	t.Logf("Redis state - Total: %d, Reserved: %d", total, reserved)

	if total != initialStock {
		t.Errorf("Redis total stock should be %d, got %d", initialStock, total)
	}
	if reserved != initialStock {
		t.Errorf("Redis reserved stock should be %d, got %d", initialStock, reserved)
	}

	dbItem, err := inventoryRepo.GetBySkuID(ctx, skuID)
	if err != nil {
		t.Fatalf("Failed to get DB inventory: %v", err)
	}

	t.Logf("DB state - Total: %d, Reserved: %d", dbItem.TotalStock, dbItem.ReservedStock)

	if dbItem.TotalStock != initialStock {
		t.Errorf("DB total stock should be %d, got %d", initialStock, dbItem.TotalStock)
	}
	if dbItem.ReservedStock != initialStock {
		t.Errorf("DB reserved stock should be %d, got %d", initialStock, dbItem.ReservedStock)
	}

	t.Log("Stress test completed successfully!")
}

func TestReservationConfirmFlow(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	ctx := context.Background()

	cacheRepo := infrastructure.NewCacheRepository(redisClient)
	inventoryRepo := infrastructure.NewInventoryRepository(db, cacheRepo)
	reservationRepo := infrastructure.NewReservationRepository(db)

	useCase := NewInventoryUseCase(
		db,
		inventoryRepo,
		reservationRepo,
		cacheRepo,
		15*time.Minute,
	)

	skuID := "SKU-B"
	item := &domain.InventoryItem{
		SkuID:         skuID,
		TotalStock:    50,
		ReservedStock: 0,
	}
	if err := useCase.CreateInventory(ctx, item); err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	orderID := "test-order-1"
	items := []domain.ReservationItem{
		{SkuID: skuID, Quantity: 10},
	}

	if err := useCase.ReserveStock(ctx, orderID, items); err != nil {
		t.Fatalf("Failed to reserve stock: %v", err)
	}

	dbItem, _ := inventoryRepo.GetBySkuID(ctx, skuID)
	if dbItem.ReservedStock != 10 {
		t.Errorf("After reserve: expected reserved=10, got %d", dbItem.ReservedStock)
	}

	if err := useCase.ConfirmStock(ctx, orderID); err != nil {
		t.Fatalf("Failed to confirm stock: %v", err)
	}

	dbItem, _ = inventoryRepo.GetBySkuID(ctx, skuID)
	if dbItem.TotalStock != 40 || dbItem.ReservedStock != 0 {
		t.Errorf("After confirm: expected total=40, reserved=0, got total=%d, reserved=%d",
			dbItem.TotalStock, dbItem.ReservedStock)
	}

	if err := useCase.ConfirmStock(ctx, orderID); err != nil {
		t.Errorf("Idempotent confirm should not return error, got: %v", err)
	}

	t.Log("Reserve-Confirm flow test completed successfully!")
}

func TestReservationReleaseFlow(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	ctx := context.Background()

	cacheRepo := infrastructure.NewCacheRepository(redisClient)
	inventoryRepo := infrastructure.NewInventoryRepository(db, cacheRepo)
	reservationRepo := infrastructure.NewReservationRepository(db)

	useCase := NewInventoryUseCase(
		db,
		inventoryRepo,
		reservationRepo,
		cacheRepo,
		15*time.Minute,
	)

	skuID := "SKU-C"
	item := &domain.InventoryItem{
		SkuID:         skuID,
		TotalStock:    50,
		ReservedStock: 0,
	}
	if err := useCase.CreateInventory(ctx, item); err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	orderID := "test-order-2"
	items := []domain.ReservationItem{
		{SkuID: skuID, Quantity: 15},
	}

	if err := useCase.ReserveStock(ctx, orderID, items); err != nil {
		t.Fatalf("Failed to reserve stock: %v", err)
	}

	dbItem, _ := inventoryRepo.GetBySkuID(ctx, skuID)
	if dbItem.ReservedStock != 15 {
		t.Errorf("After reserve: expected reserved=15, got %d", dbItem.ReservedStock)
	}

	if err := useCase.ReleaseStock(ctx, orderID); err != nil {
		t.Fatalf("Failed to release stock: %v", err)
	}

	dbItem, _ = inventoryRepo.GetBySkuID(ctx, skuID)
	if dbItem.TotalStock != 50 || dbItem.ReservedStock != 0 {
		t.Errorf("After release: expected total=50, reserved=0, got total=%d, reserved=%d",
			dbItem.TotalStock, dbItem.ReservedStock)
	}

	total, reserved, _ := cacheRepo.GetInventory(ctx, skuID)
	if total != 50 || reserved != 0 {
		t.Errorf("Redis after release: expected total=50, reserved=0, got total=%d, reserved=%d",
			total, reserved)
	}

	t.Log("Reserve-Release flow test completed successfully!")
}

func TestBatchReservationRollback(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	ctx := context.Background()

	cacheRepo := infrastructure.NewCacheRepository(redisClient)
	inventoryRepo := infrastructure.NewInventoryRepository(db, cacheRepo)
	reservationRepo := infrastructure.NewReservationRepository(db)

	useCase := NewInventoryUseCase(
		db,
		inventoryRepo,
		reservationRepo,
		cacheRepo,
		15*time.Minute,
	)

	item1 := &domain.InventoryItem{SkuID: "SKU-D1", TotalStock: 10, ReservedStock: 0}
	item2 := &domain.InventoryItem{SkuID: "SKU-D2", TotalStock: 5, ReservedStock: 0}

	if err := useCase.CreateInventory(ctx, item1); err != nil {
		t.Fatalf("Failed to create inventory 1: %v", err)
	}
	if err := useCase.CreateInventory(ctx, item2); err != nil {
		t.Fatalf("Failed to create inventory 2: %v", err)
	}

	orderID := "test-order-batch"
	items := []domain.ReservationItem{
		{SkuID: "SKU-D1", Quantity: 5},
		{SkuID: "SKU-D2", Quantity: 10},
	}

	err := useCase.ReserveStock(ctx, orderID, items)
	if err == nil {
		t.Fatal("Expected reservation to fail due to insufficient stock on second item")
	}

	total1, reserved1, _ := cacheRepo.GetInventory(ctx, "SKU-D1")
	total2, reserved2, _ := cacheRepo.GetInventory(ctx, "SKU-D2")

	t.Logf("After failed batch - SKU-D1: total=%d, reserved=%d", total1, reserved1)
	t.Logf("After failed batch - SKU-D2: total=%d, reserved=%d", total2, reserved2)

	if reserved1 != 0 {
		t.Errorf("SKU-D1 should have reserved=0 after rollback, got %d", reserved1)
	}
	if reserved2 != 0 {
		t.Errorf("SKU-D2 should have reserved=0 after rollback, got %d", reserved2)
	}

	t.Log("Batch rollback test completed successfully!")
}
