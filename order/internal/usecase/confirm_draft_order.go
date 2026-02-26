package usecase

import (
	"context"
	"time"

	"microservices/order/internal/domain"
	"microservices/order/internal/infrastructure/grpc"
	"microservices/order/internal/infrastructure/messaging"
)

// ConfirmDraftOrderUseCase handles draft order confirmation and stock reservation
type ConfirmDraftOrderUseCase struct {
	orderRepo     domain.OrderRepository
	stockReserver grpc.StockReserver
	publisher     messaging.EventPublisher
}

// NewConfirmDraftOrderUseCase creates a new ConfirmDraftOrderUseCase
func NewConfirmDraftOrderUseCase(
	orderRepo domain.OrderRepository,
	stockReserver grpc.StockReserver,
	publisher messaging.EventPublisher,
) *ConfirmDraftOrderUseCase {
	return &ConfirmDraftOrderUseCase{
		orderRepo:     orderRepo,
		stockReserver: stockReserver,
		publisher:     publisher,
	}
}

// Execute confirms a draft order, reserves stock, and transitions to PENDING
func (uc *ConfirmDraftOrderUseCase) Execute(ctx context.Context, req *ConfirmDraftOrderRequest) (*OrderResponse, error) {
	// Step 1: Load draft order
	order, err := uc.orderRepo.GetByID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, domain.ErrOrderNotFound
	}

	// Step 2: Verify ownership
	if order.UserID != req.UserID {
		return nil, domain.ErrUnauthorized
	}

	// Step 3: Verify status is DRAFT
	if order.Status != domain.StatusDraft {
		return nil, domain.ErrInvalidOrderStatusTransition
	}

	// Step 4: Reserve stock for each item via gRPC inventory
	reservedItems := make([]reservationInfo, 0, len(order.Items))

	for i := range order.Items {
		item := &order.Items[i]

		reservationID, err := uc.stockReserver.ReserveStock(
			ctx,
			item.SkuID,
			item.Quantity,
			order.ID.String(),
		)
		if err != nil {
			// Rollback: Release all previously reserved stock
			uc.releaseReservations(ctx, reservedItems)
			return nil, err
		}

		// Track reservation for potential rollback
		reservedItems = append(reservedItems, reservationInfo{
			reservationID: reservationID,
			skuID:         item.SkuID,
		})

		// Set reservation ID on item
		item.SetReservationID(reservationID)
	}

	// Step 5: Confirm draft - status becomes PENDING
	if err := order.ConfirmDraft(); err != nil {
		// Rollback: Release all reserved stock
		uc.releaseReservations(ctx, reservedItems)
		return nil, err
	}

	// Step 6: Update order in database
	if err := uc.orderRepo.Update(ctx, order); err != nil {
		// Rollback: Release all reserved stock
		uc.releaseReservations(ctx, reservedItems)
		return nil, err
	}

	// Step 7: Publish order created event
	event := &messaging.OrderCreatedEvent{
		OrderID:       order.ID,
		UserID:        order.UserID,
		FinalAmount:   order.FinalAmount,
		PaymentMethod: order.PaymentMethod,
		ItemCount:     len(order.Items),
		CreatedAt:     order.CreatedAt,
	}

	// Event publishing is best-effort, don't fail order confirmation if it fails
	_ = uc.publisher.PublishOrderCreated(ctx, event)

	// Step 8: Return response
	return uc.toOrderResponse(order), nil
}

// releaseReservations releases all reserved stock (best effort)
func (uc *ConfirmDraftOrderUseCase) releaseReservations(ctx context.Context, reservations []reservationInfo) {
	for _, r := range reservations {
		_ = uc.stockReserver.ReleaseStock(ctx, r.reservationID)
	}
}

// toOrderResponse converts Order entity to OrderResponse
func (uc *ConfirmDraftOrderUseCase) toOrderResponse(order *domain.Order) *OrderResponse {
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
		TotalAmount:     order.SubTotal,
		ShippingFee:     order.ShippingFee,
		DiscountAmount:  order.VoucherDiscount,
		FinalAmount:     order.FinalAmount,
		Status:          string(order.Status),
		PaymentMethod:   order.PaymentMethod,
		ShippingAddress: order.ShippingAddress,
		Items:           items,
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       order.UpdatedAt.Format(time.RFC3339),
	}
}
