package impl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"microservices/cart/internal/adapter"
	"microservices/cart/internal/model"
	"microservices/cart/internal/repository"
)

var (
	ErrCartNotFound       = errors.New("cart not found")
	ErrItemNotFound       = errors.New("item not found in cart")
	ErrCartLimitExceed    = errors.New("cart limit exceeded")
	ErrInvalidQuantity    = errors.New("quantity must be at least 1")
	ErrCartEmpty          = errors.New("cart is empty")
	ErrNoItemsSelected    = errors.New("no items selected in cart")
	ErrVoucherInvalid     = errors.New("voucher is invalid")
	ErrCheckoutFailed     = errors.New("checkout failed")
)

// CartService implements the cart business logic with write-behind caching strategy.
type CartService struct {
	redisRepo      repository.CartRedisRepository
	mongoRepo      repository.CartMongoRepository
	campaignClient adapter.CampaignClient
	orderClient    adapter.OrderClient
}

// NewCartService creates a new cart service.
func NewCartService(
	redisRepo repository.CartRedisRepository,
	mongoRepo repository.CartMongoRepository,
	campaignClient adapter.CampaignClient,
	orderClient adapter.OrderClient,
) *CartService {
	return &CartService{
		redisRepo:      redisRepo,
		mongoRepo:      mongoRepo,
		campaignClient: campaignClient,
		orderClient:    orderClient,
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

// getFullCart retrieves the full cart including voucher fields from MongoDB.
func (s *CartService) getFullCart(ctx context.Context, userID string) (*model.Cart, error) {
	// Get items from Redis first
	items, err := s.redisRepo.GetCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart from Redis: %w", err)
	}

	// Get full cart from MongoDB to include voucher fields
	cart, err := s.mongoRepo.GetCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart from MongoDB: %w", err)
	}

	if cart == nil {
		cart = &model.Cart{
			UserID: userID,
			Items:  items,
		}
	} else if items != nil && len(items) > 0 {
		// Use Redis items as source of truth for cart items
		cart.Items = items
	}

	return cart, nil
}

// ApplyVoucher validates and applies a voucher to the cart.
func (s *CartService) ApplyVoucher(ctx context.Context, userID, voucherCode string) (*model.ApplyVoucherResponse, error) {
	// Get cart with selected items
	cart, err := s.getFullCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	if cart == nil || len(cart.Items) == 0 {
		return &model.ApplyVoucherResponse{
			Success:   false,
			ErrorCode: "CART_EMPTY",
			Message:   ErrCartEmpty.Error(),
		}, ErrCartEmpty
	}

	// Filter selected items
	var selectedItems []adapter.CartItemForVoucher
	var originalTotal float64
	for _, item := range cart.Items {
		if item.Selected {
			selectedItems = append(selectedItems, adapter.CartItemForVoucher{
				SkuID:    item.SkuID,
				Name:     item.Name,
				Price:    item.Price,
				Quantity: item.Quantity,
				Category: "", // Category not available in cart model
			})
			originalTotal += item.Price * float64(item.Quantity)
		}
	}

	if len(selectedItems) == 0 {
		return &model.ApplyVoucherResponse{
			Success:   false,
			ErrorCode: "NO_ITEMS_SELECTED",
			Message:   ErrNoItemsSelected.Error(),
		}, ErrNoItemsSelected
	}

	// Validate voucher with Campaign Service
	result, err := s.campaignClient.ValidateVoucherForCart(ctx, voucherCode, userID, selectedItems)
	if err != nil {
		return &model.ApplyVoucherResponse{
			Success:       false,
			OriginalPrice: originalTotal,
			FinalPrice:    originalTotal,
			ErrorCode:     "CAMPAIGN_SERVICE_ERROR",
			Message:       fmt.Sprintf("failed to validate voucher: %v", err),
		}, fmt.Errorf("failed to validate voucher: %w", err)
	}

	if !result.Valid {
		return &model.ApplyVoucherResponse{
			Success:       false,
			VoucherCode:   voucherCode,
			OriginalPrice: originalTotal,
			FinalPrice:    originalTotal,
			ErrorCode:     result.ErrorCode,
			Message:       result.ErrorMessage,
		}, nil
	}

	// Update cart with voucher information
	cart.AppliedVoucherCode = voucherCode
	cart.DiscountAmount = result.DiscountValue
	cart.DiscountType = result.DiscountType
	cart.VoucherDiscountValue = result.DiscountAmount
	cart.VoucherID = result.VoucherID
	cart.CampaignID = result.CampaignID
	cart.UpdatedAt = time.Now()

	// Save to MongoDB
	if err := s.mongoRepo.UpsertCart(ctx, cart); err != nil {
		log.Printf("Warning: failed to save voucher to cart: %v", err)
	}

	return &model.ApplyVoucherResponse{
		Success:        true,
		VoucherCode:    voucherCode,
		OriginalPrice:  result.OriginalTotal,
		DiscountAmount: result.DiscountAmount,
		FinalPrice:     result.FinalPrice,
		DiscountType:   result.DiscountType,
		Message:        "Voucher applied successfully",
	}, nil
}

// RemoveVoucher clears the applied voucher from the cart.
func (s *CartService) RemoveVoucher(ctx context.Context, userID string) error {
	cart, err := s.getFullCart(ctx, userID)
	if err != nil {
		return err
	}

	if cart == nil {
		return nil // No cart to clear voucher from
	}

	// Clear voucher fields
	cart.AppliedVoucherCode = ""
	cart.DiscountAmount = 0
	cart.DiscountType = ""
	cart.VoucherDiscountValue = 0
	cart.VoucherID = ""
	cart.CampaignID = ""
	cart.UpdatedAt = time.Now()

	// Save to MongoDB
	if err := s.mongoRepo.UpsertCart(ctx, cart); err != nil {
		return fmt.Errorf("failed to remove voucher from cart: %w", err)
	}

	return nil
}

// GetCartSummaryWithDiscount returns cart summary including applied discount.
func (s *CartService) GetCartSummaryWithDiscount(ctx context.Context, userID string, selectedOnly bool) (*model.CartSummary, error) {
	cart, err := s.getFullCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	if cart == nil {
		return &model.CartSummary{
			UserID:       userID,
			Items:        []model.CartItem{},
			TotalItems:   0,
			SelectedOnly: selectedOnly,
		}, nil
	}

	var originalPrice float64
	var items []model.CartItem

	for _, item := range cart.Items {
		if selectedOnly && !item.Selected {
			continue
		}
		items = append(items, item)
		originalPrice += item.Price * float64(item.Quantity)
	}

	discountAmount := cart.VoucherDiscountValue
	finalPrice := originalPrice - discountAmount
	if finalPrice < 0 {
		finalPrice = 0
	}

	return &model.CartSummary{
		UserID:             userID,
		Items:              items,
		TotalItems:         len(items),
		TotalPrice:         originalPrice, // For backward compatibility
		OriginalPrice:      originalPrice,
		DiscountAmount:     discountAmount,
		FinalPrice:         finalPrice,
		AppliedVoucherCode: cart.AppliedVoucherCode,
		DiscountType:       cart.DiscountType,
		SelectedOnly:       selectedOnly,
	}, nil
}

// Checkout confirms the cart and creates a draft order in Order Service.
func (s *CartService) Checkout(ctx context.Context, userID string, req *model.CheckoutRequest) (*model.CheckoutResponse, error) {
	// Get cart summary with discount (selected items only)
	summary, err := s.GetCartSummaryWithDiscount(ctx, userID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart summary: %w", err)
	}

	if len(summary.Items) == 0 {
		return nil, ErrNoItemsSelected
	}

	// Get full cart to access voucher IDs
	cart, err := s.getFullCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// Re-validate voucher if one is applied
	if cart.AppliedVoucherCode != "" {
		var cartItems []adapter.CartItemForVoucher
		for _, item := range summary.Items {
			cartItems = append(cartItems, adapter.CartItemForVoucher{
				SkuID:    item.SkuID,
				Name:     item.Name,
				Price:    item.Price,
				Quantity: item.Quantity,
			})
		}

		result, err := s.campaignClient.ValidateVoucherForCart(ctx, cart.AppliedVoucherCode, userID, cartItems)
		if err != nil {
			log.Printf("Warning: failed to re-validate voucher during checkout: %v", err)
			// Continue without voucher if validation fails
			cart.AppliedVoucherCode = ""
			cart.VoucherDiscountValue = 0
			summary.DiscountAmount = 0
			summary.FinalPrice = summary.OriginalPrice
		} else if !result.Valid {
			// Voucher is no longer valid
			log.Printf("Voucher %s is no longer valid: %s", cart.AppliedVoucherCode, result.ErrorMessage)
			cart.AppliedVoucherCode = ""
			cart.VoucherDiscountValue = 0
			summary.DiscountAmount = 0
			summary.FinalPrice = summary.OriginalPrice
		}
	}

	// Build order items
	var orderItems []adapter.DraftOrderItem
	for _, item := range summary.Items {
		orderItems = append(orderItems, adapter.DraftOrderItem{
			SkuID:       item.SkuID,
			ProductName: item.Name,
			Thumbnail:   item.Thumbnail,
			Quantity:    item.Quantity,
			UnitPrice:   item.Price,
		})
	}

	// Create draft order request
	draftReq := &adapter.CreateDraftOrderRequest{
		UserID: userID,
		Items:  orderItems,
		ShippingAddress: adapter.DraftOrderShippingAddr{
			FullName:   req.ShippingAddress.FullName,
			Phone:      req.ShippingAddress.Phone,
			Address:    req.ShippingAddress.Address,
			Ward:       req.ShippingAddress.Ward,
			District:   req.ShippingAddress.District,
			City:       req.ShippingAddress.City,
			Country:    req.ShippingAddress.Country,
			PostalCode: req.ShippingAddress.PostalCode,
		},
		PaymentMethod:  req.PaymentMethod,
		CustomerNote:   req.CustomerNote,
		VoucherCode:    cart.AppliedVoucherCode,
		VoucherID:      cart.VoucherID,
		CampaignID:     cart.CampaignID,
		OriginalAmount: summary.OriginalPrice,
		DiscountAmount: summary.DiscountAmount,
		ShippingFee:    0, // TODO: Calculate from Logistics Service
		FinalAmount:    summary.FinalPrice,
	}

	// Call Order Service to create draft order
	draftResp, err := s.orderClient.CreateDraftOrder(ctx, draftReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCheckoutFailed, err)
	}

	// Clear selected items from cart after successful checkout
	go s.clearSelectedItems(userID, summary.Items)

	// Build checkout response
	var checkoutItems []model.CheckoutItemResponse
	for _, item := range summary.Items {
		checkoutItems = append(checkoutItems, model.CheckoutItemResponse{
			SkuID:     item.SkuID,
			Name:      item.Name,
			Price:     item.Price,
			Quantity:  item.Quantity,
			Thumbnail: item.Thumbnail,
			SubTotal:  item.Price * float64(item.Quantity),
		})
	}

	return &model.CheckoutResponse{
		DraftOrderID:   draftResp.DraftOrderID,
		OrderNumber:    draftResp.OrderNumber,
		OriginalPrice:  summary.OriginalPrice,
		DiscountAmount: summary.DiscountAmount,
		ShippingFee:    0,
		FinalAmount:    draftResp.FinalAmount,
		VoucherCode:    cart.AppliedVoucherCode,
		Status:         draftResp.Status,
		Items:          checkoutItems,
		CreatedAt:      draftResp.CreatedAt,
	}, nil
}

// clearSelectedItems removes selected items from the cart after checkout.
func (s *CartService) clearSelectedItems(userID string, items []model.CartItem) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in clearSelectedItems: %v", r)
		}
	}()

	ctx := context.Background()

	for _, item := range items {
		if err := s.redisRepo.RemoveItem(ctx, userID, item.SkuID); err != nil {
			log.Printf("Error removing item %s from Redis: %v", item.SkuID, err)
		}
		if err := s.mongoRepo.RemoveItem(ctx, userID, item.SkuID); err != nil {
			log.Printf("Error removing item %s from MongoDB: %v", item.SkuID, err)
		}
	}

	// Clear voucher fields after checkout
	cart, err := s.mongoRepo.GetCart(ctx, userID)
	if err != nil {
		log.Printf("Error getting cart after checkout: %v", err)
		return
	}

	if cart != nil {
		cart.AppliedVoucherCode = ""
		cart.DiscountAmount = 0
		cart.DiscountType = ""
		cart.VoucherDiscountValue = 0
		cart.VoucherID = ""
		cart.CampaignID = ""
		if err := s.mongoRepo.UpsertCart(ctx, cart); err != nil {
			log.Printf("Error clearing voucher from cart: %v", err)
		}
	}
}
