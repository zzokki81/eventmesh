# EventMesh

> **What this is:** a demo/example system built to show how a set of
> distributed-systems and observability patterns fit together in practice —
> reliable messaging, retries and dead-lettering, deduplication, metrics, and
> tracing — using "orders" as a simple, easy-to-follow stand-in domain. There
> is no real order processing, payment, or inventory logic behind it; the
> "notification" is an email sent to [Mailpit](https://mailpit.axllent.org/),
> a local mail-catcher, not a real inbox.

A small distributed system in Go that turns "place an order" into a reliable,
observable event pipeline: an **order service** writes orders through a
transactional outbox, a **relay** ships them onto **NATS JetStream**, and a
**notifier service** consumes them exactly once and sends an email. Failures
that can't be recovered automatically land in a dead-letter stream and can be
replayed by hand.

Despite the simple domain, the plumbing around it is built to a production
standard rather than cut as a toy: proper config validation, structured
logging, Prometheus metrics, OpenTelemetry tracing end-to-end, graceful
shutdown, and both unit and Postgres-backed integration tests.

## Architecture

![EventMesh architecture](docs/architecture.svg)

Two independent services connected only through Postgres and NATS — neither
knows the other exists:

- **Order service** exposes `POST /orderso` and `GET /orders/{id}`. Creating an
  order writes the `orders` row and an `outbox_events` row **in the same
  Postgres transaction**, so the order and the event it produces can never
  disagree about whether the write happened. A background **relay** polls the
  outbox (`FOR UPDATE SKIP LOCKED`, so multiple relay instances don't race each
  other) and publishes pending rows to JetStream, retrying with backoff and
  marking a row `dead` once it exhausts its attempts.
- **NATS JetStream** persists the `orders.>` stream. Delivery is
  at-least-once: a durable pull consumer only advances once the handler acks.
- **Notifier service** consumes `orders.created`, deduplicates via a two-phase
  Redis claim (a short-lived "in progress" lock, then a long-lived "completed"
  mark — see [`notifier/dedup`](notifier/dedup)) so a redelivered message never
  results in a second email, and sends the notification over SMTP.
- **Dead-letter stream + `redrive`**: a message that fails permanently (bad
  payload) or exhausts its retry budget is moved to a `dlq.>` stream instead of
  being dropped or retried forever. The [`redrive`](cmd/redrive) CLI is a
  small operator tool: run it by hand once the underlying issue is fixed, and
  it republishes every dead-lettered event to its original subject.
- **Prometheus + Jaeger**: both services export Prometheus metrics and OTLP
  traces. A trace started on an incoming HTTP request is carried through the
  outbox row (`TraceContext`), across the relay's async publish, over the NATS
  message headers, and into the notifier's consumer span — so a single order
  can be followed end-to-end in Jaeger, not just within one process.

## Why a transactional outbox?

Writing to a database and publishing to a broker are two systems that can't
commit as one atomic operation. Publish-then-write risks a message for a
write that never lands; write-then-publish risks a committed write whose
event is silently lost if the process dies in between. The outbox pattern
avoids the dilemma: the event is written as a row in the *same* transaction as
the business data, and a separate relay is the only thing that ever talks to
the broker. Worst case, the relay publishes a duplicate on crash-recovery —
which is why the notifier's dedup store exists, turning an at-least-once
pipeline into effectively-once processing at the point where it matters (an
email actually being sent).

## Tech stack

| Concern             | Choice                                                |
|---------------------|--------------------------------------------------------|
| Language             | Go 1.26                                               |
| Messaging            | [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream) |
| Database             | PostgreSQL ([pgx](https://github.com/jackc/pgx))      |
| Cache / dedup        | Redis ([go-redis](https://github.com/redis/go-redis)) |
| Rate limiting        | Redis-backed token bucket ([`redis_rate`](https://github.com/go-redis/redis_rate)) |
| Email                | SMTP ([`go-mail`](https://github.com/wneessen/go-mail)), [Mailpit](https://mailpit.axllent.org/) in dev |
| Metrics              | Prometheus (via [OpenTelemetry metrics SDK](https://opentelemetry.io/)) |
| Tracing              | OpenTelemetry → OTLP → Jaeger                         |
| Migrations           | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Config               | Environment variables ([`caarlos0/env`](https://github.com/caarlos0/env)) + [validator](https://github.com/go-playground/validator) |
| Load testing         | [k6](https://k6.io/)                                  |
| Containers (dev/test)| Docker Compose, [testcontainers-go](https://golang.testcontainers.org/) |

## Project layout

```
cmd/            entry points: order, notifier, redrive
order/          order service: domain, service, HTTP transport, Postgres repo, relay
notifier/       notifier service: domain, service, dedup, email, broker transport
pkg/            shared infrastructure: broker abstraction + JetStream impl, event
                envelope, HTTP server/middleware, logger, observability, Postgres/
                Redis/NATS clients
api/openapi/    OpenAPI spec for the order service
deployments/    docker-compose stack for local dev (Postgres, NATS, Redis,
                Mailpit, Prometheus, Jaeger)
loadtest/       k6 script + a written-up profiling exercise (see below)
```

Each service follows the same shape — `domain` (types, invariants) →
`service` (use cases) → `transports` (HTTP/broker adapters) → `repository`
(storage) — so business logic stays independent of how it's triggered or
persisted.

## Getting started

**Prerequisites:** Go 1.26+, Docker, and (for migrations) [`golang-migrate`](https://github.com/golang-migrate/migrate).

```sh
# 1. Create order/.env and notifier/.env (gitignored; docker-compose reads
#    order/.env directly for the Postgres container's own credentials, and
#    both services need it for the rest of their config). .env.example lists
#    every variable with its default, but leaves the handful that are
#    required — HTTP_ADDR, DATABASE_URL, NATS_URL, REDIS_ADDR — blank on
#    purpose, since they differ per service and per environment.
cat > order/.env <<'EOF'
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=eventmesh_db
HTTP_ADDR=:8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/eventmesh_db?sslmode=disable
NATS_URL=nats://localhost:4222
REDIS_ADDR=localhost:6379
EOF

cat > notifier/.env <<'EOF'
HTTP_ADDR=:8081
NATS_URL=nats://localhost:4222
REDIS_ADDR=localhost:6379
EOF

# 2. Bring up Postgres, NATS, Redis, Mailpit, Prometheus, Jaeger
make up

# 3. Apply database migrations
make migrate-up

# 4. Run the services (separate terminals)
make run-order
make run-notifier
```

Create an order:

```sh
curl -X POST localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"9d3b7a2e-6c1f-4e2a-8f9b-1234567890ab","user_email":"you@example.com","amount":"49.99"}'
```

Within a moment, the relay publishes the event, the notifier picks it up, and
the email shows up in Mailpit's web UI at [localhost:8025](http://localhost:8025).

Useful local endpoints once `make up` is running:

| Service      | URL                                            |
|--------------|-------------------------------------------------|
| Order API    | http://localhost:8080                          |
| Notifier API | http://localhost:8081 (health/metrics only)     |
| Mailpit UI   | http://localhost:8025                           |
| Prometheus   | http://localhost:9090                           |
| Jaeger UI    | http://localhost:16686                          |
| NATS monitor | http://localhost:8222                           |

Both services expose `/healthz`, `/readyz`, `/info`, and `/metrics`. See
[`.env.example`](.env.example) for every configuration option, its default,
and validation rule.

## API

`POST /orders` and `GET /orders/{id}` are documented in
[`api/openapi/order.yaml`](api/openapi/order.yaml) — paste it into
[Swagger Editor](https://editor.swagger.io/) or your editor's OpenAPI preview
for the full request/response schemas.

## Testing

```sh
make test-unit         # go test -race, no external dependencies
make test-integration  # Postgres repository tests via testcontainers (needs Docker)
make lint               # golangci-lint
```

CI ([`.github/workflows/ci.yaml`](.github/workflows/ci.yaml)) runs all three
on every push and pull request against `main`.

## Performance

Load tested with [k6](https://k6.io/) and profiled with Go's `pprof`.
Profiling the order write path under load surfaced a throughput ceiling at
the Postgres connection pool: raising it (25 → 100) lifted throughput ~60% at
a lower p95, with no code change.

See [`loadtest/RESULTS.md`](loadtest/RESULTS.md) for the full before/after
walkthrough and [`loadtest/README.md`](loadtest/README.md) to reproduce it.

## License

[MIT](LICENSE)
