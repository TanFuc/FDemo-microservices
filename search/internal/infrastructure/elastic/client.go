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

	// Create index with Vietnamese analyzer and advanced mapping
	mapping := c.buildIndexMapping()

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

// buildIndexMapping creates the Elasticsearch mapping with Vietnamese analyzer
func (c *Client) buildIndexMapping() map[string]interface{} {
	return map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   3,
			"number_of_replicas": 1,
			"max_result_window":  50000,
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"vietnamese_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter": []string{
							"lowercase",
							"asciifolding",
							"vietnamese_stop",
							"vietnamese_synonym",
						},
					},
					"autocomplete_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "autocomplete_tokenizer",
						"filter": []string{
							"lowercase",
							"asciifolding",
						},
					},
					"autocomplete_search": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter": []string{
							"lowercase",
							"asciifolding",
						},
					},
					"keyword_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "keyword",
						"filter":    []string{"lowercase"},
					},
				},
				"tokenizer": map[string]interface{}{
					"autocomplete_tokenizer": map[string]interface{}{
						"type":        "edge_ngram",
						"min_gram":    2,
						"max_gram":    20,
						"token_chars": []string{"letter", "digit"},
					},
				},
				"filter": map[string]interface{}{
					"vietnamese_stop": map[string]interface{}{
						"type": "stop",
						"stopwords": []string{
							"và", "của", "là", "có", "được", "trong", "cho", "với",
							"này", "các", "để", "một", "những", "không", "từ", "như",
							"khi", "theo", "vào", "ra", "lên", "về", "đến", "hoặc",
							"nhưng", "thì", "nếu", "hay", "mà", "đã", "còn", "cũng",
						},
					},
					"vietnamese_synonym": map[string]interface{}{
						"type": "synonym",
						"synonyms": []string{
							"điện thoại, dt, smartphone, phone",
							"máy tính, laptop, pc, computer",
							"tai nghe, headphone, earphone",
							"sạc, charger, cốc sạc",
							"pin, battery, ắc quy",
						},
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id":   map[string]interface{}{"type": "keyword"},
				"name": map[string]interface{}{
					"type":     "text",
					"analyzer": "vietnamese_analyzer",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type":         "keyword",
							"ignore_above": 256,
						},
						"autocomplete": map[string]interface{}{
							"type":            "text",
							"analyzer":        "autocomplete_analyzer",
							"search_analyzer": "autocomplete_search",
						},
						"exact": map[string]interface{}{
							"type":     "text",
							"analyzer": "keyword_analyzer",
						},
					},
				},
				"slug":             map[string]interface{}{"type": "keyword"},
				"description":      map[string]interface{}{"type": "text", "analyzer": "vietnamese_analyzer"},
				"shortDescription": map[string]interface{}{"type": "text", "analyzer": "vietnamese_analyzer"},
				"categoryId":       map[string]interface{}{"type": "keyword"},
				"categoryName": map[string]interface{}{
					"type":     "text",
					"analyzer": "vietnamese_analyzer",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{"type": "keyword"},
					},
				},
				"categoryPath": map[string]interface{}{"type": "keyword"},
				"brandId":      map[string]interface{}{"type": "keyword"},
				"brandName": map[string]interface{}{
					"type":     "text",
					"analyzer": "vietnamese_analyzer",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{"type": "keyword"},
					},
				},
				"shopId": map[string]interface{}{"type": "keyword"},
				"shopName": map[string]interface{}{
					"type":     "text",
					"analyzer": "vietnamese_analyzer",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{"type": "keyword"},
					},
				},
				"basePrice":  map[string]interface{}{"type": "double"},
				"minPrice":   map[string]interface{}{"type": "double"},
				"maxPrice":   map[string]interface{}{"type": "double"},
				"currency":   map[string]interface{}{"type": "keyword"},
				"thumbnail":  map[string]interface{}{"type": "keyword", "index": false},
				"images":     map[string]interface{}{"type": "keyword", "index": false},
				"status":     map[string]interface{}{"type": "keyword"},
				"visibility": map[string]interface{}{"type": "keyword"},
				"totalStock": map[string]interface{}{"type": "integer"},
				"soldCount":  map[string]interface{}{"type": "integer"},
				"viewCount":  map[string]interface{}{"type": "integer"},
				"rating":     map[string]interface{}{"type": "float"},
				"reviewCount": map[string]interface{}{"type": "integer"},
				"hasVariants":    map[string]interface{}{"type": "boolean"},
				"isDigital":      map[string]interface{}{"type": "boolean"},
				"isFreeShipping": map[string]interface{}{"type": "boolean"},
				"weight":         map[string]interface{}{"type": "float"},
				"specifications": map[string]interface{}{
					"type": "nested",
					"properties": map[string]interface{}{
						"group": map[string]interface{}{"type": "keyword"},
						"key":   map[string]interface{}{"type": "keyword"},
						"name": map[string]interface{}{
							"type":     "text",
							"analyzer": "vietnamese_analyzer",
						},
						"value": map[string]interface{}{
							"type":     "text",
							"analyzer": "vietnamese_analyzer",
							"fields": map[string]interface{}{
								"keyword": map[string]interface{}{"type": "keyword"},
							},
						},
						"unit": map[string]interface{}{"type": "keyword"},
					},
				},
				"attributes":  map[string]interface{}{"type": "object", "enabled": true},
				"metadata":    map[string]interface{}{"type": "object", "enabled": true},
				"tags":        map[string]interface{}{"type": "keyword"},
				"createdAt":   map[string]interface{}{"type": "date"},
				"updatedAt":   map[string]interface{}{"type": "date"},
				"publishedAt": map[string]interface{}{"type": "date"},
			},
		},
	}
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

// buildSearchQuery constructs an Elasticsearch bool query with multi-match and filters
func (c *Client) buildSearchQuery(params *domain.SearchParams) map[string]interface{} {
	must := []interface{}{}
	filter := []interface{}{}

	// Keyword search with multi-match (Vietnamese analyzer + autocomplete)
	if params.Keyword != "" {
		must = append(must, map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					// Exact match boost
					map[string]interface{}{
						"match": map[string]interface{}{
							"name.exact": map[string]interface{}{
								"query": params.Keyword,
								"boost": 10,
							},
						},
					},
					// Autocomplete match
					map[string]interface{}{
						"match": map[string]interface{}{
							"name.autocomplete": map[string]interface{}{
								"query": params.Keyword,
								"boost": 5,
							},
						},
					},
					// Standard Vietnamese analyzer
					map[string]interface{}{
						"match": map[string]interface{}{
							"name": map[string]interface{}{
								"query": params.Keyword,
								"boost": 3,
							},
						},
					},
					// Description match
					map[string]interface{}{
						"match": map[string]interface{}{
							"description": map[string]interface{}{
								"query": params.Keyword,
								"boost": 1,
							},
						},
					},
					// Brand and category match
					map[string]interface{}{
						"match": map[string]interface{}{
							"brandName": map[string]interface{}{
								"query": params.Keyword,
								"boost": 2,
							},
						},
					},
					map[string]interface{}{
						"match": map[string]interface{}{
							"categoryName": map[string]interface{}{
								"query": params.Keyword,
								"boost": 2,
							},
						},
					},
				},
				"minimum_should_match": 1,
			},
		})
	}

	// Filter by category
	if params.CategoryID != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"categoryId": params.CategoryID,
			},
		})
	}

	// Filter by brand
	if params.BrandID != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"brandId": params.BrandID,
			},
		})
	}

	// Filter by shop
	if params.ShopID != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"shopId": params.ShopID,
			},
		})
	}

	// Price range filter
	if params.PriceMin != nil || params.PriceMax != nil {
		rangeFilter := map[string]interface{}{}
		if params.PriceMin != nil {
			rangeFilter["gte"] = *params.PriceMin
		}
		if params.PriceMax != nil {
			rangeFilter["lte"] = *params.PriceMax
		}
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"minPrice": rangeFilter,
			},
		})
	}

	// Filter by status (default: only ACTIVE)
	if params.Status != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"status": params.Status,
			},
		})
	} else {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"status": "ACTIVE",
			},
		})
	}

	// Nested specifications filter
	for key, value := range params.Specs {
		filter = append(filter, map[string]interface{}{
			"nested": map[string]interface{}{
				"path": "specifications",
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"term": map[string]interface{}{
									"specifications.key": key,
								},
							},
							map[string]interface{}{
								"match": map[string]interface{}{
									"specifications.value": value,
								},
							},
						},
					},
				},
			},
		})
	}

	// Metadata filters
	for key, value := range params.Metadata {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				fmt.Sprintf("metadata.%s", key): value,
			},
		})
	}

	// Build final query
	boolQuery := map[string]interface{}{}
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
	sort := []interface{}{}
	if params.SortBy != "" {
		order := "asc"
		if params.SortOrder == "desc" {
			order = "desc"
		}
		sort = append(sort, map[string]interface{}{
			params.SortBy: map[string]interface{}{
				"order": order,
			},
		})
	} else {
		// Default sort by relevance (_score) and createdAt
		sort = append(sort,
			map[string]interface{}{"_score": "desc"},
			map[string]interface{}{"createdAt": "desc"},
		)
	}
	query["sort"] = sort

	return query
}

// BulkIndex indexes multiple products efficiently
func (c *Client) BulkIndex(ctx context.Context, products []domain.Product) error {
	if len(products) == 0 {
		return nil
	}

	var buf bytes.Buffer

	for _, product := range products {
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": IndexName,
				"_id":    product.ID,
			},
		}

		metaJSON, _ := json.Marshal(meta)
		productJSON, _ := json.Marshal(product)

		buf.Write(metaJSON)
		buf.WriteByte('\n')
		buf.Write(productJSON)
		buf.WriteByte('\n')
	}

	res, err := c.es.Bulk(
		bytes.NewReader(buf.Bytes()),
		c.es.Bulk.WithContext(ctx),
		c.es.Bulk.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("bulk indexing failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk indexing error: %s", res.String())
	}

	c.logger.Info("bulk indexed products", "count", len(products))
	return nil
}

// Close closes the Elasticsearch client (no-op for current client)
func (c *Client) Close() error {
	return nil
}
