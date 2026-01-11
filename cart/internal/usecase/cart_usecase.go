package usecase

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"microservices/cart/internal/adapter"
	"microservices/cart/internal/domain"
	redisrepo "microservices/cart/internal/infrastructure/redis"
)

// PriceValidationConfig holds configuration for price validation
type PriceValidationConfig struct {
	Enabled           bool    // Whether to validate prices with Catalog Service
	MaxPriceTolerance float64 // Maximum allowed price difference (e.g., 0.01 for 1%)
}

// CartUsecase handles cart business logic with write-behind caching strategy.
type CartUsecase struct {
	redisRepo       *redisrepo.CartRepository
	mongoRepo       domain.CartMongoRepository
	catalogClient   adapter.CatalogClient
	priceValidation PriceValidationConfig
}

// NewCartUsecase creates a new cart usecase.
func NewCartUsecase(redisRepo *redisrepo.CartRepository, mongoRepo domain.CartMongoRepository) *CartUsecase {
	return &CartUsecase{
		redisRepo:     redisRepo,
		mongoRepo:     mongoRepo,
		catalogClient: nil, // No price validation by default
		priceValidation: PriceValidationConfig{
			Enabled:           false,
			MaxPriceTolerance: 0.01, // 1% tolerance
		},
	}
}

// NewCartUsecaseWithCatalog creates a cart usecase with catalog client for price validation
func NewCartUsecaseWithCatalog(
	redisRepo *redisrepo.CartRepository,
	mongoRepo domain.CartMongoRepository,
	catalogClient adapter.CatalogClient,
	priceValidation PriceValidationConfig,
) *CartUsecase {
	return &CartUsecase{
		redisRepo:       redisRepo,
		mongoRepo:       mongoRepo,
		catalogClient:   catalogClient,
		priceValidation: priceValidation,
	}
}

// ErrPriceMismatch is returned when the provided price doesn't match the catalog price
var ErrPriceMismatch = fmt.Errorf("price mismatch: provided price does not match catalog price")

// AddToCart adds an item to the user's cart.
// Uses write-behind strategy: immediate Redis write, async MongoDB persistence.
func (u *CartUsecase) AddToCart(ctx context.Context, userID string, req domain.AddItemRequest) error {
	// Validate price with Catalog Service if enabled
	if u.priceValidation.Enabled && u.catalogClient != nil {
		currentPrice, err := u.catalogClient.GetProductPrice(ctx, req.SkuID)
		if err != nil {
			log.Printf("Warning: failed to validate price with catalog: %v", err)
			// Continue without validation if catalog is unavailable (graceful degradation)
		} else if currentPrice > 0 {
			// Check if prices match within tolerance
			priceDiff := math.Abs(req.Price - currentPrice)
			tolerance := currentPrice * u.priceValidation.MaxPriceTolerance
			if priceDiff > tolerance {
				return fmt.Errorf("%w: expected %.2f, got %.2f", ErrPriceMismatch, currentPrice, req.Price)
			}
			// Use the current catalog price to ensure consistency
			req.Price = currentPrice
		}
	}

	// Check max items limit
	count, err := u.redisRepo.GetItemCount(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get item count: %w", err)
	}

	// Check if item already exists
	exists, err := u.redisRepo.ItemExists(ctx, userID, req.SkuID)
	if err != nil {
		return fmt.Errorf("failed to check item existence: %w", err)
	}

	if !exists && count >= domain.MaxCartItems {
		return fmt.Errorf("cart limit exceeded: maximum %d items allowed", domain.MaxCartItems)
	}

	if exists {
		// Item exists: increment quantity
		if err := u.redisRepo.IncrementQuantity(ctx, userID, req.SkuID, req.Quantity); err != nil {
			return fmt.Errorf("failed to increment quantity: %w", err)
		}
	} else {
		// New item: add to cart
		item := domain.CartItem{
			SkuID:     req.SkuID,
			Name:      req.Name,
			Price:     req.Price,
			Quantity:  req.Quantity,
			Thumbnail: req.Thumbnail,
			Selected:  req.Selected,
			AddedAt:   time.Now().Unix(),
		}
		if err := u.redisRepo.SetItem(ctx, userID, item); err != nil {
			return fmt.Errorf("failed to add item: %w", err)
		}
	}

	// Async persistence to MongoDB (fire-and-forget with recovery)
	go u.syncToMongo(userID)

	return nil
}

// GetCart retrieves the user's cart.
// Implements lazy loading: tries Redis first, falls back to MongoDB.
func (u *CartUsecase) GetCart(ctx context.Context, userID string) (*domain.CartResponse, error) {
	// Step 1: Try Redis
	items, err := u.redisRepo.GetCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart from Redis: %w", err)
	}

	if items != nil && len(items) > 0 {
		return &domain.CartResponse{
			UserID:     userID,
			Items:      items,
			TotalItems: len(items),
		}, nil
	}

	// Step 2: Redis miss - lazy load from MongoDB
	cart, err := u.mongoRepo.GetCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart from MongoDB: %w", err)
	}

	if cart == nil || len(cart.Items) == 0 {
		return &domain.CartResponse{
			UserID:     userID,
			Items:      []domain.CartItem{},
			TotalItems: 0,
		}, nil
	}

	// Populate Redis from MongoDB
	if err := u.redisRepo.SetCartFromItems(ctx, userID, cart.Items); err != nil {
		log.Printf("Warning: failed to populate Redis from MongoDB: %v", err)
	}

	return &domain.CartResponse{
		UserID:     userID,
		Items:      cart.Items,
		TotalItems: len(cart.Items),
	}, nil
}

// RemoveItem removes an item from the user's cart.
func (u *CartUsecase) RemoveItem(ctx context.Context, userID string, skuID string) error {
	// Remove from Redis
	if err := u.redisRepo.RemoveItem(ctx, userID, skuID); err != nil {
		return fmt.Errorf("failed to remove item from Redis: %w", err)
	}

	// Async remove from MongoDB
	go u.asyncRemoveFromMongo(userID, skuID)

	return nil
}

// UpdateQuantity updates the quantity of an item in the cart.
func (u *CartUsecase) UpdateQuantity(ctx context.Context, userID string, skuID string, quantity int) error {
	if quantity < 1 {
		return fmt.Errorf("quantity must be at least 1")
	}

	// Get current item
	item, err := u.redisRepo.GetItem(ctx, userID, skuID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return fmt.Errorf("item not found in cart")
	}

	// Update quantity
	item.Quantity = quantity
	if err := u.redisRepo.SetItem(ctx, userID, *item); err != nil {
		return fmt.Errorf("failed to update quantity: %w", err)
	}

	// Async sync to MongoDB
	go u.syncToMongo(userID)

	return nil
}

// ClearCart removes all items from the user's cart (called after order placed).
func (u *CartUsecase) ClearCart(ctx context.Context, userID string) error {
	// Clear Redis
	if err := u.redisRepo.ClearCart(ctx, userID); err != nil {
		return fmt.Errorf("failed to clear cart from Redis: %w", err)
	}

	// Delete from MongoDB (sync for order confirmation)
	if err := u.mongoRepo.DeleteCart(ctx, userID); err != nil {
		log.Printf("Warning: failed to delete cart from MongoDB: %v", err)
	}

	return nil
}

// UpdateItemSelection updates the selected status of an item.
func (u *CartUsecase) UpdateItemSelection(ctx context.Context, userID string, skuID string, selected bool) error {
	item, err := u.redisRepo.GetItem(ctx, userID, skuID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return fmt.Errorf("item not found in cart")
	}

	item.Selected = selected
	if err := u.redisRepo.SetItem(ctx, userID, *item); err != nil {
		return fmt.Errorf("failed to update selection: %w", err)
	}

	go u.syncToMongo(userID)

	return nil
}

// syncToMongo syncs the entire cart to MongoDB.
// Uses detached context so it doesn't get cancelled when HTTP request finishes.
func (u *CartUsecase) syncToMongo(userID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in syncToMongo: %v", r)
		}
	}()

	// Use background context (detached from request context)
	ctx := context.Background()

	items, err := u.redisRepo.GetCart(ctx, userID)
	if err != nil {
		log.Printf("Error getting cart from Redis for sync: %v", err)
		return
	}

	cart := &domain.Cart{
		UserID: userID,
		Items:  items,
	}

	if err := u.mongoRepo.UpsertCart(ctx, cart); err != nil {
		log.Printf("Error upserting cart to MongoDB: %v", err)
	}
}

// asyncRemoveFromMongo removes an item from MongoDB asynchronously.
func (u *CartUsecase) asyncRemoveFromMongo(userID, skuID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in asyncRemoveFromMongo: %v", r)
		}
	}()

	ctx := context.Background()

	if err := u.mongoRepo.RemoveItem(ctx, userID, skuID); err != nil {
		log.Printf("Error removing item from MongoDB: %v", err)
	}
}
