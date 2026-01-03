# Application
APP_NAME := carbon
MODULE := github.com/lugondev/go-carbon

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags "-w -s \
	-X $(MODULE)/cmd/carbon/cmd.Version=$(VERSION) \
	-X $(MODULE)/cmd/carbon/cmd.GitCommit=$(GIT_COMMIT) \
	-X $(MODULE)/cmd/carbon/cmd.BuildDate=$(BUILD_DATE)"

# Directories
BUILD_DIR := ./bin
CMD_DIR := ./cmd/carbon

.PHONY: all build clean test lint fmt vet tidy docker run help

all: clean lint test build

## Build
build: ## Build the binary
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Binary built at $(BUILD_DIR)/$(APP_NAME)"

build-linux: ## Build for Linux
	@echo "Building $(APP_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_DIR)

build-darwin: ## Build for macOS
	@echo "Building $(APP_NAME) for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_DIR)

build-all: build-linux build-darwin ## Build for all platforms

## Testing
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

## Code Quality
lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

## Dependencies
tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

download: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

## Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t go-carbon:$(VERSION) \
		-t go-carbon:latest .

docker-run: ## Run Docker container
	docker run --rm -it go-carbon:latest

docker-compose-up: ## Start docker-compose services
	docker-compose up -d

docker-compose-down: ## Stop docker-compose services
	docker-compose down

docker-compose-dev: ## Start with local Solana validator
	docker-compose --profile dev up -d

## Development
run: build ## Build and run
	$(BUILD_DIR)/$(APP_NAME)

dev: ## Run in development mode
	$(GOCMD) run $(CMD_DIR) $(ARGS)

## Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## Help
help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
