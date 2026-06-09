package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"microservices/order/internal/domain"
	"microservices/order/internal/infrastructure/messaging"
)

// MarkAsPaidUseCase handles marking an order as paid
type MarkAsPaidUseCase struct {
	orderRepo domain.OrderRepository
	publisher messaging.EventPublisher
}

// NewMarkAsPaidUseCase creates a new MarkAsPaidUseCase
func NewMarkAsPaidUseCase(
	orderRepo domain.OrderRepository,
	publisher messaging.EventPublisher,
) *MarkAsPaidUseCase {
	return &MarkAsPaidUseCase{
		orderRepo: orderRepo,
		publisher: publisher,
	}
}

// Execute marks an order as paid
func (uc *MarkAsPaidUseCase) Execute(ctx context.Context, orderID uuid.UUID) error {
	// Step 1: Get order
	order, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Step 2: Mark as paid (validates status transition)
	if err := order.MarkAsPaid(order.PaymentReference); err != nil {
		return err
	}

	// Step 3: Update order status in database
	if err := uc.orderRepo.UpdateStatus(ctx, orderID, domain.StatusPaid); err != nil {
		return err
	}

	// Step 4: Publish order paid event (best effort)
	event := &messaging.OrderPaidEvent{
		OrderID: order.ID,
		UserID:  order.UserID,
		PaidAt:  time.Now(),
	}
	_ = uc.publisher.PublishOrderPaid(ctx, event)

	return nil
}

// MarkAsShippedUseCase handles marking an order as shipped
type MarkAsShippedUseCase struct {
	orderRepo domain.OrderRepository
	publisher messaging.EventPublisher
}

// NewMarkAsShippedUseCase creates a new MarkAsShippedUseCase
func NewMarkAsShippedUseCase(
	orderRepo domain.OrderRepository,
	publisher messaging.EventPublisher,
) *MarkAsShippedUseCase {
	return &MarkAsShippedUseCase{
		orderRepo: orderRepo,
		publisher: publisher,
	}
}

// Execute marks an order as shipped
func (uc *MarkAsShippedUseCase) Execute(ctx context.Context, orderID uuid.UUID) error {
	// Step 1: Get order
	order, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Step 2: Mark as shipped (validates status transition)
	if order.Status == domain.StatusPaid {
		if err := order.Confirm(); err != nil {
			return err
		}
	}
	if err := order.MarkAsShipped(order.TrackingNumber, order.ShippingCarrier); err != nil {
		return err
	}

	// Step 3: Update order status in database
	if err := uc.orderRepo.UpdateStatus(ctx, orderID, domain.StatusShipped); err != nil {
		return err
	}

	// Step 4: Publish order shipped event (best effort)
	event := &messaging.OrderShippedEvent{
		OrderID:   order.ID,
		UserID:    order.UserID,
		ShippedAt: time.Now(),
	}
	_ = uc.publisher.PublishOrderShipped(ctx, event)

	return nil
}

// MarkAsCompletedUseCase handles marking an order as completed
type MarkAsCompletedUseCase struct {
	orderRepo domain.OrderRepository
	publisher messaging.EventPublisher
}

// NewMarkAsCompletedUseCase creates a new MarkAsCompletedUseCase
func NewMarkAsCompletedUseCase(
	orderRepo domain.OrderRepository,
	publisher messaging.EventPublisher,
) *MarkAsCompletedUseCase {
	return &MarkAsCompletedUseCase{
		orderRepo: orderRepo,
		publisher: publisher,
	}
}

// Execute marks an order as completed
func (uc *MarkAsCompletedUseCase) Execute(ctx context.Context, orderID uuid.UUID) error {
	// Step 1: Get order
	order, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Step 2: Mark as completed (validates status transition)
	if order.Status == domain.StatusShipped {
		if err := order.MarkAsDelivered(); err != nil {
			return err
		}
	}
	if err := order.Complete(); err != nil {
		return err
	}

	// Step 3: Update order status in database
	if err := uc.orderRepo.UpdateStatus(ctx, orderID, domain.StatusCompleted); err != nil {
		return err
	}

	// Step 4: Publish order completed event (best effort)
	event := &messaging.OrderCompletedEvent{
		OrderID:     order.ID,
		UserID:      order.UserID,
		CompletedAt: time.Now(),
	}
	_ = uc.publisher.PublishOrderCompleted(ctx, event)

	return nil
}
