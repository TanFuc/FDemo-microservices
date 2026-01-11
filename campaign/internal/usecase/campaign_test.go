package usecase

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"microservices/campaign/internal/domain"
)

func setupTestUsecase() (*CampaignUsecase, *MockVoucherRepository, *MockUserVoucherRepository, *MockVoucherCacheRepository) {
	voucherRepo := NewMockVoucherRepository()
	userVoucherRepo := NewMockUserVoucherRepository()
	cacheRepo := NewMockVoucherCacheRepository()
	uc := NewCampaignUsecase(voucherRepo, userVoucherRepo, cacheRepo)
	return uc, voucherRepo, userVoucherRepo, cacheRepo
}

// =============================================================================
// RULE ENGINE TESTS
// =============================================================================

func TestCalculateCart_PercentageDiscount_MinOrderMet(t *testing.T) {
	uc, voucherRepo, _, _ := setupTestUsecase()
	ctx := context.Background()

	// Create voucher: 10% OFF, Min Spend 100k
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"SALE10",
		campaignID,
		100,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(10), // 10%
		domain.VoucherConditions{
			MinOrderValue: decimal.NewFromInt(100000), // 100k minimum
		},
	)
	voucherRepo.Create(ctx, voucher)

	// Test Case 1: Cart value 200k -> Should get 20k discount (10% of 200k)
	items := []domain.CartItem{
		{SKU: "SKU001", Name: "Product A", Price: decimal.NewFromInt(100000), Quantity: 2, Category: "electronics"},
	}

	result, err := uc.CalculateCart(ctx, items, "SALE10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDiscount := decimal.NewFromInt(20000)
	if !result.DiscountAmount.Equal(expectedDiscount) {
		t.Errorf("expected discount %s, got %s", expectedDiscount, result.DiscountAmount)
	}

	expectedFinal := decimal.NewFromInt(180000)
	if !result.FinalPrice.Equal(expectedFinal) {
		t.Errorf("expected final price %s, got %s", expectedFinal, result.FinalPrice)
	}

	if !result.VoucherApplied {
		t.Error("expected voucher to be applied")
	}
}

func TestCalculateCart_MinOrderNotMet(t *testing.T) {
	uc, voucherRepo, _, _ := setupTestUsecase()
	ctx := context.Background()

	// Create voucher: 10% OFF, Min Spend 100k
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"SALE10",
		campaignID,
		100,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(10),
		domain.VoucherConditions{
			MinOrderValue: decimal.NewFromInt(100000),
		},
	)
	voucherRepo.Create(ctx, voucher)

	// Test Case 2: Cart value 90k -> Should fail with "Not eligible"
	items := []domain.CartItem{
		{SKU: "SKU001", Name: "Product A", Price: decimal.NewFromInt(45000), Quantity: 2, Category: "electronics"},
	}

	_, err := uc.CalculateCart(ctx, items, "SALE10")
	if err != domain.ErrMinOrderNotMet {
		t.Errorf("expected ErrMinOrderNotMet, got %v", err)
	}
}

func TestCalculateCart_MaxDiscountCap(t *testing.T) {
	uc, voucherRepo, _, _ := setupTestUsecase()
	ctx := context.Background()

	// Create voucher: 20% OFF, Max discount 50k
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"SALE20",
		campaignID,
		100,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(20), // 20%
		domain.VoucherConditions{
			MaxDiscount: decimal.NewFromInt(50000), // Max 50k
		},
	)
	voucherRepo.Create(ctx, voucher)

	// Cart value 500k -> 20% = 100k, but capped at 50k
	items := []domain.CartItem{
		{SKU: "SKU001", Name: "Product A", Price: decimal.NewFromInt(500000), Quantity: 1, Category: "electronics"},
	}

	result, err := uc.CalculateCart(ctx, items, "SALE20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDiscount := decimal.NewFromInt(50000) // Capped at 50k
	if !result.DiscountAmount.Equal(expectedDiscount) {
		t.Errorf("expected discount %s (capped), got %s", expectedDiscount, result.DiscountAmount)
	}
}

func TestCalculateCart_FixedAmountDiscount(t *testing.T) {
	uc, voucherRepo, _, _ := setupTestUsecase()
	ctx := context.Background()

	// Create voucher: Fixed 30k OFF
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"FIXED30K",
		campaignID,
		100,
		domain.VoucherTypeFixedAmount,
		decimal.NewFromInt(30000), // 30k fixed
		domain.VoucherConditions{},
	)
	voucherRepo.Create(ctx, voucher)

	items := []domain.CartItem{
		{SKU: "SKU001", Name: "Product A", Price: decimal.NewFromInt(150000), Quantity: 1, Category: "electronics"},
	}

	result, err := uc.CalculateCart(ctx, items, "FIXED30K")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDiscount := decimal.NewFromInt(30000)
	if !result.DiscountAmount.Equal(expectedDiscount) {
		t.Errorf("expected discount %s, got %s", expectedDiscount, result.DiscountAmount)
	}

	expectedFinal := decimal.NewFromInt(120000)
	if !result.FinalPrice.Equal(expectedFinal) {
		t.Errorf("expected final price %s, got %s", expectedFinal, result.FinalPrice)
	}
}

func TestCalculateCart_CategoryRestriction(t *testing.T) {
	uc, voucherRepo, _, _ := setupTestUsecase()
	ctx := context.Background()

	// Create voucher: 15% OFF, only for electronics category
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"ELEC15",
		campaignID,
		100,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(15),
		domain.VoucherConditions{
			AllowedCategories: []string{"electronics"},
		},
	)
	voucherRepo.Create(ctx, voucher)

	// Test with clothing items -> Should fail
	items := []domain.CartItem{
		{SKU: "SKU001", Name: "T-Shirt", Price: decimal.NewFromInt(200000), Quantity: 1, Category: "clothing"},
	}

	_, err := uc.CalculateCart(ctx, items, "ELEC15")
	if err != domain.ErrCategoryNotAllowed {
		t.Errorf("expected ErrCategoryNotAllowed, got %v", err)
	}

	// Test with electronics -> Should succeed
	items = []domain.CartItem{
		{SKU: "SKU002", Name: "Headphones", Price: decimal.NewFromInt(200000), Quantity: 1, Category: "electronics"},
	}

	result, err := uc.CalculateCart(ctx, items, "ELEC15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDiscount := decimal.NewFromInt(30000) // 15% of 200k
	if !result.DiscountAmount.Equal(expectedDiscount) {
		t.Errorf("expected discount %s, got %s", expectedDiscount, result.DiscountAmount)
	}
}

func TestCalculateCart_ExcludedProducts(t *testing.T) {
	uc, voucherRepo, _, _ := setupTestUsecase()
	ctx := context.Background()

	// Create voucher: 10% OFF, excludes iPhone 15
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"NOPHONE",
		campaignID,
		100,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(10),
		domain.VoucherConditions{
			ExcludedProducts: []string{"sku_iphone_15"},
		},
	)
	voucherRepo.Create(ctx, voucher)

	// Test with excluded product -> Should fail
	items := []domain.CartItem{
		{SKU: "sku_iphone_15", Name: "iPhone 15", Price: decimal.NewFromInt(30000000), Quantity: 1, Category: "electronics"},
	}

	_, err := uc.CalculateCart(ctx, items, "NOPHONE")
	if err != domain.ErrProductExcluded {
		t.Errorf("expected ErrProductExcluded, got %v", err)
	}

	// Test with allowed product -> Should succeed
	items = []domain.CartItem{
		{SKU: "sku_samsung", Name: "Samsung Galaxy", Price: decimal.NewFromInt(20000000), Quantity: 1, Category: "electronics"},
	}

	result, err := uc.CalculateCart(ctx, items, "NOPHONE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.VoucherApplied {
		t.Error("expected voucher to be applied for allowed product")
	}
}

func TestCalculateCart_NoVoucherCode(t *testing.T) {
	uc, _, _, _ := setupTestUsecase()
	ctx := context.Background()

	items := []domain.CartItem{
		{SKU: "SKU001", Name: "Product A", Price: decimal.NewFromInt(100000), Quantity: 1, Category: "electronics"},
	}

	result, err := uc.CalculateCart(ctx, items, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.VoucherApplied {
		t.Error("voucher should not be applied when no code provided")
	}

	expectedTotal := decimal.NewFromInt(100000)
	if !result.FinalPrice.Equal(expectedTotal) {
		t.Errorf("expected final price %s, got %s", expectedTotal, result.FinalPrice)
	}
}

func TestCalculateCart_VoucherNotFound(t *testing.T) {
	uc, _, _, _ := setupTestUsecase()
	ctx := context.Background()

	items := []domain.CartItem{
		{SKU: "SKU001", Name: "Product A", Price: decimal.NewFromInt(100000), Quantity: 1, Category: "electronics"},
	}

	_, err := uc.CalculateCart(ctx, items, "INVALID_CODE")
	if err != domain.ErrVoucherNotFound {
		t.Errorf("expected ErrVoucherNotFound, got %v", err)
	}
}

// =============================================================================
// CONCURRENCY TESTS
// =============================================================================

func TestClaimVoucher_ConcurrencyStockLimit(t *testing.T) {
	uc, voucherRepo, _, cacheRepo := setupTestUsecase()
	ctx := context.Background()

	// Create voucher with stock of 50
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"FLASH50",
		campaignID,
		50, // Total stock = 50
		domain.VoucherTypePercentage,
		decimal.NewFromInt(10),
		domain.VoucherConditions{},
	)
	voucherRepo.Create(ctx, voucher)

	// Initialize stock in cache
	cacheRepo.InitializeStock(ctx, "FLASH50", 50)

	// Spawn 100 goroutines to claim
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userID := uuid.New() // Each goroutine is a different user
			err := uc.ClaimVoucher(ctx, userID, "FLASH50")
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
		}()
	}

	wg.Wait()

	// Verify exactly 50 succeeded
	if successCount != 50 {
		t.Errorf("expected exactly 50 successful claims, got %d", successCount)
	}

	if failCount != 50 {
		t.Errorf("expected exactly 50 failed claims, got %d", failCount)
	}

	// Verify remaining stock is 0
	stock, _ := cacheRepo.GetStock(ctx, "FLASH50")
	if stock != 0 {
		t.Errorf("expected stock to be 0, got %d", stock)
	}

	// Verify claimed count matches
	claimedCount := cacheRepo.GetClaimedCount("FLASH50")
	if claimedCount != 50 {
		t.Errorf("expected claimed count to be 50, got %d", claimedCount)
	}
}

func TestClaimVoucher_DoubleClaimPrevention(t *testing.T) {
	uc, voucherRepo, _, cacheRepo := setupTestUsecase()
	ctx := context.Background()

	// Create voucher with stock of 100
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"SINGLE",
		campaignID,
		100,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(10),
		domain.VoucherConditions{},
	)
	voucherRepo.Create(ctx, voucher)
	cacheRepo.InitializeStock(ctx, "SINGLE", 100)

	// Same user tries to claim multiple times
	userID := uuid.New()

	// First claim should succeed
	err := uc.ClaimVoucher(ctx, userID, "SINGLE")
	if err != nil {
		t.Fatalf("first claim should succeed: %v", err)
	}

	// Second claim should fail with "already claimed"
	err = uc.ClaimVoucher(ctx, userID, "SINGLE")
	if err != domain.ErrVoucherAlreadyClaimed {
		t.Errorf("expected ErrVoucherAlreadyClaimed, got %v", err)
	}

	// Verify stock only decreased by 1
	stock, _ := cacheRepo.GetStock(ctx, "SINGLE")
	if stock != 99 {
		t.Errorf("expected stock to be 99, got %d", stock)
	}
}

func TestClaimVoucher_ConcurrentDoubleClaimPrevention(t *testing.T) {
	uc, voucherRepo, _, cacheRepo := setupTestUsecase()
	ctx := context.Background()

	// Create voucher with stock of 100
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"RACE",
		campaignID,
		100,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(10),
		domain.VoucherConditions{},
	)
	voucherRepo.Create(ctx, voucher)
	cacheRepo.InitializeStock(ctx, "RACE", 100)

	// Same user tries to claim concurrently (race condition test)
	userID := uuid.New()
	var wg sync.WaitGroup
	var successCount int64

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := uc.ClaimVoucher(ctx, userID, "RACE")
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	// Only one should succeed
	if successCount != 1 {
		t.Errorf("expected exactly 1 successful claim for same user, got %d", successCount)
	}

	// Verify user has claimed
	claimed, _ := cacheRepo.HasUserClaimed(ctx, "RACE", userID)
	if !claimed {
		t.Error("user should be marked as having claimed")
	}
}

func TestClaimVoucher_InactiveVoucher(t *testing.T) {
	uc, voucherRepo, _, cacheRepo := setupTestUsecase()
	ctx := context.Background()

	// Create inactive voucher
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"INACTIVE",
		campaignID,
		100,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(10),
		domain.VoucherConditions{},
	)
	voucher.Status = domain.VoucherStatusInactive
	voucherRepo.Create(ctx, voucher)
	cacheRepo.InitializeStock(ctx, "INACTIVE", 100)

	userID := uuid.New()
	err := uc.ClaimVoucher(ctx, userID, "INACTIVE")
	if err != domain.ErrVoucherInactive {
		t.Errorf("expected ErrVoucherInactive, got %v", err)
	}
}

func TestClaimVoucher_OutOfStock(t *testing.T) {
	uc, voucherRepo, _, cacheRepo := setupTestUsecase()
	ctx := context.Background()

	// Create voucher with 0 stock
	campaignID := uuid.New()
	voucher := domain.NewVoucher(
		"EMPTY",
		campaignID,
		0,
		domain.VoucherTypePercentage,
		decimal.NewFromInt(10),
		domain.VoucherConditions{},
	)
	voucherRepo.Create(ctx, voucher)
	cacheRepo.InitializeStock(ctx, "EMPTY", 0)

	userID := uuid.New()
	err := uc.ClaimVoucher(ctx, userID, "EMPTY")
	if err != domain.ErrVoucherOutOfStock {
		t.Errorf("expected ErrVoucherOutOfStock, got %v", err)
	}
}
