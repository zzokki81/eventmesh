package broker

import (
	"context"

	"github.com/zzokki81/eventmesh/pkg/event"
)

// Message is a single outbound message: pre-serialized data addressed to a
// subject, with optional broker-level headers. Headers is for
// caller-supplied metadata (e.g. a dedup key, content type); implementations
// are free to add their own alongside it, such as trace context injected
// from ctx.
type Message struct {
	// Subject is the destination the message is published to.
	Subject string

	// Data is the pre-serialized message payload.
	Data []byte

	// Headers carries caller-supplied broker-level metadata alongside Data.
	Headers map[string]string
}

// Publisher emits envelopes to the broker. Implementations are responsible
// for serialization and delivery semantics.
type Publisher interface {
	// Publish sends msg and waits for the server ack.
	Publish(ctx context.Context, msg Message) error
}

// DeadLetterer moves an event that can no longer be processed to the
// dead-letter queue. It is the single, narrow capability a Subscriber needs to
// quarantine poison messages, keeping the subscriber free of the broader
// Publisher (and of how a dead letter is routed and serialized).
type DeadLetterer interface {
	// DeadLetter records dl in the dead-letter queue.
	DeadLetter(ctx context.Context, dl event.DeadLetter) error
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
