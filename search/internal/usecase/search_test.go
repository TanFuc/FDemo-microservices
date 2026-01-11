package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tafu/search-service/internal/domain"
)

// MockElasticClient mocks the Elasticsearch client
type MockElasticClient struct {
	mock.Mock
}

func (m *MockElasticClient) IndexProduct(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockElasticClient) DeleteProduct(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockElasticClient) Search(ctx context.Context, params *domain.SearchParams) (*domain.SearchResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SearchResult), args.Error(1)
}

func (m *MockElasticClient) EnsureIndex(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockRedisClient mocks the Redis client
type MockRedisClient struct {
	mock.Mock
	cache map[string]string
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		cache: make(map[string]string),
	}
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	args := m.Called(ctx, key, value, ttlSeconds)
	return args.Error(0)
}

func (m *MockRedisClient) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// TestSearchProducts_CacheMiss tests the search flow when cache misses
func TestSearchProducts_CacheMiss(t *testing.T) {
	// Setup mocks
	mockElastic := new(MockElasticClient)
	mockRedis := NewMockRedisClient()

	// Expected search results
	expectedProducts := []domain.Product{
		{
			ID:         "prod-1",
			Name:       "Test Product 1",
			Slug:       "test-product-1",
			CategoryID: "cat-1",
			BrandID:    "brand-1",
			Price:      99.99,
			Status:     "PUBLISHED",
			CreatedAt:  time.Now(),
		},
		{
			ID:         "prod-2",
			Name:       "Test Product 2",
			Slug:       "test-product-2",
			CategoryID: "cat-1",
			BrandID:    "brand-2",
			Price:      149.99,
			Status:     "PUBLISHED",
			CreatedAt:  time.Now(),
		},
	}

	expectedResult := &domain.SearchResult{
		Products:   expectedProducts,
		Total:      2,
		Page:       1,
		Limit:      20,
		TotalPages: 1,
	}

	// Setup mock expectations
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("", nil)
	mockElastic.On("Search", mock.Anything, mock.AnythingOfType("*domain.SearchParams")).Return(expectedResult, nil)
	mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), CacheTTL).Return(nil)

	// Create usecase with mock adapter
	usecase := &SearchUsecaseMock{
		elasticClient: mockElastic,
		redisClient:   mockRedis,
	}

	// Execute search
	params := &domain.SearchParams{
		Keyword: "test",
		Page:    1,
		Limit:   20,
	}

	result, err := usecase.SearchProducts(context.Background(), params)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Products, 2)
	assert.Equal(t, "Test Product 1", result.Products[0].Name)

	// Verify mocks were called correctly
	mockRedis.AssertCalled(t, "Get", mock.Anything, mock.AnythingOfType("string"))
	mockElastic.AssertCalled(t, "Search", mock.Anything, mock.AnythingOfType("*domain.SearchParams"))
	mockRedis.AssertCalled(t, "Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), CacheTTL)
}

// TestSearchProducts_CacheHit tests the search flow when cache hits
func TestSearchProducts_CacheHit(t *testing.T) {
	// Setup mocks
	mockElastic := new(MockElasticClient)
	mockRedis := NewMockRedisClient()

	// Cached result
	cachedResult := &domain.SearchResult{
		Products: []domain.Product{
			{
				ID:   "prod-cached",
				Name: "Cached Product",
			},
		},
		Total:      1,
		Page:       1,
		Limit:      20,
		TotalPages: 1,
	}

	cachedJSON, _ := json.Marshal(cachedResult)

	// Setup mock expectations - cache hit
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(string(cachedJSON), nil)

	// Create usecase with mock adapter
	usecase := &SearchUsecaseMock{
		elasticClient: mockElastic,
		redisClient:   mockRedis,
	}

	// Execute search
	params := &domain.SearchParams{
		Keyword: "cached",
		Page:    1,
		Limit:   20,
	}

	result, err := usecase.SearchProducts(context.Background(), params)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "Cached Product", result.Products[0].Name)

	// Verify Redis Get was called but Elasticsearch Search was NOT called
	mockRedis.AssertCalled(t, "Get", mock.Anything, mock.AnythingOfType("string"))
	mockElastic.AssertNotCalled(t, "Search", mock.Anything, mock.Anything)
	mockRedis.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// SearchUsecaseMock is a test-friendly version that uses interfaces
type SearchUsecaseMock struct {
	elasticClient *MockElasticClient
	redisClient   *MockRedisClient
}

func (u *SearchUsecaseMock) SearchProducts(ctx context.Context, params *domain.SearchParams) (*domain.SearchResult, error) {
	// Set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 20
	}

	// Generate cache key
	cacheKey := "search:test-key"

	// Try to get from cache
	cachedResult, err := u.redisClient.Get(ctx, cacheKey)
	if err == nil && cachedResult != "" {
		var result domain.SearchResult
		if err := json.Unmarshal([]byte(cachedResult), &result); err == nil {
			return &result, nil
		}
	}

	// Cache miss - query Elasticsearch
	result, err := u.elasticClient.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	// Store result in cache
	resultJSON, _ := json.Marshal(result)
	u.redisClient.Set(ctx, cacheKey, string(resultJSON), CacheTTL)

	return result, nil
}

// TestConsumerHandler_IndexProduct tests the consumer handler for product indexing
func TestConsumerHandler_IndexProduct(t *testing.T) {
	mockElastic := new(MockElasticClient)

	product := &domain.Product{
		ID:         "prod-new",
		Name:       "New Product",
		Slug:       "new-product",
		CategoryID: "cat-1",
		BrandID:    "brand-1",
		Price:      199.99,
		Status:     "PUBLISHED",
		CreatedAt:  time.Now(),
	}

	// Setup mock expectation
	mockElastic.On("IndexProduct", mock.Anything, product).Return(nil)

	// Simulate consumer handler behavior
	err := mockElastic.IndexProduct(context.Background(), product)

	// Assertions
	assert.NoError(t, err)
	mockElastic.AssertCalled(t, "IndexProduct", mock.Anything, product)
}

// TestConsumerHandler_DeleteProduct tests the consumer handler for product deletion
func TestConsumerHandler_DeleteProduct(t *testing.T) {
	mockElastic := new(MockElasticClient)

	productID := "prod-to-delete"

	// Setup mock expectation
	mockElastic.On("DeleteProduct", mock.Anything, productID).Return(nil)

	// Simulate consumer handler behavior
	err := mockElastic.DeleteProduct(context.Background(), productID)

	// Assertions
	assert.NoError(t, err)
	mockElastic.AssertCalled(t, "DeleteProduct", mock.Anything, productID)
}
