package infrastructure

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// Exchange names
	NotificationExchange = "notification.exchange"
	DeadLetterExchange   = "notification.dlx"

	// Queue names
	EmailQueue      = "queue.email"
	DeadLetterQueue = "queue.dead_letter"

	// Routing keys
	EmailRoutingKeyPattern = "notification.email.*"
	EmailOrderRoutingKey   = "notification.email.order"
)

// RabbitMQ holds the RabbitMQ connection and channel
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQ creates a new RabbitMQ connection
func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		channel: ch,
	}

	if err := rmq.setupTopology(); err != nil {
		rmq.Close()
		return nil, fmt.Errorf("failed to setup topology: %w", err)
	}

	log.Println("RabbitMQ connected and topology configured")
	return rmq, nil
}

// setupTopology declares exchanges and queues
func (r *RabbitMQ) setupTopology() error {
	// Declare Dead Letter Exchange (Fanout)
	if err := r.channel.ExchangeDeclare(
		DeadLetterExchange, // name
		"fanout",           // type
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // arguments
	); err != nil {
		return fmt.Errorf("failed to declare DLX: %w", err)
	}

	// Declare Notification Exchange (Topic)
	if err := r.channel.ExchangeDeclare(
		NotificationExchange, // name
		"topic",              // type
		true,                 // durable
		false,                // auto-deleted
		false,                // internal
		false,                // no-wait
		nil,                  // arguments
	); err != nil {
		return fmt.Errorf("failed to declare notification exchange: %w", err)
	}

	// Declare Dead Letter Queue
	if _, err := r.channel.QueueDeclare(
		DeadLetterQueue, // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	); err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind Dead Letter Queue to DLX
	if err := r.channel.QueueBind(
		DeadLetterQueue,    // queue name
		"",                 // routing key (empty for fanout)
		DeadLetterExchange, // exchange
		false,              // no-wait
		nil,                // arguments
	); err != nil {
		return fmt.Errorf("failed to bind DLQ: %w", err)
	}

	// Declare Email Queue with DLX configuration
	args := amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
	}
	if _, err := r.channel.QueueDeclare(
		EmailQueue, // name
		true,       // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		args,       // arguments
	); err != nil {
		return fmt.Errorf("failed to declare email queue: %w", err)
	}

	// Bind Email Queue to Notification Exchange
	if err := r.channel.QueueBind(
		EmailQueue,              // queue name
		EmailRoutingKeyPattern,  // routing key
		NotificationExchange,    // exchange
		false,                   // no-wait
		nil,                     // arguments
	); err != nil {
		return fmt.Errorf("failed to bind email queue: %w", err)
	}

	return nil
}

// Publish publishes a message to an exchange
func (r *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.channel.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
}

// Consume returns a channel of deliveries for a queue
func (r *RabbitMQ) Consume(queueName string) (<-chan amqp.Delivery, error) {
	// Set QoS - process one message at a time
	if err := r.channel.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := r.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register consumer: %w", err)
	}

	return msgs, nil
}

// Channel returns the RabbitMQ channel
func (r *RabbitMQ) Channel() *amqp.Channel {
	return r.channel
}

// Close closes the RabbitMQ connection
func (r *RabbitMQ) Close() error {
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			log.Printf("Error closing RabbitMQ channel: %v", err)
		}
	}
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			return fmt.Errorf("error closing RabbitMQ connection: %w", err)
		}
	}
	log.Println("RabbitMQ connection closed")
	return nil
}
