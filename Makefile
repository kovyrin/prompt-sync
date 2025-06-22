.PHONY: test lint watch install-tools

# Run the full Go test suite
# Usage: make test
# --------------------------------------
# Runs `go test ./...` which walks every package in the
# repository and executes its unit / integration tests.
# Suitable for local development and CI.
# --------------------------------------
test:
	go test ./...

# Run the aggregated linter suite via golangci-lint
# Usage: make lint
# --------------------------------------
# Requires golangci-lint to be installed (see docs).
# Fails the build if any linter issues are found.
# --------------------------------------
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		printf "golangci-lint not found. Installing...\n"; \
		GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2; \
	}
	$(shell go env GOPATH)/bin/golangci-lint run

# Install all developer tooling dependencies (currently only golangci-lint)
install-tools:
	GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2

# Human-only file-watcher that re-runs the test suite on every file change.
# NOTE: AI agents must NOT run this target. Use `make test` manually instead.
# Requires the `entr` utility: `brew install entr` (macOS) or `apt-get install entr` (Debian-based).
# Usage: make watch
# --------------------------------------
watch:
	find . -type f -name '*.go' | entr -c make test

export PATH := $(PATH):$(shell go env GOPATH)/bin
