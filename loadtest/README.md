# Load testing the order service

This directory holds a [k6](https://k6.io/) load test for `POST /orders` and the
recipe for profiling the service while it is under load.

Each request creates an order, which writes the order row **and** an outbox row
in one Postgres transaction; the relay drains the outbox in the background. So
this exercises the hot write path: HTTP decode/validate → transaction → publish.

## Prerequisites

1. Infrastructure up: `make up`
2. Order service running in the **development** profile (so pprof is exposed on
   `:6060`): `make run-order`
3. Docker available (the load test runs k6 in a container; no local install).

## Run the load test

```sh
make loadtest
```

This ramps to 50 virtual users, holds for a minute, then ramps down. With
`DB_MAX_CONNS=25` the hold phase intentionally pushes past the pool size, so
connection contention surfaces. k6 prints a summary with request rate, latency
percentiles (look at `p(95)`), and the pass/fail of the thresholds.

To override the script's stages — for example to reproduce the 300-VU runs in
[RESULTS.md](RESULTS.md) — pass k6 flags through `K6_ARGS`:

```sh
make loadtest K6_ARGS="--stage 30s:300 --stage 1m:300 --stage 10s:0"
```

To point it at a different address (if k6 is installed locally rather than via
Docker):

```sh
BASE_URL=http://localhost:8080 k6 run loadtest/orders.js
```

## Capture a profile while it runs

The dev pprof server is on `:6060`. **Start the load test first**, then while it
is in the hold phase capture a profile.

CPU profile (samples for 30s, then opens an interactive web UI):

```sh
go tool pprof -http=:8081 "http://localhost:6060/debug/pprof/profile?seconds=30"
```

Heap (in-use memory) profile:

```sh
go tool pprof -http=:8081 "http://localhost:6060/debug/pprof/heap"
```

Save a profile to a file to keep alongside the findings:

```sh
curl -o cpu.prof "http://localhost:6060/debug/pprof/profile?seconds=30"
go tool pprof -http=:8081 cpu.prof
```

In the web UI, the **Flame Graph** and **Top** views show where wall/CPU time
goes. Common things to look for here: time parked waiting on a database
connection (pool too small), JSON encoding/decoding, and allocation churn.

## Workflow

1. Run the load test, read the k6 summary (latency `p(95)`, error rate).
2. Capture a CPU profile during the hold phase.
3. Identify the dominant cost in the flame graph.
4. Change one thing (e.g. raise `DB_MAX_CONNS`), re-run, compare the numbers.
5. Record before/after in the project notes — that is the bottleneck story.
