.PHONY: test lint watch install-tools build help

# Set the default goal to `help` so running plain `make` shows the menu
.DEFAULT_GOAL := help

# -----------------------------------------------------------------------------
# ✨ Help: show available targets
#   Inspired by: https://marmelab.com/blog/2020/11/18/makefile-cheatsheet.html
# -----------------------------------------------------------------------------
help: ## Show this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "; printf "\nAvailable targets:\n\n"} { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }'

# Run the full Go test suite
# Usage: make test
# --------------------------------------
# Runs `go test ./...` which walks every package in the
# repository and executes its unit / integration tests.
# Suitable for local development and CI.
# --------------------------------------
test: ## Run full Go test suite (`go test ./...`)
	go test ./...

# Build the prompt-sync binary
# Usage: make build
# --------------------------------------
# Builds the CLI binary into bin/prompt-sync
# --------------------------------------
build: ## Build the prompt-sync binary
	go build -o bin/prompt-sync ./cmd/prompt-sync

# Run the aggregated linter suite via golangci-lint
# Usage: make lint
# Auto-installs golangci-lint into GOPATH/bin if missing, then runs it.
lint: ## Lint Go code with golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { \
		printf "golangci-lint not found. Installing...\n"; \
		GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2; \
	}
	$(shell go env GOPATH)/bin/golangci-lint run

# Install all developer tooling dependencies (currently only golangci-lint)
install-tools: ## Install developer tooling dependencies
	GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2

# Human-only file watcher that re-runs tests on Go file changes.
# Requires `entr` utility. Do NOT run in CI or with AI agents.
watch: ## Watch Go files and auto-test on changes (human-only)
	find . -type f -name '*.go' | entr -c make test

export PATH := $(PATH):$(shell go env GOPATH)/bin
