package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"microservices/order/internal/domain"
)

// GetOrderUseCase handles getting a single order
type GetOrderUseCase struct {
	orderRepo domain.OrderRepository
}

// NewGetOrderUseCase creates a new GetOrderUseCase
func NewGetOrderUseCase(orderRepo domain.OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{
		orderRepo: orderRepo,
	}
}

// Execute gets an order by ID
func (uc *GetOrderUseCase) Execute(ctx context.Context, orderID uuid.UUID) (*OrderResponse, error) {
	order, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return uc.toOrderResponse(order), nil
}

// toOrderResponse converts Order entity to OrderResponse
func (uc *GetOrderUseCase) toOrderResponse(order *domain.Order) *OrderResponse {
	items := make([]OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = OrderItemResponse{
			ID:          item.ID,
			ProductID:   item.ProductID,
			SkuID:       item.SkuID,
			ProductName: item.ProductName,
			SkuCode:     item.SkuCode,
			Thumbnail:   item.Thumbnail,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			SubTotal:    item.SubTotal,
		}
	}

	return &OrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		TotalAmount:     order.TotalAmount,
		ShippingFee:     order.ShippingFee,
		DiscountAmount:  order.DiscountAmount,
		FinalAmount:     order.FinalAmount,
		Status:          string(order.Status),
		PaymentMethod:   order.PaymentMethod,
		ShippingAddress: order.ShippingAddress,
		Items:           items,
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       order.UpdatedAt.Format(time.RFC3339),
	}
}

// ListOrdersUseCase handles listing orders for a user
type ListOrdersUseCase struct {
	orderRepo domain.OrderRepository
}

// NewListOrdersUseCase creates a new ListOrdersUseCase
func NewListOrdersUseCase(orderRepo domain.OrderRepository) *ListOrdersUseCase {
	return &ListOrdersUseCase{
		orderRepo: orderRepo,
	}
}

// Execute lists orders for a user
func (uc *ListOrdersUseCase) Execute(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*OrderResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	orders, err := uc.orderRepo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = uc.toOrderResponse(order)
	}

	return responses, nil
}

// toOrderResponse converts Order entity to OrderResponse
func (uc *ListOrdersUseCase) toOrderResponse(order *domain.Order) *OrderResponse {
	items := make([]OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = OrderItemResponse{
			ID:          item.ID,
			ProductID:   item.ProductID,
			SkuID:       item.SkuID,
			ProductName: item.ProductName,
			SkuCode:     item.SkuCode,
			Thumbnail:   item.Thumbnail,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			SubTotal:    item.SubTotal,
		}
	}

	return &OrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		TotalAmount:     order.TotalAmount,
		ShippingFee:     order.ShippingFee,
		DiscountAmount:  order.DiscountAmount,
		FinalAmount:     order.FinalAmount,
		Status:          string(order.Status),
		PaymentMethod:   order.PaymentMethod,
		ShippingAddress: order.ShippingAddress,
		Items:           items,
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       order.UpdatedAt.Format(time.RFC3339),
	}
}
