package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zzokki81/eventmesh/notifier/dedup"
	"github.com/zzokki81/eventmesh/notifier/domain"
	"github.com/zzokki81/eventmesh/notifier/email"
)

// testDeduplicator is a configurable dedup.Deduplicator for tests. It records
// how many times Complete and Release were called so assertions can verify the
// two-phase flow.
type testDeduplicator struct {
	claimStatus dedup.Status
	claimErr    error
	completeErr error
	releaseErr  error

	completeCalls int
	releaseCalls  int
}

func (m *testDeduplicator) Claim(context.Context, string) (dedup.Status, error) {
	return m.claimStatus, m.claimErr
}

func (m *testDeduplicator) Complete(context.Context, string) error {
	m.completeCalls++
	return m.completeErr
}

func (m *testDeduplicator) Release(context.Context, string) error {
	m.releaseCalls++
	return m.releaseErr
}

// testSender records every Send attempt and can be made to fail.
type testSender struct {
	calls int
	sent  []email.Message
	err   error
}

func (m *testSender) Send(_ context.Context, msg email.Message) error {
	m.calls++
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, msg)
	return nil
}

func newTestNotifier(d dedup.Deduplicator, s email.Sender) Notifier {
	return NewNotifier(d, s, slog.New(slog.DiscardHandler))
}

func sampleOrder() domain.OrderCreated {
	return domain.OrderCreated{
		OrderID:   uuid.New(),
		UserEmail: "buyer@example.com",
		Amount:    decimal.RequireFromString("42.00"),
	}
}

func TestProcessOrderCreated(t *testing.T) {
	errSend := errors.New("smtp down")
	errClaim := errors.New("redis down")
	errComplete := errors.New("redis blip")

	tests := []struct {
		name        string
		claimStatus dedup.Status
		claimErr    error
		sendErr     error
		completeErr error

		// wantErr is the error the call should return (matched with errors.Is);
		// nil means it should succeed. The service wraps injected errors, so the
		// same sentinel can be both injected and expected.
		wantErr      error
		wantSends    int
		wantComplete int
		wantRelease  int
	}{
		{
			name:        "new event sends and completes",
			claimStatus: dedup.StatusNew,
			wantSends:   1, wantComplete: 1,
		},
		{
			name:        "already completed skips",
			claimStatus: dedup.StatusCompleted,
		},
		{
			name:        "in progress returns ErrInProgress",
			claimStatus: dedup.StatusInProgress,
			wantErr:     domain.ErrInProgress,
		},
		{
			name:        "send failure releases claim",
			claimStatus: dedup.StatusNew,
			sendErr:     errSend,
			wantErr:     errSend,
			wantSends:   1, wantRelease: 1,
		},
		{
			name:     "claim failure does not send",
			claimErr: errClaim,
			wantErr:  errClaim,
		},
		{
			name:        "complete failure still succeeds",
			claimStatus: dedup.StatusNew,
			completeErr: errComplete,
			wantSends:   1, wantComplete: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &testDeduplicator{
				claimStatus: tt.claimStatus,
				claimErr:    tt.claimErr,
				completeErr: tt.completeErr,
			}
			s := &testSender{err: tt.sendErr}

			err := newTestNotifier(d, s).ProcessOrderCreated(context.Background(), "evt-1", sampleOrder())

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: want %v, got %v", tt.wantErr, err)
			}
			if s.calls != tt.wantSends {
				t.Errorf("sends: want %d, got %d", tt.wantSends, s.calls)
			}
			if d.completeCalls != tt.wantComplete {
				t.Errorf("completes: want %d, got %d", tt.wantComplete, d.completeCalls)
			}
			if d.releaseCalls != tt.wantRelease {
				t.Errorf("releases: want %d, got %d", tt.wantRelease, d.releaseCalls)
			}
		})
	}
}

// TestProcessOrderCreated_BuildsEmail checks the email is addressed and filled
// from the order — a content assertion distinct from the control-flow table.
func TestProcessOrderCreated_BuildsEmail(t *testing.T) {
	d := &testDeduplicator{claimStatus: dedup.StatusNew}
	s := &testSender{}
	o := sampleOrder()

	if err := newTestNotifier(d, s).ProcessOrderCreated(context.Background(), "evt-1", o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(s.sent) != 1 {
		t.Fatalf("want 1 sent message, got %d", len(s.sent))
	}
	msg := s.sent[0]
	if msg.To != o.UserEmail {
		t.Errorf("recipient: want %q, got %q", o.UserEmail, msg.To)
	}
	if !strings.Contains(msg.Subject, o.OrderID.String()) {
		t.Errorf("subject %q should contain order id %s", msg.Subject, o.OrderID)
	}
	if !strings.Contains(msg.Plain, o.Amount.String()) {
		t.Errorf("body should contain amount %s", o.Amount)
	}
}
