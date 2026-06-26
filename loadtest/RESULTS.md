# Load test results — order service `POST /orders`

A walk through finding and confirming a throughput bottleneck in the order
write path, using k6 for load and Go's pprof for profiling.

> These numbers are from a single developer machine (24 cores; Postgres,
> NATS, etc. in local Docker containers). Absolute throughput is
> environment-specific — the **relative** before/after is the takeaway.

## Setup

- Endpoint: `POST /orders`, which writes the order row **and** an outbox row in
  one Postgres transaction; the relay drains the outbox in the background.
- Load: `loadtest/orders.js` via k6, ramping to a target number of virtual
  users (VUs) and holding.
- Profiling: CPU profile captured from the dev pprof server (`:6060`) during the
  hold phase (`go tool pprof http://localhost:6060/debug/pprof/profile`).
- The only variable changed between runs is `DB_MAX_CONNS` (the Postgres
  connection pool size). No application code was changed.

## Runs

| Run | VUs | `DB_MAX_CONNS` | Throughput | p95 | avg | Errors | Cores used (of 24) |
|-----|-----|----------------|------------|-----|-----|--------|--------------------|
| 1 | 50 | 25 | 7,300/s | 7.45ms | 5.33ms | 0% | ~2.5 |
| 2 | 300 | 25 | 7,925/s | 44ms | 33ms | 0% | ~2.5 |
| 3 | 300 | 100 | **12,720/s** | **31ms** | 20ms | 0% | ~3.9 |

## Diagnosis

Going from run 1 to run 2 — **6× the VUs, but throughput barely moved**
(7,300 → 7,925/s) while p95 grew ~6× (7.45 → 44ms). That is the textbook
signature of a **serialization point**: extra concurrent users don't produce
more work per second, they just queue up waiting (Little's law: latency =
concurrency ÷ throughput).

Where the ceiling was **not**:

- **CPU** — the service used ~2.5 of 24 cores; nowhere near saturated.
- **Postgres** — not working hard.
- **A code hotspot** — the CPU profile showed no dominant application frame,
  because requests that are **blocked waiting for a pool connection are not
  on-CPU**, so they never appear in a CPU profile. The absence of a hotspot,
  combined with idle CPU and a throughput plateau, pointed away from compute and
  toward a concurrency limit.

Prime suspect: the connection pool (`DB_MAX_CONNS=25`). Each request needs a
connection for its transaction; with 25 connections and a transaction taking
~3ms, the ceiling is ≈ 25 ÷ 0.003 ≈ **8,300 tx/s** — which matches the observed
~7,900/s plateau.

## The fix (A/B)

Raise `DB_MAX_CONNS` from 25 to 100 and re-run the identical 300-VU test (run 3):

- Throughput **+60%** (7,925 → 12,720/s)
- p95 **down** (44 → 31ms)
- Still 0% errors; CPU rose to ~3.9 cores — still far from saturated

The pool was the bottleneck. Relieving it immediately yielded more throughput at
lower latency, with no code change.

## Caveats and next steps

- A bigger pool is not free: every pooled connection is a Postgres backend
  (memory, scheduling). The right value is a balance, not "as high as possible";
  past a point, more connections add contention inside Postgres rather than
  throughput. A moderate, deliberate default (e.g. 50) is more honest than
  blindly setting 100.
- At pool=100 the service still used only ~3.9 of 24 cores, so the new ceiling is
  higher again — pushing further (more VUs, larger pool) would find it, but the
  bottleneck-and-fix story is already complete.
- Real production tuning would also weigh a connection pooler (e.g. PgBouncer)
  and per-request timeouts on connection acquisition.

## How to reproduce

See `loadtest/README.md`. In short: `make up`, `make run-order`, then `make
loadtest` for the 50-VU run (run 1) or, for the 300-VU runs (2 and 3):

```sh
make loadtest K6_ARGS="--stage 30s:300 --stage 1m:300 --stage 10s:0"
```

Set the pool size with `DB_MAX_CONNS` on the order service between runs, and
capture a profile from `:6060` during the hold phase.
