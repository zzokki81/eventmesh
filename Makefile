ifneq (,$(wildcard .env))
	include .env
	export
endif

COMPOSE_FILE := deployments/docker/docker-compose.yml

VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

ORDER_LDFLAGS := \
	-X github.com/zzokki81/eventmesh/order/app.Version=$(VERSION) \
	-X github.com/zzokki81/eventmesh/order/app.CommitHash=$(COMMIT_HASH) \
	-X github.com/zzokki81/eventmesh/order/app.BuildDate=$(BUILD_DATE)

NOTIFIER_LDFLAGS := \
	-X github.com/zzokki81/eventmesh/notifier/app.Version=$(VERSION) \
	-X github.com/zzokki81/eventmesh/notifier/app.CommitHash=$(COMMIT_HASH) \
	-X github.com/zzokki81/eventmesh/notifier/app.BuildDate=$(BUILD_DATE)

.PHONY: help \
	run-order run-notifier \
	build build-notifier \
	test test-unit test-integration lint loadtest up down logs clean \
	migrate-up migrate-down migrate-status migrate-new

help:
	@echo "Available targets:"
	@echo "  run-order        - Run order service locally (requires: make up)"
	@echo "  run-notifier     - Run notifier service locally (requires: make up)"
	@echo "  build            - Build order binary into ./bin/"
	@echo "  build-notifier   - Build notifier binary into ./bin/"
	@echo "  test             - Run unit + integration tests (integration needs Docker)"
	@echo "  test-unit        - Run unit tests with race detector (no Docker)"
	@echo "  test-integration - Run integration tests against a throwaway Postgres (needs Docker)"
	@echo "  lint             - Run golangci-lint"
	@echo "  loadtest         - Run the k6 load test against a running order service (needs Docker)"
	@echo "  up               - Start docker compose services"
	@echo "  down             - Stop docker compose services"
	@echo "  logs             - Tail docker compose logs"
	@echo "  clean            - Remove build artifacts"
	@echo "  migrate-up       - Apply all pending migrations"
	@echo "  migrate-down     - Roll back the last migration"
	@echo "  migrate-status   - Show current migration version"
	@echo "  migrate-new      - Create a new migration (usage: make migrate-new name=create_outbox)"

run-order:
	go run ./cmd/order

run-notifier:
	go run ./cmd/notifier

build:
	go build -ldflags "$(ORDER_LDFLAGS)" -o bin/order ./cmd/order

build-notifier:
	go build -ldflags "$(NOTIFIER_LDFLAGS)" -o bin/notifier ./cmd/notifier

test: test-unit test-integration

test-unit:
	go test -race -v ./...

# Integration tests are guarded by the `integration` build tag so the unit run
# stays fast and Docker-free. They spin up a throwaway Postgres via
# testcontainers, so a running Docker daemon is required. -count=1 disables the
# test cache, since results depend on the live container, not just the sources.
test-integration:
	go test -race -tags integration -count=1 ./order/repository/postgres/...

lint:
	golangci-lint run ./...

# Runs k6 via Docker (no local install needed). It reaches the order service
# through host.docker.internal, which resolves to the host on macOS/Windows
# (Docker Desktop) and on Linux via --add-host=...:host-gateway, so the target
# is portable across all three. Requires the stack (make up) and the order
# service (make run-order) to be running.
#
# Pass extra k6 flags via K6_ARGS, e.g. to override the script's stages and run
# with 300 virtual users:
#   make loadtest K6_ARGS="--stage 30s:300 --stage 1m:300 --stage 10s:0"
loadtest:
	docker run --rm \
		--add-host=host.docker.internal:host-gateway \
		-e BASE_URL=http://host.docker.internal:8080 \
		-v "$(CURDIR)/loadtest:/scripts" grafana/k6 run $(K6_ARGS) /scripts/orders.js

clean:
	rm -rf bin/

up:
	docker compose -f $(COMPOSE_FILE) up -d

down:
	docker compose -f $(COMPOSE_FILE) down

logs:
	docker compose -f $(COMPOSE_FILE) logs -f

migrate-up:
	@set -a && . order/.env && set +a && migrate -path order/migrations -database "$$DATABASE_URL" up

migrate-down:
	@set -a && . order/.env && set +a && migrate -path order/migrations -database "$$DATABASE_URL" down 1

migrate-status:
	@set -a && . order/.env && set +a && migrate -path order/migrations -database "$$DATABASE_URL" version

migrate-new:
	@if [ -z "$(name)" ]; then echo "Error: name required. Usage: make migrate-new name=create_outbox"; exit 1; fi
	migrate create -ext sql -dir order/migrations -seq $(name)
