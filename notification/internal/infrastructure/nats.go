package infrastructure

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// NATS holds the NATS connection and JetStream context
type NATS struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// NewNATS creates a new NATS connection with JetStream
func NewNATS(url string) (*NATS, error) {
	opts := []nats.Option{
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Println("NATS connection closed")
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Setup streams if they don't exist
	n := &NATS{
		conn: conn,
		js:   js,
	}

	if err := n.setupStreams(); err != nil {
		log.Printf("Warning: failed to setup JetStream streams (may already exist): %v", err)
	}

	log.Println("NATS connected with JetStream")
	return n, nil
}

// setupStreams creates the required JetStream streams
func (n *NATS) setupStreams() error {
	// Create or update ORDERS stream for order events
	_, err := n.js.AddStream(&nats.StreamConfig{
		Name:        "ORDERS",
		Description: "Order events stream",
		Subjects:    []string{"order.>"},
		Retention:   nats.WorkQueuePolicy,
		MaxAge:      24 * time.Hour,
		Storage:     nats.FileStorage,
		Replicas:    1,
		Discard:     nats.DiscardOld,
	})
	if err != nil {
		// Stream might already exist, try to update
		_, err = n.js.UpdateStream(&nats.StreamConfig{
			Name:        "ORDERS",
			Description: "Order events stream",
			Subjects:    []string{"order.>"},
			Retention:   nats.WorkQueuePolicy,
			MaxAge:      24 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
			Discard:     nats.DiscardOld,
		})
		if err != nil {
			return fmt.Errorf("failed to create/update ORDERS stream: %w", err)
		}
	}

	return nil
}

// Subscribe subscribes to a NATS subject with JetStream
func (n *NATS) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return n.conn.Subscribe(subject, handler)
}

// QueueSubscribe subscribes to a NATS subject with a queue group
func (n *NATS) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return n.conn.QueueSubscribe(subject, queue, handler)
}

// JetStreamSubscribe subscribes to a JetStream subject
func (n *NATS) JetStreamSubscribe(subject string, handler nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	return n.js.Subscribe(subject, handler, opts...)
}

// JetStreamQueueSubscribe subscribes to a JetStream subject with a queue group
func (n *NATS) JetStreamQueueSubscribe(subject, queue string, handler nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	return n.js.QueueSubscribe(subject, queue, handler, opts...)
}

// Publish publishes a message to a NATS subject
func (n *NATS) Publish(subject string, data []byte) error {
	return n.conn.Publish(subject, data)
}

// JetStreamPublish publishes a message to a JetStream subject
func (n *NATS) JetStreamPublish(subject string, data []byte) (*nats.PubAck, error) {
	return n.js.Publish(subject, data)
}

// JetStream returns the JetStream context
func (n *NATS) JetStream() nats.JetStreamContext {
	return n.js
}

// Conn returns the NATS connection
func (n *NATS) Conn() *nats.Conn {
	return n.conn
}

// Close closes the NATS connection
func (n *NATS) Close() {
	if n.conn != nil {
		n.conn.Drain()
		n.conn.Close()
		log.Println("NATS connection closed")
	}
}
