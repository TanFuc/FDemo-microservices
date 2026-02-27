package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// UserRegisteredEvent mirrors auth/internal/queue/nats.go (UserRegisteredEvent)
type UserRegisteredEvent struct {
	UserID       string `json:"userId"`
	Email        string `json:"email"`
	FullName     string `json:"fullName"`
	RegisteredAt string `json:"registeredAt"`
}

// ProvisionWalletUseCase interface for provisioning wallets
type ProvisionWalletUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, email string) error
}

// UserRegisteredListener listens for user.registered events
type UserRegisteredListener struct {
	js          nats.JetStreamContext
	provisionUC ProvisionWalletUseCase
	logger      *slog.Logger
	sub         *nats.Subscription
}

// NewUserRegisteredListener creates a new user registered listener
func NewUserRegisteredListener(url string, provisionUC ProvisionWalletUseCase, logger *slog.Logger) (*UserRegisteredListener, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	return &UserRegisteredListener{
		js:          js,
		provisionUC: provisionUC,
		logger:      logger,
	}, nil
}

// Start starts the listener
func (l *UserRegisteredListener) Start(ctx context.Context) error {
	// The "USERS" stream is already created by the auth service.
	// Wallet service only creates a durable consumer on the existing stream.
	sub, err := l.js.Subscribe(
		"user.registered",
		l.handle,
		nats.Durable("wallet-user-provision-consumer"),
		nats.DeliverNew(), // only new messages after consumer creation
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(10), // retry up to 10 times on failure
	)
	if err != nil {
		return fmt.Errorf("subscribe user.registered: %w", err)
	}
	l.sub = sub
	l.logger.Info("User registered listener started")

	<-ctx.Done()
	return nil
}

// Stop stops the listener
func (l *UserRegisteredListener) Stop() {
	if l.sub != nil {
		l.sub.Unsubscribe()
	}
}

func (l *UserRegisteredListener) handle(msg *nats.Msg) {
	var event UserRegisteredEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		l.logger.Error("failed to unmarshal user.registered", "error", err)
		msg.Term() // malformed - do not retry
		return
	}

	userID, err := uuid.Parse(event.UserID)
	if err != nil {
		l.logger.Error("invalid userID in event", "user_id", event.UserID, "error", err)
		msg.Term()
		return
	}

	if err := l.provisionUC.Execute(context.Background(), userID, event.Email); err != nil {
		l.logger.Error("failed to provision wallet", "user_id", event.UserID, "error", err)
		msg.Nak() // retry
		return
	}

	l.logger.Info("wallet provisioned for user", "user_id", event.UserID)
	msg.Ack()
}
