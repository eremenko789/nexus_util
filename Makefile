# Nexus Util - Makefile for cross-platform builds
#
# `make help` lists every target. Targets fall into two groups:
#   read-only  : help, fmt-check, lint, lint-full, vet, test, test-race,
#                test-cover, build, verify, generate, vulncheck
#   stateful   : bootstrap/tools (installs into .artifacts), fmt, tidy, deps,
#                generate, clean, build-all, release, run-container
#
# There are no //go:generate directives in this repository yet; `make generate`
# reports that and succeeds.

# Application name
APP_NAME = nexus-util

# Version information
VERSION ?= 1.0.0
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Pure-Go build (no glibc / CGO)
export CGO_ENABLED = 0
GO_TAGS = netgo osusergo

# Build flags
GO_BUILD_FLAGS = -tags "$(GO_TAGS)" -trimpath
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION) -X main.build=$(GIT_COMMIT)"

# Local, git-ignored outputs
ARTIFACTS_DIR ?= .artifacts
TOOLS_DIR     ?= $(ARTIFACTS_DIR)/bin
COVERAGE_FILE ?= $(ARTIFACTS_DIR)/coverage.out

# Pinned tool versions. Keep in sync with scripts/install-tools.sh.
# golangci-lint v1 is required: .golangci.yml uses the v1 configuration schema.
GOLANGCI_LINT_VERSION ?= v1.62.2
# govulncheck >= v1.1.4 is required: v1.1.3 fails to build on Go >= 1.22 (its
# golang.org/x/tools pin rejects newer compilers). v1.1.4 needs Go >= 1.22, so on
# the go 1.21 directive from go.mod it relies on Go's default toolchain switching.
GOVULNCHECK_VERSION   ?= v1.1.4

# Prefer a golangci-lint already on PATH, otherwise use the pinned local copy.
GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || echo $(TOOLS_DIR)/golangci-lint)
GOLANGCI_LINT_FLAGS ?= --timeout=5m
# Formatting-only rules; the full rule set lives in .golangci.yml. golangci-lint
# merges `linters.enable` from the main config with CLI flags, so `--disable-all`
# alone cannot narrow the full config down to the formatters.
FMT_LINT_CONFIG ?= .golangci-fmt.yml

# Baseline for the incremental lint. Defaults to the merge base with origin/main
# so only issues introduced by the current branch/working tree are reported.
# Left empty when origin/main cannot be resolved (no remote, shallow clone); the
# lint target then falls back to a full lint with a warning rather than silently
# reporting nothing. Override explicitly, e.g. `make lint LINT_NEW_FROM=origin/main`.
LINT_NEW_FROM ?= $(shell git merge-base HEAD origin/main 2>/dev/null || true)

# Extra flags for go test, e.g. GO_TEST_FLAGS='-count=1 -v'
GO_TEST_FLAGS ?=

# Paths owned by code generation. Empty because no generators exist yet; set it
# when generators are added so `make generate-check` detects uncommitted output.
GENERATED_PATHS ?=

# Supported platforms and architectures
PLATFORMS = linux/amd64 linux/arm64 linux/arm \
            windows/amd64 \
            darwin/amd64 darwin/arm64 \
            freebsd/amd64 \
            openbsd/amd64 \
            netbsd/amd64

# Default target
.PHONY: all
all: clean build

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

.PHONY: bootstrap
bootstrap: ## Download module dependencies and install pinned local tools
	@echo "Downloading Go module dependencies..."
	go mod download
	@$(MAKE) --no-print-directory tools

.PHONY: tools
tools: ## Install the pinned developer tools into .artifacts/bin
	@TOOLS_DIR=$(TOOLS_DIR) GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) \
		./scripts/install-tools.sh golangci-lint

.PHONY: deps
deps: ## Alias kept for compatibility (go mod download + go mod tidy)
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

.PHONY: tidy
tidy: ## Rewrite go.mod/go.sum (mutating; review the diff afterwards)
	go mod tidy

# ---------------------------------------------------------------------------
# Formatting and static analysis
# ---------------------------------------------------------------------------

.PHONY: fmt
fmt: $(GOLANGCI_LINT) ## Format Go sources (gofmt -s, goimports, whitespace)
	@echo "Formatting Go sources..."
	@$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS) -c $(FMT_LINT_CONFIG) --fix

.PHONY: fmt-check
fmt-check: $(GOLANGCI_LINT) ## Fail if any Go file is not formatted
	@unformatted=$$(gofmt -s -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt -s clean:"; \
		echo "$$unformatted"; \
		echo "Run: make fmt"; \
		exit 1; \
	fi
	@$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS) -c $(FMT_LINT_CONFIG)

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint issues introduced since origin/main (fast, no legacy debt)
	@if [ -n "$(LINT_NEW_FROM)" ]; then \
		echo "Linting changes since $(LINT_NEW_FROM) (whole repository: make lint-full)"; \
		$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS) --new-from-rev=$(LINT_NEW_FROM); \
	else \
		echo "WARNING: origin/main is unavailable, so there is no baseline for an"; \
		echo "incremental lint. Running a full lint instead - the pre-existing findings"; \
		echo "in AGENTS.md will be reported. Fetch the default branch for a diff-only lint."; \
		$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS); \
	fi

.PHONY: lint-full
lint-full: $(GOLANGCI_LINT) ## Lint the whole repository (reports ~48 known pre-existing issues)
	@echo "Linting the whole repository (pre-existing issues included)..."
	@$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS)

.PHONY: vet
vet: ## Run go vet on all packages
	go vet ./...

.PHONY: vulncheck
vulncheck: ## Run govulncheck (requires network access to vuln.go.dev)
	@echo "govulncheck contacts vuln.go.dev, so it is not part of 'make verify'."
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

$(GOLANGCI_LINT):
	@$(MAKE) --no-print-directory tools

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

.PHONY: test
test: ## Run the unit tests
	@echo "Running tests..."
	go test $(GO_TEST_FLAGS) ./...

.PHONY: test-race
test-race: ## Run the unit tests with the race detector (needs cgo + a C toolchain)
	@echo "Running tests with the race detector..."
	CGO_ENABLED=1 go test -race $(GO_TEST_FLAGS) ./...

.PHONY: test-cover
test-cover: ## Write a coverage profile to .artifacts/coverage.out and print the total
	@mkdir -p $(ARTIFACTS_DIR)
	go test $(GO_TEST_FLAGS) -covermode=atomic -coverpkg=./... \
		-coverprofile=$(COVERAGE_FILE) ./...
	@echo "Total coverage:"
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	@echo "HTML report: go tool cover -html=$(COVERAGE_FILE)"

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

.PHONY: build
build: ## Build the binary for the current platform into bin/
	@echo "Building $(APP_NAME) for current platform..."
	go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o bin/$(APP_NAME) .

# Build for all platforms
.PHONY: build-all
build-all: clean ## Cross-compile every supported platform into bin/
	@echo "Building $(APP_NAME) for all platforms..."
	@mkdir -p bin
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		OUTPUT_NAME=$(APP_NAME); \
		if [ "$$OS" = "windows" ]; then OUTPUT_NAME=$(APP_NAME).exe; fi; \
		echo "Building for $$OS/$$ARCH..."; \
		GOOS=$$OS GOARCH=$$ARCH go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o bin/$(APP_NAME)-$$OS-$$ARCH$$(if [ "$$OS" = "windows" ]; then echo .exe; fi) .; \
	done
	@echo "Build completed! Binaries are in bin/ directory"

# Build for specific platform
.PHONY: build-linux-amd64
build-linux-amd64: ## Cross-compile for linux/amd64
	@echo "Building for linux/amd64..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 .

.PHONY: build-windows-amd64
build-windows-amd64: ## Cross-compile for windows/amd64
	@echo "Building for windows/amd64..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe .

.PHONY: build-darwin-amd64
build-darwin-amd64: ## Cross-compile for darwin/amd64
	@echo "Building for darwin/amd64..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 .

.PHONY: build-darwin-arm64
build-darwin-arm64: ## Cross-compile for darwin/arm64
	@echo "Building for darwin/arm64..."
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64 .

# ---------------------------------------------------------------------------
# Code generation
# ---------------------------------------------------------------------------

.PHONY: generate
generate: ## Run go generate, or report that this repository has no generators
	@directives=$$(grep -rl 'go:generate' --include='*.go' . 2>/dev/null | grep -v '^\./\.artifacts/' || true); \
	if [ -z "$$directives" ]; then \
		echo "No //go:generate directives in this repository - nothing to generate."; \
	else \
		echo "Running: go generate ./..."; \
		echo "$$directives" | sed 's/^/  generator: /'; \
		go generate ./...; \
	fi

.PHONY: generate-check
generate-check: generate ## Fail if code generation leaves uncommitted changes
	@directives=$$(grep -rl 'go:generate' --include='*.go' . 2>/dev/null | grep -v '^\./\.artifacts/' || true); \
	if [ -z "$$directives" ]; then \
		echo "No generators - skipping generated-code drift check."; \
	elif [ -z "$(GENERATED_PATHS)" ]; then \
		echo "Generators exist but GENERATED_PATHS is empty; set it in the Makefile"; \
		echo "to the generated paths so drift can be detected."; \
	elif ! git diff --quiet -- $(GENERATED_PATHS); then \
		echo "Generation changed files that should be committed:"; \
		git --no-pager diff --stat -- $(GENERATED_PATHS); \
		exit 1; \
	fi

# ---------------------------------------------------------------------------
# Verification
# ---------------------------------------------------------------------------

.PHONY: verify
verify: fmt-check vet lint test build generate-check ## Run the checks required before a commit or PR
	@echo ""
	@echo "verify: all checks passed."

# ---------------------------------------------------------------------------
# Housekeeping
# ---------------------------------------------------------------------------

.PHONY: clean
clean: ## Remove local build, tooling and coverage artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(ARTIFACTS_DIR) bin release
	go clean

# Run the application
.PHONY: run
run: build ## Build and run `nexus-util --help`
	@echo "Running $(APP_NAME)..."
	./bin/$(APP_NAME) --help

# Create release packages
.PHONY: release
release: build-all ## Create release tarballs in release/ (local only, never pushes)
	@echo "Creating release packages..."
	@mkdir -p release
	@for binary in bin/$(APP_NAME)-*; do \
		OS=$$(echo $$binary | sed 's/.*-\([^-]*\)-[^-]*$$/\1/'); \
		ARCH=$$(echo $$binary | sed 's/.*-\([^-]*\)$$/\1/' | sed 's/\.exe$$//'); \
		EXT=""; \
		if [ "$$OS" = "windows" ]; then EXT=".exe"; fi; \
		PACKAGE_NAME=$(APP_NAME)-$(VERSION)-$$OS-$$ARCH; \
		mkdir -p release/$$PACKAGE_NAME; \
		cp $$binary release/$$PACKAGE_NAME/$(APP_NAME)$$EXT; \
		cp README.md release/$$PACKAGE_NAME/; \
		cd release && tar -czf $$PACKAGE_NAME.tar.gz $$PACKAGE_NAME/; \
		cd ..; \
		rm -rf release/$$PACKAGE_NAME; \
	done
	@echo "Release packages created in release/ directory"

.PHONY: run-container
run-container: ## Build and enter the Dockerfile toolchain image (needs Docker + registry access)
	docker build . -t go_build:latest
	docker run -it --rm -v $(CURDIR):$(CURDIR) -w $(CURDIR) --entrypoint bash go_build:latest

# Show help
.PHONY: help
help: ## Show this help
	@echo "Nexus Util - available targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_.-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
	@echo ""
	@echo "Variables (override on the command line, e.g. make test GO_TEST_FLAGS=-count=1):"
	@echo "  VERSION=$(VERSION)  ARTIFACTS_DIR=$(ARTIFACTS_DIR)  TOOLS_DIR=$(TOOLS_DIR)"
	@echo "  GO_TEST_FLAGS='$(GO_TEST_FLAGS)'  LINT_NEW_FROM=$(LINT_NEW_FROM)"
	@echo "  GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION)  GOVULNCHECK_VERSION=$(GOVULNCHECK_VERSION)"
	@echo ""
	@echo "Pinned linter in use: $(GOLANGCI_LINT)"
