package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"microservices/analytic/internal/config"
	"microservices/analytic/internal/domain"
	"microservices/analytic/internal/infrastructure/clickhouse"
	natsClient "microservices/analytic/internal/infrastructure/nats"
	"microservices/analytic/internal/worker"
)

// TestBatchIngestion tests that 5000 events can be ingested without data loss.
// This test requires running ClickHouse and NATS instances.
func TestBatchIngestion(t *testing.T) {
	if testing.Short() || os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := config.Load()

	// Connect to ClickHouse
	chConn, err := clickhouse.NewConnection(cfg.ClickHouse)
	if err != nil {
		t.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer chConn.Close()

	// Initialize schema
	if err := clickhouse.InitSchema(ctx, chConn); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Clear existing test data
	testRunID := uuid.New().String()
	repo := clickhouse.NewEventRepository(chConn)

	// Connect to NATS
	nats, err := natsClient.NewClient(cfg.NATS)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nats.Close()

	// Start batch worker with faster flush for testing
	testCfg := config.WorkerConfig{
		BatchSize:     1000,
		FlushInterval: 1 * time.Second,
	}
	batchWorker := worker.NewBatchWorker(nats, repo, testCfg)

	workerCtx, workerCancel := context.WithCancel(ctx)
	var workerWg sync.WaitGroup
	workerWg.Add(1)
	go func() {
		defer workerWg.Done()
		batchWorker.Start(workerCtx)
	}()

	// Fire 5000 events in ~1 second using goroutines
	const eventCount = 5000
	const concurrency = 50

	t.Logf("Publishing %d events with test run ID: %s", eventCount, testRunID)

	var wg sync.WaitGroup
	eventsPerGoroutine := eventCount / concurrency

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := domain.UserEvent{
					EventID:   uuid.New(),
					UserID:    fmt.Sprintf("test-user-%d", workerID),
					EventType: "view_item",
					Metadata:  fmt.Sprintf(`{"sku_id":"test-sku","test_run":"%s"}`, testRunID),
					URL:       "https://example.com/products/test",
					IPAddress: "127.0.0.1",
					UserAgent: "LoadTest/1.0",
					CreatedAt: time.Now().UTC(),
				}

				if err := nats.Publish(ctx, event); err != nil {
					t.Errorf("Failed to publish event: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	publishDuration := time.Since(startTime)
	t.Logf("Published %d events in %v", eventCount, publishDuration)

	// Wait for events to be flushed (2 seconds as per spec + buffer)
	t.Log("Waiting for events to be flushed to ClickHouse...")
	time.Sleep(3 * time.Second)

	// Stop worker to ensure final flush
	workerCancel()
	workerWg.Wait()

	// Query ClickHouse for count
	var count uint64
	row := chConn.QueryRow(ctx, `
		SELECT count(*)
		FROM analytics.user_events
		WHERE JSONExtractString(metadata, 'test_run') = $1
	`, testRunID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count events: %v", err)
	}

	t.Logf("Found %d events in ClickHouse", count)

	// Verify count matches
	if count != eventCount {
		t.Errorf("Expected %d events, got %d (data loss detected)", eventCount, count)
	} else {
		t.Logf("SUCCESS: All %d events were ingested without data loss", eventCount)
	}

	// Cleanup: Delete test data
	if err := chConn.Exec(ctx, `
		ALTER TABLE analytics.user_events DELETE
		WHERE JSONExtractString(metadata, 'test_run') = $1
	`, testRunID); err != nil {
		t.Logf("Warning: Failed to cleanup test data: %v", err)
	}
}

// TestHighThroughput tests sustained high throughput ingestion.
func TestHighThroughput(t *testing.T) {
	if testing.Short() || os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := config.Load()

	chConn, err := clickhouse.NewConnection(cfg.ClickHouse)
	if err != nil {
		t.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer chConn.Close()

	if err := clickhouse.InitSchema(ctx, chConn); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	repo := clickhouse.NewEventRepository(chConn)

	nats, err := natsClient.NewClient(cfg.NATS)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nats.Close()

	testCfg := config.WorkerConfig{
		BatchSize:     1000,
		FlushInterval: 2 * time.Second,
	}
	batchWorker := worker.NewBatchWorker(nats, repo, testCfg)

	workerCtx, workerCancel := context.WithCancel(ctx)
	go batchWorker.Start(workerCtx)

	testRunID := uuid.New().String()
	const eventsPerSecond = 2000
	const durationSeconds = 5
	const totalEvents = eventsPerSecond * durationSeconds

	t.Logf("Sustained throughput test: %d events/sec for %d seconds", eventsPerSecond, durationSeconds)

	var publishedCount int64
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	deadline := startTime.Add(time.Duration(durationSeconds) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ticker.C:
			// Publish batch
			for i := 0; i < eventsPerSecond; i++ {
				event := domain.UserEvent{
					EventID:   uuid.New(),
					UserID:    "throughput-test-user",
					EventType: "view_item",
					Metadata:  fmt.Sprintf(`{"sku_id":"throughput-test","test_run":"%s"}`, testRunID),
					URL:       "https://example.com/throughput-test",
					IPAddress: "127.0.0.1",
					UserAgent: "ThroughputTest/1.0",
					CreatedAt: time.Now().UTC(),
				}
				if err := nats.Publish(ctx, event); err != nil {
					t.Errorf("Publish error: %v", err)
					continue
				}
				publishedCount++
			}
		case <-ctx.Done():
			t.Fatal("Context cancelled")
		}
	}

	t.Logf("Published %d events", publishedCount)

	// Wait for flush
	time.Sleep(5 * time.Second)
	workerCancel()

	var count uint64
	row := chConn.QueryRow(ctx, `
		SELECT count(*)
		FROM analytics.user_events
		WHERE JSONExtractString(metadata, 'test_run') = $1
	`, testRunID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count: %v", err)
	}

	t.Logf("Ingested %d/%d events (%.2f%%)", count, publishedCount, float64(count)/float64(publishedCount)*100)

	if count < uint64(publishedCount)*95/100 {
		t.Errorf("Too much data loss: expected at least 95%% of %d events", publishedCount)
	}

	// Cleanup
	chConn.Exec(ctx, `ALTER TABLE analytics.user_events DELETE WHERE JSONExtractString(metadata, 'test_run') = $1`, testRunID)
}

// BenchmarkEventPublish benchmarks the event publishing throughput.
func BenchmarkEventPublish(b *testing.B) {
	ctx := context.Background()
	cfg := config.Load()

	nats, err := natsClient.NewClient(cfg.NATS)
	if err != nil {
		b.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nats.Close()

	event := domain.UserEvent{
		EventID:   uuid.New(),
		UserID:    "bench-user",
		EventType: "view_item",
		Metadata:  `{"sku_id":"bench-sku"}`,
		URL:       "https://example.com/bench",
		IPAddress: "127.0.0.1",
		UserAgent: "Benchmark/1.0",
		CreatedAt: time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.EventID = uuid.New()
		if err := nats.Publish(ctx, event); err != nil {
			b.Fatalf("Publish failed: %v", err)
		}
	}
}

// BenchmarkEventSerialization benchmarks JSON serialization of events.
func BenchmarkEventSerialization(b *testing.B) {
	event := domain.UserEvent{
		EventID:   uuid.New(),
		UserID:    "bench-user",
		EventType: "view_item",
		Metadata:  `{"sku_id":"bench-sku","category":"electronics"}`,
		URL:       "https://example.com/products/bench-product",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		CreatedAt: time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(event)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}
