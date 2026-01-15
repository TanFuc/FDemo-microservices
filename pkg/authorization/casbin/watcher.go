package casbin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"microservices/pkg/authorization"
)

// WatcherMessage represents a message sent through the watcher
type WatcherMessage struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "policy_change", "role_change", "invalidate"
	Method    string    `json:"method"`
	Params    []string  `json:"params,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // Instance ID that sent the message
}

// Watcher implements a Redis-based distributed watcher for Casbin
type Watcher struct {
	client     *redis.Client
	channel    string
	instanceID string
	callback   func(string)
	pubsub     *redis.PubSub
	logger     *slog.Logger

	mu      sync.RWMutex
	running bool
	cancel  context.CancelFunc
}

// WatcherConfig represents the configuration for the watcher
type WatcherConfig struct {
	Addr         string
	Password     string
	DB           int
	Channel      string
	InstanceID   string
	PoolSize     int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultWatcherConfig returns the default watcher configuration
func DefaultWatcherConfig() *WatcherConfig {
	return &WatcherConfig{
		Addr:         "localhost:6379",
		DB:           0,
		Channel:      "casbin-watcher",
		PoolSize:     5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// NewWatcher creates a new Redis-based watcher
func NewWatcher(cfg *WatcherConfig, logger *slog.Logger) (*Watcher, error) {
	if cfg == nil {
		cfg = DefaultWatcherConfig()
	}

	if cfg.InstanceID == "" {
		cfg.InstanceID = fmt.Sprintf("instance-%d", time.Now().UnixNano())
	}

	if logger == nil {
		logger = slog.Default()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%w: failed to connect to Redis for watcher: %v", authorization.ErrConnectionFailed, err)
	}

	return &Watcher{
		client:     client,
		channel:    cfg.Channel,
		instanceID: cfg.InstanceID,
		logger:     logger,
	}, nil
}

// NewWatcherWithClient creates a new watcher with an existing Redis client
func NewWatcherWithClient(client *redis.Client, channel, instanceID string, logger *slog.Logger) *Watcher {
	if channel == "" {
		channel = "casbin-watcher"
	}
	if instanceID == "" {
		instanceID = fmt.Sprintf("instance-%d", time.Now().UnixNano())
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Watcher{
		client:     client,
		channel:    channel,
		instanceID: instanceID,
		logger:     logger,
	}
}

// SetUpdateCallback sets the callback function that is called when a policy update is received
func (w *Watcher) SetUpdateCallback(callback func(string)) error {
	w.mu.Lock()
	w.callback = callback
	w.mu.Unlock()
	return nil
}

// Update publishes a policy update to all watchers
func (w *Watcher) Update() error {
	return w.UpdateWithParams("Update", nil)
}

// UpdateForAddPolicy publishes a notification for added policy
func (w *Watcher) UpdateForAddPolicy(sec, ptype string, params ...string) error {
	return w.UpdateWithParams("AddPolicy", append([]string{sec, ptype}, params...))
}

// UpdateForRemovePolicy publishes a notification for removed policy
func (w *Watcher) UpdateForRemovePolicy(sec, ptype string, params ...string) error {
	return w.UpdateWithParams("RemovePolicy", append([]string{sec, ptype}, params...))
}

// UpdateForRemoveFilteredPolicy publishes a notification for filtered policy removal
func (w *Watcher) UpdateForRemoveFilteredPolicy(sec, ptype string, fieldIndex int, fieldValues ...string) error {
	params := append([]string{sec, ptype, fmt.Sprintf("%d", fieldIndex)}, fieldValues...)
	return w.UpdateWithParams("RemoveFilteredPolicy", params)
}

// UpdateForSavePolicy publishes a notification for saved policies
func (w *Watcher) UpdateForSavePolicy(model interface{}) error {
	return w.UpdateWithParams("SavePolicy", nil)
}

// UpdateForAddPolicies publishes a notification for added policies
func (w *Watcher) UpdateForAddPolicies(sec, ptype string, rules ...[]string) error {
	return w.UpdateWithParams("AddPolicies", []string{sec, ptype})
}

// UpdateForRemovePolicies publishes a notification for removed policies
func (w *Watcher) UpdateForRemovePolicies(sec, ptype string, rules ...[]string) error {
	return w.UpdateWithParams("RemovePolicies", []string{sec, ptype})
}

// UpdateWithParams publishes an update with parameters
func (w *Watcher) UpdateWithParams(method string, params []string) error {
	msg := &WatcherMessage{
		ID:        fmt.Sprintf("%s-%d", w.instanceID, time.Now().UnixNano()),
		Type:      "policy_change",
		Method:    method,
		Params:    params,
		Timestamp: time.Now(),
		Source:    w.instanceID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal watcher message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.client.Publish(ctx, w.channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish watcher message: %w", err)
	}

	w.logger.Debug("published watcher message", "method", method, "params", params)
	return nil
}

// Start starts listening for policy updates
func (w *Watcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.pubsub = w.client.Subscribe(ctx, w.channel)
	w.running = true
	w.mu.Unlock()

	// Wait for subscription confirmation
	_, err := w.pubsub.Receive(ctx)
	if err != nil {
		w.Stop()
		return fmt.Errorf("failed to subscribe to watcher channel: %w", err)
	}

	go w.listen(ctx)

	w.logger.Info("watcher started", "channel", w.channel, "instance", w.instanceID)
	return nil
}

// Stop stops listening for policy updates
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	w.running = false
	if w.cancel != nil {
		w.cancel()
	}
	if w.pubsub != nil {
		w.pubsub.Close()
	}

	w.logger.Info("watcher stopped", "instance", w.instanceID)
	return nil
}

// Close closes the watcher and its connections
func (w *Watcher) Close() error {
	if err := w.Stop(); err != nil {
		return err
	}
	return w.client.Close()
}

// listen listens for messages from Redis pub/sub
func (w *Watcher) listen(ctx context.Context) {
	ch := w.pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.handleMessage(msg.Payload)
		}
	}
}

// handleMessage handles a received message
func (w *Watcher) handleMessage(payload string) {
	var msg WatcherMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		w.logger.Error("failed to unmarshal watcher message", "error", err)
		return
	}

	// Ignore messages from ourselves
	if msg.Source == w.instanceID {
		w.logger.Debug("ignoring self message", "id", msg.ID)
		return
	}

	w.logger.Debug("received watcher message",
		"id", msg.ID,
		"method", msg.Method,
		"source", msg.Source,
	)

	// Call the callback
	w.mu.RLock()
	callback := w.callback
	w.mu.RUnlock()

	if callback != nil {
		callback(msg.Method)
	}
}

// IsRunning returns whether the watcher is currently running
func (w *Watcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetInstanceID returns the instance ID of this watcher
func (w *Watcher) GetInstanceID() string {
	return w.instanceID
}

// Ping checks the watcher connection
func (w *Watcher) Ping(ctx context.Context) error {
	return w.client.Ping(ctx).Err()
}
