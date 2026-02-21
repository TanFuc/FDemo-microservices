package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"microservices/media/internal/config"
	"microservices/media/internal/repository"
	"microservices/pkg/logger"
)

// QueueRepo implements the MessageQueueRepository interface using NATS JetStream.
type QueueRepo struct {
	conn       *nats.Conn
	js         jetstream.JetStream
	streamName string
	subject    string
}

// Ensure QueueRepo implements MessageQueueRepository.
var _ repository.MessageQueueRepository = (*QueueRepo)(nil)

// NewQueueRepo creates a new NATS queue repository.
func NewQueueRepo(cfg config.NATSConfig) (*QueueRepo, error) {
	conn, err := nats.Connect(cfg.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	logger.Info().
		Str("url", cfg.URL).
		Str("stream", cfg.StreamName).
		Msg("NATS JetStream connected")

	client := &QueueRepo{
		conn:       conn,
		js:         js,
		streamName: cfg.StreamName,
		subject:    cfg.Subject,
	}

	if err := client.ensureStream(context.Background()); err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

// ensureStream ensures the JetStream stream exists.
func (q *QueueRepo) ensureStream(ctx context.Context) error {
	_, err := q.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        q.streamName,
		Description: "Media processing events",
		Subjects:    []string{q.subject},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		return fmt.Errorf("failed to create/update stream: %w", err)
	}

	logger.Info().
		Str("stream", q.streamName).
		Msg("JetStream stream ensured")

	return nil
}

// Publish sends a message to the specified subject.
func (q *QueueRepo) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := q.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	logger.Debug().
		Str("subject", subject).
		Int("size", len(data)).
		Msg("Message published")

	return nil
}

// Subscribe subscribes to messages on the specified subject.
func (q *QueueRepo) Subscribe(ctx context.Context, subject string, handler func(data []byte) error) error {
	consumer, err := q.js.CreateOrUpdateConsumer(ctx, q.streamName, jetstream.ConsumerConfig{
		Durable:       "media-processor",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    3,
		FilterSubject: subject,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	cons, err := consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			logger.Error().
				Err(err).
				Str("subject", subject).
				Msg("Error processing message")
			msg.Nak()
			return
		}
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go func() {
		<-ctx.Done()
		cons.Stop()
	}()

	logger.Info().
		Str("subject", subject).
		Msg("Started consuming messages")

	return nil
}

// Close closes the connection to the message queue.
func (q *QueueRepo) Close() error {
	q.conn.Close()
	logger.Info().Msg("NATS connection closed")
	return nil
}
