package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/tafu/search-service/internal/domain"
	"github.com/tafu/search-service/internal/infrastructure/elastic"
)

const (
	StreamName       = "CATALOG"
	ConsumerName     = "search-service-consumer"
	SubjectPattern   = "catalog.product.>"
	WorkerPoolSize   = 10
	MaxPendingMsgs   = 100
	AckWaitDuration  = 30 * time.Second
	MaxDeliver       = 5
)

// Consumer handles NATS JetStream event consumption
type Consumer struct {
	nc            *nats.Conn
	js            jetstream.JetStream
	elasticClient *elastic.Client
	logger        *slog.Logger
	wg            sync.WaitGroup
	cancel        context.CancelFunc
	consumer      jetstream.Consumer
}

// NewConsumer creates a new NATS JetStream consumer
func NewConsumer(natsURL string, elasticClient *elastic.Client, logger *slog.Logger) (*Consumer, error) {
	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				logger.Warn("nats disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info("nats reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create jetstream context: %w", err)
	}

	logger.Info("connected to nats", "url", natsURL)

	return &Consumer{
		nc:            nc,
		js:            js,
		elasticClient: elasticClient,
		logger:        logger,
	}, nil
}

// EnsureStream creates the CATALOG stream if it doesn't exist
func (c *Consumer) EnsureStream(ctx context.Context) error {
	_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        StreamName,
		Description: "Catalog product events for search indexing",
		Subjects:    []string{SubjectPattern},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		return fmt.Errorf("failed to create/update stream: %w", err)
	}
	c.logger.Info("stream ensured", "stream", StreamName)
	return nil
}

// Start begins consuming events with a worker pool
func (c *Consumer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// Create or update durable consumer
	consumer, err := c.js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Durable:       ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       AckWaitDuration,
		MaxDeliver:    MaxDeliver,
		FilterSubject: SubjectPattern,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	c.consumer = consumer

	c.logger.Info("starting worker pool", "workers", WorkerPoolSize)

	// Create message channel for worker pool
	msgChan := make(chan jetstream.Msg, MaxPendingMsgs)

	// Start worker goroutines
	for i := 0; i < WorkerPoolSize; i++ {
		c.wg.Add(1)
		go c.worker(ctx, i, msgChan)
	}

	// Start message fetcher
	c.wg.Add(1)
	go c.fetchMessages(ctx, msgChan)

	return nil
}

// fetchMessages continuously fetches messages and sends to worker channel
func (c *Consumer) fetchMessages(ctx context.Context, msgChan chan<- jetstream.Msg) {
	defer c.wg.Done()
	defer close(msgChan)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("message fetcher stopping")
			return
		default:
			msgs, err := c.consumer.Fetch(10, jetstream.FetchMaxWait(5*time.Second))
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Warn("fetch error", "error", err)
				continue
			}

			for msg := range msgs.Messages() {
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}
			}

			if msgs.Error() != nil && msgs.Error() != context.DeadlineExceeded {
				c.logger.Warn("messages iteration error", "error", msgs.Error())
			}
		}
	}
}

// worker processes messages from the channel
func (c *Consumer) worker(ctx context.Context, id int, msgChan <-chan jetstream.Msg) {
	defer c.wg.Done()
	c.logger.Debug("worker started", "workerId", id)

	for {
		select {
		case <-ctx.Done():
			c.logger.Debug("worker stopping", "workerId", id)
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			c.processMessage(ctx, msg)
		}
	}
}

// processMessage handles a single NATS message
func (c *Consumer) processMessage(ctx context.Context, msg jetstream.Msg) {
	subject := msg.Subject()
	traceID := generateTraceID()
	logger := c.logger.With("traceId", traceID, "subject", subject)

	logger.Debug("processing message")

	var event domain.ProductEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		logger.Error("failed to unmarshal event", "error", err)
		// Ack to prevent redelivery of malformed messages
		if err := msg.Ack(); err != nil {
			logger.Error("failed to ack malformed message", "error", err)
		}
		return
	}

	var processErr error
	switch subject {
	case "catalog.product.created", "catalog.product.updated":
		if event.Product == nil {
			logger.Error("product data missing in event")
			if err := msg.Ack(); err != nil {
				logger.Error("failed to ack", "error", err)
			}
			return
		}
		logger = logger.With("productId", event.Product.ID)
		processErr = c.elasticClient.IndexProduct(ctx, event.Product)
		if processErr == nil {
			logger.Info("product indexed successfully")
		}

	case "catalog.product.deleted":
		productID := event.ProductID
		if productID == "" && event.Product != nil {
			productID = event.Product.ID
		}
		if productID == "" {
			logger.Error("product ID missing in delete event")
			if err := msg.Ack(); err != nil {
				logger.Error("failed to ack", "error", err)
			}
			return
		}
		logger = logger.With("productId", productID)
		processErr = c.elasticClient.DeleteProduct(ctx, productID)
		if processErr == nil {
			logger.Info("product deleted successfully")
		}

	default:
		logger.Warn("unknown event subject, acknowledging")
		if err := msg.Ack(); err != nil {
			logger.Error("failed to ack unknown event", "error", err)
		}
		return
	}

	if processErr != nil {
		logger.Error("failed to process event", "error", processErr)
		// NAK to retry - Elasticsearch might be temporarily unavailable
		if err := msg.Nak(); err != nil {
			logger.Error("failed to nak message", "error", err)
		}
		return
	}

	// ACK only after successful processing
	if err := msg.Ack(); err != nil {
		logger.Error("failed to ack message", "error", err)
	}
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() error {
	c.logger.Info("stopping consumer")

	if c.cancel != nil {
		c.cancel()
	}

	// Wait for all workers to finish
	c.wg.Wait()

	if c.nc != nil {
		c.nc.Close()
	}

	c.logger.Info("consumer stopped")
	return nil
}

// generateTraceID creates a simple trace ID for logging
func generateTraceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
