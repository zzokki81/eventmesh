package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/zzokki81/eventmesh/pkg/broker"
	"github.com/zzokki81/eventmesh/pkg/event"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// tracerName identifies this package's spans in the trace backend.
const tracerName = "github.com/zzokki81/eventmesh/pkg/broker/jetstream"

// subscriber is the JetStream-backed broker.Subscriber implementation.
// It owns a durable pull consumer and dispatches each decoded envelope to
// the configured handler.
type subscriber struct {
	js           jetstream.JetStream
	cfg          SubscriberConfig
	handler      broker.MessageHandler
	deadLetterer broker.DeadLetterer
	tracer       trace.Tracer
	logger       *slog.Logger

	// consumeCtx is the runtime handle returned by Consume; held so that
	// shutdown can drain it without restarting the consumer.
	consumeCtx jetstream.ConsumeContext
}

// NewSubscriber assembles a JetStream-backed broker.Subscriber that will
// dispatch messages from cfg.Subject to handler. The deadLetterer is used to
// move messages that can no longer be processed to the dead-letter queue.
func NewSubscriber(js jetstream.JetStream, cfg SubscriberConfig, handler broker.MessageHandler, deadLetterer broker.DeadLetterer, logger *slog.Logger) broker.Subscriber {
	return &subscriber{
		js:           js,
		cfg:          cfg,
		handler:      handler,
		deadLetterer: deadLetterer,
		tracer:       otel.Tracer(tracerName),
		logger:       logger,
	}
}

// Run creates or attaches to the durable consumer and starts consuming.
// It blocks until ctx is canceled, then drains in-flight handlers.
func (s *subscriber) Run(ctx context.Context) error {
	stream, err := s.js.Stream(ctx, s.cfg.StreamName)
	if err != nil {
		return fmt.Errorf("get stream %s: %w", s.cfg.StreamName, err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       s.cfg.ConsumerName,
		FilterSubject: s.cfg.Subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       s.cfg.AckWait,
		// MaxDeliver is unlimited on purpose. The delivery budget is enforced in
		// dispatch against cfg.MaxDeliver, and a message is removed only once it is
		// safely in the DLQ (Term) or successfully republished. If the server
		// capped deliveries itself, a DLQ-publish failure on the final attempt
		// would be Nak'd at the limit and dropped instead of retried.
		MaxDeliver: -1,
		// Deliberately no BackOff here: the growing pause is driven explicitly
		// per-attempt via NakWithDelay (see dispatch). Setting the consumer's
		// BackOff as well distorts those explicit delays.
	})
	if err != nil {
		return fmt.Errorf("create consumer %s: %w", s.cfg.ConsumerName, err)
	}

	s.consumeCtx, err = consumer.Consume(func(msg jetstream.Msg) {
		s.dispatch(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}

	s.logger.Info("subscriber started",
		"stream", s.cfg.StreamName,
		"consumer", s.cfg.ConsumerName,
		"subject", s.cfg.Subject,
	)

	<-ctx.Done()
	return s.shutdown()
}

// dispatch decodes the envelope, runs the handler under panic recovery,
// and acks/naks/terms based on the outcome.
func (s *subscriber) dispatch(ctx context.Context, msg jetstream.Msg) {
	// Continue the producer's trace: extract the context propagated through the
	// NATS headers and open a consumer span around this message's handling.
	ctx = otel.GetTextMapPropagator().Extract(ctx, natsHeaderCarrier(msg.Headers()))
	ctx, span := s.tracer.Start(ctx, "jetstream.consume",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", msg.Subject()),
		),
	)
	defer span.End()

	var env event.Envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		s.logger.ErrorContext(ctx, "malformed envelope, terminating message",
			"err", err, "subject", msg.Subject(),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "malformed envelope")
		s.term(ctx, msg, err.Error())
		return
	}
	span.SetAttributes(attribute.String("event.id", env.ID.String()))

	err := s.safeHandle(ctx, &env)
	if err == nil {
		if ackErr := msg.Ack(); ackErr != nil {
			s.logger.ErrorContext(ctx, "failed to ack message",
				"err", ackErr, "event_id", env.ID,
			)
		}
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, "handler failed")

	// Decide between retry and dead-letter. A retryable error is naked for
	// redelivery with an explicit per-attempt pause (NakWithDelay, see
	// backoffFor) until the delivery budget is spent. A non-retryable error, or a
	// retryable one that has exhausted its attempts, is dead-lettered.
	var deliveries uint64
	if meta, metaErr := msg.Metadata(); metaErr == nil {
		deliveries = meta.NumDelivered
	} else {
		s.logger.ErrorContext(ctx, "failed to read message metadata",
			"err", metaErr, "event_id", env.ID,
		)
	}

	if s.handler.IsRetryable(err) && deliveries < uint64(s.cfg.MaxDeliver) {
		delay := s.backoffFor(deliveries)
		s.logger.WarnContext(ctx, "handler failed, nak for retry",
			"err", err, "event_id", env.ID, "type", env.Type,
			"delivery", deliveries, "retry_in", delay,
		)
		if nakErr := msg.NakWithDelay(delay); nakErr != nil {
			s.logger.ErrorContext(ctx, "failed to nak message",
				"err", nakErr, "event_id", env.ID,
			)
		}
		return
	}

	s.deadLetter(ctx, msg, &env, err, deliveries)
}

// backoffFor returns the pause to apply before the next attempt, given how many
// times the message has already been delivered (1-based). The schedule slot for
// the attempt just made is used; once the schedule is exhausted its last entry
// is reused. A plain Nak ignores any pause, so the returned delay is passed to
// NakWithDelay to actually space out retries.
func (s *subscriber) backoffFor(deliveries uint64) time.Duration {
	schedule := s.cfg.RetryBackoff
	if len(schedule) == 0 {
		return 0
	}
	idx := int(deliveries) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(schedule) {
		idx = len(schedule) - 1
	}
	return schedule[idx]
}

// deadLetter moves a message that can no longer be processed to the dead-letter
// queue and then terms the original so it stops being redelivered. If the
// hand-off fails, the original is naked with backoff (not termed), so a
// transient DLQ outage does not silently drop the message and does not spin the
// hand-off in a tight loop while the DLQ is down.
func (s *subscriber) deadLetter(ctx context.Context, msg jetstream.Msg, env *event.Envelope, cause error, deliveries uint64) {
	dl := event.NewDeadLetter(*env, msg.Subject(), cause.Error(), deliveries)
	if err := s.deadLetterer.DeadLetter(ctx, dl); err != nil {
		s.logger.ErrorContext(ctx, "failed to dead-letter message, will retry",
			"err", err, "event_id", env.ID,
		)
		if nakErr := msg.NakWithDelay(s.backoffFor(deliveries)); nakErr != nil {
			s.logger.ErrorContext(ctx, "failed to nak message after dead-letter failure",
				"err", nakErr, "event_id", env.ID,
			)
		}
		return
	}

	s.logger.WarnContext(ctx, "message dead-lettered",
		"event_id", env.ID, "type", env.Type, "reason", cause.Error(),
		"deliveries", deliveries,
	)
	s.term(ctx, msg, "dead-lettered")
}

// safeHandle runs the handler with panic recovery. A panic is converted into
// an error so the dispatch branches handle it uniformly.
func (s *subscriber) safeHandle(ctx context.Context, env *event.Envelope) (err error) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.ErrorContext(ctx, "panic in handler",
				"panic", r, "event_id", env.ID, "type", env.Type,
			)
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return s.handler.Handle(ctx, env)
}

// term tells the broker to stop redelivery for this message; called when
// the payload or handler error is unrecoverable. Failures to issue Term
// are logged but not propagated.
func (s *subscriber) term(ctx context.Context, msg jetstream.Msg, reason string) {
	if termErr := msg.Term(); termErr != nil {
		s.logger.ErrorContext(ctx, "failed to term message",
			"err", termErr, "reason", reason,
		)
	}
}

// shutdown waits for in-flight handlers to complete after draining the
// consumer, ensuring no message is acknowledged or terminated mid-process.
func (s *subscriber) shutdown() error {
	s.logger.Info("subscriber draining")
	s.consumeCtx.Drain()
	s.logger.Info("subscriber stopped")
	return nil
}
