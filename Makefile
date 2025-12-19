include ./scripts/env.sh

# -----------------------------
# Go parameters
# -----------------------------
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Main backend binary
BINARY_NAME=strategyexecutor
MAIN_PATH=./cmd/strategyexecutor

# Phemex CLI
PHEMEX_CLI_BINARY=phemex-cli
PHEMEX_CLI_PATH=./cmd/Phemex

# Build flags
LDFLAGS=-ldflags "-s -w"

# Test flags
TEST_FLAGS=-v -race

# Linter configuration
GOLINT=golangci-lint
LINT_FLAGS=run --timeout=5m

# Colors for help and messages
YELLOW := \033[1;33m
GREEN  := \033[1;32m
RED    := \033[1;31m
BLUE   := \033[1;34m
NC     := \033[0m # No Color

.PHONY: all build clean test coverage lint deps tidy vendor run run-phemex cli help

# -----------------------------
# Project Management
# -----------------------------
all: clean lint test build ## Run clean, lint, test, and build

help: ## Show this help message
	@echo '${YELLOW}Usage:${NC}'
	@echo '  make ${GREEN}<target>${NC}'
	@echo ''
	@echo '${YELLOW}Targets:${NC}'
	@awk '/^[a-zA-Z\-\_\.0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  ${GREEN}%-20s${NC} %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)
	@echo ''
	@echo '${YELLOW}Examples:${NC}'
	@echo '  ${BLUE}make build${NC}          - Build the backend'
	@echo '  ${BLUE}make run${NC}            - Build and run the backend'
	@echo '  ${BLUE}make run-phemex${NC}     - Run the Phemex CLI'
	@echo '  ${BLUE}make test${NC}           - Run all tests'
	@echo ''
	@echo '${YELLOW}Development Workflow:${NC}'
	@echo '  1. ${BLUE}make deps${NC}'
	@echo '  2. ${BLUE}make lint${NC}'
	@echo '  3. ${BLUE}make test${NC}'
	@echo '  4. ${BLUE}make build${NC}'
	@echo ''

# -----------------------------
# Build Operations
# -----------------------------
build: ## Build the backend binary
	@echo "${BLUE}Building ${BINARY_NAME}...${NC}"
	@$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "${GREEN}Build successful!${NC}"

build-phemex: ## Build the Phemex CLI
	@echo "${BLUE}Building ${PHEMEX_CLI_BINARY}...${NC}"
	@$(GOBUILD) $(LDFLAGS) -o $(PHEMEX_CLI_BINARY) $(PHEMEX_CLI_PATH)
	@echo "${GREEN}Phemex CLI build successful!${NC}"

clean: ## Remove build artifacts and temporary files
	@echo "${BLUE}Cleaning up...${NC}"
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -f $(PHEMEX_CLI_BINARY)
	@rm -f coverage.out
	@echo "${GREEN}Cleanup complete!${NC}"

# -----------------------------
# Testing and Quality
# -----------------------------
install_tooling: ## Install linters
	go install mvdan.cc/gofumpt@v0.1.1
	go install golang.org/x/tools/cmd/goimports@v0.1.1
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
	go install gotest.tools/gotestsum@v1.6.4
	go install github.com/jstemmer/go-junit-report@v0.9.1
	go install github.com/axw/gocov/gocov@v1.0.0
	go install github.com/AlekSi/gocov-xml@v0.0.0-20190121064608-3a14fb1c4737
	go install golang.org/x/vuln/cmd/govulncheck@latest

test: ## Run tests with race detection
	which gotestsum || ( \
		make install_tooling \
	)
	ENVIRONMENT=prod gotestsum --format testname --junitfile ./test.xml -- -timeout=7m -coverprofile=coverage.out -count=1 -test.short ./...
	tail -n +2 coverage.out >> ./sonar/coverage-all.out
	go tool cover -html=./coverage.out -o coverage.html
	gocov convert ./coverage.out | gocov-xml > ./coverage.xml
	mv coverage.out  ./sonar/${TEST_FOLDER}_coverage.out
	mv coverage.xml  ./sonar/${TEST_FOLDER}_coverage.xml
	mv coverage.html  ./sonar/${TEST_FOLDER}_coverage.html
	mv test.xml  ./sonar/${TEST_FOLDER}_test.xml

coverage: ## Generate test coverage report
	@echo "${BLUE}Generating coverage report...${NC}"
	@$(GOTEST) -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out
	@echo "${GREEN}Coverage report generated!${NC}"

lint: ## Run code linter
	@echo "${BLUE}Running linter...${NC}"
	@$(GOLINT) $(LINT_FLAGS)
	@echo "${GREEN}Lint check completed!${NC}"

# -----------------------------
# Dependency Management
# -----------------------------
deps: ## Download project dependencies
	@echo "${BLUE}Downloading dependencies...${NC}"
	@$(GOGET) -v ./...
	@echo "${GREEN}Dependencies downloaded!${NC}"

tidy: ## Tidy up module dependencies
	@echo "${BLUE}Tidying up dependencies...${NC}"
	@$(GOMOD) tidy
	@echo "${GREEN}Dependencies tidied!${NC}"

vendor: ## Vendor dependencies
	@echo "${BLUE}Vendoring dependencies...${NC}"
	@$(GOMOD) vendor
	@echo "${GREEN}Dependencies vendored!${NC}"

# -----------------------------
# Run targets
# -----------------------------
run: build ## Build and run the backend
	@echo "${BLUE}Running ${BINARY_NAME}...${NC}"
	@./$(BINARY_NAME)

run-phemex: ## Build and run the Phemex CLI
	@$(MAKE) build-phemex
	@echo "${BLUE}Running ${PHEMEX_CLI_BINARY}...${NC}"
	@./$(PHEMEX_CLI_BINARY)


start: ## Start.
	#./scripts/clean_db.sh
	@echo "Sourcing env.sh..."
	$(shell . ./scripts/env.sh; go run main.go)

start_dev: ## Start.
	@echo "Sourcing env.sh..."
	$(shell . ./scripts/env_dev.sh; go run main.go)



cmd_tv_news: ## Start.
	@echo "Sourcing env.sh..."
	$(shell . ./scripts/env.sh; go run cmd/main.go tvnews)


cmd_ohlcv_crypto_btc_1h:
	$(shell . ./scripts/scheduler/cmd_ohlcv_crypto_btc_1h.sh; go run cmd/main.go ohlcv_crypto)

cmd_ohlcv_crypto_btc_1m:
	$(shell . ./scripts/scheduler/cmd_ohlcv_crypto_btc_1m.sh; go run cmd/main.go ohlcv_crypto)


docker-build:
	docker build --build-arg -t strategyexecutor -f Dockerfile .

docker: docker-build
	docker run --env-file ./scripts/env_docker.sh -it -p 3010:3010 strategyexecutor

docker_cmd:
	docker build -t strategyexecutor-cmd -f Dockerfile.cmd .

docker_tvnews: docker_cmd
	docker run --env-file ./scripts/env_docker.sh -it -p 3003:3003 strategyexecutor-cmd -help

secret:
	kubectl apply -f helm/secret.yaml


# Default target
.DEFAULT_GOAL := help


send_long_manual:
	curl -X POST https://api.biidin.com/api/tv/signal \
	  -H "Content-Type: application/json" \
	  -d '{ "id": "Long", "symbol": "BTCUSD", "action": "buy", "qty": "1.402926", "price": "87068.01", "type": "Market", "marketPosition": "long", "prevMarketPosition": "short", "marketPositionSize": "0.700223", "prevMarketPositionSize": "0.702703", "signalToken": "manual_trade", "timestamp": "2025-12-17T13:30:01Z", "comment": "Long", "message": "", "exchange_name": "phemex"}'

	curl -X POST https://api.biidin.com/api/tv/signal \
	  -H "Content-Type: application/json" \
	  -d '{ "id": "Long", "symbol": "BTCUSD", "action": "buy", "qty": "1.402926", "price": "87068.01", "type": "Market", "marketPosition": "long", "prevMarketPosition": "short", "marketPositionSize": "0.700223", "prevMarketPositionSize": "0.702703", "signalToken": "manual_trade", "timestamp": "2025-12-17T13:30:01Z", "comment": "Long", "message": "", "exchange_name": "kraken"}'

	curl -X POST https://api.biidin.com/api/tv/signal \
	  -H "Content-Type: application/json" \
	  -d '{ "id": "Long", "symbol": "BTCUSD", "action": "buy", "qty": "1.402926", "price": "87068.01", "type": "Market", "marketPosition": "long", "prevMarketPosition": "short", "marketPositionSize": "0.700223", "prevMarketPositionSize": "0.702703", "signalToken": "manual_trade", "timestamp": "2025-12-17T13:30:01Z", "comment": "Long", "message": "", "exchange_name": "coinbase"}'

	curl -X POST https://api.biidin.com/api/tv/signal \
	  -H "Content-Type: application/json" \
	  -d '{ "id": "Long", "symbol": "BTCUSD", "action": "buy", "qty": "1.402926", "price": "87068.01", "type": "Market", "marketPosition": "long", "prevMarketPosition": "short", "marketPositionSize": "0.700223", "prevMarketPositionSize": "0.702703", "signalToken": "manual_trade", "timestamp": "2025-12-17T13:30:01Z", "comment": "Long", "message": "", "exchange_name": "hydra"}'

