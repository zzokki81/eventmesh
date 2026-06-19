package dedup_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/zzokki81/eventmesh/notifier/dedup"
)

// newTestStore returns a Store backed by an in-memory Redis (miniredis), along
// with the miniredis handle so tests can fast-forward its clock for TTL checks.
func newTestStore(t *testing.T, lockTTL, doneTTL time.Duration) (*dedup.Store, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return dedup.NewStore(client, lockTTL, doneTTL), mr
}

func TestClaim_NewThenInProgress(t *testing.T) {
	store, _ := newTestStore(t, time.Minute, time.Hour)
	ctx := context.Background()

	status, err := store.Claim(ctx, "evt-1")
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if status != dedup.StatusNew {
		t.Fatalf("first claim: want StatusNew, got %v", status)
	}

	// A second claim while the first is still held must report in-progress.
	status, err = store.Claim(ctx, "evt-1")
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if status != dedup.StatusInProgress {
		t.Fatalf("second claim: want StatusInProgress, got %v", status)
	}
}

func TestClaim_AfterComplete(t *testing.T) {
	store, _ := newTestStore(t, time.Minute, time.Hour)
	ctx := context.Background()

	if _, err := store.Claim(ctx, "evt-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := store.Complete(ctx, "evt-1"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	status, err := store.Claim(ctx, "evt-1")
	if err != nil {
		t.Fatalf("claim after complete: %v", err)
	}
	if status != dedup.StatusCompleted {
		t.Fatalf("want StatusCompleted, got %v", status)
	}
}

func TestClaim_AfterRelease(t *testing.T) {
	store, _ := newTestStore(t, time.Minute, time.Hour)
	ctx := context.Background()

	if _, err := store.Claim(ctx, "evt-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := store.Release(ctx, "evt-1"); err != nil {
		t.Fatalf("release: %v", err)
	}

	// After release the event looks new again, so a retry can re-process it.
	status, err := store.Claim(ctx, "evt-1")
	if err != nil {
		t.Fatalf("claim after release: %v", err)
	}
	if status != dedup.StatusNew {
		t.Fatalf("want StatusNew, got %v", status)
	}
}

func TestClaim_LockExpiresAllowsReclaim(t *testing.T) {
	lockTTL := time.Minute
	store, mr := newTestStore(t, lockTTL, time.Hour)
	ctx := context.Background()

	if _, err := store.Claim(ctx, "evt-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	// Simulate a crashed attempt: the claim is never completed, but its TTL
	// lapses. A later delivery must be able to re-claim and process the event.
	mr.FastForward(lockTTL + time.Second)

	status, err := store.Claim(ctx, "evt-1")
	if err != nil {
		t.Fatalf("claim after lock expiry: %v", err)
	}
	if status != dedup.StatusNew {
		t.Fatalf("want StatusNew after lock expiry, got %v", status)
	}
}

func TestClaim_CompletionOutlivesLock(t *testing.T) {
	lockTTL := time.Minute
	doneTTL := time.Hour
	store, mr := newTestStore(t, lockTTL, doneTTL)
	ctx := context.Background()

	if _, err := store.Claim(ctx, "evt-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := store.Complete(ctx, "evt-1"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Past the short lock window but well within the completion window: a late
	// redelivery must still be recognized as already processed.
	mr.FastForward(lockTTL + time.Minute)

	status, err := store.Claim(ctx, "evt-1")
	if err != nil {
		t.Fatalf("claim after lock window: %v", err)
	}
	if status != dedup.StatusCompleted {
		t.Fatalf("want StatusCompleted, got %v", status)
	}
}
