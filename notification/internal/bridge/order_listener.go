package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	"notification-service/internal/infrastructure"
	"notification-service/internal/models"
)

const (
	OrderCreatedSubject = "order.created"
	NotificationQueue   = "notification-service"
)

// OrderEventListener listens to order events from NATS and forwards to RabbitMQ
type OrderEventListener struct {
	nats     *infrastructure.NATS
	rabbitmq *infrastructure.RabbitMQ
	sub      *nats.Subscription
}

// NewOrderEventListener creates a new order event listener
func NewOrderEventListener(n *infrastructure.NATS, rmq *infrastructure.RabbitMQ) *OrderEventListener {
	return &OrderEventListener{
		nats:     n,
		rabbitmq: rmq,
	}
}

// Start starts listening to order events
func (l *OrderEventListener) Start(ctx context.Context) error {
	var err error

	// Try JetStream subscription first
	l.sub, err = l.nats.JetStreamQueueSubscribe(
		OrderCreatedSubject,
		NotificationQueue,
		l.handleOrderCreated,
		nats.Durable("notification-service-consumer"),
		nats.ManualAck(),
		nats.AckExplicit(),
	)

	if err != nil {
		// Fallback to regular NATS subscription
		log.Printf("JetStream subscription failed, falling back to regular NATS: %v", err)
		l.sub, err = l.nats.QueueSubscribe(OrderCreatedSubject, NotificationQueue, l.handleOrderCreatedRegular)
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", OrderCreatedSubject, err)
		}
	}

	log.Printf("Subscribed to NATS subject: %s", OrderCreatedSubject)
	return nil
}

// handleOrderCreated handles order created events from JetStream
func (l *OrderEventListener) handleOrderCreated(msg *nats.Msg) {
	log.Printf("Received order.created event: %s", string(msg.Data))

	// Parse the order event
	var event models.OrderCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Failed to unmarshal order event: %v", err)
		// Ack to prevent redelivery of invalid messages
		msg.Ack()
		return
	}

	// Transform to notification job
	job := models.NotificationJob{
		Type:      models.NotificationTypeEmail,
		Recipient: event.UserEmail,
		Template:  "order_confirmation",
		UserID:    event.UserID,
		Data: map[string]interface{}{
			"order_id": event.OrderID,
			"total":    event.Amount,
			"currency": event.Currency,
			"items":    event.Items,
			"user_id":  event.UserID,
		},
	}

	// Forward to RabbitMQ
	if err := l.forwardToRabbitMQ(job); err != nil {
		log.Printf("Failed to forward to RabbitMQ: %v", err)
		// NAK so NATS will redeliver
		msg.Nak()
		return
	}

	// ACK the NATS message
	if err := msg.Ack(); err != nil {
		log.Printf("Failed to ACK NATS message: %v", err)
	}

	log.Printf("Order event forwarded to RabbitMQ: %s", event.OrderID)
}

// handleOrderCreatedRegular handles order created events from regular NATS
func (l *OrderEventListener) handleOrderCreatedRegular(msg *nats.Msg) {
	log.Printf("Received order.created event (regular NATS): %s", string(msg.Data))

	// Parse the order event
	var event models.OrderCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Failed to unmarshal order event: %v", err)
		return
	}

	// Transform to notification job
	job := models.NotificationJob{
		Type:      models.NotificationTypeEmail,
		Recipient: event.UserEmail,
		Template:  "order_confirmation",
		UserID:    event.UserID,
		Data: map[string]interface{}{
			"order_id": event.OrderID,
			"total":    event.Amount,
			"currency": event.Currency,
			"items":    event.Items,
			"user_id":  event.UserID,
		},
	}

	// Forward to RabbitMQ
	if err := l.forwardToRabbitMQ(job); err != nil {
		log.Printf("Failed to forward to RabbitMQ: %v", err)
		return
	}

	log.Printf("Order event forwarded to RabbitMQ: %s", event.OrderID)
}

// forwardToRabbitMQ forwards a notification job to RabbitMQ
func (l *OrderEventListener) forwardToRabbitMQ(job models.NotificationJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal notification job: %w", err)
	}

	ctx := context.Background()
	if err := l.rabbitmq.Publish(
		ctx,
		infrastructure.NotificationExchange,
		infrastructure.EmailOrderRoutingKey,
		data,
	); err != nil {
		return fmt.Errorf("failed to publish to RabbitMQ: %w", err)
	}

	return nil
}

// Stop stops the listener
func (l *OrderEventListener) Stop() error {
	if l.sub != nil {
		if err := l.sub.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
		log.Println("Order event listener stopped")
	}
	return nil
}
