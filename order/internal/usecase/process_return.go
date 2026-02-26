package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"microservices/order/internal/domain"
	"microservices/order/internal/infrastructure/grpc"
)

// ProcessReturnUseCase handles return request processing
type ProcessReturnUseCase struct {
	orderRepo     domain.OrderRepository
	returnRepo    domain.ReturnRepository
	stockRestorer grpc.StockRestorer
}

// NewProcessReturnUseCase creates a new ProcessReturnUseCase
func NewProcessReturnUseCase(
	orderRepo domain.OrderRepository,
	returnRepo domain.ReturnRepository,
	stockRestorer grpc.StockRestorer,
) *ProcessReturnUseCase {
	return &ProcessReturnUseCase{
		orderRepo:     orderRepo,
		returnRepo:    returnRepo,
		stockRestorer: stockRestorer,
	}
}

// CreateReturn creates a new return request
func (uc *ProcessReturnUseCase) CreateReturn(ctx context.Context, req *CreateReturnRequest) (*ReturnResponse, error) {
	// Step 1: Get the order
	order, err := uc.orderRepo.GetByID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}

	// Step 2: Validate order belongs to user
	if order.UserID != req.UserID {
		return nil, domain.ErrUnauthorized
	}

	// Step 3: Validate order can be returned (must be delivered/completed)
	if !uc.canOrderBeReturned(order) {
		return nil, domain.ErrOrderCannotBeReturned
	}

	// Step 4: Validate return items
	orderItemMap := make(map[uuid.UUID]*domain.OrderItem)
	for i := range order.Items {
		orderItemMap[order.Items[i].ID] = &order.Items[i]
	}

	returnItems := make([]domain.ReturnItem, 0, len(req.Items))
	for _, item := range req.Items {
		orderItem, exists := orderItemMap[item.OrderItemID]
		if !exists {
			return nil, fmt.Errorf("order item %s not found in order", item.OrderItemID)
		}

		// Check quantity doesn't exceed available (original - already returned)
		availableQty := orderItem.Quantity - orderItem.ReturnedQuantity
		if item.Quantity > availableQty {
			return nil, fmt.Errorf("%w: requested %d but only %d available for item %s",
				domain.ErrInvalidRefundQuantity, item.Quantity, availableQty, item.OrderItemID)
		}

		returnItems = append(returnItems, domain.ReturnItem{
			OrderItemID: item.OrderItemID,
			Quantity:    item.Quantity,
			Reason:      item.Reason,
		})
	}

	// Step 5: Create return request
	returnType := domain.ReturnType(req.ReturnType)
	returnReason := domain.ReturnReason(req.Reason)

	returnReq, err := domain.NewReturnRequest(
		req.OrderID,
		req.UserID,
		returnType,
		returnReason,
		returnItems,
		req.CustomerReason,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create return request: %w", err)
	}

	// Set optional fields
	returnReq.CustomerNote = req.CustomerNote
	if len(req.EvidenceURLs) > 0 {
		returnReq.EvidenceURLs = strings.Join(req.EvidenceURLs, ",")
	}

	// Step 6: Persist return request
	if err := uc.returnRepo.Create(ctx, returnReq); err != nil {
		return nil, err
	}

	return uc.toReturnResponse(returnReq), nil
}

// ApproveReturn approves a return request
func (uc *ProcessReturnUseCase) ApproveReturn(ctx context.Context, req *ApproveReturnRequest) (*ReturnResponse, error) {
	// Get return request
	returnReq, err := uc.returnRepo.GetByID(ctx, req.ReturnID)
	if err != nil {
		return nil, err
	}

	// Approve
	if err := returnReq.Approve(req.ApprovedBy, req.ApprovedByID, req.Note); err != nil {
		return nil, err
	}

	// Update
	if err := uc.returnRepo.Update(ctx, returnReq); err != nil {
		return nil, err
	}

	return uc.toReturnResponse(returnReq), nil
}

// RejectReturn rejects a return request
func (uc *ProcessReturnUseCase) RejectReturn(ctx context.Context, req *RejectReturnRequest) (*ReturnResponse, error) {
	// Get return request
	returnReq, err := uc.returnRepo.GetByID(ctx, req.ReturnID)
	if err != nil {
		return nil, err
	}

	// Reject
	if err := returnReq.Reject(req.Reason, req.RejectedBy); err != nil {
		return nil, err
	}

	// Update
	if err := uc.returnRepo.Update(ctx, returnReq); err != nil {
		return nil, err
	}

	return uc.toReturnResponse(returnReq), nil
}

// CompleteReturn completes a return and restores stock to inventory
func (uc *ProcessReturnUseCase) CompleteReturn(ctx context.Context, req *CompleteReturnRequest) (*ReturnResponse, error) {
	// Step 1: Get return request
	returnReq, err := uc.returnRepo.GetByID(ctx, req.ReturnID)
	if err != nil {
		return nil, err
	}

	// Step 2: Mark as completed
	if err := returnReq.Complete(); err != nil {
		return nil, err
	}

	// Step 3: Get the order to map item IDs to SKUs
	order, err := uc.orderRepo.GetByID(ctx, returnReq.OrderID)
	if err != nil {
		return nil, err
	}

	orderItemMap := make(map[uuid.UUID]*domain.OrderItem)
	for i := range order.Items {
		orderItemMap[order.Items[i].ID] = &order.Items[i]
	}

	// Step 4: Get return items and restore stock
	returnItems, err := returnReq.GetReturnItems()
	if err != nil {
		return nil, fmt.Errorf("failed to parse return items: %w", err)
	}

	stockRestoreErrors := make([]string, 0)
	for _, item := range returnItems {
		orderItem, exists := orderItemMap[item.OrderItemID]
		if !exists {
			stockRestoreErrors = append(stockRestoreErrors,
				fmt.Sprintf("order item %s not found", item.OrderItemID))
			continue
		}

		// Restore stock via gRPC
		err := uc.stockRestorer.RestoreStock(
			ctx,
			orderItem.SkuID,
			item.Quantity,
			returnReq.ID.String(),
			"RETURN",
			fmt.Sprintf("Return %s - %s", returnReq.ReturnNumber, item.Reason),
		)
		if err != nil {
			stockRestoreErrors = append(stockRestoreErrors,
				fmt.Sprintf("failed to restore stock for SKU %s: %v", orderItem.SkuID, err))
		}
	}

	// Step 5: Mark stock as restored (even if partial failure, we track it)
	if len(stockRestoreErrors) == 0 {
		now := time.Now()
		returnReq.StockRestored = true
		returnReq.StockRestoredAt = &now
	}

	// Step 6: Update return request
	if err := uc.returnRepo.Update(ctx, returnReq); err != nil {
		return nil, err
	}

	// Step 7: Update order items' returned quantities
	for _, item := range returnItems {
		if orderItem, exists := orderItemMap[item.OrderItemID]; exists {
			orderItem.ReturnedQuantity += item.Quantity
		}
	}
	if err := uc.orderRepo.Update(ctx, order); err != nil {
		// Log error but don't fail - stock was already restored
		// In production, use outbox pattern or saga for consistency
	}

	// Return response with any errors noted
	response := uc.toReturnResponse(returnReq)
	if len(stockRestoreErrors) > 0 {
		// In production, you'd handle this differently (retry queue, alerts, etc.)
		// For now, the response still succeeds but stock_restored will be false
	}

	return response, nil
}

// GetReturn retrieves a return request by ID
func (uc *ProcessReturnUseCase) GetReturn(ctx context.Context, returnID, userID uuid.UUID) (*ReturnResponse, error) {
	returnReq, err := uc.returnRepo.GetByID(ctx, returnID)
	if err != nil {
		return nil, err
	}

	// Validate ownership (unless admin - in production, check roles)
	if returnReq.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	return uc.toReturnResponse(returnReq), nil
}

// ListReturns lists return requests with filtering
func (uc *ProcessReturnUseCase) ListReturns(ctx context.Context, req *ListReturnsRequest) (*ListReturnsResponse, error) {
	filter := domain.ReturnFilter{
		UserID:  req.UserID,
		OrderID: req.OrderID,
		Status:  req.Status,
		Page:    req.Page,
		Limit:   req.Limit,
	}

	returns, total, err := uc.returnRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	responses := make([]ReturnResponse, len(returns))
	for i, r := range returns {
		responses[i] = *uc.toReturnResponse(r)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &ListReturnsResponse{
		Returns:    responses,
		Total:      total,
		Page:       req.Page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// GetReturnsByOrderID retrieves all returns for an order
func (uc *ProcessReturnUseCase) GetReturnsByOrderID(ctx context.Context, orderID, userID uuid.UUID) ([]ReturnResponse, error) {
	// Validate order ownership
	order, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	returns, err := uc.returnRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	responses := make([]ReturnResponse, len(returns))
	for i, r := range returns {
		responses[i] = *uc.toReturnResponse(r)
	}

	return responses, nil
}

// canOrderBeReturned checks if an order can have returns created
func (uc *ProcessReturnUseCase) canOrderBeReturned(order *domain.Order) bool {
	returnableStatuses := map[domain.OrderStatus]bool{
		domain.StatusDelivered: true,
		domain.StatusCompleted: true,
	}
	return returnableStatuses[order.Status]
}

// toReturnResponse converts domain ReturnRequest to ReturnResponse
func (uc *ProcessReturnUseCase) toReturnResponse(r *domain.ReturnRequest) *ReturnResponse {
	var items []ReturnItemResponse
	if err := json.Unmarshal(r.ReturnItems, &items); err == nil {
		// Parse succeeded
	}

	// Convert domain.ReturnItem to ReturnItemResponse
	var returnItems []domain.ReturnItem
	_ = json.Unmarshal(r.ReturnItems, &returnItems)

	itemResponses := make([]ReturnItemResponse, len(returnItems))
	for i, item := range returnItems {
		itemResponses[i] = ReturnItemResponse{
			OrderItemID: item.OrderItemID,
			Quantity:    item.Quantity,
			Reason:      item.Reason,
			Condition:   item.Condition,
		}
	}

	response := &ReturnResponse{
		ID:             r.ID,
		OrderID:        r.OrderID,
		UserID:         r.UserID,
		ReturnNumber:   r.ReturnNumber,
		Type:           string(r.Type),
		Status:         string(r.Status),
		Reason:         string(r.Reason),
		Items:          itemResponses,
		CustomerReason: r.CustomerReason,
		CustomerNote:   r.CustomerNote,
		EvidenceURLs:   r.EvidenceURLs,
		SellerNote:     r.SellerNote,
		RejectionReason: r.RejectionReason,
		StockRestored:  r.StockRestored,
		RequestedAt:    r.RequestedAt.Format(time.RFC3339),
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      r.UpdatedAt.Format(time.RFC3339),
	}

	if r.ApprovedAt != nil {
		response.ApprovedAt = r.ApprovedAt.Format(time.RFC3339)
	}
	if r.RejectedAt != nil {
		response.RejectedAt = r.RejectedAt.Format(time.RFC3339)
	}
	if r.CompletedAt != nil {
		response.CompletedAt = r.CompletedAt.Format(time.RFC3339)
	}

	return response
}
