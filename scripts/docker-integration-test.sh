#!/usr/bin/env bash
# docker-integration-test.sh - Run integration tests in Docker containers
#
# Usage:
#   ./scripts/docker-integration-test.sh [options] [test-suite]
#
# Test Suites:
#   unit        Run unit tests only (default)
#   apt         Run APT provider integration tests
#   brew        Run Homebrew provider integration tests
#   files       Run files provider integration tests
#   ssh         Run SSH provider integration tests
#   full        Run full integration test suite
#   all         Run all test suites
#
# Options:
#   -v, --verbose    Enable verbose output
#   -c, --coverage   Generate coverage report
#   -h, --help       Show this help message

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
VERBOSE=false
COVERAGE=false
TEST_SUITE="unit"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Docker compose command
DOCKER_COMPOSE="docker compose -f ${PROJECT_ROOT}/docker-compose.test.yml"

# Print colored message
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Show usage
usage() {
    cat << EOF
Docker Integration Test Runner for Preflight

Usage: $0 [options] [test-suite]

Test Suites:
  unit        Run unit tests only (default)
  apt         Run APT provider integration tests
  brew        Run Homebrew provider integration tests
  files       Run files provider integration tests
  ssh         Run SSH provider integration tests
  full        Run full integration test suite
  all         Run all test suites sequentially

Options:
  -v, --verbose    Enable verbose output
  -c, --coverage   Generate coverage report after tests
  -h, --help       Show this help message

Examples:
  $0                    # Run unit tests
  $0 apt                # Run APT integration tests
  $0 -c full            # Run full integration tests with coverage
  $0 all                # Run all test suites

EOF
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--coverage)
                COVERAGE=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            unit|apt|brew|files|ssh|full|all)
                TEST_SUITE="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Check Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    log_info "Docker is available"
}

# Build test containers if needed
build_containers() {
    log_info "Building test containers..."

    local build_args=""
    if [ "$VERBOSE" = true ]; then
        build_args="--progress=plain"
    fi

    $DOCKER_COMPOSE build $build_args
    log_success "Containers built successfully"
}

# Run a specific test suite
run_suite() {
    local suite=$1
    local start_time=$(date +%s)

    log_info "Running $suite tests..."

    local run_args="--rm"
    if [ "$VERBOSE" = true ]; then
        run_args="$run_args -e GOTEST_FLAGS=-v"
    fi

    if $DOCKER_COMPOSE run $run_args "$suite"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "$suite tests passed in ${duration}s"
        return 0
    else
        log_error "$suite tests failed"
        return 1
    fi
}

# Run all test suites
run_all_suites() {
    local failed=0
    local suites=("unit" "files" "ssh" "apt")

    for suite in "${suites[@]}"; do
        if ! run_suite "$suite"; then
            ((failed++))
        fi
    done

    if [ $failed -gt 0 ]; then
        log_error "$failed test suite(s) failed"
        return 1
    fi

    log_success "All test suites passed"
    return 0
}

# Generate coverage report
generate_coverage() {
    log_info "Generating coverage report..."

    mkdir -p "${PROJECT_ROOT}/coverage"

    if $DOCKER_COMPOSE run --rm coverage; then
        log_success "Coverage report generated: coverage/coverage.html"
    else
        log_warning "Coverage generation failed"
    fi
}

# Cleanup
cleanup() {
    log_info "Cleaning up..."
    $DOCKER_COMPOSE down --remove-orphans &> /dev/null || true
}

# Main entry point
main() {
    parse_args "$@"

    # Setup cleanup trap
    trap cleanup EXIT

    # Change to project root
    cd "$PROJECT_ROOT"

    # Check prerequisites
    check_docker

    # Build containers
    build_containers

    # Run tests
    local exit_code=0
    case $TEST_SUITE in
        all)
            if ! run_all_suites; then
                exit_code=1
            fi
            ;;
        *)
            if ! run_suite "$TEST_SUITE"; then
                exit_code=1
            fi
            ;;
    esac

    # Generate coverage if requested
    if [ "$COVERAGE" = true ]; then
        generate_coverage
    fi

    exit $exit_code
}

main "$@"
