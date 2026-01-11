package nats

import (
	"context"
	"encoding/json"

	"catalog-service/internal/domain"
	"catalog-service/internal/repository"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var _ repository.EventPublisher = (*Publisher)(nil)

type Publisher struct {
	nc *nats.Conn
	js jetstream.JetStream
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
	ctx := context.Background()
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{"catalog.>"},
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

func (p *Publisher) PublishProductCreated(ctx context.Context, product *domain.Product) error {
	event := domain.ProductEvent{Product: product}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.js.Publish(ctx, domain.SubjectProductCreated, data)
	return err
}

func (p *Publisher) PublishProductUpdated(ctx context.Context, product *domain.Product) error {
	event := domain.ProductEvent{Product: product}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.js.Publish(ctx, domain.SubjectProductUpdated, data)
	return err
}

func (p *Publisher) Close() error {
	p.nc.Close()
	return nil
}
