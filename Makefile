include ./scripts/env.sh

APP_NAME ?= trading-journal
BIN_DIR ?= bin
BINARY_NAME ?= $(BIN_DIR)/$(APP_NAME)
KUCOIN_BINARY ?= $(BIN_DIR)/kucoin-example
MAIN_PATH ?= ./main.go
SWAGGER_FILE ?= ./docs/swagger.yaml
SWAGGER_PORT ?= 8080
SWAGGER_IMAGE ?= swaggerapi/swagger-ui

.PHONY: help run run_server build build_server clean test test-modules test-handlers test-repositories swagger kucoin mexc build_kucoin

help: ## Show available commands
	@echo "Usage: make <target>" && echo && echo "Available targets:" && \
	awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

#run: kucoin ## Backwards-compatible alias for the KuCoin example

kucoin: ## Run the KuCoin balances example
	go run ./cmd/kucoin/main.go

mexc: ## Run the MEXC balances checker
	go run ./cmd/mexc/main.go

phemex: ## Run the phemex balances checker
	go run ./cmd/phemex/main.go

run_server: ## Run the HTTP server (main.go)
	APP_NAME=$(APP_NAME) PORT=$(PORT) go run $(MAIN_PATH)

#build: build_kucoin ## Build the KuCoin example binary

build_kucoin: ## Build the KuCoin example binary
	mkdir -p $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(KUCOIN_BINARY) ./cmd/kucoin/main.go

build_server: ## Build the HTTP server binary
	mkdir -p $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY_NAME) $(MAIN_PATH)

clean: ## Remove built binaries
	rm -rf $(BIN_DIR)

test: ## Run all tests
	go test ./... -v

test-modules: ## Run tests for application modules (src/*)
	go test ./src/... -v

test-handlers: ## Run tests for HTTP handlers only
	go test ./src/handler/... -v

test-repositories: ## Run tests for repositories only
	go test ./src/repository/... -v

swagger: ## Serve Swagger UI using docs/swagger.yaml
	docker run --rm -p $(SWAGGER_PORT):8080 -e SWAGGER_JSON=/swagger.yaml -v "$(PWD)/$(SWAGGER_FILE):/swagger.yaml" $(SWAGGER_IMAGE)
