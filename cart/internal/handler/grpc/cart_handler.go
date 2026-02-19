package grpc

import (
	"context"

	"microservices/cart/internal/model"
	"microservices/cart/internal/service"
)

// CartHandler handles gRPC requests for cart operations.
type CartHandler struct {
	// Uncomment when pb is generated:
	// pb.UnimplementedCartServiceServer
	cartService service.CartService
}

// NewCartHandler creates a new gRPC cart handler.
func NewCartHandler(cartService service.CartService) *CartHandler {
	return &CartHandler{
		cartService: cartService,
	}
}

// GetCart retrieves the user's cart.
// When pb is generated, implement this method:
//
//	func (h *CartHandler) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.GetCartResponse, error) {
//		cart, err := h.cartService.GetCart(ctx, req.UserId)
//		if err != nil {
//			return nil, err
//		}
//
//		items := make([]*pb.CartItem, len(cart.Items))
//		for i, item := range cart.Items {
//			items[i] = &pb.CartItem{
//				SkuId:     item.SkuID,
//				Name:      item.Name,
//				Price:     item.Price,
//				Quantity:  int32(item.Quantity),
//				Thumbnail: item.Thumbnail,
//				Selected:  item.Selected,
//				AddedAt:   item.AddedAt,
//			}
//		}
//
//		return &pb.GetCartResponse{
//			UserId:     cart.UserID,
//			Items:      items,
//			TotalItems: int32(cart.TotalItems),
//		}, nil
//	}

// GetCartInternal is an internal method for service-to-service communication.
func (h *CartHandler) GetCartInternal(ctx context.Context, userID string) (*model.CartResponse, error) {
	return h.cartService.GetCart(ctx, userID)
}

// AddToCartInternal is an internal method for service-to-service communication.
func (h *CartHandler) AddToCartInternal(ctx context.Context, userID string, req *model.AddItemRequest) error {
	return h.cartService.AddToCart(ctx, userID, req)
}

// RemoveItemInternal is an internal method for service-to-service communication.
func (h *CartHandler) RemoveItemInternal(ctx context.Context, userID, skuID string) error {
	return h.cartService.RemoveItem(ctx, userID, skuID)
}

// UpdateQuantityInternal is an internal method for service-to-service communication.
func (h *CartHandler) UpdateQuantityInternal(ctx context.Context, userID, skuID string, quantity int) error {
	return h.cartService.UpdateQuantity(ctx, userID, skuID, quantity)
}

// ClearCartInternal is an internal method for service-to-service communication.
func (h *CartHandler) ClearCartInternal(ctx context.Context, userID string) error {
	return h.cartService.ClearCart(ctx, userID)
}

// GetCartSummaryInternal is an internal method for service-to-service communication.
func (h *CartHandler) GetCartSummaryInternal(ctx context.Context, userID string, selectedOnly bool) (*model.CartSummary, error) {
	return h.cartService.GetCartSummary(ctx, userID, selectedOnly)
}
