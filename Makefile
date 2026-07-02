include .env

export

BINARY ?= api.exe
MAIN_PATH := ./cmd/api
MIGRATIONS_PATH := ./migrations

.DEFAULT_GOAL := help

.PHONY: run build build-run migrate-up migrate-down clean test help

run:
	go run $(MAIN_PATH) -port=$(PORT) -env=$(ENVIRONMENT)

build:
	go build -o ./bin/$(BINARY) $(MAIN_PATH)

build-run: build
	./bin/$(BINARY) -port=$(PORT) -env=$(ENVIRONMENT)

migrate-up:
	migrate -path=$(MIGRATIONS_PATH) -database=$(DATABASE_URL) up

migrate-down:
	migrate -path=$(MIGRATIONS_PATH) -database=$(DATABASE_URL) down

test:
	go test ./... -v

clean:
	@echo "Cleaning up..."
	@rm -f ./bin/$(BINARY)
	@go clean
	@echo "Done!"

help:
	@echo "Commands:"
	@echo "  make run         - Runs the API"
	@echo "  make build       - Builds the API into an executable"
	@echo "  make build-run   - Runs the API executable"
	@echo "  make clean       - Removes the compiled executable and cleans cache"