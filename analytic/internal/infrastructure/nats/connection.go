package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"microservices/analytic/internal/config"
	"microservices/analytic/internal/domain"
)

type Client struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
	cfg    config.NATSConfig
}

func NewClient(cfg config.NATSConfig) (*Client, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &Client{
		conn: conn,
		js:   js,
		cfg:  cfg,
	}

	if err := client.setupStream(context.Background()); err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

func (c *Client) setupStream(ctx context.Context) error {
	streamConfig := jetstream.StreamConfig{
		Name:      c.cfg.StreamName,
		Subjects:  []string{c.cfg.Subject},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    0, // No max age
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	}

	stream, err := c.js.CreateOrUpdateStream(ctx, streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create/update stream: %w", err)
	}

	c.stream = stream
	return nil
}

// Publish publishes an event to the NATS JetStream subject.
func (c *Client) Publish(ctx context.Context, event domain.UserEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = c.js.Publish(ctx, c.cfg.Subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Subscribe creates a durable consumer and returns messages through a channel.
func (c *Client) Subscribe(ctx context.Context, consumerName string) (<-chan jetstream.Msg, error) {
	consumer, err := c.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       consumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	msgCh := make(chan jetstream.Msg, 1000)

	cons, err := consumer.Consume(func(msg jetstream.Msg) {
		select {
		case msgCh <- msg:
		case <-ctx.Done():
			return
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	go func() {
		<-ctx.Done()
		cons.Stop()
		close(msgCh)
	}()

	return msgCh, nil
}

// Close closes the NATS connection.
func (c *Client) Close() {
	c.conn.Close()
}

// JetStream returns the JetStream context for direct access.
func (c *Client) JetStream() jetstream.JetStream {
	return c.js
}
