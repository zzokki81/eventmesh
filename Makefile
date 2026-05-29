ifneq (,$(wildcard .env))
	include .env
	export
endif

COMPOSE_FILE := deployments/docker/docker-compose.yml

VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X github.com/zzokki81/eventmesh/order/app.Version=$(VERSION) \
           -X github.com/zzokki81/eventmesh/order/app.CommitHash=$(COMMIT_HASH) \
           -X github.com/zzokki81/eventmesh/order/app.BuildDate=$(BUILD_DATE)

.PHONY: help run build test lint up down logs clean migrate-up migrate-down migrate-status migrate-new

help:
	@echo "Available targets:"
	@echo "  run             - Run the application locally"
	@echo "  build           - Build the binary into ./bin/"
	@echo "  test            - Run tests with race detector"
	@echo "  lint            - Run golangci-lint"
	@echo "  up              - Start docker compose services"
	@echo "  down            - Stop docker compose services"
	@echo "  logs            - Tail docker compose logs"
	@echo "  clean           - Remove build artifacts"
	@echo "  migrate-up      - Apply all pending migrations"
	@echo "  migrate-down    - Roll back the last migration"
	@echo "  migrate-status  - Show current migration version"
	@echo "  migrate-new     - Create a new migration (usage: make migrate-new name=create_outbox)"

run:
	go run ./cmd/order

build:
	go build -ldflags "$(LDFLAGS)" -o bin/order ./cmd/order

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
	migrate -path order/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path order/migrations -database "$(DATABASE_URL)" down 1

migrate-status:
	migrate -path order/migrations -database "$(DATABASE_URL)" version

migrate-new:
	@if [ -z "$(name)" ]; then echo "Error: name required. Usage: make migrate-new name=create_outbox"; exit 1; fi
	migrate create -ext sql -dir order/migrations -seq $(name)
