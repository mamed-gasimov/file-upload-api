-include .env

POSTGRES_DSN ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSLMODE)
MIGRATIONS_DIR = migrations
GOOSE = goose

.PHONY: migrate-up migrate-down migrate-status migrate-create

migrate-up:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" up

migrate-down:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" down

migrate-status:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" status

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=<migration_name>"; exit 1; fi
	$(GOOSE) -dir $(MIGRATIONS_DIR) create $(name) sql
