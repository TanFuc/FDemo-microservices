package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "microservices/inventory/pkg/pb"
	"microservices/order/internal/config"
	"microservices/order/internal/domain"
)

// InventoryClient wraps the gRPC client for Inventory Service
type InventoryClient struct {
	client  pb.InventoryServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

// NewInventoryClient creates a new InventoryClient
func NewInventoryClient(cfg *config.InventoryConfig) (*InventoryClient, error) {
	conn, err := grpc.NewClient(
		cfg.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to inventory service: %w", err)
	}

	return &InventoryClient{
		client:  pb.NewInventoryServiceClient(conn),
		conn:    conn,
		timeout: time.Duration(cfg.Timeout) * time.Second,
	}, nil
}

// Close closes the gRPC connection
func (c *InventoryClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ReserveStock reserves stock for an order item
func (c *InventoryClient) ReserveStock(ctx context.Context, skuID string, quantity int, orderID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ReserveStock(ctx, &pb.ReserveStockRequest{
		OrderId: orderID,
		Items: []*pb.ReservationItem{{
			SkuId:    skuID,
			Quantity: int32(quantity),
		}},
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if !resp.Success {
		if resp.Message != "" {
			return "", fmt.Errorf("%w: %s", domain.ErrOutOfStock, resp.Message)
		}
		return "", domain.ErrOutOfStock
	}

	if len(resp.Reservations) != 1 {
		return "", fmt.Errorf("inventory returned %d reservations for one item", len(resp.Reservations))
	}
	return resp.Reservations[0].Id, nil
}

// ReleaseStock releases all reservations for an order.
func (c *InventoryClient) ReleaseStock(ctx context.Context, orderID string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ReleaseStock(ctx, &pb.ReleaseStockRequest{
		OrderId: orderID,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if !resp.Success {
		if resp.Message != "" {
			return fmt.Errorf("%w: %s", domain.ErrStockReleaseFailed, resp.Message)
		}
		return domain.ErrStockReleaseFailed
	}

	return nil
}

// CheckStock checks available stock for a SKU
func (c *InventoryClient) CheckStock(ctx context.Context, skuID string) (bool, int, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetInventory(ctx, &pb.GetInventoryRequest{
		SkuId: skuID,
	})
	if err != nil {
		return false, 0, fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if resp.Inventory == nil {
		return false, 0, nil
	}
	quantity := int(resp.Inventory.AvailableStock)
	return quantity > 0, quantity, nil
}

// ConfirmStock confirms all reservations for an order (after payment success)
func (c *InventoryClient) ConfirmStock(ctx context.Context, orderID string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ConfirmStock(ctx, &pb.ConfirmStockRequest{
		OrderId: orderID,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if !resp.Success {
		if resp.Message != "" {
			return fmt.Errorf("stock confirmation failed: %s", resp.Message)
		}
		return fmt.Errorf("stock confirmation failed for order %s", orderID)
	}

	return nil
}

// ReleaseStockByOrderID releases all reservations for an order (after payment failed)
func (c *InventoryClient) ReleaseStockByOrderID(ctx context.Context, orderID string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ReleaseStock(ctx, &pb.ReleaseStockRequest{
		OrderId: orderID,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if !resp.Success {
		if resp.Message != "" {
			return fmt.Errorf("%w: %s", domain.ErrStockReleaseFailed, resp.Message)
		}
		return domain.ErrStockReleaseFailed
	}

	return nil
}

// StockReserver interface for dependency injection
type StockReserver interface {
	ReserveStock(ctx context.Context, skuID string, quantity int, orderID string) (string, error)
	ReleaseStock(ctx context.Context, orderID string) error
}

// StockConfirmer interface for payment flow
type StockConfirmer interface {
	ConfirmStock(ctx context.Context, orderID string) error
	ReleaseStockByOrderID(ctx context.Context, orderID string) error
}

// Ensure InventoryClient implements StockReserver
var _ StockReserver = (*InventoryClient)(nil)

// Ensure InventoryClient implements StockConfirmer
var _ StockConfirmer = (*InventoryClient)(nil)
