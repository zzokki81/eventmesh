package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zzokki81/eventmesh/notifier/domain"
	"github.com/zzokki81/eventmesh/pkg/event"
)

// testNotifier records the call it received and returns a configured error.
type testNotifier struct {
	called   bool
	gotEvent string
	gotOrder domain.OrderCreated
	err      error
}

func (m *testNotifier) ProcessOrderCreated(_ context.Context, eventID string, o domain.OrderCreated) error {
	m.called = true
	m.gotEvent = eventID
	m.gotOrder = o
	return m.err
}

func newTestHandler(n *testNotifier) *OrderCreatedHandler {
	return NewOrderCreatedHandler(n, slog.New(slog.DiscardHandler))
}

func mustMarshal(t *testing.T, p orderCreatedPayload) []byte {
	t.Helper()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return data
}

func TestHandle(t *testing.T) {
	validData := mustMarshal(t, orderCreatedPayload{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		UserEmail: "buyer@example.com",
		Amount:    decimal.RequireFromString("9.99"),
		Status:    "pending",
		CreatedAt: time.Now(),
	})

	tests := []struct {
		name      string
		data      []byte
		svcErr    error
		wantErr   error
		wantCalls bool
	}{
		{
			name:    "malformed payload is not delegated",
			data:    []byte("{not json"),
			wantErr: errMalformedPayload,
			// service must not run on an undecodable payload.
		},
		{
			name:      "valid payload delegates successfully",
			data:      validData,
			wantCalls: true,
		},
		{
			name:      "service error is returned",
			data:      validData,
			svcErr:    domain.ErrInProgress,
			wantErr:   domain.ErrInProgress,
			wantCalls: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &testNotifier{err: tt.svcErr}
			env := &event.Envelope{ID: uuid.New(), Data: tt.data}

			err := newTestHandler(m).Handle(context.Background(), env)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: want %v, got %v", tt.wantErr, err)
			}
			if m.called != tt.wantCalls {
				t.Errorf("service called: want %v, got %v", tt.wantCalls, m.called)
			}
		})
	}
}

// TestHandle_MapsEnvelopeToOrder checks the handler passes the envelope id and
// maps the wire payload onto the domain order.
func TestHandle_MapsEnvelopeToOrder(t *testing.T) {
	payload := orderCreatedPayload{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		UserEmail: "buyer@example.com",
		Amount:    decimal.RequireFromString("9.99"),
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	env := &event.Envelope{ID: uuid.New(), Data: mustMarshal(t, payload)}
	m := &testNotifier{}

	if err := newTestHandler(m).Handle(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.gotEvent != env.ID.String() {
		t.Errorf("event id: want %s, got %s", env.ID, m.gotEvent)
	}
	if m.gotOrder.OrderID != payload.ID {
		t.Errorf("order id: want %s, got %s", payload.ID, m.gotOrder.OrderID)
	}
	if m.gotOrder.UserEmail != payload.UserEmail {
		t.Errorf("email: want %s, got %s", payload.UserEmail, m.gotOrder.UserEmail)
	}
	if !m.gotOrder.Amount.Equal(payload.Amount) {
		t.Errorf("amount: want %s, got %s", payload.Amount, m.gotOrder.Amount)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"malformed payload is permanent", errMalformedPayload, false},
		{"wrapped malformed payload is permanent", errors.Join(errMalformedPayload, errors.New("bad json")), false},
		{"in-progress is retryable", domain.ErrInProgress, true},
		{"generic error is retryable", errors.New("redis down"), true},
	}

	var h OrderCreatedHandler
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable(%v): want %v, got %v", tt.err, tt.want, got)
			}
		})
	}
}
