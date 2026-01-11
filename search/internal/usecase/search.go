package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"

	"microservices/search/internal/domain"
	"microservices/search/internal/infrastructure/cache"
	"microservices/search/internal/infrastructure/elastic"
)

const (
	CacheKeyPrefix = "search:"
	CacheTTL       = 120 // 2 minutes TTL as specified
)

// SearchUsecase handles search operations with caching
type SearchUsecase struct {
	elasticClient *elastic.Client
	redisClient   *cache.RedisClient
	logger        *slog.Logger
}

// NewSearchUsecase creates a new search usecase
func NewSearchUsecase(elasticClient *elastic.Client, redisClient *cache.RedisClient, logger *slog.Logger) *SearchUsecase {
	return &SearchUsecase{
		elasticClient: elasticClient,
		redisClient:   redisClient,
		logger:        logger,
	}
}

// SearchProducts performs a search with read-through caching
func (u *SearchUsecase) SearchProducts(ctx context.Context, params *domain.SearchParams) (*domain.SearchResult, error) {
	// Set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// Generate cache key
	cacheKey := u.generateCacheKey(params)
	logger := u.logger.With("cacheKey", cacheKey)

	// Try to get from cache
	cachedResult, err := u.redisClient.Get(ctx, cacheKey)
	if err != nil {
		logger.Warn("cache get error, falling back to Elasticsearch", "error", err)
	} else if cachedResult != "" {
		// Cache hit - unmarshal and return
		var result domain.SearchResult
		if err := json.Unmarshal([]byte(cachedResult), &result); err != nil {
			logger.Warn("failed to unmarshal cached result", "error", err)
		} else {
			logger.Debug("cache hit")
			return &result, nil
		}
	}

	// Cache miss - query Elasticsearch
	logger.Debug("cache miss, querying elasticsearch")
	result, err := u.elasticClient.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search failed: %w", err)
	}

	// Store result in cache
	resultJSON, err := json.Marshal(result)
	if err != nil {
		logger.Warn("failed to marshal result for caching", "error", err)
	} else {
		if err := u.redisClient.Set(ctx, cacheKey, string(resultJSON), CacheTTL); err != nil {
			logger.Warn("failed to cache result", "error", err)
		}
	}

	return result, nil
}

// generateCacheKey creates a deterministic cache key from search params
func (u *SearchUsecase) generateCacheKey(params *domain.SearchParams) string {
	// Serialize params to JSON for hashing
	paramsJSON, _ := json.Marshal(params)
	hash := sha256.Sum256(paramsJSON)
	return CacheKeyPrefix + hex.EncodeToString(hash[:16])
}

// InvalidateCache clears cached search results (useful when data changes)
func (u *SearchUsecase) InvalidateCache(ctx context.Context, key string) error {
	return u.redisClient.Delete(ctx, key)
}
