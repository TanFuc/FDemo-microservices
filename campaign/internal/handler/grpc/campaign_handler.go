package grpc

import (
	"context"

	"github.com/google/uuid"

	"microservices/campaign/internal/model"
	"microservices/campaign/internal/service"
)

// CampaignHandler handles gRPC requests for campaign operations.
type CampaignHandler struct {
	// Uncomment when pb is generated:
	// pb.UnimplementedCampaignServiceServer
	campaignService service.CampaignService
	voucherService  service.VoucherService
}

// NewCampaignHandler creates a new gRPC campaign handler.
func NewCampaignHandler(
	campaignService service.CampaignService,
	voucherService service.VoucherService,
) *CampaignHandler {
	return &CampaignHandler{
		campaignService: campaignService,
		voucherService:  voucherService,
	}
}

// GetCampaignInternal is an internal method for service-to-service communication.
func (h *CampaignHandler) GetCampaignInternal(ctx context.Context, id string) (*model.CampaignResponse, error) {
	return h.campaignService.GetCampaign(ctx, id)
}

// ListActiveCampaignsInternal is an internal method for service-to-service communication.
func (h *CampaignHandler) ListActiveCampaignsInternal(ctx context.Context) ([]*model.CampaignResponse, error) {
	return h.campaignService.ListActiveCampaigns(ctx)
}

// GetVoucherByCodeInternal is an internal method for service-to-service communication.
func (h *CampaignHandler) GetVoucherByCodeInternal(ctx context.Context, code string) (*model.VoucherResponse, error) {
	return h.voucherService.GetVoucherByCode(ctx, code)
}

// ClaimVoucherInternal is an internal method for service-to-service communication.
func (h *CampaignHandler) ClaimVoucherInternal(ctx context.Context, userID uuid.UUID, code string) error {
	return h.voucherService.ClaimVoucher(ctx, userID, code)
}

// CalculateCartInternal is an internal method for service-to-service communication.
func (h *CampaignHandler) CalculateCartInternal(ctx context.Context, items []model.CartItem, voucherCode string) (*model.CalculateCartResult, error) {
	return h.voucherService.CalculateCart(ctx, items, voucherCode)
}
