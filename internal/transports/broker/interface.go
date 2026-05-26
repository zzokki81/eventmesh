package broker

import (
	"context"

	"github.com/zzokki81/eventmesh/internal/pkg/event"
)

// Publisher emits envelopes to the broker. Implementations are responsible
// for serialization and delivery semantics.
type Publisher interface {
	// Publish sends env to subject and returns once the broker has
	// acknowledged the message.
	Publish(ctx context.Context, subject string, env *event.Envelope) error
}

// Subscriber binds a consumer to a MessageHandler and drives the dispatch
// loop. Implementations decode envelopes, invoke the handler, and translate
// the result into ack/nak/term against the underlying broker.
type Subscriber interface {
	// Run starts consumption and blocks until ctx is canceled; on cancel
	// it drains in-flight handlers before returning.
	Run(ctx context.Context) error
}

// MessageHandler processes decoded envelopes. Together with the retry
// classification it returns, it forms the contract a Subscriber relies on
// to decide between Ack, Nak, and Term for each message.
type MessageHandler interface {
	// Handle processes env. A nil return is treated as success and results in Ack.
	Handle(ctx context.Context, env *event.Envelope) error

	// IsRetryable classifies a handler error. Retryable errors trigger
	// redelivery (Nak); non-retryable errors are dead-lettered (Term).
	IsRetryable(err error) bool
}
