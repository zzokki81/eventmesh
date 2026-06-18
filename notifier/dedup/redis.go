// Package dedup provides a Redis-backed, two-phase deduplication store that
// keeps an event from being processed more than once across redeliveries and
// across multiple notifier instances.
//
// The two phases are:
//   - a short-lived "in progress" claim taken before processing, and
//   - a long-lived "completed" mark written after processing succeeds.
//
// The short claim bounds how long a crashed attempt blocks redelivery: once it
// expires, another delivery can re-claim and finish the work, so a crash mid-
// processing does not silently drop the event.
package dedup

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix = "notifier:dedup:"

	valueInProgress = "in_progress"
	valueCompleted  = "completed"
)

// Status reports the outcome of a Claim.
type Status int

const (
	// StatusNew means the caller acquired the claim and must process the event.
	StatusNew Status = iota

	// StatusInProgress means another attempt currently holds the claim; the
	// caller should retry later rather than process or skip.
	StatusInProgress

	// StatusCompleted means the event was already fully processed; skip it.
	StatusCompleted
)

// Store is a Redis-backed deduplication store.
type Store struct {
	client *redis.Client

	// lockTTL bounds how long an "in progress" claim is held. It must be longer
	// than the worst-case processing time but shorter than the broker's total
	// redelivery budget (AckWait * MaxDeliver), so a crashed attempt's claim
	// clears in time for a redelivery to re-claim and finish.
	lockTTL time.Duration

	// doneTTL bounds how long a "completed" mark is kept. It should be at least
	// as long as the stream retention so a late redelivery is still recognized.
	doneTTL time.Duration
}

// NewStore returns a Store using client, with the given claim and completion TTLs.
func NewStore(client *redis.Client, lockTTL, doneTTL time.Duration) *Store {
	return &Store{client: client, lockTTL: lockTTL, doneTTL: doneTTL}
}

// Claim attempts to acquire the processing claim for eventID. It returns
// StatusNew if the claim was acquired, StatusCompleted if the event was already
// processed, or StatusInProgress if another attempt holds the claim.
func (s *Store) Claim(ctx context.Context, eventID string) (Status, error) {
	key := keyPrefix + eventID

	ok, err := s.client.SetNX(ctx, key, valueInProgress, s.lockTTL).Result()
	if err != nil {
		return 0, fmt.Errorf("dedup claim setnx: %w", err)
	}
	if ok {
		return StatusNew, nil
	}

	// The key already exists; read its value to tell "in progress" from "completed".
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		// The claim expired between SetNX and Get. Treat as in-progress so the
		// caller retries; the next delivery will re-claim cleanly.
		if errors.Is(err, redis.Nil) {
			return StatusInProgress, nil
		}
		return 0, fmt.Errorf("dedup claim get: %w", err)
	}

	if val == valueCompleted {
		return StatusCompleted, nil
	}

	return StatusInProgress, nil
}

// Complete marks eventID as fully processed, replacing the short claim with the
// long-lived completion mark.
func (s *Store) Complete(ctx context.Context, eventID string) error {
	if err := s.client.Set(ctx, keyPrefix+eventID, valueCompleted, s.doneTTL).Err(); err != nil {
		return fmt.Errorf("dedup complete: %w", err)
	}
	return nil
}

// Release removes the claim for eventID so a later delivery can retry. It is
// called when processing fails before completion.
func (s *Store) Release(ctx context.Context, eventID string) error {
	if err := s.client.Del(ctx, keyPrefix+eventID).Err(); err != nil {
		return fmt.Errorf("dedup release: %w", err)
	}
	return nil
}
