.PHONY: build format lint run test test-race help

SOURCE_CODE ?= ./cmd/... ./internal/...
TEST_CODE ?= ./test/...
REV := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_OUTPUT_DIR := ./dist

help: ## Show available make targets
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "%-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build guesschema binary to ./dist/guesschema
build:
	mkdir -p $(BUILD_OUTPUT_DIR)
	go build -trimpath -ldflags "-s -w -X github.com/Olian04/guesschema/cmd/guesschema/version.Revision=$(REV) -X github.com/Olian04/guesschema/cmd/guesschema/version.BuildTime=$(BUILD_TIME)" -o $(BUILD_OUTPUT_DIR)/guesschema ./cmd/guesschema

run: ## Run guesschema from source (stdin JSONL → stdout schema)
	go run ./cmd/guesschema

format: ## Run go fmt and gofmt
	go fmt ./...
	gofmt -w .

lint: ## Run go vet, module verify, vuln scan, golangci
	go vet ./...
	go mod verify
	go tool govulncheck $(SOURCE_CODE)
	go tool golangci-lint run $(SOURCE_CODE)

test: ## Run unit tests
	go test -shuffle=on -timeout 120s $(TEST_CODE)

test-race: ## Run unit tests with race detector
	go test -race -shuffle=on -timeout 180s $(TEST_CODE)
