package grpc

import (
	"context"
	"errors"

	"microservices/inventory/internal/model"
	"microservices/inventory/internal/service"
	"microservices/inventory/pkg/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InventoryHandler handles gRPC inventory operations
// Note: This requires generated protobuf code from inventory.proto
// For now, we'll implement the handler logic that can be connected
// once the pb package is generated

type InventoryHandler struct {
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
func InventoryToProto(item *model.InventoryItem) *InventoryItemProto {
	return &InventoryItemProto{
		SkuId:          item.SkuID,
		TotalStock:     int32(item.TotalStock),
		ReservedStock:  int32(item.ReservedStock),
		AvailableStock: int32(item.AvailableStock()),
		UpdatedAt:      timestamppb.New(item.UpdatedAt),
	}
}

// ReservationToProto converts model to proto message
func ReservationToProto(r *model.StockReservation) *StockReservationProto {
	return &StockReservationProto{
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

// Proto message types (placeholder - will be replaced by generated code)
type InventoryItemProto struct {
	SkuId          string
	TotalStock     int32
	ReservedStock  int32
	AvailableStock int32
	UpdatedAt      *timestamppb.Timestamp
}

type StockReservationProto struct {
	Id        string
	OrderId   string
	SkuId     string
	Quantity  int32
	Status    string
	ExpiresAt *timestamppb.Timestamp
	CreatedAt *timestamppb.Timestamp
	UpdatedAt *timestamppb.Timestamp
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
