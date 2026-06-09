package grpc

import (
	"context"
	"errors"

	"microservices/inventory/internal/model"
	"microservices/inventory/internal/service"
	"microservices/inventory/pkg/logger"
	pb "microservices/inventory/pkg/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InventoryHandler handles gRPC inventory operations
// Note: This requires generated protobuf code from inventory.proto
// For now, we'll implement the handler logic that can be connected
// once the pb package is generated

type InventoryHandler struct {
	pb.UnimplementedInventoryServiceServer
	inventoryService   service.InventoryService
	reservationService service.ReservationService
}

func NewInventoryHandler(
	inventoryService service.InventoryService,
	reservationService service.ReservationService,
) *InventoryHandler {
	return &InventoryHandler{
		inventoryService:   inventoryService,
		reservationService: reservationService,
	}
}

// InventoryToProto converts model to proto message
func InventoryToProto(item *model.InventoryItem) *pb.InventoryItem {
	return &pb.InventoryItem{
		SkuId:          item.SkuID,
		TotalStock:     int32(item.TotalStock),
		ReservedStock:  int32(item.ReservedStock),
		AvailableStock: int32(item.AvailableStock()),
		UpdatedAt:      timestamppb.New(item.UpdatedAt),
	}
}

// ReservationToProto converts model to proto message
func ReservationToProto(r *model.StockReservation) *pb.StockReservation {
	return &pb.StockReservation{
		Id:        r.ID.String(),
		OrderId:   r.OrderID,
		SkuId:     r.SkuID,
		Quantity:  int32(r.Quantity),
		Status:    string(r.Status),
		ExpiresAt: timestamppb.New(r.ExpiresAt),
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

// HandleCreateInventory handles create inventory request
func (h *InventoryHandler) HandleCreateInventory(ctx context.Context, skuID string, totalStock int32) (*model.InventoryItem, error) {
	req := &model.CreateInventoryRequest{
		SkuID:      skuID,
		TotalStock: int(totalStock),
	}

	item, err := h.inventoryService.CreateInventory(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}

	return item, nil
}

// HandleGetInventory handles get inventory request
func (h *InventoryHandler) HandleGetInventory(ctx context.Context, skuID string) (*model.InventoryItem, error) {
	item, err := h.inventoryService.GetInventory(ctx, skuID)
	if err != nil {
		return nil, mapError(err)
	}

	return item, nil
}

// HandleUpdateInventory handles update inventory request
func (h *InventoryHandler) HandleUpdateInventory(ctx context.Context, skuID string, totalStock int32) (*model.InventoryItem, error) {
	req := &model.UpdateInventoryRequest{
		TotalStock: int(totalStock),
	}

	item, err := h.inventoryService.UpdateInventory(ctx, skuID, req)
	if err != nil {
		return nil, mapError(err)
	}

	return item, nil
}

// HandleDeleteInventory handles delete inventory request
func (h *InventoryHandler) HandleDeleteInventory(ctx context.Context, skuID string) error {
	if err := h.inventoryService.DeleteInventory(ctx, skuID); err != nil {
		return mapError(err)
	}

	return nil
}

// HandleListInventory handles list inventory request
func (h *InventoryHandler) HandleListInventory(ctx context.Context, page, limit int32) ([]model.InventoryItem, int64, error) {
	filter := &model.InventoryFilter{
		Page:  int(page),
		Limit: int(limit),
	}

	items, total, err := h.inventoryService.ListInventory(ctx, filter)
	if err != nil {
		return nil, 0, mapError(err)
	}

	return items, total, nil
}

// HandleReserveStock handles reserve stock request
func (h *InventoryHandler) HandleReserveStock(ctx context.Context, orderID string, items []model.ReservationItem) ([]model.StockReservation, error) {
	req := &model.ReserveStockRequest{
		OrderID: orderID,
		Items:   items,
	}

	reservations, err := h.reservationService.ReserveStock(ctx, req)
	if err != nil {
		return nil, mapError(err)
	}

	return reservations, nil
}

// HandleConfirmStock handles confirm stock request
func (h *InventoryHandler) HandleConfirmStock(ctx context.Context, orderID string) error {
	if err := h.reservationService.ConfirmStock(ctx, orderID); err != nil {
		return mapError(err)
	}

	return nil
}

// HandleReleaseStock handles release stock request
func (h *InventoryHandler) HandleReleaseStock(ctx context.Context, orderID string) error {
	if err := h.reservationService.ReleaseStock(ctx, orderID); err != nil {
		return mapError(err)
	}

	return nil
}

// HandleSyncInventory handles sync inventory request
func (h *InventoryHandler) HandleSyncInventory(ctx context.Context, skuID string) (*model.InventoryItem, error) {
	item, err := h.inventoryService.SyncInventory(ctx, skuID)
	if err != nil {
		return nil, mapError(err)
	}

	return item, nil
}

// HandleGetReservation handles get reservation request
func (h *InventoryHandler) HandleGetReservation(ctx context.Context, orderID string) ([]model.StockReservation, error) {
	reservations, err := h.reservationService.GetReservation(ctx, orderID)
	if err != nil {
		return nil, mapError(err)
	}

	return reservations, nil
}

// HandleListReservations handles list reservations request
func (h *InventoryHandler) HandleListReservations(ctx context.Context, statusFilter string, page, limit int32) ([]model.StockReservation, int64, error) {
	filter := &model.ReservationFilter{
		Status: statusFilter,
		Page:   int(page),
		Limit:  int(limit),
	}

	reservations, total, err := h.reservationService.ListReservations(ctx, filter)
	if err != nil {
		return nil, 0, mapError(err)
	}

	return reservations, total, nil
}

func (h *InventoryHandler) CreateInventory(ctx context.Context, req *pb.CreateInventoryRequest) (*pb.CreateInventoryResponse, error) {
	item, err := h.HandleCreateInventory(ctx, req.SkuId, req.TotalStock)
	if err != nil {
		return nil, err
	}
	return &pb.CreateInventoryResponse{Inventory: InventoryToProto(item)}, nil
}

func (h *InventoryHandler) GetInventory(ctx context.Context, req *pb.GetInventoryRequest) (*pb.GetInventoryResponse, error) {
	item, err := h.HandleGetInventory(ctx, req.SkuId)
	if err != nil {
		return nil, err
	}
	return &pb.GetInventoryResponse{Inventory: InventoryToProto(item)}, nil
}

func (h *InventoryHandler) UpdateInventory(ctx context.Context, req *pb.UpdateInventoryRequest) (*pb.UpdateInventoryResponse, error) {
	item, err := h.HandleUpdateInventory(ctx, req.SkuId, req.TotalStock)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateInventoryResponse{Inventory: InventoryToProto(item)}, nil
}

func (h *InventoryHandler) DeleteInventory(ctx context.Context, req *pb.DeleteInventoryRequest) (*pb.DeleteInventoryResponse, error) {
	if err := h.HandleDeleteInventory(ctx, req.SkuId); err != nil {
		return nil, err
	}
	return &pb.DeleteInventoryResponse{Success: true}, nil
}

func (h *InventoryHandler) ListInventory(ctx context.Context, req *pb.ListInventoryRequest) (*pb.ListInventoryResponse, error) {
	items, total, err := h.HandleListInventory(ctx, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}
	result := make([]*pb.InventoryItem, len(items))
	for i := range items {
		result[i] = InventoryToProto(&items[i])
	}
	return &pb.ListInventoryResponse{Items: result, Total: total, Page: req.Page, Limit: req.Limit}, nil
}

func (h *InventoryHandler) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	items := make([]model.ReservationItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = model.ReservationItem{SkuID: item.SkuId, Quantity: int(item.Quantity)}
	}
	reservations, err := h.HandleReserveStock(ctx, req.OrderId, items)
	if err != nil {
		return nil, err
	}
	result := make([]*pb.StockReservation, len(reservations))
	for i := range reservations {
		result[i] = ReservationToProto(&reservations[i])
	}
	return &pb.ReserveStockResponse{Success: true, Reservations: result}, nil
}

func (h *InventoryHandler) ConfirmStock(ctx context.Context, req *pb.ConfirmStockRequest) (*pb.ConfirmStockResponse, error) {
	if err := h.HandleConfirmStock(ctx, req.OrderId); err != nil {
		return nil, err
	}
	return &pb.ConfirmStockResponse{Success: true}, nil
}

func (h *InventoryHandler) ReleaseStock(ctx context.Context, req *pb.ReleaseStockRequest) (*pb.ReleaseStockResponse, error) {
	if err := h.HandleReleaseStock(ctx, req.OrderId); err != nil {
		return nil, err
	}
	return &pb.ReleaseStockResponse{Success: true}, nil
}

func (h *InventoryHandler) SyncInventory(ctx context.Context, req *pb.SyncInventoryRequest) (*pb.SyncInventoryResponse, error) {
	item, err := h.HandleSyncInventory(ctx, req.SkuId)
	if err != nil {
		return nil, err
	}
	return &pb.SyncInventoryResponse{Success: true, Inventory: InventoryToProto(item)}, nil
}

func (h *InventoryHandler) GetReservation(ctx context.Context, req *pb.GetReservationRequest) (*pb.GetReservationResponse, error) {
	reservations, err := h.HandleGetReservation(ctx, req.OrderId)
	if err != nil {
		return nil, err
	}
	result := make([]*pb.StockReservation, len(reservations))
	for i := range reservations {
		result[i] = ReservationToProto(&reservations[i])
	}
	return &pb.GetReservationResponse{Reservations: result}, nil
}

func (h *InventoryHandler) ListReservations(ctx context.Context, req *pb.ListReservationsRequest) (*pb.ListReservationsResponse, error) {
	reservations, total, err := h.HandleListReservations(ctx, req.Status, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}
	result := make([]*pb.StockReservation, len(reservations))
	for i := range reservations {
		result[i] = ReservationToProto(&reservations[i])
	}
	return &pb.ListReservationsResponse{Reservations: result, Total: total, Page: req.Page, Limit: req.Limit}, nil
}

func mapError(err error) error {
	logger.Error("gRPC error", err)

	if errors.Is(err, model.ErrInventoryNotFound) || errors.Is(err, model.ErrReservationNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, model.ErrInvalidSkuID) || errors.Is(err, model.ErrInvalidOrderID) || errors.Is(err, model.ErrInvalidQuantity) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, model.ErrInsufficientStock) || errors.Is(err, model.ErrAlreadyConfirmed) || errors.Is(err, model.ErrAlreadyCancelled) || errors.Is(err, model.ErrInventoryAlreadyExists) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	return status.Error(codes.Internal, "internal server error")
}
