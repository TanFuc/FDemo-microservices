package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrOrderNotCompleted    = errors.New("order is not completed")
	ErrProductNotInOrder    = errors.New("product not found in order")
	ErrUnauthorized         = errors.New("user is not authorized to access this order")
	ErrOrderServiceDown     = errors.New("order service is unavailable")
)

type OrderItem struct {
	ProductID   string
	ProductName string
	Quantity    int32
	Price       float64
}

type OrderDetail struct {
	OrderID   string
	UserID    string
	Status    string
	Items     []OrderItem
	CreatedAt string
}

type OrderServiceClient interface {
	GetOrderDetail(ctx context.Context, orderID, userID string) (*OrderDetail, error)
	Close() error
}

type orderServiceClient struct {
	conn   *grpc.ClientConn
	target string
}

func NewOrderServiceClient(target string) (OrderServiceClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &orderServiceClient{
		conn:   conn,
		target: target,
	}, nil
}

func (c *orderServiceClient) GetOrderDetail(ctx context.Context, orderID, userID string) (*OrderDetail, error) {
	// This is a placeholder implementation
	// In production, this would call the actual gRPC service
	// For now, we return an error indicating the service is not implemented
	// The actual implementation would look like:
	//
	// client := pb.NewOrderServiceClient(c.conn)
	// resp, err := client.GetOrderDetail(ctx, &pb.GetOrderDetailRequest{
	//     OrderId: orderID,
	//     UserId:  userID,
	// })
	// if err != nil {
	//     return nil, ErrOrderServiceDown
	// }
	// ...

	return nil, ErrOrderServiceDown
}

func (c *orderServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// VerifyPurchase checks if the user has purchased the product in a completed order
func VerifyPurchase(order *OrderDetail, productID, userID string) error {
	if order == nil {
		return ErrOrderNotFound
	}

	if order.UserID != userID {
		return ErrUnauthorized
	}

	// Check if order is completed or delivered
	if order.Status != "COMPLETED" && order.Status != "DELIVERED" {
		return ErrOrderNotCompleted
	}

	// Check if product is in order items
	for _, item := range order.Items {
		if item.ProductID == productID {
			return nil
		}
	}

	return ErrProductNotInOrder
}
