# Nexus Util - Makefile for cross-platform builds

# Application name
APP_NAME = nexus-util

# Version information
VERSION ?= 1.0.0
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.build=$(GIT_COMMIT)"

# Supported platforms and architectures
PLATFORMS = linux/amd64 linux/arm64 linux/arm linux/386 \
            windows/amd64 windows/386 \
            darwin/amd64 darwin/arm64 \
            freebsd/amd64 freebsd/386 \
            openbsd/amd64 openbsd/386 \
            netbsd/amd64 netbsd/386

# Default target
.PHONY: all
all: clean build

# Build for current platform
.PHONY: build
build:
	@echo "Building $(APP_NAME) for current platform..."
	go build $(LDFLAGS) -o bin/$(APP_NAME) .

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "Building $(APP_NAME) for all platforms..."
	@mkdir -p bin
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		OUTPUT_NAME=$(APP_NAME); \
		if [ "$$OS" = "windows" ]; then OUTPUT_NAME=$(APP_NAME).exe; fi; \
		echo "Building for $$OS/$$ARCH..."; \
		GOOS=$$OS GOARCH=$$ARCH go build $(LDFLAGS) -o bin/$(APP_NAME)-$$OS-$$ARCH$$(if [ "$$OS" = "windows" ]; then echo .exe; fi) .; \
	done
	@echo "Build completed! Binaries are in bin/ directory"

# Build for specific platform
.PHONY: build-linux-amd64
build-linux-amd64:
	@echo "Building for linux/amd64..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 .

.PHONY: build-windows-amd64
build-windows-amd64:
	@echo "Building for windows/amd64..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe .

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@echo "Building for darwin/amd64..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 .

.PHONY: build-darwin-arm64
build-darwin-arm64:
	@echo "Building for darwin/arm64..."
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64 .

# Test
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Lint
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run the application
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	./bin/$(APP_NAME) --help

# Create release packages
.PHONY: release
release: build-all
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

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-all      - Build for all supported platforms"
	@echo "  build-linux-amd64    - Build for Linux AMD64"
	@echo "  build-windows-amd64  - Build for Windows AMD64"
	@echo "  build-darwin-amd64   - Build for macOS AMD64"
	@echo "  build-darwin-arm64   - Build for macOS ARM64"
	@echo "  test           - Run tests"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  run            - Run the application"
	@echo "  release        - Create release packages"
	@echo "  help           - Show this help"