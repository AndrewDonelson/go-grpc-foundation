.PHONY: help build build-debug build-prod test test-coverage coverage example example-server example-client clean install-deps proto proto-example lint vet fmt

# Variables
BINARY_NAME := go-grpc-foundation
EXAMPLE_SERVER := example-server
EXAMPLE_CLIENT := example-client
BUILD_DIR := bin
COVERAGE_DIR := coverage
PROTO_DIR := example/proto
PB_DIR := example/pkg/pb

# Go build flags
LDFLAGS := -s -w
LDFLAGS_DEBUG := -X main.version=dev -X main.buildTime=$(shell date +%Y-%m-%dT%H:%M:%S)
LDFLAGS_PROD := -X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown") -X main.buildTime=$(shell date +%Y-%m-%dT%H:%M:%S)

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m
COLOR_CYAN := \033[36m

# Default target
.DEFAULT_GOAL := help

# ============================================================================
# Help / Usage
# ============================================================================

help: ## Show this help message
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)go-grpc-foundation - Makefile Commands$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Available targets:$(COLOR_RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(COLOR_BOLD)Examples:$(COLOR_RESET)"
	@echo "  $(COLOR_YELLOW)make build-example$(COLOR_RESET)   - Build example binaries (server + client)"
	@echo "  $(COLOR_YELLOW)make build-debug$(COLOR_RESET)     - Verify packages compile (library)"
	@echo "  $(COLOR_YELLOW)make build-prod$(COLOR_RESET)      - Verify packages compile (library)"
	@echo "  $(COLOR_YELLOW)make test$(COLOR_RESET)            - Run all tests"
	@echo "  $(COLOR_YELLOW)make coverage$(COLOR_RESET)        - Generate and show coverage report"
	@echo "  $(COLOR_YELLOW)make example$(COLOR_RESET)         - Run example (server + client)"
	@echo ""
	@echo "$(COLOR_YELLOW)Note:$(COLOR_RESET) This is a library/framework. Use 'make build-example' to build binaries."
	@echo ""

# ============================================================================
# Build Targets
# ============================================================================

build: build-debug ## Build debug version (default)

build-debug: ## Verify all packages compile (library - no binary produced)
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Verifying package compilation (debug)...$(COLOR_RESET)"
	@go build -ldflags "$(LDFLAGS_DEBUG)" ./pkg/...
	@echo "$(COLOR_GREEN)✓ All packages compile successfully$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)ℹ This is a library - no binary is produced. Use 'make build-example' for binaries.$(COLOR_RESET)"

build-prod: ## Verify all packages compile (library - no binary produced)
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Verifying package compilation (production)...$(COLOR_RESET)"
	@go build -ldflags "$(LDFLAGS) $(LDFLAGS_PROD)" ./pkg/...
	@echo "$(COLOR_GREEN)✓ All packages compile successfully$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)ℹ This is a library - no binary is produced. Use 'make build-example' for binaries.$(COLOR_RESET)"

# ============================================================================
# Example Build Targets
# ============================================================================

build-example: proto-example ## Build example server and client
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Building example server and client...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(EXAMPLE_SERVER) ./example/server
	@go build -o $(BUILD_DIR)/$(EXAMPLE_CLIENT) ./example/client
	@echo "$(COLOR_GREEN)✓ Example builds complete:$(COLOR_RESET)"
	@echo "  - $(BUILD_DIR)/$(EXAMPLE_SERVER)"
	@echo "  - $(BUILD_DIR)/$(EXAMPLE_CLIENT)"

# ============================================================================
# Test Targets
# ============================================================================

test: ## Run all tests
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Running all tests...$(COLOR_RESET)"
	@go test -v -timeout 60s ./pkg/... ./example/...
	@echo "$(COLOR_GREEN)✓ All tests passed$(COLOR_RESET)"

test-coverage: ## Run tests with coverage
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Running tests with coverage...$(COLOR_RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@go test -v -timeout 60s -coverprofile=$(COVERAGE_DIR)/coverage.out ./pkg/... ./example/...
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage.txt
	@echo "$(COLOR_GREEN)✓ Coverage report generated: $(COVERAGE_DIR)/coverage.out$(COLOR_RESET)"

coverage: test-coverage ## Generate and display coverage report
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Coverage Report:$(COLOR_RESET)"
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)$(shell printf '=%.0s' {1..80})$(COLOR_RESET)"
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Package Coverage:$(COLOR_RESET)"
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep -E "pkg/(server|config|database|middleware|logging)" | grep -v "test" | awk '{printf "  %-50s %s\n", $$1, $$3}'
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Coverage below 100%:$(COLOR_RESET)"
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep -v "100.0%" | grep -E "pkg/" | grep -v "test" | head -10 || echo "  All packages at 100%!"
	@echo ""
	@echo "$(COLOR_GREEN)Full report: $(COVERAGE_DIR)/coverage.txt$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)HTML report: make coverage-html$(COLOR_RESET)"

coverage-html: test-coverage ## Generate HTML coverage report
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Generating HTML coverage report...$(COLOR_RESET)"
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(COLOR_GREEN)✓ HTML report generated: $(COVERAGE_DIR)/coverage.html$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Open with: open $(COVERAGE_DIR)/coverage.html$(COLOR_RESET)"

# ============================================================================
# Example Targets
# ============================================================================

example: proto-example build-example ## Run example (starts server, then runs client with all logs)
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)========================================$(COLOR_RESET)"
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Running gRPC Communication Types Example$(COLOR_RESET)"
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)========================================$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_YELLOW)Cleaning up any existing servers...$(COLOR_RESET)"
	@if [ -f /tmp/example-server.pid ]; then \
		kill $$(cat /tmp/example-server.pid) 2>/dev/null || true; \
		rm -f /tmp/example-server.pid; \
	fi
	@if command -v lsof >/dev/null 2>&1; then \
		lsof -ti:50051 | xargs kill -9 2>/dev/null || true; \
	elif command -v fuser >/dev/null 2>&1; then \
		fuser -k 50051/tcp 2>/dev/null || true; \
	fi
	@rm -f /tmp/example-server.log
	@echo "$(COLOR_GREEN)✓ Cleanup complete$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_YELLOW)Starting server in background...$(COLOR_RESET)"
	@$(BUILD_DIR)/$(EXAMPLE_SERVER) > /tmp/example-server.log 2>&1 & \
		echo $$! > /tmp/example-server.pid && \
		echo "$(COLOR_GREEN)✓ Server started (PID: $$(cat /tmp/example-server.pid))$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Waiting for server to be ready...$(COLOR_RESET)"
	@for i in 1 2 3 4 5; do \
		sleep 1; \
		if command -v nc >/dev/null 2>&1 && nc -z localhost 50051 2>/dev/null; then \
			echo "$(COLOR_GREEN)✓ Server is ready$(COLOR_RESET)"; \
			break; \
		elif command -v timeout >/dev/null 2>&1 && timeout 0.1 bash -c "echo >/dev/tcp/localhost/50051" 2>/dev/null; then \
			echo "$(COLOR_GREEN)✓ Server is ready$(COLOR_RESET)"; \
			break; \
		fi; \
		if [ $$i -eq 5 ]; then \
			echo "$(COLOR_YELLOW)⚠ Server may not be ready, continuing anyway...$(COLOR_RESET)"; \
		fi; \
	done
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Server Logs:$(COLOR_RESET)"
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)$(shell printf '=%.0s' {1..80})$(COLOR_RESET)"
	@tail -20 /tmp/example-server.log || echo "No server logs yet"
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Running client...$(COLOR_RESET)"
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)$(shell printf '=%.0s' {1..80})$(COLOR_RESET)"
	@$(BUILD_DIR)/$(EXAMPLE_CLIENT) || true
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Server Logs (full):$(COLOR_RESET)"
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)$(shell printf '=%.0s' {1..80})$(COLOR_RESET)"
	@cat /tmp/example-server.log || echo "No server logs"
	@echo ""
	@echo "$(COLOR_YELLOW)Stopping server...$(COLOR_RESET)"
	@if [ -f /tmp/example-server.pid ]; then \
		kill $$(cat /tmp/example-server.pid) 2>/dev/null || true; \
		rm -f /tmp/example-server.pid; \
		echo "$(COLOR_GREEN)✓ Server stopped$(COLOR_RESET)"; \
	fi
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_GREEN)Example complete!$(COLOR_RESET)"

example-server: proto-example ## Start example server only (foreground with logs)
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Starting example server...$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Press Ctrl+C to stop$(COLOR_RESET)"
	@echo ""
	@cd example/server && go run main.go

example-client: proto-example ## Run example client only
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Running example client...$(COLOR_RESET)"
	@echo ""
	@cd example/client && go run main.go

# ============================================================================
# Protocol Buffer Targets
# ============================================================================

proto: ## Generate protocol buffer code for example
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Generating protocol buffer code...$(COLOR_RESET)"
	@if ! command -v protoc >/dev/null 2>&1; then \
		echo "$(COLOR_YELLOW)⚠ protoc not found. Installing dependencies...$(COLOR_RESET)"; \
		$(MAKE) install-deps; \
	fi
	@export PATH=$$PATH:$$(go env GOPATH)/bin && \
		mkdir -p $(PB_DIR) && \
		protoc --go_out=$(PB_DIR) \
			--go_opt=paths=source_relative \
			--go-grpc_out=$(PB_DIR) \
			--go-grpc_opt=paths=source_relative \
			--proto_path=$(PROTO_DIR) \
			$(PROTO_DIR)/demo/v1/demo.proto && \
		mv $(PB_DIR)/demo/v1/*.pb.go $(PB_DIR)/ 2>/dev/null || true && \
		rmdir $(PB_DIR)/demo/v1 $(PB_DIR)/demo 2>/dev/null || true
	@echo "$(COLOR_GREEN)✓ Protocol buffer code generated$(COLOR_RESET)"

proto-example: proto ## Alias for proto (for example targets)

# ============================================================================
# Development Targets
# ============================================================================

install-deps: ## Install required dependencies (protoc plugins)
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Installing dependencies...$(COLOR_RESET)"
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "$(COLOR_GREEN)✓ Dependencies installed$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Note: protoc must be installed separately$(COLOR_RESET)"
	@echo "  macOS: brew install protobuf"
	@echo "  Linux: apt-get install protobuf-compiler"
	@echo "  Windows: Download from https://github.com/protocolbuffers/protobuf/releases"

lint: ## Run linter (golangci-lint if available, else go vet)
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint not found, using go vet instead$(COLOR_RESET)"; \
		$(MAKE) vet; \
	fi

vet: ## Run go vet
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	@go vet ./pkg/... ./example/...
	@echo "$(COLOR_GREEN)✓ go vet passed$(COLOR_RESET)"

fmt: ## Format code with gofmt
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	@go fmt ./pkg/... ./example/...
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

fmt-check: ## Check if code is formatted correctly
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Checking code formatting...$(COLOR_RESET)"
	@if [ $$(gofmt -l ./pkg ./example | wc -l) -gt 0 ]; then \
		echo "$(COLOR_YELLOW)⚠ Code is not formatted. Run 'make fmt' to fix.$(COLOR_RESET)"; \
		gofmt -l ./pkg ./example; \
		exit 1; \
	else \
		echo "$(COLOR_GREEN)✓ Code is properly formatted$(COLOR_RESET)"; \
	fi

# ============================================================================
# Clean Targets
# ============================================================================

clean: ## Clean build artifacts
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -f /tmp/example-server.log /tmp/example-server.pid
	@go clean -cache -testcache
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

clean-all: clean ## Clean everything including generated code
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Cleaning all artifacts including generated code...$(COLOR_RESET)"
	@rm -rf $(PB_DIR)/*.pb.go
	@echo "$(COLOR_GREEN)✓ Full clean complete$(COLOR_RESET)"

# ============================================================================
# CI/CD Targets
# ============================================================================

ci: fmt-check vet test-coverage ## Run CI checks (format, vet, tests, coverage)
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_GREEN)✓ All CI checks passed!$(COLOR_RESET)"

# ============================================================================
# Info Targets
# ============================================================================

info: ## Show build information
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Build Information:$(COLOR_RESET)"
	@echo "  Go version:     $$(go version)"
	@echo "  Go module:      $$(go list -m)"
	@echo "  Build dir:      $(BUILD_DIR)"
	@echo "  Coverage dir:   $(COVERAGE_DIR)"
	@echo "  Binary name:    $(BINARY_NAME)"
	@echo ""
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)Available binaries:$(COLOR_RESET)"
	@ls -lh $(BUILD_DIR)/* 2>/dev/null || echo "  (none - run 'make build' first)"

