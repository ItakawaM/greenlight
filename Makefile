include .env

export

BINARY_PATH ?= ./bin/api
MAIN_PATH := ./cmd/api
MIGRATIONS_PATH := ./migrations

POSTGRES_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
REDIS_URL := redis://:${REDIS_PASSWORD}@${REDIS_HOST}:${REDIS_PORT}/0

RUN_FLAGS = -port=$(SERVER_PORT) -env=$(SERVER_ENVIRONMENT) \
	-postgres-max-open-conns=$(POSTGRES_MAX_OPEN_CONNS) -postgres-max-idle-time=$(POSTGRES_MAX_IDLE_TIME) \
	-limiter-rps=$(LIMITER_RPS) -limiter-burst=$(LIMITER_BURST) -limiter-enabled=$(LIMITER_ENABLED)

.DEFAULT_GOAL := help

.PHONY: \
	build \
	build-run \
	clean \
	compose-down \
	compose-up \
	compose-up-rebuild \
	help \
	migrate-down \
	migrate-up \
	run \
	test

run:
	go run $(MAIN_PATH) $(RUN_FLAGS)

build:
	go build -o $(BINARY_PATH) $(MAIN_PATH)

build-run: build
	$(BINARY_PATH) $(RUN_FLAGS)

compose-up:
	docker compose up -d

compose-up-rebuild:
	docker compose up -d --build

compose-down:
	docker compose down

migrate-up:
	migrate -path=$(MIGRATIONS_PATH) -database="$(POSTGRES_URL)" up

migrate-down:
	migrate -path=$(MIGRATIONS_PATH) -database="$(POSTGRES_URL)" down

test:
	go test ./... -v

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_PATH)
	@go clean
	@echo "Done!"

help:
	@echo "Commands:"
	@echo "  make run                 - Runs the API"
	@echo "  make build               - Builds the API into an executable"
	@echo "  make build-run           - Builds and runs the API executable"
	@echo "  make compose-up          - Starts Docker Compose services"
	@echo "  make compose-up-rebuild  - Rebuilds and starts Docker Compose services"
	@echo "  make compose-down        - Stops Docker Compose services"
	@echo "  make migrate-up          - Applies database migrations"
	@echo "  make migrate-down        - Rolls back database migrations"
	@echo "  make test                - Runs the test suite"
	@echo "  make clean               - Removes the compiled executable and cleans cache"