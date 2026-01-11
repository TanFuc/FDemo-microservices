package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"microservices/profile/internal/config"
	"microservices/profile/internal/domain/service"
	"microservices/profile/pkg/logger"
)

const (
	StreamName   = "USERS"
	ConsumerName = "profile-service-user-consumer"
	Subject      = "user.registered"
)

type UserRegisteredEvent struct {
	UserID       string `json:"userId"`
	Email        string `json:"email"`
	FullName     string `json:"fullName"`
	RegisteredAt string `json:"registeredAt"`
}

type UserRegisteredListener struct {
	cfg            *config.Config
	profileService *service.ProfileService
	conn           *nats.Conn
	js             jetstream.JetStream
	consumer       jetstream.Consumer
	stopCh         chan struct{}
}

func NewUserRegisteredListener(cfg *config.Config, profileService *service.ProfileService) *UserRegisteredListener {
	return &UserRegisteredListener{
		cfg:            cfg,
		profileService: profileService,
		stopCh:         make(chan struct{}),
	}
}

func (l *UserRegisteredListener) Start(ctx context.Context) error {
	// Connect to NATS
	conn, err := nats.Connect(l.cfg.NATS.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn().Err(err).Msg("NATS disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info().Msg("NATS reconnected")
		}),
	)
	if err != nil {
		return err
	}
	l.conn = conn

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return err
	}
	l.js = js

	// Get or wait for stream
	stream, err := l.waitForStream(ctx)
	if err != nil {
		conn.Close()
		return err
	}

	// Create consumer
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          ConsumerName,
		Durable:       ConsumerName,
		FilterSubject: Subject,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
	})
	if err != nil {
		conn.Close()
		return err
	}
	l.consumer = consumer

	logger.Info().Msg("User registered listener started")

	// Start consuming
	go l.consume(ctx)

	return nil
}

func (l *UserRegisteredListener) waitForStream(ctx context.Context) (jetstream.Stream, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(2 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, nats.ErrStreamNotFound
		case <-ticker.C:
			stream, err := l.js.Stream(ctx, StreamName)
			if err == nil {
				return stream, nil
			}
			logger.Debug().Msg("Waiting for USERS stream...")
		}
	}
}

func (l *UserRegisteredListener) consume(ctx context.Context) {
	for {
		select {
		case <-l.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			msgs, err := l.consumer.Fetch(10, jetstream.FetchMaxWait(5*time.Second))
			if err != nil {
				if err != nats.ErrTimeout && err != context.DeadlineExceeded {
					logger.Error().Err(err).Msg("Error fetching messages")
				}
				time.Sleep(time.Second)
				continue
			}

			for msg := range msgs.Messages() {
				l.handleMessage(ctx, msg)
			}
		}
	}
}

func (l *UserRegisteredListener) handleMessage(ctx context.Context, msg jetstream.Msg) {
	var event UserRegisteredEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal event")
		msg.Nak()
		return
	}

	logger.Info().
		Str("userId", event.UserID).
		Str("email", event.Email).
		Msg("Processing user.registered event")

	// Create profile
	_, err := l.profileService.CreateInitialProfile(ctx, event.UserID, event.FullName, event.Email)
	if err != nil {
		logger.Error().Err(err).
			Str("userId", event.UserID).
			Msg("Failed to create profile")
		msg.Nak()
		return
	}

	logger.Info().
		Str("userId", event.UserID).
		Msg("Profile created successfully")

	msg.Ack()
}

func (l *UserRegisteredListener) Stop() {
	close(l.stopCh)
	if l.conn != nil {
		l.conn.Drain()
		l.conn.Close()
	}
	logger.Info().Msg("User registered listener stopped")
}
