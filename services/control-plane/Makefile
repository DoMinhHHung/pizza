.PHONY: run tidy proto migrate-up migrate-down

run:
	go run ./cmd/server

tidy:
	go mod tidy

proto:
	@echo "proto generation will be added in next milestone"

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down
