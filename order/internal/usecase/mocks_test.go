package usecase

import (
	"context"

	"github.com/google/uuid"
	"microservices/order/internal/domain"
	"microservices/order/internal/infrastructure/messaging"
)

// MockOrderRepository is a mock implementation of domain.OrderRepository
type MockOrderRepository struct {
	CreateFunc       func(ctx context.Context, order *domain.Order) error
	GetByIDFunc      func(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetByUserIDFunc  func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Order, error)
	UpdateFunc       func(ctx context.Context, order *domain.Order) error
	UpdateStatusFunc func(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error
}

func (m *MockOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, order)
	}
	return nil
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockOrderRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Order, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (m *MockOrderRepository) Update(ctx context.Context, order *domain.Order) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, order)
	}
	return nil
}

func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
}

// MockStockReserver is a mock implementation of grpc.StockReserver
type MockStockReserver struct {
	ReserveStockFunc func(ctx context.Context, skuID string, quantity int, orderID string) (string, error)
	ReleaseStockFunc func(ctx context.Context, reservationID string) error
}

func (m *MockStockReserver) ReserveStock(ctx context.Context, skuID string, quantity int, orderID string) (string, error) {
	if m.ReserveStockFunc != nil {
		return m.ReserveStockFunc(ctx, skuID, quantity, orderID)
	}
	return uuid.New().String(), nil
}

func (m *MockStockReserver) ReleaseStock(ctx context.Context, reservationID string) error {
	if m.ReleaseStockFunc != nil {
		return m.ReleaseStockFunc(ctx, reservationID)
	}
	return nil
}

// MockEventPublisher is a mock implementation of messaging.EventPublisher
type MockEventPublisher struct {
	PublishOrderCreatedFunc   func(ctx context.Context, event *messaging.OrderCreatedEvent) error
	PublishOrderCancelledFunc func(ctx context.Context, event *messaging.OrderCancelledEvent) error
	PublishOrderPaidFunc      func(ctx context.Context, event *messaging.OrderPaidEvent) error
	PublishOrderShippedFunc   func(ctx context.Context, event *messaging.OrderShippedEvent) error
	PublishOrderCompletedFunc func(ctx context.Context, event *messaging.OrderCompletedEvent) error
}

func (m *MockEventPublisher) PublishOrderCreated(ctx context.Context, event *messaging.OrderCreatedEvent) error {
	if m.PublishOrderCreatedFunc != nil {
		return m.PublishOrderCreatedFunc(ctx, event)
	}
	return nil
}

func (m *MockEventPublisher) PublishOrderCancelled(ctx context.Context, event *messaging.OrderCancelledEvent) error {
	if m.PublishOrderCancelledFunc != nil {
		return m.PublishOrderCancelledFunc(ctx, event)
	}
	return nil
}

func (m *MockEventPublisher) PublishOrderPaid(ctx context.Context, event *messaging.OrderPaidEvent) error {
	if m.PublishOrderPaidFunc != nil {
		return m.PublishOrderPaidFunc(ctx, event)
	}
	return nil
}

func (m *MockEventPublisher) PublishOrderShipped(ctx context.Context, event *messaging.OrderShippedEvent) error {
	if m.PublishOrderShippedFunc != nil {
		return m.PublishOrderShippedFunc(ctx, event)
	}
	return nil
}

func (m *MockEventPublisher) PublishOrderCompleted(ctx context.Context, event *messaging.OrderCompletedEvent) error {
	if m.PublishOrderCompletedFunc != nil {
		return m.PublishOrderCompletedFunc(ctx, event)
	}
	return nil
}
