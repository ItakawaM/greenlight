include .env

export

BINARY ?= api.exe
MAIN_PATH := ./cmd/api
MIGRATIONS_PATH := ./migrations
DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable

.DEFAULT_GOAL := help

.PHONY: run build build-run compose-up compose-up-rebuild compose-down migrate-up migrate-down clean test help

run:
	go run $(MAIN_PATH) -port=$(PORT) -env=$(ENVIRONMENT)

build:
	go build -o ./bin/$(BINARY) $(MAIN_PATH)

build-run: build
	./bin/$(BINARY) -port=$(PORT) -env=$(ENVIRONMENT)

compose-up:
	docker compose up -d

compose-up-rebuild:
	docker compose up -d --build

compose-down:
	docker compose down

migrate-up:
	migrate -path=$(MIGRATIONS_PATH) -database="$(DATABASE_URL)" up

migrate-down:
	migrate -path=$(MIGRATIONS_PATH) -database="$(DATABASE_URL)" down

test:
	go test ./... -v

clean:
	@echo "Cleaning up..."
	@rm -f ./bin/$(BINARY)
	@go clean
	@echo "Done!"

help:
	@echo "Commands:"
	@echo "  make run                - Runs the API"
	@echo "  make build              - Builds the API into an executable"
	@echo "  make build-run          - Runs the API executable"
	@echo "  make compose-up         - Starts Docker Compose services"
	@echo "  make compose-up-rebuild - Rebuilds and starts Docker Compose services"
	@echo "  make compose-down       - Stops Docker Compose services"
	@echo "  make migrate-up         - Applies database migrations"
	@echo "  make migrate-down       - Rolls back database migrations"
	@echo "  make test               - Runs the test suite"
	@echo "  make clean              - Removes the compiled executable and cleans cache"