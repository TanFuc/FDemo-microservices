package nats

import (
	"context"
	"encoding/json"
	"time"

	"microservices/catalog/internal/domain"
	"microservices/catalog/internal/repository"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var _ repository.EventPublisher = (*Publisher)(nil)

type Publisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// ProductEventPayload is the wire format for product events that search-service expects
type ProductEventPayload struct {
	ID               string                 `json:"id"`
	ShopID           string                 `json:"shopId,omitempty"`
	ShopName         string                 `json:"shopName,omitempty"`
	Name             string                 `json:"name"`
	Slug             string                 `json:"slug"`
	Description      string                 `json:"description,omitempty"`
	ShortDescription string                 `json:"shortDescription,omitempty"`
	CategoryID       string                 `json:"categoryId"`
	CategoryName     string                 `json:"categoryName,omitempty"`
	CategoryPath     string                 `json:"categoryPath,omitempty"`
	BrandID          string                 `json:"brandId"`
	BrandName        string                 `json:"brandName,omitempty"`
	Thumbnail        string                 `json:"thumbnail"`
	Images           []string               `json:"images,omitempty"`
	BasePrice        float64                `json:"basePrice"`
	MinPrice         float64                `json:"minPrice"`
	MaxPrice         float64                `json:"maxPrice"`
	Currency         string                 `json:"currency,omitempty"`
	TotalStock       int                    `json:"totalStock"`
	Status           string                 `json:"status"`
	Visibility       string                 `json:"visibility,omitempty"`
	HasVariants      bool                   `json:"hasVariants"`
	IsDigital        bool                   `json:"isDigital"`
	IsFreeShipping   bool                   `json:"isFreeShipping"`
	Weight           float64                `json:"weight,omitempty"`
	Specifications   []SpecificationPayload `json:"specifications,omitempty"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
}

// SpecificationPayload is the wire format for product specifications
type SpecificationPayload struct {
	Group string `json:"group,omitempty"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
	Order int    `json:"order"`
}

// CatalogEventPayload is the wire format for NATS events that search-service expects
type CatalogEventPayload struct {
	EventType string               `json:"eventType"`
	ProductID string               `json:"productId,omitempty"`
	Product   *ProductEventPayload `json:"product,omitempty"`
}

func NewPublisher(url string, streamName string) (*Publisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, err
	}

	// Create stream if it doesn't exist
	// Match search-service consumer config
	ctx := context.Background()
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{"catalog.>"},
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		nc.Close()
		return nil, err
	}

	return &Publisher{
		nc: nc,
		js: js,
	}, nil
}

// mapProductToPayload converts domain.Product to the wire format
func mapProductToPayload(product *domain.Product) *ProductEventPayload {
	// Extract image URLs from ProductImage slice
	images := make([]string, 0, len(product.Images))
	for _, img := range product.Images {
		images = append(images, img.URL)
	}

	// Convert specifications
	specs := make([]SpecificationPayload, 0, len(product.Specifications))
	for _, spec := range product.Specifications {
		specs = append(specs, SpecificationPayload{
			Group: spec.Group,
			Key:   spec.Key,
			Name:  spec.Name,
			Value: spec.Value,
			Unit:  spec.Unit,
			Order: spec.Order,
		})
	}

	return &ProductEventPayload{
		ID:               product.ID.Hex(),
		ShopID:           product.ShopID,
		ShopName:         product.ShopName,
		Name:             product.Name,
		Slug:             product.Slug,
		Description:      product.Description,
		ShortDescription: product.ShortDescription,
		CategoryID:       product.CategoryID.Hex(),
		CategoryName:     "", // Not stored in domain.Product, will be empty
		CategoryPath:     product.CategoryPath,
		BrandID:          product.BrandID.Hex(),
		BrandName:        product.BrandName,
		Thumbnail:        product.Thumbnail,
		Images:           images,
		BasePrice:        product.BasePrice.InexactFloat64(),
		MinPrice:         product.MinPrice.InexactFloat64(),
		MaxPrice:         product.MaxPrice.InexactFloat64(),
		Currency:         product.Currency,
		TotalStock:       product.TotalStock,
		Status:           product.Status,
		Visibility:       product.Visibility,
		HasVariants:      product.HasVariants,
		IsDigital:        product.IsDigital,
		IsFreeShipping:   product.IsFreeShipping,
		Weight:           product.Weight,
		Specifications:   specs,
		Attributes:       product.Attributes,
		Metadata:         product.Metadata,
		Tags:             product.Tags,
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}
}

func (p *Publisher) PublishProductCreated(ctx context.Context, product *domain.Product) error {
	event := CatalogEventPayload{
		EventType: domain.EventTypeCreated,
		Product:   mapProductToPayload(product),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.js.Publish(ctx, domain.SubjectProductCreated, data)
	return err
}

func (p *Publisher) PublishProductUpdated(ctx context.Context, product *domain.Product) error {
	event := CatalogEventPayload{
		EventType: domain.EventTypeUpdated,
		Product:   mapProductToPayload(product),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.js.Publish(ctx, domain.SubjectProductUpdated, data)
	return err
}

func (p *Publisher) PublishProductDeleted(ctx context.Context, productID string) error {
	event := CatalogEventPayload{
		EventType: domain.EventTypeDeleted,
		ProductID: productID,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.js.Publish(ctx, domain.SubjectProductDeleted, data)
	return err
}

func (p *Publisher) Close() error {
	p.nc.Close()
	return nil
}

// JetStream returns the JetStream context for use by workers
func (p *Publisher) JetStream() jetstream.JetStream {
	return p.js
}
