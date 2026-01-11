package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"microservices/order/internal/domain"
	"microservices/order/internal/infrastructure/grpc"
	"microservices/order/internal/infrastructure/messaging"
)

// CancelOrderUseCase handles order cancellation logic
type CancelOrderUseCase struct {
	orderRepo     domain.OrderRepository
	stockReserver grpc.StockReserver
	publisher     messaging.EventPublisher
}

// NewCancelOrderUseCase creates a new CancelOrderUseCase
func NewCancelOrderUseCase(
	orderRepo domain.OrderRepository,
	stockReserver grpc.StockReserver,
	publisher messaging.EventPublisher,
) *CancelOrderUseCase {
	return &CancelOrderUseCase{
		orderRepo:     orderRepo,
		stockReserver: stockReserver,
		publisher:     publisher,
	}
}

// Execute cancels an order
func (uc *CancelOrderUseCase) Execute(ctx context.Context, orderID uuid.UUID) error {
	// Step 1: Get order
	order, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Step 2: Check if order can be cancelled
	if !order.CanCancel() {
		return domain.ErrOrderCannotBeCancelled
	}

	// Step 3: Cancel order
	if err := order.Cancel(); err != nil {
		return err
	}

	// Step 4: Update order in database
	if err := uc.orderRepo.UpdateStatus(ctx, orderID, domain.StatusCancelled); err != nil {
		return err
	}

	// Step 5: Release stock for all items
	reservationIDs := make([]string, 0, len(order.Items))
	for _, item := range order.Items {
		if item.ReservationID != "" {
			reservationIDs = append(reservationIDs, item.ReservationID)
			// Best effort release - don't fail cancellation if release fails
			_ = uc.stockReserver.ReleaseStock(ctx, item.ReservationID)
		}
	}

	// Step 6: Publish order cancelled event
	event := &messaging.OrderCancelledEvent{
		OrderID:        order.ID,
		UserID:         order.UserID,
		ReservationIDs: reservationIDs,
		CancelledAt:    time.Now(),
	}
	_ = uc.publisher.PublishOrderCancelled(ctx, event)

	return nil
}
