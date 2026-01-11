package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"tafu-logistic/logistics-service/internal/core/domain"
	"tafu-logistic/logistics-service/internal/core/ports"
	"tafu-logistic/logistics-service/pkg/retry"
)

// ShippingService handles shipping-related use cases
type ShippingService struct {
	providerFactory ports.ProviderFactory
	repository      ports.ShippingOrderRepository
	cache           ports.FeeCache
	publisher       ports.EventPublisher
	retryConfig     retry.Config
}

// NewShippingService creates a new shipping service
func NewShippingService(
	providerFactory ports.ProviderFactory,
	repository ports.ShippingOrderRepository,
	cache ports.FeeCache,
	publisher ports.EventPublisher,
) *ShippingService {
	return &ShippingService{
		providerFactory: providerFactory,
		repository:      repository,
		cache:           cache,
		publisher:       publisher,
		retryConfig:     retry.DefaultConfig(),
	}
}

// CalculateFeeRequest contains parameters for fee calculation
type CalculateFeeRequest struct {
	Provider       domain.ProviderName `json:"provider"`
	FromDistrictID int                 `json:"from_district_id"`
	ToDistrictID   int                 `json:"to_district_id"`
	WeightGram     int                 `json:"weight_gram"`
	InsuranceValue int                 `json:"insurance_value"`
}

// CalculateFeeResponse contains the calculated fee
type CalculateFeeResponse struct {
	Provider    domain.ProviderName `json:"provider"`
	Fee         float64             `json:"fee"`
	FromCache   bool                `json:"from_cache"`
}

// CalculateFee calculates shipping fee with caching
func (s *ShippingService) CalculateFee(ctx context.Context, req *CalculateFeeRequest) (*CalculateFeeResponse, error) {
	// Check cache first
	if s.cache != nil {
		fee, found, err := s.cache.GetFee(ctx, string(req.Provider), req.FromDistrictID, req.ToDistrictID, req.WeightGram)
		if err == nil && found {
			return &CalculateFeeResponse{
				Provider:  req.Provider,
				Fee:       fee,
				FromCache: true,
			}, nil
		}
	}

	// Get provider
	provider, err := s.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return nil, fmt.Errorf("provider not available: %w", err)
	}

	// Calculate fee from provider
	rateReq := &ports.RateRequest{
		FromDistrictID: req.FromDistrictID,
		ToDistrictID:   req.ToDistrictID,
		WeightGram:     req.WeightGram,
		InsuranceValue: req.InsuranceValue,
	}

	fee, err := provider.CalculateFee(ctx, rateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fee: %w", err)
	}

	// Cache the result
	if s.cache != nil {
		_ = s.cache.SetFee(ctx, string(req.Provider), req.FromDistrictID, req.ToDistrictID, req.WeightGram, fee)
	}

	return &CalculateFeeResponse{
		Provider:  req.Provider,
		Fee:       fee,
		FromCache: false,
	}, nil
}

// CreateShipmentRequest contains parameters for creating a shipment
type CreateShipmentRequest struct {
	InternalOrderID uuid.UUID           `json:"internal_order_id"`
	Provider        domain.ProviderName `json:"provider"`
	Sender          domain.ContactInfo  `json:"sender"`
	Receiver        domain.ContactInfo  `json:"receiver"`
	Parcels         []domain.Parcel     `json:"parcels"`
	IsCOD           bool                `json:"is_cod"`
	CODAmount       float64             `json:"cod_amount"`
	Note            string              `json:"note"`
}

// CreateShipmentResponse contains the created shipment details
type CreateShipmentResponse struct {
	ID           uuid.UUID           `json:"id"`
	TrackingCode string              `json:"tracking_code"`
	LabelURL     string              `json:"label_url"`
	ShippingFee  float64             `json:"shipping_fee"`
	Provider     domain.ProviderName `json:"provider"`
}

// CreateShipment creates a new shipment with retry
func (s *ShippingService) CreateShipment(ctx context.Context, req *CreateShipmentRequest) (*CreateShipmentResponse, error) {
	// Get provider
	provider, err := s.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return nil, fmt.Errorf("provider not available: %w", err)
	}

	// Build ship request
	shipReq := &ports.ShipRequest{
		InternalOrderID: req.InternalOrderID.String(),
		Sender:          req.Sender,
		Receiver:        req.Receiver,
		Parcels:         req.Parcels,
		IsCOD:           req.IsCOD,
		CODAmount:       req.CODAmount,
		Note:            req.Note,
	}

	// Create order with retry
	var shipResp *ports.ShipResponse
	err = retry.Do(ctx, s.retryConfig, func() error {
		var createErr error
		shipResp, createErr = provider.CreateOrder(ctx, shipReq)
		return createErr
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create shipment after retries: %w", err)
	}

	// Create shipping order record
	order := domain.NewShippingOrder(req.InternalOrderID, req.Provider)
	order.SetTrackingInfo(shipResp.TrackingCode, shipResp.LabelURL)
	order.SetFees(shipResp.ShippingFee, req.CODAmount)

	// Store raw response as metadata
	metadata, _ := json.Marshal(shipResp)
	order.SetMetadata(metadata)

	// Save to database
	if err := s.repository.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save shipping order: %w", err)
	}

	// Publish event
	if s.publisher != nil {
		event := &ports.ShipmentCreatedEvent{
			InternalOrderID: req.InternalOrderID,
			TrackingCode:    shipResp.TrackingCode,
			Provider:        req.Provider,
			ShippingFee:     shipResp.ShippingFee,
			LabelURL:        shipResp.LabelURL,
		}
		_ = s.publisher.PublishShipmentCreated(ctx, event)
	}

	return &CreateShipmentResponse{
		ID:           order.ID,
		TrackingCode: shipResp.TrackingCode,
		LabelURL:     shipResp.LabelURL,
		ShippingFee:  shipResp.ShippingFee,
		Provider:     req.Provider,
	}, nil
}

// GetShipment retrieves a shipment by ID
func (s *ShippingService) GetShipment(ctx context.Context, id uuid.UUID) (*domain.ShippingOrder, error) {
	order, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("shipment not found")
	}
	return order, nil
}

// GetShipmentByTrackingCode retrieves a shipment by tracking code
func (s *ShippingService) GetShipmentByTrackingCode(ctx context.Context, trackingCode string) (*domain.ShippingOrder, error) {
	order, err := s.repository.GetByTrackingCode(ctx, trackingCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("shipment not found")
	}
	return order, nil
}

// ListProviders returns all available providers
func (s *ShippingService) ListProviders() []domain.ProviderName {
	return s.providerFactory.ListProviders()
}
