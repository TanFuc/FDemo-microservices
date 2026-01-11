package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/elastic/go-elasticsearch/v8"
	"microservices/search/internal/domain"
)

const (
	IndexName = "products_index"
)

// Client wraps Elasticsearch operations
type Client struct {
	es     *elasticsearch.Client
	logger *slog.Logger
}

// NewClient creates a new Elasticsearch client wrapper
func NewClient(addresses []string, logger *slog.Logger) (*Client, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Test connection
	res, err := es.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch error: %s", res.String())
	}

	logger.Info("connected to elasticsearch", "addresses", addresses)

	return &Client{
		es:     es,
		logger: logger,
	}, nil
}

// EnsureIndex creates the products index with proper mapping if it doesn't exist
func (c *Client) EnsureIndex(ctx context.Context) error {
	// Check if index exists
	res, err := c.es.Indices.Exists([]string{IndexName})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		c.logger.Info("index already exists", "index", IndexName)
		return nil
	}

	// Create index with mapping
	mapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id":         map[string]interface{}{"type": "keyword"},
				"name":       map[string]interface{}{"type": "text", "analyzer": "standard"},
				"slug":       map[string]interface{}{"type": "keyword"},
				"categoryId": map[string]interface{}{"type": "keyword"},
				"brandId":    map[string]interface{}{"type": "keyword"},
				"price":      map[string]interface{}{"type": "double"},
				"thumbnail":  map[string]interface{}{"type": "keyword"},
				"status":     map[string]interface{}{"type": "keyword"},
				"createdAt":  map[string]interface{}{"type": "date"},
				"specs":      map[string]interface{}{"type": "object", "dynamic": true},
				"metadata":   map[string]interface{}{"type": "object", "dynamic": true},
			},
		},
		"settings": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	}

	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	res, err = c.es.Indices.Create(
		IndexName,
		c.es.Indices.Create.WithBody(bytes.NewReader(mappingJSON)),
		c.es.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	c.logger.Info("index created successfully", "index", IndexName)
	return nil
}

// IndexProduct indexes or updates a product document (idempotent using product ID)
func (c *Client) IndexProduct(ctx context.Context, product *domain.Product) error {
	data, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	res, err := c.es.Index(
		IndexName,
		bytes.NewReader(data),
		c.es.Index.WithDocumentID(product.ID),
		c.es.Index.WithContext(ctx),
		c.es.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to index product: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index product: %s", res.String())
	}

	c.logger.Debug("product indexed", "productId", product.ID, "name", product.Name)
	return nil
}

// DeleteProduct removes a product document by ID
func (c *Client) DeleteProduct(ctx context.Context, id string) error {
	res, err := c.es.Delete(
		IndexName,
		id,
		c.es.Delete.WithContext(ctx),
		c.es.Delete.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	defer res.Body.Close()

	// 404 is acceptable - document might not exist
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("failed to delete product: %s", res.String())
	}

	c.logger.Debug("product deleted", "productId", id)
	return nil
}

// Search performs a search query and returns results
func (c *Client) Search(ctx context.Context, params *domain.SearchParams) (*domain.SearchResult, error) {
	query := c.buildSearchQuery(params)

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	c.logger.Debug("executing search", "query", string(queryJSON))

	from := (params.Page - 1) * params.Limit
	if from < 0 {
		from = 0
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(IndexName),
		c.es.Search.WithBody(bytes.NewReader(queryJSON)),
		c.es.Search.WithFrom(from),
		c.es.Search.WithSize(params.Limit),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.String())
	}

	var searchResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source domain.Product `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	products := make([]domain.Product, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		products = append(products, hit.Source)
	}

	total := searchResponse.Hits.Total.Value
	totalPages := int(total) / params.Limit
	if int(total)%params.Limit > 0 {
		totalPages++
	}

	return &domain.SearchResult{
		Products:   products,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// buildSearchQuery constructs an Elasticsearch bool query from search params
func (c *Client) buildSearchQuery(params *domain.SearchParams) map[string]interface{} {
	boolQuery := map[string]interface{}{}
	must := []interface{}{}
	filter := []interface{}{}

	// Multi-match for keyword search with boosted name field
	if params.Keyword != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  params.Keyword,
				"fields": []string{"name^3", "slug"},
				"type":   "best_fields",
			},
		})
	}

	// Filter by status = PUBLISHED
	filter = append(filter, map[string]interface{}{
		"term": map[string]interface{}{
			"status": "PUBLISHED",
		},
	})

	// Filter by categoryId
	if params.CategoryID != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"categoryId": params.CategoryID,
			},
		})
	}

	// Filter by brandId
	if params.BrandID != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"brandId": params.BrandID,
			},
		})
	}

	// Price range filter
	if params.PriceMin != nil || params.PriceMax != nil {
		priceRange := map[string]interface{}{}
		if params.PriceMin != nil {
			priceRange["gte"] = *params.PriceMin
		}
		if params.PriceMax != nil {
			priceRange["lte"] = *params.PriceMax
		}
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"price": priceRange,
			},
		})
	}

	// Dynamic specs filters
	for key, value := range params.Specs {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				fmt.Sprintf("specs.%s.keyword", key): value,
			},
		})
	}

	// Metadata filters (e.g., isFlashSale)
	for key, value := range params.Metadata {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				fmt.Sprintf("metadata.%s", key): value,
			},
		})
	}

	if len(must) > 0 {
		boolQuery["must"] = must
	}
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
	}

	// Sorting
	if params.SortBy != "" {
		order := "asc"
		if params.SortOrder == "desc" {
			order = "desc"
		}
		query["sort"] = []interface{}{
			map[string]interface{}{
				params.SortBy: map[string]interface{}{
					"order": order,
				},
			},
		}
	}

	return query
}

// Close closes the Elasticsearch client (no-op for current client)
func (c *Client) Close() error {
	return nil
}
