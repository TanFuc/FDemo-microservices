package hub

import (
	"context"
	"strings"

	"github.com/nats-io/nats.go"
	"microservices/pkg/realtime"
)

// Backplane coordinates distributed events across NATS and local Hub connections.
type Backplane struct {
	nc  *nats.Conn
	hub *Hub
	sub *nats.Subscription
}

func NewBackplane(nc *nats.Conn, hub *Hub) *Backplane {
	return &Backplane{
		nc:  nc,
		hub: hub,
	}
}

// Start begins listening to the distributed NATS subjects.
func (b *Backplane) Start(ctx context.Context) error {
	if b.nc == nil {
		return nil
	}

	sub, err := b.nc.Subscribe("realtime.>", func(msg *nats.Msg) {
		b.handleIncomingMessage(msg.Subject, msg.Data)
	})
	if err != nil {
		return err
	}

	b.sub = sub
	return nil
}

func (b *Backplane) handleIncomingMessage(subject string, data []byte) {
	// Parse event envelope to check scope
	event, err := realtime.DeserializeEvent(data)
	if err != nil {
		return
	}

	if strings.HasPrefix(subject, "realtime.user.") {
		userID := strings.TrimPrefix(subject, "realtime.user.")
		b.hub.SendToUser(userID, data)
	} else if strings.HasPrefix(subject, "realtime.room.") {
		roomID := strings.TrimPrefix(subject, "realtime.room.")
		b.hub.SendToRoom(roomID, data)
	} else {
		// Broadcast
		b.hub.Broadcast(data)
	}

	_ = event
}

// Stop unsubscribes from the backplane.
func (b *Backplane) Stop() error {
	if b.sub != nil {
		return b.sub.Unsubscribe()
	}
	return nil
}
