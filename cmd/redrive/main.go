// Command redrive drains the dead-letter stream: for each dead-lettered event
// it republishes the original envelope onto its original subject (so the normal
// consumer picks it up again) and then removes it from the dead-letter stream.
//
// It is meant to be run by hand after the underlying cause of the failures has
// been fixed. Messages whose republish fails are left in the dead-letter stream
// so a later run can retry them.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/zzokki81/eventmesh/pkg/broker"
	"github.com/zzokki81/eventmesh/pkg/broker/jetstream"
	"github.com/zzokki81/eventmesh/pkg/event"
	"github.com/zzokki81/eventmesh/pkg/nats"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load("notifier/.env")

	cfg := nats.Config{}
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if cfg.URL == "" {
		cfg.URL = "nats://localhost:4222"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nc, err := nats.NewConnection(cfg)
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}
	defer nc.Drain() //nolint:errcheck

	js, err := nats.NewJetStream(ctx, nc)
	if err != nil {
		return fmt.Errorf("init jetstream: %w", err)
	}

	stream, err := js.Stream(ctx, cfg.DLQStreamName)
	if err != nil {
		return fmt.Errorf("get dlq stream %q: %w", cfg.DLQStreamName, err)
	}

	info, err := stream.Info(ctx)
	if err != nil {
		return fmt.Errorf("read dlq stream info: %w", err)
	}
	if info.State.Msgs == 0 {
		fmt.Printf("dead-letter stream %q is empty, nothing to redrive\n", cfg.DLQStreamName)
		return nil
	}

	publisher := jetstream.NewPublisher(js)

	var redriven, failed int
	for seq := info.State.FirstSeq; seq <= info.State.LastSeq; seq++ {
		raw, err := stream.GetMsg(ctx, seq)
		if err != nil {
			// Sequence was already deleted (e.g. a previous redrive); skip it.
			continue
		}

		if err := redriveOne(ctx, publisher, raw.Data); err != nil {
			fmt.Fprintf(os.Stderr, "seq %d: redrive failed, leaving in dlq: %v\n", seq, err)
			failed++
			continue
		}

		if err := stream.DeleteMsg(ctx, seq); err != nil {
			fmt.Fprintf(os.Stderr, "seq %d: republished but failed to remove from dlq: %v\n", seq, err)
			failed++
			continue
		}
		redriven++
	}

	fmt.Printf("redrive complete: %d redriven, %d failed, from stream %q\n", redriven, failed, cfg.DLQStreamName)
	return nil
}

// redriveOne unwraps a dead letter and republishes the original envelope onto
// its original subject.
func redriveOne(ctx context.Context, publisher broker.Publisher, data []byte) error {
	var dl event.DeadLetter
	if err := json.Unmarshal(data, &dl); err != nil {
		return fmt.Errorf("decode dead letter: %w", err)
	}
	if dl.Subject == "" {
		return errors.New("dead letter has no original subject")
	}

	payload, err := json.Marshal(dl.Event)
	if err != nil {
		return fmt.Errorf("encode envelope: %w", err)
	}

	if err := publisher.Publish(ctx, dl.Subject, payload); err != nil {
		return fmt.Errorf("republish to %s: %w", dl.Subject, err)
	}
	return nil
}
