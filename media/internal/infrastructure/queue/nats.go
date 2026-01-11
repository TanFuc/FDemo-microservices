package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/tafu-media/media-service/internal/config"
)

type NATSClient struct {
	conn       *nats.Conn
	js         jetstream.JetStream
	streamName string
	subject    string
}

func NewNATSClient(cfg config.NATSConfig) (*NATSClient, error) {
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

	client := &NATSClient{
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

func (n *NATSClient) ensureStream(ctx context.Context) error {
	_, err := n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        n.streamName,
		Description: "Media processing events",
		Subjects:    []string{n.subject},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		return fmt.Errorf("failed to create/update stream: %w", err)
	}
	return nil
}

func (n *NATSClient) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := n.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

func (n *NATSClient) Subscribe(ctx context.Context, subject string, handler func(data []byte) error) error {
	consumer, err := n.js.CreateOrUpdateConsumer(ctx, n.streamName, jetstream.ConsumerConfig{
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
			fmt.Printf("Error processing message: %v\n", err)
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

	return nil
}

func (n *NATSClient) Close() error {
	n.conn.Close()
	return nil
}
