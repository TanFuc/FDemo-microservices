package usercontext

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// UserReplicationSubscriber handles NATS events for user context replication
type UserReplicationSubscriber struct {
	nc           *nats.Conn
	js           nats.JetStreamContext
	repository   UserContextRepository
	serviceName  string
	subscriptions []*nats.Subscription
}

// NewUserReplicationSubscriber creates a new user replication subscriber
func NewUserReplicationSubscriber(
	nc *nats.Conn,
	repository UserContextRepository,
	serviceName string,
) (*UserReplicationSubscriber, error) {
	js, err := nc.JetStream()
	if err != nil {
		// If JetStream is not available, continue without it
		log.Printf("[%s] JetStream not available, using core NATS", serviceName)
	}

	return &UserReplicationSubscriber{
		nc:          nc,
		js:          js,
		repository:  repository,
		serviceName: serviceName,
	}, nil
}

// Start begins listening for user identity events
func (s *UserReplicationSubscriber) Start(ctx context.Context) error {
	// Subscribe to user created events
	sub1, err := s.nc.Subscribe(SubjectUserCreated, s.handleUserCreated)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectUserCreated, err)
	}
	s.subscriptions = append(s.subscriptions, sub1)

	// Subscribe to user role changed events
	sub2, err := s.nc.Subscribe(SubjectUserRoleChanged, s.handleUserRoleChanged)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectUserRoleChanged, err)
	}
	s.subscriptions = append(s.subscriptions, sub2)

	// Subscribe to user status changed events
	sub3, err := s.nc.Subscribe(SubjectUserStatusChanged, s.handleUserStatusChanged)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectUserStatusChanged, err)
	}
	s.subscriptions = append(s.subscriptions, sub3)

	// Subscribe to user deleted events
	sub4, err := s.nc.Subscribe(SubjectUserDeleted, s.handleUserDeleted)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectUserDeleted, err)
	}
	s.subscriptions = append(s.subscriptions, sub4)

	// Subscribe to profile updated events
	sub5, err := s.nc.Subscribe(SubjectUserUpdated, s.handleUserUpdated)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectUserUpdated, err)
	}
	s.subscriptions = append(s.subscriptions, sub5)

	// Subscribe to shop registered events
	sub6, err := s.nc.Subscribe(SubjectShopRegistered, s.handleShopRegistered)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectShopRegistered, err)
	}
	s.subscriptions = append(s.subscriptions, sub6)

	// Subscribe to shop approved events
	sub7, err := s.nc.Subscribe(SubjectShopApproved, s.handleShopApproved)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectShopApproved, err)
	}
	s.subscriptions = append(s.subscriptions, sub7)

	log.Printf("[%s] User replication subscriber started", s.serviceName)
	return nil
}

// Stop stops all subscriptions
func (s *UserReplicationSubscriber) Stop() {
	for _, sub := range s.subscriptions {
		_ = sub.Unsubscribe()
	}
	log.Printf("[%s] User replication subscriber stopped", s.serviceName)
}

func (s *UserReplicationSubscriber) handleUserCreated(msg *nats.Msg) {
	var event UserCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[%s] Failed to unmarshal UserCreatedEvent: %v", s.serviceName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user := &UserContext{
		UserID:      event.UserID,
		Email:       event.Email,
		DisplayName: event.DisplayName,
		Role:        event.Role,
		Status:      StatusActive,
		Version:     1,
		CreatedAt:   event.Timestamp,
		UpdatedAt:   event.Timestamp,
	}

	if err := s.repository.Upsert(ctx, user); err != nil {
		log.Printf("[%s] Failed to upsert user %s: %v", s.serviceName, event.UserID, err)
		return
	}

	log.Printf("[%s] Replicated new user: %s", s.serviceName, event.UserID)
}

func (s *UserReplicationSubscriber) handleUserRoleChanged(msg *nats.Msg) {
	var event UserRoleChangedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[%s] Failed to unmarshal UserRoleChangedEvent: %v", s.serviceName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get existing user or create new
	user, err := s.repository.FindByUserID(ctx, event.UserID)
	if err != nil {
		// Create new if not found
		user = &UserContext{
			UserID:    event.UserID,
			Role:      event.NewRole,
			Status:    StatusActive,
			CreatedAt: event.Timestamp,
		}
	}

	user.Role = event.NewRole
	user.UpdatedAt = event.Timestamp
	user.Version++

	if err := s.repository.Upsert(ctx, user); err != nil {
		log.Printf("[%s] Failed to update user role %s: %v", s.serviceName, event.UserID, err)
		return
	}

	log.Printf("[%s] Updated user role: %s -> %s", s.serviceName, event.UserID, event.NewRole)
}

func (s *UserReplicationSubscriber) handleUserStatusChanged(msg *nats.Msg) {
	var event UserStatusChangedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[%s] Failed to unmarshal UserStatusChangedEvent: %v", s.serviceName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := s.repository.FindByUserID(ctx, event.UserID)
	if err != nil {
		user = &UserContext{
			UserID:    event.UserID,
			Role:      RoleCustomer,
			CreatedAt: event.Timestamp,
		}
	}

	user.Status = event.NewStatus
	user.UpdatedAt = event.Timestamp
	user.Version++

	if err := s.repository.Upsert(ctx, user); err != nil {
		log.Printf("[%s] Failed to update user status %s: %v", s.serviceName, event.UserID, err)
		return
	}

	log.Printf("[%s] Updated user status: %s -> %s", s.serviceName, event.UserID, event.NewStatus)
}

func (s *UserReplicationSubscriber) handleUserDeleted(msg *nats.Msg) {
	var event UserDeletedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[%s] Failed to unmarshal UserDeletedEvent: %v", s.serviceName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := s.repository.FindByUserID(ctx, event.UserID)
	if err != nil {
		return // User not found, nothing to do
	}

	user.Status = StatusInactive
	user.DisplayName = "Deleted User"
	user.Email = ""
	user.UpdatedAt = event.Timestamp
	user.Version++

	if err := s.repository.Upsert(ctx, user); err != nil {
		log.Printf("[%s] Failed to soft-delete user %s: %v", s.serviceName, event.UserID, err)
		return
	}

	log.Printf("[%s] Soft-deleted user: %s", s.serviceName, event.UserID)
}

func (s *UserReplicationSubscriber) handleUserUpdated(msg *nats.Msg) {
	var event UserUpdatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[%s] Failed to unmarshal UserUpdatedEvent: %v", s.serviceName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := s.repository.FindByUserID(ctx, event.UserID)
	if err != nil {
		user = &UserContext{
			UserID:    event.UserID,
			Role:      RoleCustomer,
			Status:    StatusActive,
			CreatedAt: event.Timestamp,
		}
	}

	// Only update if event version is newer (idempotency)
	if event.Version > 0 && event.Version <= user.Version {
		log.Printf("[%s] Skipping stale update for user %s (event version: %d, local version: %d)",
			s.serviceName, event.UserID, event.Version, user.Version)
		return
	}

	if event.DisplayName != "" {
		user.DisplayName = event.DisplayName
	}
	if event.AvatarURL != "" {
		user.AvatarURL = event.AvatarURL
	}
	if event.Email != "" {
		user.Email = event.Email
	}
	user.UpdatedAt = event.Timestamp
	if event.Version > 0 {
		user.Version = event.Version
	} else {
		user.Version++
	}

	if err := s.repository.Upsert(ctx, user); err != nil {
		log.Printf("[%s] Failed to update user profile %s: %v", s.serviceName, event.UserID, err)
		return
	}

	log.Printf("[%s] Updated user profile: %s", s.serviceName, event.UserID)
}

func (s *UserReplicationSubscriber) handleShopRegistered(msg *nats.Msg) {
	var event ShopRegisteredEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[%s] Failed to unmarshal ShopRegisteredEvent: %v", s.serviceName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := s.repository.FindByUserID(ctx, event.UserID)
	if err != nil {
		user = &UserContext{
			UserID:    event.UserID,
			Role:      RoleSeller,
			Status:    StatusActive,
			CreatedAt: event.Timestamp,
		}
	}

	user.ShopID = event.ShopID
	user.Role = RoleSeller
	user.UpdatedAt = event.Timestamp
	user.Version++

	if err := s.repository.Upsert(ctx, user); err != nil {
		log.Printf("[%s] Failed to update user shop %s: %v", s.serviceName, event.UserID, err)
		return
	}

	log.Printf("[%s] Updated user shop: %s -> %s", s.serviceName, event.UserID, event.ShopID)
}

func (s *UserReplicationSubscriber) handleShopApproved(msg *nats.Msg) {
	var event ShopApprovedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[%s] Failed to unmarshal ShopApprovedEvent: %v", s.serviceName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := s.repository.FindByUserID(ctx, event.UserID)
	if err != nil {
		log.Printf("[%s] User not found for shop approval: %s", s.serviceName, event.UserID)
		return
	}

	user.ShopID = event.ShopID
	user.UpdatedAt = event.Timestamp
	user.Version++

	if err := s.repository.Upsert(ctx, user); err != nil {
		log.Printf("[%s] Failed to update user shop approval %s: %v", s.serviceName, event.UserID, err)
		return
	}

	log.Printf("[%s] Shop approved for user: %s -> %s", s.serviceName, event.UserID, event.ShopID)
}
