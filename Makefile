# Load .env if present and export all variables to recipes.
ifneq (,$(wildcard .env))
	include .env
	export
endif

COMPOSE_FILE := deployments/docker/docker-compose.yml

.PHONY: help run build test lint up down logs clean \
        migrate-up migrate-down migrate-status migrate-new

# Default target — list available commands.
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

# --- Application ---

run:
	go run ./cmd/eventmesh

build:
	go build -o bin/eventmesh ./cmd/eventmesh

test:
	go test -race -v ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

# --- Infrastructure ---

up:
	docker compose -f $(COMPOSE_FILE) up -d

down:
	docker compose -f $(COMPOSE_FILE) down

logs:
	docker compose -f $(COMPOSE_FILE) logs -f

# --- Migrations ---

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-status:
	migrate -path migrations -database "$(DATABASE_URL)" version

migrate-new:
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make migrate-new name=create_outbox"; \
		exit 1; \
	fi
	migrate create -ext sql -dir migrations -seq $(name)
