PROTO_DIR=api/proto
GEN_DIR=api/gen

.PHONY: run tidy proto migrate-up migrate-down

run:
	go run ./cmd/server

tidy:
	go mod tidy

proto:
	mkdir -p $(GEN_DIR)
	protoc \
	  -I $(PROTO_DIR) \
	  --go_out=$(GEN_DIR) --go_opt=paths=source_relative \
	  --go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
	  $(PROTO_DIR)/control/v1/*.proto

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

.PHONY: test test-unit test-integration

test-unit:
	go test ./internal/auth/ ./internal/org/ -count=1 -short

test-integration:
	go test ./internal/org/ ./internal/project/ ./internal/testutil/ -count=1 -timeout 120s

test:
	go test ./... -count=1 -timeout 120s
