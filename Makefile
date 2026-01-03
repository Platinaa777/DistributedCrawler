include vendor.proto.mk
include .env

LOCAL_BIN := $(CURDIR)/bin
LOCAL_BIN_WIN := $(shell cygpath -w "$(LOCAL_BIN)")

GOOSE := $(LOCAL_BIN)/goose
LOCAL_MIGRATION_DIR=$(MIGRATION_DIR)
LOCAL_MIGRATION_DSN="host=localhost port=$(PG_PORT) dbname=$(PG_DATABASE_NAME) user=$(PG_USER) password=$(PG_PASSWORD) sslmode=disable"

APP_NAME := distributed-crawler
GO_FILES := $(shell find . -name '*.go' -type f)

# .PHONY объявляет "фиктивные" цели, которые не создают файлы с таким именем.
# Без этого, если в директории есть файл с именем "build" или "test",
# make подумает что цель уже выполнена и не запустит команду.
# Это защищает от конфликтов имен и улучшает производительность.
.PHONY: help build run test clean docker-up docker-down all info

.bin-deps: export GOBIN := $(LOCAL_BIN_WIN)
.bin-deps:
	$(info Installing dependencies....)
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0
	GOBIN=$(LOCAL_BIN) go install github.com/pressly/goose/v3/cmd/goose@v3.14.0

.buf-generate: .bin-deps
	$(info run buf generate...)
	PATH="$(LOCAL_BIN_WIN);$(PATH)" buf generate

.tidy:
	go mod tidy

generate: .tidy .buf-generate

build:
	@echo "building project ... "
	go build -o $(LOCAL_BIN)/$(APP_NAME) ./cmd/http_server/main.go
	@echo "Build completed! File: $(LOCAL_BIN)/$(APP_NAME)"

run:
	@echo "Run app..."
	go run ./cmd/http_server/main.go

test:
	@echo "Run test..."
	go test -v ./...
	@echo "Test passed!"

local-migration-status:
	$(GOOSE) -dir $(LOCAL_MIGRATION_DIR) postgres $(LOCAL_MIGRATION_DSN) status -v

local-migration-up:
	$(GOOSE) -dir $(LOCAL_MIGRATION_DIR) postgres $(LOCAL_MIGRATION_DSN) up -v

local-migration-down:
	$(GOOSE) -dir $(LOCAL_MIGRATION_DIR) postgres $(LOCAL_MIGRATION_DSN) down -v
