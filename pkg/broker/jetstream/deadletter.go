package jetstream

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zzokki81/eventmesh/pkg/broker"
	"github.com/zzokki81/eventmesh/pkg/event"
)

// deadLetterer is the JetStream-backed broker.DeadLetterer. It owns how a dead
// letter is routed and serialized: the wrapper is marshaled to JSON and
// published to "<subjectPrefix>.<original subject>", which the dead-letter
// stream binds via its "<subjectPrefix>.>" wildcard.
type deadLetterer struct {
	publisher     broker.Publisher
	subjectPrefix string
}

// NewDeadLetterer returns a broker.DeadLetterer that publishes dead letters
// through publisher, prefixing each original subject with subjectPrefix.
func NewDeadLetterer(publisher broker.Publisher, subjectPrefix string) broker.DeadLetterer {
	return &deadLetterer{publisher: publisher, subjectPrefix: subjectPrefix}
}

// DeadLetter marshals dl and publishes it to its dead-letter subject.
func (d *deadLetterer) DeadLetter(ctx context.Context, dl event.DeadLetter) error {
	data, err := json.Marshal(dl)
	if err != nil {
		return fmt.Errorf("marshal dead letter: %w", err)
	}

	subject := d.subjectPrefix + "." + dl.Subject
	if err := d.publisher.Publish(ctx, broker.Message{Subject: subject, Data: data}); err != nil {
		return fmt.Errorf("publish dead letter to %s: %w", subject, err)
	}
	return nil
}
