package usecase

import (
	"context"

	"github.com/google/uuid"

	"microservices/order/internal/domain"
)

// DeleteDraftOrderUseCase handles soft-deleting draft orders
type DeleteDraftOrderUseCase struct {
	orderRepo domain.OrderRepository
}

// NewDeleteDraftOrderUseCase creates a new DeleteDraftOrderUseCase
func NewDeleteDraftOrderUseCase(orderRepo domain.OrderRepository) *DeleteDraftOrderUseCase {
	return &DeleteDraftOrderUseCase{orderRepo: orderRepo}
}

// Execute soft-deletes a draft order after verifying ownership
func (uc *DeleteDraftOrderUseCase) Execute(ctx context.Context, orderID, userID uuid.UUID) error {
	// Load order and verify ownership in one query
	order, err := uc.orderRepo.GetByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		return err // ErrOrderNotFound if not found or wrong owner
	}

	// Only DRAFT orders can be deleted this way
	if order.Status != domain.StatusDraft {
		return domain.ErrOrderCannotBeCancelled
	}

	return uc.orderRepo.SoftDeleteDraft(ctx, orderID)
}
