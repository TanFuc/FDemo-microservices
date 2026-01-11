package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tafu/order-service/internal/domain"
	"github.com/tafu/order-service/internal/infrastructure/messaging"
)

func TestCreateOrderUseCase_Execute_Success(t *testing.T) {
	// Arrange
	var createdOrder *domain.Order

	mockRepo := &MockOrderRepository{
		CreateFunc: func(ctx context.Context, order *domain.Order) error {
			createdOrder = order
			return nil
		},
	}

	reservationCount := 0
	mockStock := &MockStockReserver{
		ReserveStockFunc: func(ctx context.Context, skuID string, quantity int, orderID string) (string, error) {
			reservationCount++
			return "reservation-" + skuID, nil
		},
	}

	eventPublished := false
	mockPublisher := &MockEventPublisher{
		PublishOrderCreatedFunc: func(ctx context.Context, event *messaging.OrderCreatedEvent) error {
			eventPublished = true
			return nil
		},
	}

	uc := NewCreateOrderUseCase(mockRepo, mockStock, mockPublisher)

	req := &CreateOrderRequest{
		UserID:        uuid.New(),
		PaymentMethod: "MOMO",
		ShippingAddress: ShippingAddressDTO{
			FullName: "John Doe",
			Phone:    "0123456789",
			Address:  "123 Main St",
			District: "District 1",
			City:     "Ho Chi Minh",
			Country:  "Vietnam",
		},
		ShippingFee:    decimal.NewFromInt(30000),
		DiscountAmount: decimal.NewFromInt(10000),
		Items: []CreateOrderItemDTO{
			{
				ProductID:   "prod-1",
				SkuID:       "sku-1",
				ProductName: "Test Product",
				SkuCode:     "SKU001",
				Thumbnail:   "http://example.com/image.jpg",
				Quantity:    2,
				UnitPrice:   decimal.NewFromInt(100000),
			},
		},
	}

	// Act
	resp, err := uc.Execute(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if createdOrder == nil {
		t.Fatal("Expected order to be created in repository")
	}

	if createdOrder.Status != domain.StatusPending {
		t.Errorf("Expected status PENDING, got %s", createdOrder.Status)
	}

	if len(createdOrder.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(createdOrder.Items))
	}

	// Check totals
	expectedTotal := decimal.NewFromInt(200000) // 2 * 100000
	if !createdOrder.TotalAmount.Equal(expectedTotal) {
		t.Errorf("Expected total amount %s, got %s", expectedTotal, createdOrder.TotalAmount)
	}

	expectedFinal := decimal.NewFromInt(220000) // 200000 + 30000 - 10000
	if !createdOrder.FinalAmount.Equal(expectedFinal) {
		t.Errorf("Expected final amount %s, got %s", expectedFinal, createdOrder.FinalAmount)
	}

	if reservationCount != 1 {
		t.Errorf("Expected 1 stock reservation, got %d", reservationCount)
	}

	if !eventPublished {
		t.Error("Expected order created event to be published")
	}
}

func TestCreateOrderUseCase_Execute_OutOfStock(t *testing.T) {
	// Arrange
	var createdOrder *domain.Order

	mockRepo := &MockOrderRepository{
		CreateFunc: func(ctx context.Context, order *domain.Order) error {
			createdOrder = order
			return nil
		},
	}

	mockStock := &MockStockReserver{
		ReserveStockFunc: func(ctx context.Context, skuID string, quantity int, orderID string) (string, error) {
			return "", domain.ErrOutOfStock
		},
	}

	mockPublisher := &MockEventPublisher{}

	uc := NewCreateOrderUseCase(mockRepo, mockStock, mockPublisher)

	req := &CreateOrderRequest{
		UserID:        uuid.New(),
		PaymentMethod: "COD",
		ShippingAddress: ShippingAddressDTO{
			FullName: "John Doe",
			Phone:    "0123456789",
			Address:  "123 Main St",
			District: "District 1",
			City:     "Ho Chi Minh",
			Country:  "Vietnam",
		},
		Items: []CreateOrderItemDTO{
			{
				ProductID:   "prod-1",
				SkuID:       "sku-1",
				ProductName: "Test Product",
				SkuCode:     "SKU001",
				Quantity:    100,
				UnitPrice:   decimal.NewFromInt(100000),
			},
		},
	}

	// Act
	resp, err := uc.Execute(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != domain.ErrOutOfStock {
		t.Errorf("Expected ErrOutOfStock, got %v", err)
	}

	if resp != nil {
		t.Error("Expected nil response on error")
	}

	if createdOrder != nil {
		t.Error("Expected no order to be created when out of stock")
	}
}

func TestCreateOrderUseCase_Execute_RollbackOnSecondItemOutOfStock(t *testing.T) {
	// Arrange
	mockRepo := &MockOrderRepository{}

	reservedSkus := make([]string, 0)
	releasedSkus := make([]string, 0)

	mockStock := &MockStockReserver{
		ReserveStockFunc: func(ctx context.Context, skuID string, quantity int, orderID string) (string, error) {
			if skuID == "sku-2" {
				return "", domain.ErrOutOfStock
			}
			reservedSkus = append(reservedSkus, skuID)
			return "reservation-" + skuID, nil
		},
		ReleaseStockFunc: func(ctx context.Context, reservationID string) error {
			releasedSkus = append(releasedSkus, reservationID)
			return nil
		},
	}

	mockPublisher := &MockEventPublisher{}

	uc := NewCreateOrderUseCase(mockRepo, mockStock, mockPublisher)

	req := &CreateOrderRequest{
		UserID:        uuid.New(),
		PaymentMethod: "STRIPE",
		ShippingAddress: ShippingAddressDTO{
			FullName: "John Doe",
			Phone:    "0123456789",
			Address:  "123 Main St",
			District: "District 1",
			City:     "Ho Chi Minh",
			Country:  "Vietnam",
		},
		Items: []CreateOrderItemDTO{
			{
				ProductID:   "prod-1",
				SkuID:       "sku-1",
				ProductName: "Product 1",
				SkuCode:     "SKU001",
				Quantity:    1,
				UnitPrice:   decimal.NewFromInt(100000),
			},
			{
				ProductID:   "prod-2",
				SkuID:       "sku-2", // This will fail
				ProductName: "Product 2",
				SkuCode:     "SKU002",
				Quantity:    1,
				UnitPrice:   decimal.NewFromInt(200000),
			},
		},
	}

	// Act
	_, err := uc.Execute(context.Background(), req)

	// Assert
	if err != domain.ErrOutOfStock {
		t.Errorf("Expected ErrOutOfStock, got %v", err)
	}

	if len(reservedSkus) != 1 || reservedSkus[0] != "sku-1" {
		t.Errorf("Expected only sku-1 to be reserved, got %v", reservedSkus)
	}

	if len(releasedSkus) != 1 || releasedSkus[0] != "reservation-sku-1" {
		t.Errorf("Expected sku-1 reservation to be released, got %v", releasedSkus)
	}
}

func TestCreateOrderUseCase_Execute_InvalidUserID(t *testing.T) {
	mockRepo := &MockOrderRepository{}
	mockStock := &MockStockReserver{}
	mockPublisher := &MockEventPublisher{}

	uc := NewCreateOrderUseCase(mockRepo, mockStock, mockPublisher)

	req := &CreateOrderRequest{
		UserID:        uuid.Nil, // Invalid
		PaymentMethod: "MOMO",
		ShippingAddress: ShippingAddressDTO{
			FullName: "John Doe",
			Phone:    "0123456789",
			Address:  "123 Main St",
			District: "District 1",
			City:     "Ho Chi Minh",
			Country:  "Vietnam",
		},
		Items: []CreateOrderItemDTO{
			{
				ProductID:   "prod-1",
				SkuID:       "sku-1",
				ProductName: "Test Product",
				SkuCode:     "SKU001",
				Quantity:    1,
				UnitPrice:   decimal.NewFromInt(100000),
			},
		},
	}

	_, err := uc.Execute(context.Background(), req)

	if err != domain.ErrInvalidUserID {
		t.Errorf("Expected ErrInvalidUserID, got %v", err)
	}
}

func TestCreateOrderUseCase_Execute_EmptyItems(t *testing.T) {
	mockRepo := &MockOrderRepository{}
	mockStock := &MockStockReserver{}
	mockPublisher := &MockEventPublisher{}

	uc := NewCreateOrderUseCase(mockRepo, mockStock, mockPublisher)

	req := &CreateOrderRequest{
		UserID:        uuid.New(),
		PaymentMethod: "MOMO",
		ShippingAddress: ShippingAddressDTO{
			FullName: "John Doe",
			Phone:    "0123456789",
			Address:  "123 Main St",
			District: "District 1",
			City:     "Ho Chi Minh",
			Country:  "Vietnam",
		},
		Items: []CreateOrderItemDTO{}, // Empty
	}

	_, err := uc.Execute(context.Background(), req)

	if err != domain.ErrEmptyOrderItems {
		t.Errorf("Expected ErrEmptyOrderItems, got %v", err)
	}
}

func TestCreateOrderUseCase_Execute_InvalidPaymentMethod(t *testing.T) {
	mockRepo := &MockOrderRepository{}
	mockStock := &MockStockReserver{}
	mockPublisher := &MockEventPublisher{}

	uc := NewCreateOrderUseCase(mockRepo, mockStock, mockPublisher)

	req := &CreateOrderRequest{
		UserID:        uuid.New(),
		PaymentMethod: "INVALID",
		ShippingAddress: ShippingAddressDTO{
			FullName: "John Doe",
			Phone:    "0123456789",
			Address:  "123 Main St",
			District: "District 1",
			City:     "Ho Chi Minh",
			Country:  "Vietnam",
		},
		Items: []CreateOrderItemDTO{
			{
				ProductID:   "prod-1",
				SkuID:       "sku-1",
				ProductName: "Test Product",
				SkuCode:     "SKU001",
				Quantity:    1,
				UnitPrice:   decimal.NewFromInt(100000),
			},
		},
	}

	_, err := uc.Execute(context.Background(), req)

	if err != domain.ErrInvalidPaymentMethod {
		t.Errorf("Expected ErrInvalidPaymentMethod, got %v", err)
	}
}

func TestCreateOrderUseCase_Execute_InvalidQuantity(t *testing.T) {
	mockRepo := &MockOrderRepository{}
	mockStock := &MockStockReserver{}
	mockPublisher := &MockEventPublisher{}

	uc := NewCreateOrderUseCase(mockRepo, mockStock, mockPublisher)

	req := &CreateOrderRequest{
		UserID:        uuid.New(),
		PaymentMethod: "MOMO",
		ShippingAddress: ShippingAddressDTO{
			FullName: "John Doe",
			Phone:    "0123456789",
			Address:  "123 Main St",
			District: "District 1",
			City:     "Ho Chi Minh",
			Country:  "Vietnam",
		},
		Items: []CreateOrderItemDTO{
			{
				ProductID:   "prod-1",
				SkuID:       "sku-1",
				ProductName: "Test Product",
				SkuCode:     "SKU001",
				Quantity:    0, // Invalid
				UnitPrice:   decimal.NewFromInt(100000),
			},
		},
	}

	_, err := uc.Execute(context.Background(), req)

	if err != domain.ErrInvalidQuantity {
		t.Errorf("Expected ErrInvalidQuantity, got %v", err)
	}
}
