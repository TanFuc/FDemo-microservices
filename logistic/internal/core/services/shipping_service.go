package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"microservices/logistic/internal/core/domain"
	"microservices/logistic/internal/core/ports"
	"microservices/logistic/pkg/retry"
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

// CompareFeesRequest contains parameters for comparing fees across providers
type CompareFeesRequest struct {
	FromDistrictID int                   `json:"from_district_id"`
	ToDistrictID   int                   `json:"to_district_id"`
	WeightGram     int                   `json:"weight_gram"`
	InsuranceValue int                   `json:"insurance_value"`
	Providers      []domain.ProviderName `json:"providers,omitempty"`
}

// FeeQuote represents a fee quote from a single provider
type FeeQuote struct {
	Provider  domain.ProviderName `json:"provider"`
	Fee       float64             `json:"fee,omitempty"`
	FromCache bool                `json:"from_cache"`
	Error     string              `json:"error,omitempty"`
}

// CompareFeesResponse contains fee quotes from all requested providers
type CompareFeesResponse struct {
	Quotes   []FeeQuote          `json:"quotes"`
	Cheapest domain.ProviderName `json:"cheapest,omitempty"`
}

// CompareFees compares shipping fees across multiple providers concurrently
func (s *ShippingService) CompareFees(ctx context.Context, req *CompareFeesRequest) (*CompareFeesResponse, error) {
	// If no providers specified, use all registered providers
	providers := req.Providers
	if len(providers) == 0 {
		providers = s.providerFactory.ListProviders()
	}

	// Create channels for results
	quotes := make([]FeeQuote, len(providers))
	var wg sync.WaitGroup

	// Query all providers concurrently
	for i, providerName := range providers {
		wg.Add(1)
		go func(idx int, pName domain.ProviderName) {
			defer wg.Done()

			quote := FeeQuote{
				Provider:  pName,
				FromCache: false,
			}

			// Try cache first
			if s.cache != nil {
				fee, found, err := s.cache.GetFee(ctx, string(pName), req.FromDistrictID, req.ToDistrictID, req.WeightGram)
				if err == nil && found {
					quote.Fee = fee
					quote.FromCache = true
					quotes[idx] = quote
					return
				}
			}

			// Get provider and calculate fee
			provider, err := s.providerFactory.GetProvider(pName)
			if err != nil {
				quote.Error = err.Error()
				quotes[idx] = quote
				return
			}

			rateReq := &ports.RateRequest{
				FromDistrictID: req.FromDistrictID,
				ToDistrictID:   req.ToDistrictID,
				WeightGram:     req.WeightGram,
				InsuranceValue: req.InsuranceValue,
			}

			fee, err := provider.CalculateFee(ctx, rateReq)
			if err != nil {
				quote.Error = err.Error()
				quotes[idx] = quote
				return
			}

			quote.Fee = fee

			// Cache the result
			if s.cache != nil {
				_ = s.cache.SetFee(ctx, string(pName), req.FromDistrictID, req.ToDistrictID, req.WeightGram, fee)
			}

			quotes[idx] = quote
		}(i, providerName)
	}

	wg.Wait()

	// Find cheapest provider
	var cheapest domain.ProviderName
	var lowestFee float64 = -1

	for _, quote := range quotes {
		if quote.Error == "" && (lowestFee < 0 || quote.Fee < lowestFee) {
			lowestFee = quote.Fee
			cheapest = quote.Provider
		}
	}

	return &CompareFeesResponse{
		Quotes:   quotes,
		Cheapest: cheapest,
	}, nil
}

// CancelShipment cancels a shipment
func (s *ShippingService) CancelShipment(ctx context.Context, id uuid.UUID) error {
	// Load shipping order from repository
	order, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get shipment: %w", err)
	}
	if order == nil {
		return fmt.Errorf("shipment not found")
	}

	// Validate status - cannot cancel if already cancelled or delivered
	if order.SystemStatus == domain.StatusCancelled {
		return fmt.Errorf("shipment is already cancelled")
	}
	if order.SystemStatus == domain.StatusDelivered {
		return fmt.Errorf("cannot cancel delivered shipment")
	}

	// Get provider
	provider, err := s.providerFactory.GetProvider(order.Provider)
	if err != nil {
		return fmt.Errorf("provider not available: %w", err)
	}

	// Call provider to cancel order
	if err := provider.CancelOrder(ctx, order.TrackingCode); err != nil {
		return fmt.Errorf("failed to cancel shipment with provider: %w", err)
	}

	// Update order status
	order.SystemStatus = domain.StatusCancelled

	// Save to database
	if err := s.repository.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to update shipping order: %w", err)
	}

	// Publish NATS event
	if s.publisher != nil {
		event := &ports.StatusUpdatedEvent{
			InternalOrderID: order.InternalOrderID,
			TrackingCode:    order.TrackingCode,
			Provider:        order.Provider,
			CarrierStatus:   "cancelled",
			SystemStatus:    domain.StatusCancelled,
		}
		_ = s.publisher.PublishStatusUpdated(ctx, event)
	}

	return nil
}
