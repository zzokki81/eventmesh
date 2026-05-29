package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/zzokki81/eventmesh/pkg/event"
	"github.com/zzokki81/eventmesh/pkg/broker"
)

// subscriber is the JetStream-backed broker.Subscriber implementation.
// It owns a durable pull consumer and dispatches each decoded envelope to
// the configured handler.
type subscriber struct {
	js      jetstream.JetStream
	cfg     SubscriberConfig
	handler broker.MessageHandler
	logger  *slog.Logger

	// consumeCtx is the runtime handle returned by Consume; held so that
	// shutdown can drain it without restarting the consumer.
	consumeCtx jetstream.ConsumeContext
}

// NewSubscriber assembles a JetStream-backed broker.Subscriber that will
// dispatch messages from cfg.Subject to handler.
func NewSubscriber(js jetstream.JetStream, cfg SubscriberConfig, handler broker.MessageHandler, logger *slog.Logger) broker.Subscriber {
	return &subscriber{
		js:      js,
		cfg:     cfg,
		handler: handler,
		logger:  logger,
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
		MaxDeliver:    s.cfg.MaxDeliver,
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
	var env event.Envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		s.logger.ErrorContext(ctx, "malformed envelope, terminating message",
			"err", err, "subject", msg.Subject(),
		)
		s.term(ctx, msg, err.Error())
		return
	}

	err := s.safeHandle(ctx, &env)
	if err == nil {
		if ackErr := msg.Ack(); ackErr != nil {
			s.logger.ErrorContext(ctx, "failed to ack message",
				"err", ackErr, "event_id", env.ID,
			)
		}
		return
	}

	if s.handler.IsRetryable(err) {
		s.logger.WarnContext(ctx, "handler failed, nak for retry",
			"err", err, "event_id", env.ID, "type", env.Type,
		)
		if nakErr := msg.Nak(); nakErr != nil {
			s.logger.ErrorContext(ctx, "failed to nak message",
				"err", nakErr, "event_id", env.ID,
			)
		}
		return
	}

	s.logger.ErrorContext(ctx, "handler failed with non-retryable error, terminating",
		"err", err, "event_id", env.ID, "type", env.Type,
	)
	s.term(ctx, msg, err.Error())
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
