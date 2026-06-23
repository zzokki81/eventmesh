package observability

// Config holds the observability settings parsed from the environment.
type Config struct {
	// OTLPEndpoint is the host:port of the OTLP gRPC receiver (e.g. Jaeger)
	// that traces are exported to. Defaults to the local all-in-one Jaeger.
	OTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"127.0.0.1:4317"`
}
