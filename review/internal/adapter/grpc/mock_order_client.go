package grpc

import (
	"context"
)

// MockOrderServiceClient is a mock implementation for testing
type MockOrderServiceClient struct {
	Orders map[string]*OrderDetail
}

func NewMockOrderServiceClient() *MockOrderServiceClient {
	return &MockOrderServiceClient{
		Orders: make(map[string]*OrderDetail),
	}
}

func (m *MockOrderServiceClient) AddOrder(order *OrderDetail) {
	m.Orders[order.OrderID] = order
}

func (m *MockOrderServiceClient) GetOrderDetail(ctx context.Context, orderID, userID string) (*OrderDetail, error) {
	order, exists := m.Orders[orderID]
	if !exists {
		return nil, ErrOrderNotFound
	}

	if order.UserID != userID {
		return nil, ErrUnauthorized
	}

	return order, nil
}

func (m *MockOrderServiceClient) Close() error {
	return nil
}
