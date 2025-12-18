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
test: ## Run tests with race detection
	@echo "${BLUE}Running tests...${NC}"
	@$(GOTEST) $(TEST_FLAGS) ./...

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
