package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/tafu/analytics-service/internal/config"
	"github.com/tafu/analytics-service/internal/domain"
	"github.com/tafu/analytics-service/internal/infrastructure/clickhouse"
	natsClient "github.com/tafu/analytics-service/internal/infrastructure/nats"
)

type BatchWorker struct {
	nats       *natsClient.Client
	repo       *clickhouse.EventRepository
	cfg        config.WorkerConfig
	buffer     []domain.UserEvent
	bufferLock sync.Mutex
	flushCh    chan struct{}
}

func NewBatchWorker(
	nats *natsClient.Client,
	repo *clickhouse.EventRepository,
	cfg config.WorkerConfig,
) *BatchWorker {
	return &BatchWorker{
		nats:    nats,
		repo:    repo,
		cfg:     cfg,
		buffer:  make([]domain.UserEvent, 0, cfg.BatchSize),
		flushCh: make(chan struct{}, 1),
	}
}

// Start begins processing messages from NATS and batching them for ClickHouse.
func (w *BatchWorker) Start(ctx context.Context) error {
	msgCh, err := w.nats.Subscribe(ctx, "analytics-worker")
	if err != nil {
		return err
	}

	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()

	log.Printf("BatchWorker started - batch size: %d, flush interval: %v",
		w.cfg.BatchSize, w.cfg.FlushInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("BatchWorker shutting down, performing final flush...")
			w.flush(context.Background()) // Use background context for final flush
			return nil

		case msg, ok := <-msgCh:
			if !ok {
				log.Println("Message channel closed, performing final flush...")
				w.flush(context.Background())
				return nil
			}
			w.handleMessage(ctx, msg)

		case <-ticker.C:
			w.flush(ctx)

		case <-w.flushCh:
			w.flush(ctx)
		}
	}
}

func (w *BatchWorker) handleMessage(ctx context.Context, msg jetstream.Msg) {
	var event domain.UserEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		log.Printf("Failed to unmarshal event: %v", err)
		msg.Nak() // Negative acknowledge - will be redelivered
		return
	}

	w.bufferLock.Lock()
	w.buffer = append(w.buffer, event)
	bufferLen := len(w.buffer)
	w.bufferLock.Unlock()

	// Acknowledge the message
	if err := msg.Ack(); err != nil {
		log.Printf("Failed to ack message: %v", err)
	}

	// Trigger flush if buffer is full
	if bufferLen >= w.cfg.BatchSize {
		select {
		case w.flushCh <- struct{}{}:
		default:
			// Flush already pending
		}
	}
}

func (w *BatchWorker) flush(ctx context.Context) {
	w.bufferLock.Lock()
	if len(w.buffer) == 0 {
		w.bufferLock.Unlock()
		return
	}

	// Take ownership of current buffer and create new one
	events := w.buffer
	w.buffer = make([]domain.UserEvent, 0, w.cfg.BatchSize)
	w.bufferLock.Unlock()

	log.Printf("Flushing %d events to ClickHouse...", len(events))

	if err := w.repo.BatchInsert(ctx, events); err != nil {
		log.Printf("Failed to batch insert events: %v", err)
		// In production, you might want to:
		// 1. Retry with exponential backoff
		// 2. Send to dead letter queue
		// 3. Write to local file for recovery
		return
	}

	log.Printf("Successfully flushed %d events", len(events))
}

// BufferSize returns the current number of events in the buffer.
func (w *BatchWorker) BufferSize() int {
	w.bufferLock.Lock()
	defer w.bufferLock.Unlock()
	return len(w.buffer)
}
