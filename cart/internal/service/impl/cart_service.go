package impl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"microservices/cart/internal/model"
	"microservices/cart/internal/repository"
)

var (
	ErrCartNotFound    = errors.New("cart not found")
	ErrItemNotFound    = errors.New("item not found in cart")
	ErrCartLimitExceed = errors.New("cart limit exceeded")
	ErrInvalidQuantity = errors.New("quantity must be at least 1")
)

// CartService implements the cart business logic with write-behind caching strategy.
type CartService struct {
	redisRepo repository.CartRedisRepository
	mongoRepo repository.CartMongoRepository
}

// NewCartService creates a new cart service.
func NewCartService(
	redisRepo repository.CartRedisRepository,
	mongoRepo repository.CartMongoRepository,
) *CartService {
	return &CartService{
		redisRepo: redisRepo,
		mongoRepo: mongoRepo,
	}
}

// GetCart retrieves the user's cart.
// Implements lazy loading: tries Redis first, falls back to MongoDB.
func (s *CartService) GetCart(ctx context.Context, userID string) (*model.CartResponse, error) {
	// Step 1: Try Redis
	items, err := s.redisRepo.GetCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart from Redis: %w", err)
	}

	if items != nil && len(items) > 0 {
		return &model.CartResponse{
			UserID:     userID,
			Items:      items,
			TotalItems: len(items),
		}, nil
	}

	// Step 2: Redis miss - lazy load from MongoDB
	cart, err := s.mongoRepo.GetCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart from MongoDB: %w", err)
	}

	if cart == nil || len(cart.Items) == 0 {
		return &model.CartResponse{
			UserID:     userID,
			Items:      []model.CartItem{},
			TotalItems: 0,
		}, nil
	}

	// Populate Redis from MongoDB
	if err := s.redisRepo.SetCartFromItems(ctx, userID, cart.Items); err != nil {
		log.Printf("Warning: failed to populate Redis from MongoDB: %v", err)
	}

	return &model.CartResponse{
		UserID:     userID,
		Items:      cart.Items,
		TotalItems: len(cart.Items),
	}, nil
}

// AddToCart adds an item to the user's cart.
// Uses write-behind strategy: immediate Redis write, async MongoDB persistence.
func (s *CartService) AddToCart(ctx context.Context, userID string, req *model.AddItemRequest) error {
	// Check max items limit
	count, err := s.redisRepo.GetItemCount(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get item count: %w", err)
	}

	// Check if item already exists
	exists, err := s.redisRepo.ItemExists(ctx, userID, req.SkuID)
	if err != nil {
		return fmt.Errorf("failed to check item existence: %w", err)
	}

	if !exists && count >= model.MaxCartItems {
		return fmt.Errorf("%w: maximum %d items allowed", ErrCartLimitExceed, model.MaxCartItems)
	}

	if exists {
		// Item exists: increment quantity
		if err := s.redisRepo.IncrementQuantity(ctx, userID, req.SkuID, req.Quantity); err != nil {
			return fmt.Errorf("failed to increment quantity: %w", err)
		}
	} else {
		// New item: add to cart
		item := model.CartItem{
			SkuID:     req.SkuID,
			Name:      req.Name,
			Price:     req.Price,
			Quantity:  req.Quantity,
			Thumbnail: req.Thumbnail,
			Selected:  req.Selected,
			AddedAt:   time.Now().Unix(),
		}
		if err := s.redisRepo.SetItem(ctx, userID, item); err != nil {
			return fmt.Errorf("failed to add item: %w", err)
		}
	}

	// Async persistence to MongoDB (fire-and-forget with recovery)
	go s.syncToMongo(userID)

	return nil
}

// RemoveItem removes an item from the user's cart.
func (s *CartService) RemoveItem(ctx context.Context, userID string, skuID string) error {
	// Remove from Redis
	if err := s.redisRepo.RemoveItem(ctx, userID, skuID); err != nil {
		return fmt.Errorf("failed to remove item from Redis: %w", err)
	}

	// Async remove from MongoDB
	go s.asyncRemoveFromMongo(userID, skuID)

	return nil
}

// UpdateQuantity updates the quantity of an item in the cart.
func (s *CartService) UpdateQuantity(ctx context.Context, userID string, skuID string, quantity int) error {
	if quantity < 1 {
		return ErrInvalidQuantity
	}

	// Get current item
	item, err := s.redisRepo.GetItem(ctx, userID, skuID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return ErrItemNotFound
	}

	// Update quantity
	item.Quantity = quantity
	if err := s.redisRepo.SetItem(ctx, userID, *item); err != nil {
		return fmt.Errorf("failed to update quantity: %w", err)
	}

	// Async sync to MongoDB
	go s.syncToMongo(userID)

	return nil
}

// UpdateSelection updates the selection status of an item.
func (s *CartService) UpdateSelection(ctx context.Context, userID string, skuID string, selected bool) error {
	item, err := s.redisRepo.GetItem(ctx, userID, skuID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return ErrItemNotFound
	}

	item.Selected = selected
	if err := s.redisRepo.SetItem(ctx, userID, *item); err != nil {
		return fmt.Errorf("failed to update selection: %w", err)
	}

	go s.syncToMongo(userID)

	return nil
}

// ClearCart removes all items from the user's cart (called after order placed).
func (s *CartService) ClearCart(ctx context.Context, userID string) error {
	// Clear Redis
	if err := s.redisRepo.ClearCart(ctx, userID); err != nil {
		return fmt.Errorf("failed to clear cart from Redis: %w", err)
	}

	// Delete from MongoDB (sync for order confirmation)
	if err := s.mongoRepo.DeleteCart(ctx, userID); err != nil {
		log.Printf("Warning: failed to delete cart from MongoDB: %v", err)
	}

	return nil
}

// GetCartSummary retrieves cart summary with total price.
func (s *CartService) GetCartSummary(ctx context.Context, userID string, selectedOnly bool) (*model.CartSummary, error) {
	cartResp, err := s.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	var totalPrice float64
	var items []model.CartItem

	for _, item := range cartResp.Items {
		if selectedOnly && !item.Selected {
			continue
		}
		items = append(items, item)
		totalPrice += item.Price * float64(item.Quantity)
	}

	return &model.CartSummary{
		UserID:       userID,
		Items:        items,
		TotalItems:   len(items),
		TotalPrice:   totalPrice,
		SelectedOnly: selectedOnly,
	}, nil
}

// syncToMongo syncs the entire cart to MongoDB.
// Uses detached context so it doesn't get cancelled when HTTP request finishes.
func (s *CartService) syncToMongo(userID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in syncToMongo: %v", r)
		}
	}()

	// Use background context (detached from request context)
	ctx := context.Background()

	items, err := s.redisRepo.GetCart(ctx, userID)
	if err != nil {
		log.Printf("Error getting cart from Redis for sync: %v", err)
		return
	}

	cart := &model.Cart{
		UserID:    userID,
		Items:     items,
		UpdatedAt: time.Now(),
	}

	if err := s.mongoRepo.UpsertCart(ctx, cart); err != nil {
		log.Printf("Error upserting cart to MongoDB: %v", err)
	}
}

// asyncRemoveFromMongo removes an item from MongoDB asynchronously.
func (s *CartService) asyncRemoveFromMongo(userID, skuID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in asyncRemoveFromMongo: %v", r)
		}
	}()

	ctx := context.Background()

	if err := s.mongoRepo.RemoveItem(ctx, userID, skuID); err != nil {
		log.Printf("Error removing item from MongoDB: %v", err)
	}
}
