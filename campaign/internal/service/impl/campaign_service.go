package impl

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"microservices/campaign/internal/model"
	"microservices/campaign/internal/repository"
)

var (
	ErrCampaignNotFound = errors.New("campaign not found")
	ErrInvalidID        = errors.New("invalid ID format")
)

// CampaignService implements campaign business logic
type CampaignService struct {
	campaignRepo repository.CampaignRepository
}

// NewCampaignService creates a new campaign service
func NewCampaignService(campaignRepo repository.CampaignRepository) *CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
	}
}

// CreateCampaign creates a new campaign
func (s *CampaignService) CreateCampaign(ctx context.Context, req *model.CreateCampaignRequest) (*model.CampaignResponse, error) {
	campaign := model.NewCampaign(req.Name, req.StartTime, req.EndTime)

	if err := s.campaignRepo.Create(ctx, campaign); err != nil {
		return nil, err
	}

	return campaign.ToResponse(), nil
}

// GetCampaign retrieves a campaign by ID
func (s *CampaignService) GetCampaign(ctx context.Context, id string) (*model.CampaignResponse, error) {
	campaignID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, ErrCampaignNotFound
	}

	return campaign.ToResponse(), nil
}

// ListCampaigns retrieves campaigns with pagination
func (s *CampaignService) ListCampaigns(ctx context.Context, page, limit int) ([]*model.CampaignResponse, int64, error) {
	offset := (page - 1) * limit

	campaigns, total, err := s.campaignRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*model.CampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		responses[i] = campaign.ToResponse()
	}

	return responses, total, nil
}

// ListActiveCampaigns retrieves all active campaigns
func (s *CampaignService) ListActiveCampaigns(ctx context.Context) ([]*model.CampaignResponse, error) {
	campaigns, err := s.campaignRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]*model.CampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		responses[i] = campaign.ToResponse()
	}

	return responses, nil
}
