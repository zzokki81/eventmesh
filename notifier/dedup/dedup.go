package dedup

import "context"

// Deduplicator guards against processing the same event more than once.
// Consumers depend on this interface; *Store is its Redis-backed implementation.
type Deduplicator interface {
	// Claim attempts to acquire the processing claim for eventID.
	Claim(ctx context.Context, eventID string) (Status, error)

	// Complete marks eventID as fully processed.
	Complete(ctx context.Context, eventID string) error

	// Release relinquishes the claim for eventID so a later delivery can retry.
	Release(ctx context.Context, eventID string) error
}

// Compile-time check that *Store implements Deduplicator.
var _ Deduplicator = (*Store)(nil)
