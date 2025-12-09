.PHONY: build test run lint migrate clean help

# Build settings
BINARY_NAME=server
BINARY_CLI=fo
BUILD_DIR=bin
GO=go

# Build the server
build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	$(GO) build -o $(BUILD_DIR)/$(BINARY_CLI) ./cmd/fo

# Build server only
build-server:
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Build CLI only
build-cli:
	$(GO) build -o $(BUILD_DIR)/$(BINARY_CLI) ./cmd/fo

# Run the server
run:
	$(GO) run ./cmd/server

# Run the CLI
run-cli:
	$(GO) run ./cmd/fo

# Run tests
test:
	$(GO) test -v ./...

# Run integration tests
test-integration:
	$(GO) test -v -tags=integration ./tests/integration/...

# Run unit tests
test-unit:
	$(GO) test -v ./internal/... ./pkg/...

# Run linter
lint:
	golangci-lint run

# Run database migrations up
migrate:
	$(GO) run ./cmd/migrate up

# Run database migrations down
migrate-down:
	$(GO) run ./cmd/migrate down

# Run database migrations to specific version
migrate-to:
	$(GO) run ./cmd/migrate to $(VERSION)

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Generate test coverage
coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Start development environment
dev-up:
	docker compose up -d postgres redis

# Stop development environment
dev-down:
	docker compose down

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build server and CLI binaries"
	@echo "  build-server    - Build server binary only"
	@echo "  build-cli       - Build CLI binary only"
	@echo "  run             - Run the server"
	@echo "  run-cli         - Run the CLI"
	@echo "  test            - Run all tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-unit       - Run unit tests"
	@echo "  lint            - Run linter"
	@echo "  migrate         - Run database migrations up"
	@echo "  migrate-down    - Run database migrations down"
	@echo "  clean           - Clean build artifacts"
	@echo "  coverage        - Generate test coverage report"
	@echo "  dev-up          - Start development dependencies"
	@echo "  dev-down        - Stop development dependencies"
