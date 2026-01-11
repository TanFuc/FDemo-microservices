package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/tafu/payment-service/internal/usecase"
)

// ReconciliationJob runs periodic reconciliation of pending payments
type ReconciliationJob struct {
	uc       *usecase.PaymentUseCase
	cron     *cron.Cron
	logger   *slog.Logger
	schedule string
	timeout  time.Duration
	olderThan time.Duration
}

// ReconciliationConfig holds configuration for the reconciliation job
type ReconciliationConfig struct {
	Schedule  string        // Cron schedule (e.g., "*/10 * * * *" for every 10 minutes)
	Timeout   time.Duration // Maximum execution time
	OlderThan time.Duration // Only reconcile transactions older than this
}

// NewReconciliationJob creates a new reconciliation job
func NewReconciliationJob(
	uc *usecase.PaymentUseCase,
	config ReconciliationConfig,
	logger *slog.Logger,
) *ReconciliationJob {
	// Set defaults
	if config.Schedule == "" {
		config.Schedule = "*/10 * * * *" // Every 10 minutes
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.OlderThan == 0 {
		config.OlderThan = 10 * time.Minute
	}

	return &ReconciliationJob{
		uc:        uc,
		cron:      cron.New(cron.WithSeconds()),
		logger:    logger,
		schedule:  config.Schedule,
		timeout:   config.Timeout,
		olderThan: config.OlderThan,
	}
}

// Start starts the reconciliation job
func (j *ReconciliationJob) Start() error {
	_, err := j.cron.AddFunc(j.schedule, j.run)
	if err != nil {
		return err
	}

	j.cron.Start()
	j.logger.Info("reconciliation job started",
		"schedule", j.schedule,
		"older_than", j.olderThan,
	)
	return nil
}

// Stop stops the reconciliation job
func (j *ReconciliationJob) Stop() context.Context {
	return j.cron.Stop()
}

// Run executes the reconciliation job immediately
func (j *ReconciliationJob) Run() {
	j.run()
}

// run is the internal execution function
func (j *ReconciliationJob) run() {
	ctx, cancel := context.WithTimeout(context.Background(), j.timeout)
	defer cancel()

	j.logger.Info("starting reconciliation job")
	startTime := time.Now()

	synced, failed, err := j.uc.ReconcilePendingPayments(ctx, j.olderThan)

	duration := time.Since(startTime)

	if err != nil {
		j.logger.Error("reconciliation job failed",
			"error", err,
			"duration", duration,
		)
		return
	}

	j.logger.Info("reconciliation job completed",
		"synced", synced,
		"failed", failed,
		"duration", duration,
	)
}
