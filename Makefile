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
	test lint up down logs clean \
	migrate-up migrate-down migrate-status migrate-new

help:
	@echo "Available targets:"
	@echo "  run-order        - Run order service locally (requires: make up)"
	@echo "  run-notifier     - Run notifier service locally (requires: make up)"
	@echo "  build            - Build order binary into ./bin/"
	@echo "  build-notifier   - Build notifier binary into ./bin/"
	@echo "  test             - Run tests with race detector"
	@echo "  lint             - Run golangci-lint"
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

test:
	go test -race -v ./...

lint:
	golangci-lint run ./...

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
