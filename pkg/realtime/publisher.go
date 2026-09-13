package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

var (
	ErrNilEvent     = errors.New("realtime: event envelope cannot be nil")
	ErrClosed       = errors.New("realtime: publisher is closed")
)

// Publisher defines the contract for microservices to emit real-time events.
type Publisher interface {
	Publish(ctx context.Context, event *EventEnvelope) error
	Close() error
}

// MemoryPublisher provides an in-memory event publisher for hermetic testing.
type MemoryPublisher struct {
	events []*EventEnvelope
	mu     sync.RWMutex
	closed bool
}

// NewMemoryPublisher returns a new thread-safe in-memory publisher.
func NewMemoryPublisher() *MemoryPublisher {
	return &MemoryPublisher{
		events: make([]*EventEnvelope, 0),
	}
}

func (m *MemoryPublisher) Publish(ctx context.Context, event *EventEnvelope) error {
	if event == nil {
		return ErrNilEvent
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	m.events = append(m.events, event)
	return nil
}

func (m *MemoryPublisher) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// GetEvents returns all published events for assertion in unit tests.
func (m *MemoryPublisher) GetEvents() []*EventEnvelope {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copied := make([]*EventEnvelope, len(m.events))
	copy(copied, m.events)
	return copied
}

// Clear clears the published events buffer.
func (m *MemoryPublisher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]*EventEnvelope, 0)
}

// NatsSubjectForEvent determines the NATS subject for routing
func NatsSubjectForEvent(event *EventEnvelope) string {
	if event.UserID != "" {
		return "realtime.user." + event.UserID
	}
	if event.RoomID != "" {
		return "realtime.room." + event.RoomID
	}
	return "realtime.broadcast"
}

// SerializeEvent encodes an EventEnvelope into wire JSON
func SerializeEvent(event *EventEnvelope) ([]byte, error) {
	if event == nil {
		return nil, ErrNilEvent
	}
	return json.Marshal(event)
}

// DeserializeEvent decodes an EventEnvelope from wire JSON
func DeserializeEvent(data []byte) (*EventEnvelope, error) {
	var event EventEnvelope
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
