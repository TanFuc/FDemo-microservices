package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	orderpb "microservices/review/proto/order"
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
	conn       *grpc.ClientConn
	grpcClient orderpb.OrderServiceClient
	target     string
}

func NewOrderServiceClient(target string) (OrderServiceClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &orderServiceClient{
		conn:       conn,
		grpcClient: orderpb.NewOrderServiceClient(conn),
		target:     target,
	}, nil
}

func (c *orderServiceClient) GetOrderDetail(ctx context.Context, orderID, userID string) (*OrderDetail, error) {
	resp, err := c.grpcClient.GetOrderDetail(ctx, &orderpb.GetOrderDetailRequest{
		OrderId: orderID,
		UserId:  userID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, ErrOrderNotFound
			case codes.PermissionDenied, codes.Unauthenticated:
				return nil, ErrUnauthorized
			case codes.Unavailable:
				return nil, ErrOrderServiceDown
			}
		}
		return nil, ErrOrderServiceDown
	}

	items := make([]OrderItem, len(resp.Items))
	for i, it := range resp.Items {
		items[i] = OrderItem{
			ProductID:   it.ProductId,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			Price:       it.Price,
		}
	}

	return &OrderDetail{
		OrderID:   resp.OrderId,
		UserID:    resp.UserId,
		Status:    resp.Status,
		Items:     items,
		CreatedAt: resp.CreatedAt,
	}, nil
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
