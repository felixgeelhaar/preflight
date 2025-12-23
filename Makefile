.PHONY: build test lint coverage clean install release-local security

# Binary name
BINARY_NAME=preflight
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/preflight

## test: Run all tests
test:
	$(GOTEST) -v ./...

## test-race: Run tests with race detector
test-race:
	$(GOTEST) -race -v ./...

## coverage: Generate coverage report
coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## coverage-check: Check coverage meets threshold (requires coverctl)
coverage-check:
	$(GOTEST) -coverprofile=coverage.out ./...
	coverctl check --threshold 80

## lint: Run linter
lint:
	golangci-lint run

## lint-fix: Run linter with auto-fix
lint-fix:
	golangci-lint run --fix

## tidy: Tidy go modules
tidy:
	$(GOMOD) tidy

## clean: Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## install: Install binary to GOPATH/bin
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

## deps: Download dependencies
deps:
	$(GOMOD) download

## release-local: Build release artifacts for all platforms locally
release-local:
	./scripts/build-artifacts.sh $(VERSION) $(COMMIT) $(BUILD_DATE)

## security: Run security vulnerability check
security:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

# Default target
.DEFAULT_GOAL := help
