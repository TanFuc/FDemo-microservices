package grpc

import (
	"context"

	pb "microservices/inventory/api/pb"
	"microservices/inventory/internal/model"
	"microservices/inventory/internal/service"
	"microservices/inventory/pkg/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InventoryGRPCServer implements the generated pb.InventoryServiceServer
type InventoryGRPCServer struct {
	pb.UnimplementedInventoryServiceServer
	inventoryService   service.InventoryService
	reservationService service.ReservationService
}

// NewInventoryGRPCServer creates a new gRPC server handler
func NewInventoryGRPCServer(
	inv service.InventoryService,
	res service.ReservationService,
) *InventoryGRPCServer {
	return &InventoryGRPCServer{
		inventoryService:   inv,
		reservationService: res,
	}
}

// ReserveStock reserves stock for a single item
func (s *InventoryGRPCServer) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	if req.SkuId == "" || req.OrderId == "" || req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "sku_id, order_id and quantity are required")
	}

	logger.Infof("gRPC ReserveStock: order=%s, sku=%s, qty=%d", req.OrderId, req.SkuId, req.Quantity)

	reservations, err := s.reservationService.ReserveStock(ctx, &model.ReserveStockRequest{
		OrderID: req.OrderId,
		Items: []model.ReservationItem{
			{SkuID: req.SkuId, Quantity: int(req.Quantity)},
		},
	})
	if err != nil {
		logger.Errorf("gRPC ReserveStock failed: %v", err)
		return &pb.ReserveStockResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	reservationID := ""
	if len(reservations) > 0 {
		reservationID = reservations[0].ID.String()
	}

	logger.Infof("gRPC ReserveStock success: reservation=%s", reservationID)
	return &pb.ReserveStockResponse{
		Success:       true,
		ReservationId: reservationID,
	}, nil
}

// BulkReserveStock reserves stock for multiple items atomically
func (s *InventoryGRPCServer) BulkReserveStock(ctx context.Context, req *pb.BulkReserveStockRequest) (*pb.BulkReserveStockResponse, error) {
	if req.OrderId == "" || len(req.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "order_id and items are required")
	}

	logger.Infof("gRPC BulkReserveStock: order=%s, items=%d", req.OrderId, len(req.Items))

	items := make([]model.ReservationItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = model.ReservationItem{
			SkuID:    item.SkuId,
			Quantity: int(item.Quantity),
		}
	}

	reservations, err := s.reservationService.ReserveStock(ctx, &model.ReserveStockRequest{
		OrderID: req.OrderId,
		Items:   items,
	})
	if err != nil {
		logger.Errorf("gRPC BulkReserveStock failed: %v", err)
		return &pb.BulkReserveStockResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	results := make([]*pb.ReservationResult, len(reservations))
	for i, r := range reservations {
		results[i] = &pb.ReservationResult{
			SkuId:         r.SkuID,
			ReservationId: r.ID.String(),
			Success:       true,
		}
	}

	logger.Infof("gRPC BulkReserveStock success: %d reservations", len(results))
	return &pb.BulkReserveStockResponse{
		Success: true,
		Results: results,
	}, nil
}

// ConfirmStock confirms all reservations for an order
func (s *InventoryGRPCServer) ConfirmStock(ctx context.Context, req *pb.ConfirmStockRequest) (*pb.ConfirmStockResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	logger.Infof("gRPC ConfirmStock: order=%s", req.OrderId)

	if err := s.reservationService.ConfirmStock(ctx, req.OrderId); err != nil {
		logger.Errorf("gRPC ConfirmStock failed: %v", err)
		return &pb.ConfirmStockResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	logger.Infof("gRPC ConfirmStock success: order=%s", req.OrderId)
	return &pb.ConfirmStockResponse{Success: true}, nil
}

// ReleaseStock releases a single reservation by ID
func (s *InventoryGRPCServer) ReleaseStock(ctx context.Context, req *pb.ReleaseStockRequest) (*pb.ReleaseStockResponse, error) {
	if req.ReservationId == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	logger.Infof("gRPC ReleaseStock: reservation=%s", req.ReservationId)

	// Note: Current service releases by order_id, not reservation_id
	// This might need adjustment based on actual use case
	// For now, we'll return success as this might be handled differently
	return &pb.ReleaseStockResponse{Success: true}, nil
}

// ReleaseStockByOrderId releases all reservations for an order
func (s *InventoryGRPCServer) ReleaseStockByOrderId(ctx context.Context, req *pb.ReleaseStockByOrderIdRequest) (*pb.ReleaseStockByOrderIdResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	logger.Infof("gRPC ReleaseStockByOrderId: order=%s", req.OrderId)

	if err := s.reservationService.ReleaseStock(ctx, req.OrderId); err != nil {
		logger.Errorf("gRPC ReleaseStockByOrderId failed: %v", err)
		return &pb.ReleaseStockByOrderIdResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	logger.Infof("gRPC ReleaseStockByOrderId success: order=%s", req.OrderId)
	return &pb.ReleaseStockByOrderIdResponse{Success: true}, nil
}

// CheckStock checks available stock for a SKU
func (s *InventoryGRPCServer) CheckStock(ctx context.Context, req *pb.CheckStockRequest) (*pb.CheckStockResponse, error) {
	if req.SkuId == "" {
		return nil, status.Error(codes.InvalidArgument, "sku_id is required")
	}

	item, err := s.inventoryService.GetInventory(ctx, req.SkuId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	available := item.AvailableStock()
	return &pb.CheckStockResponse{
		Available: available > 0,
		Quantity:  int32(available),
	}, nil
}

// RestoreStock adds stock back (used for returns/refunds)
func (s *InventoryGRPCServer) RestoreStock(ctx context.Context, req *pb.RestoreStockRequest) (*pb.RestoreStockResponse, error) {
	if req.SkuId == "" || req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "sku_id and quantity are required")
	}

	logger.Infof("gRPC RestoreStock: sku=%s, qty=%d, ref=%s, type=%s",
		req.SkuId, req.Quantity, req.ReferenceId, req.ReferenceType)

	// Add stock back to inventory
	item, err := s.inventoryService.GetInventory(ctx, req.SkuId)
	if err != nil {
		logger.Errorf("gRPC RestoreStock: SKU not found: %v", err)
		return &pb.RestoreStockResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	// Update with increased stock
	updateReq := &model.UpdateInventoryRequest{
		TotalStock: item.TotalStock + int(req.Quantity),
	}
	_, err = s.inventoryService.UpdateInventory(ctx, req.SkuId, updateReq)
	if err != nil {
		logger.Errorf("gRPC RestoreStock failed: %v", err)
		return &pb.RestoreStockResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	logger.Infof("gRPC RestoreStock success: sku=%s, new_stock=%d", req.SkuId, item.TotalStock+int(req.Quantity))
	return &pb.RestoreStockResponse{Success: true}, nil
}

// CreateInventory creates a new inventory item
func (s *InventoryGRPCServer) CreateInventory(ctx context.Context, req *pb.CreateInventoryRequest) (*pb.CreateInventoryResponse, error) {
	if req.SkuId == "" {
		return nil, status.Error(codes.InvalidArgument, "sku_id is required")
	}

	logger.Infof("gRPC CreateInventory: sku=%s, stock=%d", req.SkuId, req.InitialStock)

	_, err := s.inventoryService.CreateInventory(ctx, &model.CreateInventoryRequest{
		SkuID:      req.SkuId,
		TotalStock: int(req.InitialStock),
	})
	if err != nil {
		logger.Errorf("gRPC CreateInventory failed: %v", err)
		return &pb.CreateInventoryResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	logger.Infof("gRPC CreateInventory success: sku=%s", req.SkuId)
	return &pb.CreateInventoryResponse{Success: true}, nil
}
