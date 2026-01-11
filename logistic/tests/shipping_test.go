package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"tafu-logistic/logistics-service/internal/adapters/providers"
	"tafu-logistic/logistics-service/internal/adapters/providers/mock"
	"tafu-logistic/logistics-service/internal/api/dto"
	apihttp "tafu-logistic/logistics-service/internal/api/http"
	"tafu-logistic/logistics-service/internal/core/domain"
	"tafu-logistic/logistics-service/internal/core/ports"
	"tafu-logistic/logistics-service/internal/core/services"
)

// MockRepository implements ports.ShippingOrderRepository for testing
type MockRepository struct {
	orders map[uuid.UUID]*domain.ShippingOrder
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		orders: make(map[uuid.UUID]*domain.ShippingOrder),
	}
}

func (r *MockRepository) Create(ctx context.Context, order *domain.ShippingOrder) error {
	r.orders[order.ID] = order
	return nil
}

func (r *MockRepository) Update(ctx context.Context, order *domain.ShippingOrder) error {
	r.orders[order.ID] = order
	return nil
}

func (r *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ShippingOrder, error) {
	return r.orders[id], nil
}

func (r *MockRepository) GetByInternalOrderID(ctx context.Context, internalOrderID uuid.UUID) (*domain.ShippingOrder, error) {
	for _, order := range r.orders {
		if order.InternalOrderID == internalOrderID {
			return order, nil
		}
	}
	return nil, nil
}

func (r *MockRepository) GetByTrackingCode(ctx context.Context, trackingCode string) (*domain.ShippingOrder, error) {
	for _, order := range r.orders {
		if order.TrackingCode == trackingCode {
			return order, nil
		}
	}
	return nil, nil
}

func (r *MockRepository) ListByStatus(ctx context.Context, status domain.SystemStatus, limit, offset int) ([]*domain.ShippingOrder, error) {
	var result []*domain.ShippingOrder
	for _, order := range r.orders {
		if order.SystemStatus == status {
			result = append(result, order)
		}
	}
	return result, nil
}

func (r *MockRepository) UpdateStatus(ctx context.Context, id uuid.UUID, carrierStatus string, systemStatus domain.SystemStatus) error {
	if order, ok := r.orders[id]; ok {
		order.CarrierStatus = carrierStatus
		order.SystemStatus = systemStatus
	}
	return nil
}

// MockCache implements ports.FeeCache for testing
type MockCache struct {
	fees map[string]float64
}

func NewMockCache() *MockCache {
	return &MockCache{
		fees: make(map[string]float64),
	}
}

func (c *MockCache) GetFee(ctx context.Context, provider string, fromDistrict, toDistrict, weight int) (float64, bool, error) {
	return 0, false, nil // Always cache miss for testing
}

func (c *MockCache) SetFee(ctx context.Context, provider string, fromDistrict, toDistrict, weight int, fee float64) error {
	return nil
}

// MockPublisher implements ports.EventPublisher for testing
type MockPublisher struct {
	shipmentCreatedEvents []*ports.ShipmentCreatedEvent
	statusUpdatedEvents   []*ports.StatusUpdatedEvent
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{}
}

func (p *MockPublisher) PublishShipmentCreated(ctx context.Context, event *ports.ShipmentCreatedEvent) error {
	p.shipmentCreatedEvents = append(p.shipmentCreatedEvents, event)
	return nil
}

func (p *MockPublisher) PublishStatusUpdated(ctx context.Context, event *ports.StatusUpdatedEvent) error {
	p.statusUpdatedEvents = append(p.statusUpdatedEvents, event)
	return nil
}

func (p *MockPublisher) Close() error {
	return nil
}

// MockProviderFactory implements ports.ProviderFactory for testing
type MockProviderFactory struct {
	mockProvider *mock.MockProvider
}

func NewMockProviderFactory() *MockProviderFactory {
	return &MockProviderFactory{
		mockProvider: mock.NewMockProvider(),
	}
}

func (f *MockProviderFactory) GetProvider(name domain.ProviderName) (ports.Provider, error) {
	return f.mockProvider, nil
}

func (f *MockProviderFactory) ListProviders() []domain.ProviderName {
	return []domain.ProviderName{domain.ProviderMock}
}

func setupTestRouter() (*gin.Engine, *MockRepository, *MockPublisher) {
	gin.SetMode(gin.TestMode)

	repo := NewMockRepository()
	cache := NewMockCache()
	publisher := NewMockPublisher()
	factory := NewMockProviderFactory()

	shippingService := services.NewShippingService(factory, repo, cache, publisher)
	webhookService := services.NewWebhookService(factory, repo, publisher)

	router := apihttp.NewRouter(shippingService, webhookService)
	return router.Engine(), repo, publisher
}

func TestCalculateFee_MockProvider(t *testing.T) {
	router, _, _ := setupTestRouter()

	reqBody := dto.CalculateFeeRequest{
		Provider:       "MOCK",
		FromDistrictID: 1,
		ToDistrictID:   2,
		WeightGram:     500,
		InsuranceValue: 100000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/calculate-fee", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success, got error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["provider"] != "MOCK" {
		t.Errorf("Expected provider MOCK, got %s", data["provider"])
	}
	if data["fee"].(float64) <= 0 {
		t.Errorf("Expected positive fee, got %f", data["fee"])
	}

	t.Logf("Calculated fee: %.2f VND", data["fee"])
}

func TestCreateShipment_MockProvider(t *testing.T) {
	router, repo, publisher := setupTestRouter()

	internalOrderID := uuid.New()

	reqBody := dto.CreateShipmentRequest{
		InternalOrderID: internalOrderID.String(),
		Provider:        "MOCK",
		Sender: dto.ContactInfo{
			Name:       "Sender Name",
			Phone:      "0901234567",
			Address:    "123 Sender St",
			WardCode:   "00001",
			DistrictID: 1,
			ProvinceID: 1,
		},
		Receiver: dto.ContactInfo{
			Name:       "Receiver Name",
			Phone:      "0907654321",
			Address:    "456 Receiver Ave",
			WardCode:   "00002",
			DistrictID: 2,
			ProvinceID: 2,
		},
		Parcels: []dto.Parcel{
			{
				Name:       "Test Item",
				Quantity:   1,
				WeightGram: 500,
				Value:      100000,
			},
		},
		IsCOD:     true,
		CODAmount: 50000,
		Note:      "Test shipment",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success, got error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	trackingCode := data["tracking_code"].(string)

	if trackingCode == "" {
		t.Error("Expected tracking code, got empty")
	}

	// Verify tracking code format
	if len(trackingCode) < 5 || trackingCode[:5] != "MOCK-" {
		t.Errorf("Expected tracking code to start with MOCK-, got %s", trackingCode)
	}

	// Verify order was saved
	if len(repo.orders) != 1 {
		t.Errorf("Expected 1 order in repo, got %d", len(repo.orders))
	}

	// Verify event was published
	if len(publisher.shipmentCreatedEvents) != 1 {
		t.Errorf("Expected 1 shipment created event, got %d", len(publisher.shipmentCreatedEvents))
	}

	t.Logf("Created shipment with tracking code: %s", trackingCode)
}

func TestHandleWebhook_MockProvider(t *testing.T) {
	router, repo, publisher := setupTestRouter()

	// First create a shipment
	internalOrderID := uuid.New()
	order := domain.NewShippingOrder(internalOrderID, domain.ProviderMock)
	order.SetTrackingInfo("MOCK-12345", "http://label.url")
	_ = repo.Create(context.Background(), order)

	// Send webhook
	webhookPayload := map[string]interface{}{
		"tracking_code":     "MOCK-12345",
		"internal_order_id": internalOrderID.String(),
		"status":            "delivered",
	}
	body, _ := json.Marshal(webhookPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/MOCK", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success, got error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["new_status"] != "DELIVERED" {
		t.Errorf("Expected new status DELIVERED, got %s", data["new_status"])
	}

	// Verify event was published
	if len(publisher.statusUpdatedEvents) != 1 {
		t.Errorf("Expected 1 status updated event, got %d", len(publisher.statusUpdatedEvents))
	}

	event := publisher.statusUpdatedEvents[0]
	if event.SystemStatus != domain.StatusDelivered {
		t.Errorf("Expected system status DELIVERED, got %s", event.SystemStatus)
	}

	t.Logf("Webhook processed: status changed to %s", data["new_status"])
}

func TestListProviders(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/providers", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success, got error")
	}

	data := resp.Data.(map[string]interface{})
	providers := data["providers"].([]interface{})

	if len(providers) == 0 {
		t.Error("Expected at least one provider")
	}

	t.Logf("Available providers: %v", providers)
}

func TestHealthCheck(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", resp["status"])
	}
}

func TestStatusMapping(t *testing.T) {
	testCases := []struct {
		provider       domain.ProviderName
		carrierStatus  string
		expectedStatus domain.SystemStatus
	}{
		{domain.ProviderGHN, "ready_to_pick", domain.StatusPending},
		{domain.ProviderGHN, "picking", domain.StatusPicking},
		{domain.ProviderGHN, "delivering", domain.StatusShipping},
		{domain.ProviderGHN, "delivered", domain.StatusDelivered},
		{domain.ProviderGHN, "return", domain.StatusReturned},
		{domain.ProviderGHN, "cancel", domain.StatusCancelled},
		{domain.ProviderGHTK, "1", domain.StatusPending},
		{domain.ProviderGHTK, "5", domain.StatusDelivered},
		{domain.ProviderGHTK, "6", domain.StatusReturned},
		{domain.ProviderMock, "delivered", domain.StatusDelivered},
		{domain.ProviderMock, "shipping", domain.StatusShipping},
	}

	for _, tc := range testCases {
		result := domain.MapCarrierStatus(tc.provider, tc.carrierStatus)
		if result != tc.expectedStatus {
			t.Errorf("Provider %s, status %s: expected %s, got %s",
				tc.provider, tc.carrierStatus, tc.expectedStatus, result)
		}
	}
}

// Integration test helper - run with: go test -v -tags=integration ./tests/...
func TestIntegration_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router, repo, publisher := setupTestRouter()

	// Step 1: Calculate fee
	t.Run("CalculateFee", func(t *testing.T) {
		reqBody := dto.CalculateFeeRequest{
			Provider:       "MOCK",
			FromDistrictID: 1,
			ToDistrictID:   2,
			WeightGram:     1000,
			InsuranceValue: 500000,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/calculate-fee", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Calculate fee failed: %s", w.Body.String())
		}
	})

	// Step 2: Create shipment
	var trackingCode string
	internalOrderID := uuid.New()

	t.Run("CreateShipment", func(t *testing.T) {
		reqBody := dto.CreateShipmentRequest{
			InternalOrderID: internalOrderID.String(),
			Provider:        "MOCK",
			Sender: dto.ContactInfo{
				Name:       "Shop ABC",
				Phone:      "0901234567",
				Address:    "123 Shop St, District 1",
				WardCode:   "00001",
				DistrictID: 1,
				ProvinceID: 79,
			},
			Receiver: dto.ContactInfo{
				Name:       "Customer XYZ",
				Phone:      "0907654321",
				Address:    "456 Customer Ave, District 7",
				WardCode:   "00002",
				DistrictID: 2,
				ProvinceID: 79,
			},
			Parcels: []dto.Parcel{
				{Name: "Product A", Quantity: 2, WeightGram: 500, Value: 250000},
			},
			IsCOD:     true,
			CODAmount: 500000,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/create", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Create shipment failed: %s", w.Body.String())
		}

		var resp dto.APIResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp.Data.(map[string]interface{})
		trackingCode = data["tracking_code"].(string)
	})

	// Step 3: Get shipment by tracking code
	t.Run("GetShipmentByTracking", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/track/"+trackingCode, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Get shipment failed: %s", w.Body.String())
		}
	})

	// Step 4: Simulate webhook for status update
	t.Run("HandleWebhook", func(t *testing.T) {
		webhookPayload := map[string]interface{}{
			"tracking_code":     trackingCode,
			"internal_order_id": internalOrderID.String(),
			"status":            "shipping",
		}
		body, _ := json.Marshal(webhookPayload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/MOCK", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Webhook failed: %s", w.Body.String())
		}
	})

	// Verify final state
	if len(repo.orders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(repo.orders))
	}

	if len(publisher.shipmentCreatedEvents) != 1 {
		t.Errorf("Expected 1 shipment created event, got %d", len(publisher.shipmentCreatedEvents))
	}

	if len(publisher.statusUpdatedEvents) != 1 {
		t.Errorf("Expected 1 status updated event, got %d", len(publisher.statusUpdatedEvents))
	}

	t.Log("Full integration flow completed successfully")
}
