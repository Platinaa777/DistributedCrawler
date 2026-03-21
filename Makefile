include vendor.proto.mk
include .env

LOCAL_BIN := $(CURDIR)/bin
LOCAL_BIN_WIN := $(shell cygpath -w "$(LOCAL_BIN)")

GOOSE := $(LOCAL_BIN)/goose
LOCAL_MIGRATION_DIR=$(MIGRATION_DIR)
LOCAL_MIGRATION_DSN="host=localhost port=$(PG_PORT) dbname=$(PG_DATABASE_NAME) user=$(PG_USER) password=$(PG_PASSWORD) sslmode=disable"
MIGRATE := go run ./cmd/migrate

APP_NAME := distributed-crawler
GO_FILES := $(shell find . -name '*.go' -type f)

# .PHONY объявляет "фиктивные" цели, которые не создают файлы с таким именем.
# Без этого, если в директории есть файл с именем "build" или "test",
# make подумает что цель уже выполнена и не запустит команду.
# Это защищает от конфликтов имен и улучшает производительность.
.PHONY: help build run test clean docker-up docker-down docker-logs all info docker-build

.bin-deps: export GOBIN := $(LOCAL_BIN_WIN)
.bin-deps:
	$(info Installing dependencies....)
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0
	GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.27.4
	GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.27.4
	GOBIN=$(LOCAL_BIN) go install github.com/envoyproxy/protoc-gen-validate@v1.3.0
	GOBIN=$(LOCAL_BIN) go install github.com/pressly/goose/v3/cmd/goose@v3.14.0
	GOBIN=$(LOCAL_BIN) go install github.com/rakyll/statik@v0.1.7

.buf-generate: .bin-deps
	$(info run buf generate...)
	PATH="$(LOCAL_BIN_WIN);$(PATH)" buf generate

.statik-generate: .bin-deps
	$(info Embedding swagger files with statik...)
	$(LOCAL_BIN)/statik -src=api/v1/swagger/ -include='*.css,*.html,*.js,*.json,*.png'

.tidy:
	go mod tidy

generate: .tidy .buf-generate .statik-generate

build:
	@echo "building project ... "
	go build -o $(LOCAL_BIN)/$(APP_NAME) ./cmd/grpc_server/main.go
	@echo "Build completed! File: $(LOCAL_BIN)/$(APP_NAME)"

run-grpc-server:
	@echo "Run grpc app..."
	go run ./cmd/grpc_server/main.go --config-path=.env

run-fetcher:
	@echo "Run fetcher..."
	go run ./cmd/fetch_worker/main.go --worker-config-path=.worker.env

run-parser:
	@echo "Run parser..."
	go run ./cmd/parser_worker/main.go --worker-config-path=.worker.env

run-export:
	@echo "Run export worker..."
	go run ./cmd/export_worker/main.go --worker-config-path=.worker.env

test:
	go clean -testcache
	go test ./... -covermode count -coverpkg=distributed-crawler/... -count 5

test-coverage:
	go clean -testcache
	go test ./... -coverprofile=coverage.tmp.out -covermode count -coverpkg=distributed-crawler/... -count 5
	grep -v 'mocks\|config' coverage.tmp.out  > coverage.out
	rm coverage.tmp.out
	go tool cover -html=coverage.out;
	go tool cover -func=./coverage.out | grep "total";
	grep -sqFx "/coverage.out" .gitignore || echo "/coverage.out" >> .gitignore

local-migration-status:
	$(GOOSE) -dir $(LOCAL_MIGRATION_DIR) postgres $(LOCAL_MIGRATION_DSN) status -v

local-migration-up:
	$(GOOSE) -dir $(LOCAL_MIGRATION_DIR) postgres $(LOCAL_MIGRATION_DSN) up -v

local-migration-down:
	$(GOOSE) -dir $(LOCAL_MIGRATION_DIR) postgres $(LOCAL_MIGRATION_DSN) down -v

local-migration-create:
	$(GOOSE) -dir $(LOCAL_MIGRATION_DIR) create $(NAME) sql

migration-status:
	$(MIGRATE) --dsn $(LOCAL_MIGRATION_DSN) status

migration-up:
	$(MIGRATE) --dsn $(LOCAL_MIGRATION_DSN) up

migration-down:
	$(MIGRATE) --dsn $(LOCAL_MIGRATION_DSN) down

migration-create:
	$(MIGRATE) --dsn $(LOCAL_MIGRATION_DSN) create $(NAME)

docker-build:
	@echo "Building docker image for $(APP)..."
	@if [ -z "$(APP)" ]; then echo "APP is required (e.g. APP=grpc_server)"; exit 1; fi
	docker build -f docker/$(APP)/Dockerfile -t distributed-crawler-$(APP) .

docker-up:
	docker compose -f docker/docker-compose.yaml up -d

docker-down:
	docker compose -f docker/docker-compose.yaml down

docker-logs:
	docker compose -f docker/docker-compose.yaml logs -f
