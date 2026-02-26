package grpc

import (
	"context"
	"fmt"
	"time"

	"microservices/order/internal/config"
	"microservices/order/internal/domain"
	pb "microservices/order/proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		SkuId:    skuID,
		Quantity: int32(quantity),
		OrderId:  orderID,
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if !resp.Success {
		if resp.ErrorMessage != "" {
			return "", fmt.Errorf("%w: %s", domain.ErrOutOfStock, resp.ErrorMessage)
		}
		return "", domain.ErrOutOfStock
	}

	return resp.ReservationId, nil
}

// ReleaseStock releases previously reserved stock
func (c *InventoryClient) ReleaseStock(ctx context.Context, reservationID string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ReleaseStock(ctx, &pb.ReleaseStockRequest{
		ReservationId: reservationID,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if !resp.Success {
		if resp.ErrorMessage != "" {
			return fmt.Errorf("%w: %s", domain.ErrStockReleaseFailed, resp.ErrorMessage)
		}
		return domain.ErrStockReleaseFailed
	}

	return nil
}

// CheckStock checks available stock for a SKU
func (c *InventoryClient) CheckStock(ctx context.Context, skuID string) (bool, int, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.CheckStock(ctx, &pb.CheckStockRequest{
		SkuId: skuID,
	})
	if err != nil {
		return false, 0, fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	return resp.Available, int(resp.Quantity), nil
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
		if resp.ErrorMessage != "" {
			return fmt.Errorf("stock confirmation failed: %s", resp.ErrorMessage)
		}
		return fmt.Errorf("stock confirmation failed for order %s", orderID)
	}

	return nil
}

// ReleaseStockByOrderID releases all reservations for an order (after payment failed)
func (c *InventoryClient) ReleaseStockByOrderID(ctx context.Context, orderID string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ReleaseStockByOrderId(ctx, &pb.ReleaseStockByOrderIdRequest{
		OrderId: orderID,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if !resp.Success {
		if resp.ErrorMessage != "" {
			return fmt.Errorf("%w: %s", domain.ErrStockReleaseFailed, resp.ErrorMessage)
		}
		return domain.ErrStockReleaseFailed
	}

	return nil
}

// RestoreStock adds stock back to inventory (used for returns/refunds)
func (c *InventoryClient) RestoreStock(ctx context.Context, skuID string, quantity int, referenceID, referenceType, note string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.RestoreStock(ctx, &pb.RestoreStockRequest{
		SkuId:         skuID,
		Quantity:      int32(quantity),
		ReferenceId:   referenceID,
		ReferenceType: referenceType,
		Note:          note,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInventoryServiceUnavailable, err)
	}

	if !resp.Success {
		if resp.ErrorMessage != "" {
			return fmt.Errorf("stock restore failed: %s", resp.ErrorMessage)
		}
		return fmt.Errorf("stock restore failed for SKU %s", skuID)
	}

	return nil
}

// StockReserver interface for dependency injection
type StockReserver interface {
	ReserveStock(ctx context.Context, skuID string, quantity int, orderID string) (string, error)
	ReleaseStock(ctx context.Context, reservationID string) error
}

// StockConfirmer interface for payment flow
type StockConfirmer interface {
	ConfirmStock(ctx context.Context, orderID string) error
	ReleaseStockByOrderID(ctx context.Context, orderID string) error
}

// StockRestorer interface for returns/refunds
type StockRestorer interface {
	RestoreStock(ctx context.Context, skuID string, quantity int, referenceID, referenceType, note string) error
}

// Ensure InventoryClient implements StockReserver
var _ StockReserver = (*InventoryClient)(nil)

// Ensure InventoryClient implements StockConfirmer
var _ StockConfirmer = (*InventoryClient)(nil)

// Ensure InventoryClient implements StockRestorer
var _ StockRestorer = (*InventoryClient)(nil)
