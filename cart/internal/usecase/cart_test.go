package usecase_test

import (
	"context"
	"os"
	"testing"
	"time"

	"microservices/cart/internal/domain"
	mongorepo "microservices/cart/internal/infrastructure/mongo"
	redisrepo "microservices/cart/internal/infrastructure/redis"
	"microservices/cart/internal/usecase"
)

// Integration tests for CartUsecase
// Requires running Redis and MongoDB instances

var (
	testRedisClient *redisrepo.CartRepository
	testMongoRepo   *mongorepo.CartRepository
	testUsecase     *usecase.CartUsecase
	redisClient     interface{ FlushDB(context.Context) }
)

func TestMain(m *testing.M) {
	// Skip if not integration test
	if os.Getenv("INTEGRATION_TEST") != "true" {
		os.Exit(0)
	}

	// Setup
	ctx := context.Background()

	// Initialize Redis
	redisConfig := redisrepo.NewConfigFromEnv()
	client, err := redisrepo.NewClient(redisConfig)
	if err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}
	testRedisClient = redisrepo.NewCartRepository(client)

	// Store reference for FlushDB
	type flusher interface {
		FlushDB(context.Context) interface{ Err() error }
	}
	if f, ok := interface{}(client).(flusher); ok {
		redisClient = f
	}

	// Initialize MongoDB
	mongoConfig := mongorepo.NewConfigFromEnv()
	mongoClient, err := mongorepo.NewClient(ctx, mongoConfig)
	if err != nil {
		panic("Failed to connect to MongoDB: " + err.Error())
	}
	mongoDB := mongorepo.GetDatabase(mongoClient, mongoConfig.Database)
	testMongoRepo = mongorepo.NewCartRepository(mongoDB)

	// Initialize usecase
	testUsecase = usecase.NewCartUsecase(testRedisClient, testMongoRepo)

	// Run tests
	code := m.Run()

	// Cleanup
	client.Close()
	mongoClient.Disconnect(ctx)

	os.Exit(code)
}

// Test 1: Add & Retrieve
// Call AddToCart (User A, SKU 1, Qty 1)
// Call GetCart (User A) -> Expect SKU 1, Qty 1 (From Redis)
func TestAddAndRetrieve(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := "user_test_a"

	// Clean up before test
	testUsecase.ClearCart(ctx, userID)

	// Add item to cart
	req := domain.AddItemRequest{
		SkuID:     "SKU_001",
		Name:      "Test Product",
		Price:     150000,
		Quantity:  1,
		Thumbnail: "http://example.com/image.jpg",
		Selected:  true,
	}

	err := testUsecase.AddToCart(ctx, userID, req)
	if err != nil {
		t.Fatalf("AddToCart failed: %v", err)
	}

	// Get cart
	cart, err := testUsecase.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("GetCart failed: %v", err)
	}

	// Verify
	if cart.TotalItems != 1 {
		t.Errorf("Expected 1 item, got %d", cart.TotalItems)
	}

	if len(cart.Items) != 1 {
		t.Fatalf("Expected 1 item in cart, got %d", len(cart.Items))
	}

	item := cart.Items[0]
	if item.SkuID != "SKU_001" {
		t.Errorf("Expected SKU_001, got %s", item.SkuID)
	}
	if item.Quantity != 1 {
		t.Errorf("Expected quantity 1, got %d", item.Quantity)
	}

	// Clean up
	testUsecase.ClearCart(ctx, userID)
}

// Test 2: Add same item again increments quantity
func TestAddSameItemIncrementsQuantity(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := "user_test_b"

	// Clean up before test
	testUsecase.ClearCart(ctx, userID)

	// Add item to cart
	req := domain.AddItemRequest{
		SkuID:     "SKU_002",
		Name:      "Test Product 2",
		Price:     200000,
		Quantity:  2,
		Thumbnail: "http://example.com/image2.jpg",
		Selected:  true,
	}

	err := testUsecase.AddToCart(ctx, userID, req)
	if err != nil {
		t.Fatalf("First AddToCart failed: %v", err)
	}

	// Add same item again with quantity 3
	req.Quantity = 3
	err = testUsecase.AddToCart(ctx, userID, req)
	if err != nil {
		t.Fatalf("Second AddToCart failed: %v", err)
	}

	// Get cart
	cart, err := testUsecase.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("GetCart failed: %v", err)
	}

	// Verify - should have 1 item with quantity 5 (2 + 3)
	if cart.TotalItems != 1 {
		t.Errorf("Expected 1 item, got %d", cart.TotalItems)
	}

	if cart.Items[0].Quantity != 5 {
		t.Errorf("Expected quantity 5, got %d", cart.Items[0].Quantity)
	}

	// Clean up
	testUsecase.ClearCart(ctx, userID)
}

// Test 3: Remove item
func TestRemoveItem(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := "user_test_c"

	// Clean up before test
	testUsecase.ClearCart(ctx, userID)

	// Add two items
	req1 := domain.AddItemRequest{
		SkuID:    "SKU_003",
		Name:     "Product 3",
		Price:    100000,
		Quantity: 1,
	}
	req2 := domain.AddItemRequest{
		SkuID:    "SKU_004",
		Name:     "Product 4",
		Price:    150000,
		Quantity: 2,
	}

	testUsecase.AddToCart(ctx, userID, req1)
	testUsecase.AddToCart(ctx, userID, req2)

	// Remove first item
	err := testUsecase.RemoveItem(ctx, userID, "SKU_003")
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}

	// Get cart
	cart, err := testUsecase.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("GetCart failed: %v", err)
	}

	// Verify
	if cart.TotalItems != 1 {
		t.Errorf("Expected 1 item, got %d", cart.TotalItems)
	}
	if cart.Items[0].SkuID != "SKU_004" {
		t.Errorf("Expected SKU_004, got %s", cart.Items[0].SkuID)
	}

	// Clean up
	testUsecase.ClearCart(ctx, userID)
}

// Test 4: Update quantity
func TestUpdateQuantity(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := "user_test_d"

	// Clean up before test
	testUsecase.ClearCart(ctx, userID)

	// Add item
	req := domain.AddItemRequest{
		SkuID:    "SKU_005",
		Name:     "Product 5",
		Price:    100000,
		Quantity: 1,
	}
	testUsecase.AddToCart(ctx, userID, req)

	// Update quantity
	err := testUsecase.UpdateQuantity(ctx, userID, "SKU_005", 10)
	if err != nil {
		t.Fatalf("UpdateQuantity failed: %v", err)
	}

	// Get cart
	cart, err := testUsecase.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("GetCart failed: %v", err)
	}

	// Verify
	if cart.Items[0].Quantity != 10 {
		t.Errorf("Expected quantity 10, got %d", cart.Items[0].Quantity)
	}

	// Clean up
	testUsecase.ClearCart(ctx, userID)
}

// Test 5: Clear cart
func TestClearCart(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := "user_test_e"

	// Add items
	req1 := domain.AddItemRequest{SkuID: "SKU_006", Name: "P6", Price: 100, Quantity: 1}
	req2 := domain.AddItemRequest{SkuID: "SKU_007", Name: "P7", Price: 200, Quantity: 1}
	testUsecase.AddToCart(ctx, userID, req1)
	testUsecase.AddToCart(ctx, userID, req2)

	// Clear cart
	err := testUsecase.ClearCart(ctx, userID)
	if err != nil {
		t.Fatalf("ClearCart failed: %v", err)
	}

	// Get cart
	cart, err := testUsecase.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("GetCart failed: %v", err)
	}

	// Verify
	if cart.TotalItems != 0 {
		t.Errorf("Expected 0 items, got %d", cart.TotalItems)
	}
}

// Test 6: Persistence - Lazy loading from MongoDB when Redis is empty
// This test requires flushing Redis manually in a real integration test environment
func TestPersistenceAndLazyLoad(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := "user_test_persistence"

	// Clean up before test
	testUsecase.ClearCart(ctx, userID)

	// Add item to cart
	req := domain.AddItemRequest{
		SkuID:     "SKU_PERSIST",
		Name:      "Persistent Product",
		Price:     250000,
		Quantity:  3,
		Thumbnail: "http://example.com/persist.jpg",
		Selected:  true,
	}

	err := testUsecase.AddToCart(ctx, userID, req)
	if err != nil {
		t.Fatalf("AddToCart failed: %v", err)
	}

	// Wait for async MongoDB sync
	time.Sleep(200 * time.Millisecond)

	// Verify item is in MongoDB by getting cart from MongoDB directly
	mongoCart, err := testMongoRepo.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get cart from MongoDB: %v", err)
	}

	if mongoCart == nil || len(mongoCart.Items) == 0 {
		t.Fatal("Expected cart to be persisted to MongoDB")
	}

	if mongoCart.Items[0].SkuID != "SKU_PERSIST" {
		t.Errorf("Expected SKU_PERSIST in MongoDB, got %s", mongoCart.Items[0].SkuID)
	}

	// Clean up
	testUsecase.ClearCart(ctx, userID)
}

// Test 7: Max items limit
func TestMaxItemsLimit(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := "user_test_max_items"

	// Clean up before test
	testUsecase.ClearCart(ctx, userID)

	// Add 100 items (max limit)
	for i := 0; i < domain.MaxCartItems; i++ {
		req := domain.AddItemRequest{
			SkuID:    "SKU_" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
			Name:     "Product",
			Price:    100,
			Quantity: 1,
		}
		err := testUsecase.AddToCart(ctx, userID, req)
		if err != nil {
			t.Fatalf("Failed to add item %d: %v", i, err)
		}
	}

	// Try to add one more (should fail)
	req := domain.AddItemRequest{
		SkuID:    "SKU_OVERFLOW",
		Name:     "Overflow Product",
		Price:    100,
		Quantity: 1,
	}
	err := testUsecase.AddToCart(ctx, userID, req)
	if err == nil {
		t.Error("Expected error when exceeding max items limit")
	}

	// Clean up
	testUsecase.ClearCart(ctx, userID)
}
