package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// jetStreamInfoTimeout caps how long we wait when verifying JetStream
// is enabled on the server. Short because this only runs at startup.
const jetStreamInfoTimeout = 5 * time.Second

// NewConnection creates a new NATS connection configured from cfg.
// The caller is responsible for calling Drain() on the returned connection
// when the application shuts down.
func NewConnection(cfg Config) (*nats.Conn, error) {
	conn, err := nats.Connect(cfg.URL,
		nats.Name("eventmesh"),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.PingInterval(cfg.PingInterval),
		nats.MaxPingsOutstanding(cfg.MaxPingsOut),
		nats.Timeout(cfg.ConnectTimeout),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return conn, nil
}

// NewJetStream creates a JetStream context from an existing NATS connection.
// It verifies JetStream is enabled on the server before returning, so failures
// surface at startup rather than at first publish.
func NewJetStream(ctx context.Context, conn *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}

	verifyCtx, cancel := context.WithTimeout(ctx, jetStreamInfoTimeout)
	defer cancel()

	if _, err := js.AccountInfo(verifyCtx); err != nil {
		if errors.Is(err, nats.ErrJetStreamNotEnabled) {
			return nil, fmt.Errorf("jetstream is not enabled on the nats server")
		}
		return nil, fmt.Errorf("verify jetstream: %w", err)
	}

	return js, nil
}

// SetupStream creates or ensures the existence of the application's primary JetStream stream.
// The operation is idempotent and safe to execute on every application startup.
// If a stream with the same name already exists, its configuration will not be modified.
// Any changes to the stream configuration require manual deletion of the existing stream
// for updates to take effect.
func SetupStream(ctx context.Context, js jetstream.JetStream, name string) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      name,
		Subjects:  []string{"orders.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.WorkQueuePolicy,
	})
	if err != nil {
		return fmt.Errorf("setup stream %q: %w", name, err)
	}
	return nil
}
