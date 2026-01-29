.DEFAULT_GOAL := test
.PHONY: test test-unit test-integration test-coverage \
        build build-dev build-release \
        lint fmt deadcode ci clean \
        change change-new change-preview \
        deps-tools

# CI detection
ifdef CI
  LINT_FIX :=
  RACE := -race
else
  LINT_FIX := --fix
  RACE :=
endif

# GH_TOKEN auto-detection for integration tests
GH_TOKEN ?= $(shell gh auth token 2>/dev/null)
export GH_TOKEN

# Platforms for release builds
PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64

# ──────────────────────────────────────────────────────────────────────────────
# Test targets
# ──────────────────────────────────────────────────────────────────────────────

test: test-unit

test-unit:
	@echo "Running unit tests..."
	@gotestsum -- -tags=!integration -short ./...

test-integration:
	@echo "Running integration tests..."
	@echo "→ Installing grove binary..."
	@go install ./cmd/grove
	@[ -z "$(GH_TOKEN)" ] || echo "→ Using gh CLI authentication for PR tests"
	@gotestsum -- -tags=integration -timeout=300s ./cmd/grove/...

test-coverage:
	@echo "Running unit tests with coverage..."
	@mkdir -p coverage
	@go test -v -tags=!integration -short -coverprofile=coverage/coverage.out -covermode=atomic $(RACE) ./...
	@go tool cover -func=coverage/coverage.out

# ──────────────────────────────────────────────────────────────────────────────
# Build targets
# ──────────────────────────────────────────────────────────────────────────────

build: build-dev

build-dev:
	@echo "Building Grove..."
	@mkdir -p bin
	@go build -o bin/grove ./cmd/grove

build-release:
	@echo "Building release binaries..."
	@mkdir -p bin
	@for platform in $(PLATFORMS); do \
		os=$${platform%%-*}; \
		arch=$${platform#*-}; \
		echo "Building bin/grove-$$platform..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags="-s -w" -o bin/grove-$$platform ./cmd/grove; \
	done

# ──────────────────────────────────────────────────────────────────────────────
# Code quality
# ──────────────────────────────────────────────────────────────────────────────

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run $(LINT_FIX)

fmt:
	@echo "Formatting code..."
	@gofmt -w -s .
	@echo "Formatting Markdown/JSON/YAML..."
	@npx prettier --write .

deadcode:
	@echo "Checking for dead code..."
	@go run golang.org/x/tools/cmd/deadcode@latest ./...

# ──────────────────────────────────────────────────────────────────────────────
# CI pipeline
# ──────────────────────────────────────────────────────────────────────────────

ci: clean lint test-coverage test-integration build-dev
	@echo "CI pipeline completed successfully!"

clean:
	@echo "Cleaning all artifacts..."
	@rm -rf coverage bin
	@go clean -testcache

# ──────────────────────────────────────────────────────────────────────────────
# Changelog management
# ──────────────────────────────────────────────────────────────────────────────

change: change-new

change-new:
	@command -v changie >/dev/null 2>&1 || go install github.com/miniscruff/changie@latest
	@echo "Creating change entry..."
	@changie new

change-preview:
	@command -v changie >/dev/null 2>&1 || go install github.com/miniscruff/changie@latest
	@echo "Unreleased changes:"
	@entries=$$(ls -1 .changes/unreleased/*.yaml 2>/dev/null | wc -l); \
	if [ "$$entries" -eq 0 ]; then \
		echo "  (none)"; \
	else \
		for f in .changes/unreleased/*.yaml; do \
			echo "  • $$(basename $$f)"; \
		done; \
		echo ""; \
		echo "Next version: $$(changie next auto)"; \
	fi

# ──────────────────────────────────────────────────────────────────────────────
# Development tools
# ──────────────────────────────────────────────────────────────────────────────

deps-tools:
	@echo "Installing development tools..."
	@echo "  Installing gotestsum..."
	@go install gotest.tools/gotestsum@v1.13.0
	@echo "  Installing changie..."
	@go install github.com/miniscruff/changie@v1.24.0
	@echo "✓ All tools installed"
