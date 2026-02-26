package usecase

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"microservices/order/internal/domain"
	"microservices/order/internal/infrastructure/grpc"
	"microservices/order/internal/infrastructure/messaging"
)

// CreateOrderUseCase handles order creation logic
type CreateOrderUseCase struct {
	orderRepo    domain.OrderRepository
	stockReserver grpc.StockReserver
	publisher    messaging.EventPublisher
}

// NewCreateOrderUseCase creates a new CreateOrderUseCase
func NewCreateOrderUseCase(
	orderRepo domain.OrderRepository,
	stockReserver grpc.StockReserver,
	publisher messaging.EventPublisher,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orderRepo:    orderRepo,
		stockReserver: stockReserver,
		publisher:    publisher,
	}
}

// Execute creates a new order
func (uc *CreateOrderUseCase) Execute(ctx context.Context, req *CreateOrderRequest) (*OrderResponse, error) {
	// Step 1: Validate request
	if err := uc.validateRequest(req); err != nil {
		return nil, err
	}

	// Step 2: Convert shipping address to JSON
	shippingJSON, err := req.ShippingAddress.ToJSON()
	if err != nil {
		return nil, domain.ErrInvalidAddress
	}

	// Step 3: Create order entity
	order := domain.NewOrder(req.UserID, req.PaymentMethod, shippingJSON)

	// Step 4: Create order items and reserve stock
	reservedItems := make([]reservationInfo, 0, len(req.Items))

	for _, itemDTO := range req.Items {
		// Reserve stock via gRPC
		reservationID, err := uc.stockReserver.ReserveStock(
			ctx,
			itemDTO.SkuID,
			itemDTO.Quantity,
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
			skuID:         itemDTO.SkuID,
		})

		// Create order item
		item := domain.NewOrderItem(
			itemDTO.ProductID,
			itemDTO.SkuID,
			itemDTO.ProductName,
			itemDTO.SkuCode,
			itemDTO.Thumbnail,
			itemDTO.Quantity,
			itemDTO.UnitPrice,
		)
		item.SetReservationID(reservationID)
		order.AddItem(*item)
	}

	// Step 5: Calculate totals
	order.CalculateTotals(req.ShippingFee, req.DiscountAmount)

	// Step 6: Persist order to database
	if err := uc.orderRepo.Create(ctx, order); err != nil {
		// Rollback: Release all reserved stock on DB failure
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

	// Event publishing is best-effort, don't fail order creation if it fails
	_ = uc.publisher.PublishOrderCreated(ctx, event)

	// Step 8: Return response
	return uc.toOrderResponse(order), nil
}

// validateRequest validates the create order request
func (uc *CreateOrderUseCase) validateRequest(req *CreateOrderRequest) error {
	if req.UserID.String() == "" || req.UserID.String() == "00000000-0000-0000-0000-000000000000" {
		return domain.ErrInvalidUserID
	}

	if len(req.Items) == 0 {
		return domain.ErrEmptyOrderItems
	}

	validPaymentMethods := map[string]bool{
		"MOMO":   true,
		"COD":    true,
		"STRIPE": true,
	}
	if !validPaymentMethods[req.PaymentMethod] {
		return domain.ErrInvalidPaymentMethod
	}

	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return domain.ErrInvalidQuantity
		}
		if item.UnitPrice.LessThanOrEqual(decimalZero) {
			return domain.ErrInvalidPrice
		}
	}

	return nil
}

// reservationInfo tracks reservation for rollback
type reservationInfo struct {
	reservationID string
	skuID         string
}

// releaseReservations releases all reserved stock (best effort)
func (uc *CreateOrderUseCase) releaseReservations(ctx context.Context, reservations []reservationInfo) {
	for _, r := range reservations {
		_ = uc.stockReserver.ReleaseStock(ctx, r.reservationID)
	}
}

// toOrderResponse converts Order entity to OrderResponse
func (uc *CreateOrderUseCase) toOrderResponse(order *domain.Order) *OrderResponse {
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

// decimalZero for comparisons
var decimalZero = decimal.Zero
