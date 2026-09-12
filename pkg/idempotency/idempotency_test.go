package idempotency

import (
	"context"
	"testing"
	"time"
)

func TestEmptyKeyHandling(t *testing.T) {
	mgr := &redisManager{keyPrefix: "test:"}
	ctx := context.Background()

	_, _, err := mgr.TryAcquire(ctx, "", time.Second)
	if err != ErrEmptyKey {
		t.Fatalf("expected ErrEmptyKey, got: %v", err)
	}

	err = mgr.SaveResult(ctx, "", CachedResponse{}, time.Second)
	if err != ErrEmptyKey {
		t.Fatalf("expected ErrEmptyKey, got: %v", err)
	}

	err = mgr.Release(ctx, "")
	if err != ErrEmptyKey {
		t.Fatalf("expected ErrEmptyKey, got: %v", err)
	}
}

func TestKeyFormatting(t *testing.T) {
	mgr := &redisManager{keyPrefix: "nexus:idemp:"}
	if mgr.lockKey("req-123") != "nexus:idemp:lock:req-123" {
		t.Fatalf("unexpected lockKey: %s", mgr.lockKey("req-123"))
	}
	if mgr.dataKey("req-123") != "nexus:idemp:data:req-123" {
		t.Fatalf("unexpected dataKey: %s", mgr.dataKey("req-123"))
	}
}
