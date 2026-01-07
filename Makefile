.PHONY: build test lint coverage clean install release-local security \
	docker-test docker-test-unit docker-test-apt docker-test-brew docker-test-full \
	docker-test-files docker-test-e2e docker-coverage docker-lint docker-build docker-clean \
	test-e2e

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
	coverctl check --fail-under 80

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

# =============================================================================
# Docker Testing Targets
# =============================================================================

DOCKER_COMPOSE=docker compose -f docker-compose.test.yml

## docker-test: Run all tests in Docker (unit + integration)
docker-test: docker-test-unit docker-test-files

## docker-test-unit: Run unit tests in Docker
docker-test-unit:
	$(DOCKER_COMPOSE) run --rm unit

## docker-test-apt: Run APT provider integration tests in Docker
docker-test-apt:
	$(DOCKER_COMPOSE) run --rm apt

## docker-test-brew: Run Homebrew provider integration tests in Docker
docker-test-brew:
	$(DOCKER_COMPOSE) run --rm brew

## docker-test-files: Run files provider integration tests in Docker
docker-test-files:
	$(DOCKER_COMPOSE) run --rm files

## docker-test-ssh: Run SSH provider integration tests in Docker
docker-test-ssh:
	$(DOCKER_COMPOSE) run --rm ssh

## docker-test-full: Run full integration test suite in Docker
docker-test-full:
	$(DOCKER_COMPOSE) run --rm full

## docker-test-e2e: Run end-to-end tests in Docker
docker-test-e2e:
	$(DOCKER_COMPOSE) run --rm e2e

## test-e2e: Run end-to-end tests locally
test-e2e: build
	$(GOTEST) -v -tags=e2e ./test/e2e/...

## docker-coverage: Generate coverage report in Docker
docker-coverage:
	mkdir -p coverage
	$(DOCKER_COMPOSE) run --rm coverage
	@echo "Coverage report: coverage/coverage.html"

## docker-lint: Run linter in Docker
docker-lint:
	$(DOCKER_COMPOSE) run --rm lint

## docker-build: Build Linux binary in Docker
docker-build:
	mkdir -p bin
	$(DOCKER_COMPOSE) run --rm build
	@echo "Binary: bin/preflight-linux"

## docker-clean: Clean Docker test resources
docker-clean:
	$(DOCKER_COMPOSE) down -v --rmi local
	rm -rf coverage/

## docker-shell: Open shell in test container for debugging
docker-shell:
	$(DOCKER_COMPOSE) run --rm --entrypoint /bin/bash unit

# Default target
.DEFAULT_GOAL := help
