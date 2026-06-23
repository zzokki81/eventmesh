package jetstream

import "github.com/nats-io/nats.go"

// natsHeaderCarrier adapts nats.Header to the OTel TextMapCarrier interface so
// trace context can be injected into (publisher) and extracted from (subscriber)
// NATS message headers. It is the carrier used by the global propagator for the
// broker hop.
type natsHeaderCarrier nats.Header

// Get returns the value for key, or "" if absent.
func (c natsHeaderCarrier) Get(key string) string {
	return nats.Header(c).Get(key)
}

// Set stores value under key, replacing any existing value.
func (c natsHeaderCarrier) Set(key, value string) {
	nats.Header(c).Set(key, value)
}

// Keys lists every header name present in the carrier.
func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
