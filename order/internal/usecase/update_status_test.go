package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/tafu/order-service/internal/domain"
)

func TestMarkAsPaidUseCase_Execute(t *testing.T) {
	tests := []struct {
		name        string
		orderStatus domain.OrderStatus
		wantErr     error
	}{
		{
			name:        "Success - pending order marked as paid",
			orderStatus: domain.StatusPending,
			wantErr:     nil,
		},
		{
			name:        "Fail - already paid order",
			orderStatus: domain.StatusPaid,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - cancelled order",
			orderStatus: domain.StatusCancelled,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - shipped order",
			orderStatus: domain.StatusShipped,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - completed order",
			orderStatus: domain.StatusCompleted,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderID := uuid.New()
			userID := uuid.New()

			mockRepo := &MockOrderRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
					return &domain.Order{
						ID:     orderID,
						UserID: userID,
						Status: tt.orderStatus,
					}, nil
				},
				UpdateStatusFunc: func(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
					return nil
				},
			}

			mockPublisher := &MockEventPublisher{}

			uc := NewMarkAsPaidUseCase(mockRepo, mockPublisher)
			err := uc.Execute(context.Background(), orderID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestMarkAsPaidUseCase_OrderNotFound(t *testing.T) {
	orderID := uuid.New()

	mockRepo := &MockOrderRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
			return nil, domain.ErrOrderNotFound
		},
	}

	mockPublisher := &MockEventPublisher{}

	uc := NewMarkAsPaidUseCase(mockRepo, mockPublisher)
	err := uc.Execute(context.Background(), orderID)

	if err != domain.ErrOrderNotFound {
		t.Errorf("Expected ErrOrderNotFound, got %v", err)
	}
}

func TestMarkAsShippedUseCase_Execute(t *testing.T) {
	tests := []struct {
		name        string
		orderStatus domain.OrderStatus
		wantErr     error
	}{
		{
			name:        "Success - paid order marked as shipped",
			orderStatus: domain.StatusPaid,
			wantErr:     nil,
		},
		{
			name:        "Fail - pending order",
			orderStatus: domain.StatusPending,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - cancelled order",
			orderStatus: domain.StatusCancelled,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - already shipped order",
			orderStatus: domain.StatusShipped,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - completed order",
			orderStatus: domain.StatusCompleted,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderID := uuid.New()
			userID := uuid.New()

			mockRepo := &MockOrderRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
					return &domain.Order{
						ID:     orderID,
						UserID: userID,
						Status: tt.orderStatus,
					}, nil
				},
				UpdateStatusFunc: func(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
					return nil
				},
			}

			mockPublisher := &MockEventPublisher{}

			uc := NewMarkAsShippedUseCase(mockRepo, mockPublisher)
			err := uc.Execute(context.Background(), orderID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestMarkAsShippedUseCase_OrderNotFound(t *testing.T) {
	orderID := uuid.New()

	mockRepo := &MockOrderRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
			return nil, domain.ErrOrderNotFound
		},
	}

	mockPublisher := &MockEventPublisher{}

	uc := NewMarkAsShippedUseCase(mockRepo, mockPublisher)
	err := uc.Execute(context.Background(), orderID)

	if err != domain.ErrOrderNotFound {
		t.Errorf("Expected ErrOrderNotFound, got %v", err)
	}
}

func TestMarkAsCompletedUseCase_Execute(t *testing.T) {
	tests := []struct {
		name        string
		orderStatus domain.OrderStatus
		wantErr     error
	}{
		{
			name:        "Success - shipped order marked as completed",
			orderStatus: domain.StatusShipped,
			wantErr:     nil,
		},
		{
			name:        "Fail - pending order",
			orderStatus: domain.StatusPending,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - paid order",
			orderStatus: domain.StatusPaid,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - cancelled order",
			orderStatus: domain.StatusCancelled,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
		{
			name:        "Fail - already completed order",
			orderStatus: domain.StatusCompleted,
			wantErr:     domain.ErrInvalidOrderStatusTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderID := uuid.New()
			userID := uuid.New()

			mockRepo := &MockOrderRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
					return &domain.Order{
						ID:     orderID,
						UserID: userID,
						Status: tt.orderStatus,
					}, nil
				},
				UpdateStatusFunc: func(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
					return nil
				},
			}

			mockPublisher := &MockEventPublisher{}

			uc := NewMarkAsCompletedUseCase(mockRepo, mockPublisher)
			err := uc.Execute(context.Background(), orderID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestMarkAsCompletedUseCase_OrderNotFound(t *testing.T) {
	orderID := uuid.New()

	mockRepo := &MockOrderRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
			return nil, domain.ErrOrderNotFound
		},
	}

	mockPublisher := &MockEventPublisher{}

	uc := NewMarkAsCompletedUseCase(mockRepo, mockPublisher)
	err := uc.Execute(context.Background(), orderID)

	if err != domain.ErrOrderNotFound {
		t.Errorf("Expected ErrOrderNotFound, got %v", err)
	}
}
