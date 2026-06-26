# eventmesh
Distributed event processing platform built in Go with NATS JetStream.

## Performance

Load tested with [k6](https://k6.io/) and profiled with Go's pprof. Profiling
the order write path under load surfaced a throughput ceiling at the Postgres
connection pool: raising it (25 → 100) lifted throughput ~60% at a lower p95,
with no code change.

See [loadtest/RESULTS.md](loadtest/RESULTS.md) for the full before/after and
[loadtest/README.md](loadtest/README.md) to reproduce.
