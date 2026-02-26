package worker

import (
	"context"
	"log/slog"
	"time"

	"microservices/order/internal/domain"
)

// DraftCleanupWorker deletes expired draft orders every day at 3:00 AM.
type DraftCleanupWorker struct {
	orderRepo domain.OrderRepository
	logger    *slog.Logger
	draftTTL  time.Duration // How old a draft must be before deletion (default: 24h)
}

// NewDraftCleanupWorker creates a new worker instance.
func NewDraftCleanupWorker(
	orderRepo domain.OrderRepository,
	logger *slog.Logger,
	draftTTL time.Duration,
) *DraftCleanupWorker {
	return &DraftCleanupWorker{
		orderRepo: orderRepo,
		logger:    logger,
		draftTTL:  draftTTL,
	}
}

// Start begins the worker loop. Blocks until ctx is cancelled.
func (w *DraftCleanupWorker) Start(ctx context.Context) {
	w.logger.Info("DraftCleanupWorker started, scheduled to run daily at 03:00 AM")

	for {
		next := nextRunAt(3, 0, 0) // 3:00:00 AM server local time
		w.logger.Info("DraftCleanupWorker: next run scheduled",
			"nextRun", next.Format(time.RFC3339),
			"waitDuration", time.Until(next).String(),
		)

		select {
		case <-ctx.Done():
			w.logger.Info("DraftCleanupWorker: stopped due to context cancellation")
			return
		case <-time.After(time.Until(next)):
			w.runCleanup(ctx)
		}
	}
}

// runCleanup performs the actual deletion of expired drafts in batches.
func (w *DraftCleanupWorker) runCleanup(ctx context.Context) {
	w.logger.Info("DraftCleanupWorker: cleanup run started")
	start := time.Now()

	cutoff := time.Now().Add(-w.draftTTL) // e.g., now - 24h
	totalDeleted := 0
	batchSize := 100

	for {
		deleted, err := w.orderRepo.DeleteExpiredDrafts(ctx, cutoff, batchSize)
		if err != nil {
			w.logger.Error("DraftCleanupWorker: error deleting batch",
				"error", err,
				"deletedSoFar", totalDeleted,
			)
			break
		}

		totalDeleted += deleted
		w.logger.Info("DraftCleanupWorker: batch completed",
			"batchDeleted", deleted,
			"totalDeleted", totalDeleted,
		)

		if deleted < batchSize {
			break // No more records to delete
		}

		// Brief pause between batches to avoid overloading the DB
		time.Sleep(100 * time.Millisecond)
	}

	w.logger.Info("DraftCleanupWorker: cleanup run finished",
		"totalDeleted", totalDeleted,
		"duration", time.Since(start).String(),
	)
}

// nextRunAt returns the next occurrence of hour:minute:second in local time.
// If the time has already passed today, it returns tomorrow's occurrence.
func nextRunAt(hour, minute, second int) time.Time {
	now := time.Now()
	loc := now.Location() // Inherit server timezone (Vietnam = UTC+7)

	target := time.Date(
		now.Year(), now.Month(), now.Day(),
		hour, minute, second, 0,
		loc,
	)

	if now.After(target) {
		target = target.Add(24 * time.Hour)
	}

	return target
}
