#!/usr/bin/env bash
# test_all_commands.sh - Comprehensive CLI command smoke test
# Runs every preflight command in a safe Docker environment and verifies output.
#
# Exit codes:
#   0 - All tests passed
#   1 - One or more tests failed

set -euo pipefail

PREFLIGHT="${PREFLIGHT_BINARY:-./bin/preflight}"
PASSED=0
FAILED=0
SKIPPED=0
FAILURES=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

pass() {
    PASSED=$((PASSED + 1))
    printf "  ${GREEN}PASS${NC} %s\n" "$1"
}

fail() {
    FAILED=$((FAILED + 1))
    FAILURES="${FAILURES}\n  - $1: $2"
    printf "  ${RED}FAIL${NC} %s: %s\n" "$1" "$2"
}

skip() {
    SKIPPED=$((SKIPPED + 1))
    printf "  ${YELLOW}SKIP${NC} %s: %s\n" "$1" "$2"
}

section() {
    printf "\n${CYAN}=== %s ===${NC}\n" "$1"
}

# Run a command and check exit code only
run_test_exit() {
    local name="$1"; shift
    local expected_exit="$1"; shift

    local actual_exit=0
    "$@" >/dev/null 2>&1 || actual_exit=$?

    if [ "$actual_exit" -eq "$expected_exit" ]; then
        pass "$name"
    else
        fail "$name" "exit code $actual_exit (expected $expected_exit)"
    fi
}

# Run a command, check exit code and that stdout+stderr contains pattern
run_test() {
    local name="$1"; shift
    local expected_exit="$1"; shift
    local pattern="$1"; shift

    local output
    local actual_exit=0
    output=$("$@" 2>&1) || actual_exit=$?

    if [ "$actual_exit" -ne "$expected_exit" ]; then
        fail "$name" "exit code $actual_exit (expected $expected_exit). Output: $(echo "$output" | head -3)"
        return
    fi

    if [ -z "$pattern" ]; then
        pass "$name"
        return
    fi

    if echo "$output" | grep -qF "$pattern"; then
        pass "$name"
    elif echo "$output" | grep -qi "$pattern"; then
        pass "$name"
    else
        fail "$name" "output missing '$pattern'. Got: $(echo "$output" | head -3)"
    fi
}

# Run a command, accept any of the listed exit codes
run_test_any_exit() {
    local name="$1"; shift
    local exits="$1"; shift  # comma-separated, e.g. "0,1,2"
    local pattern="$1"; shift

    local output
    local actual_exit=0
    output=$("$@" 2>&1) || actual_exit=$?

    local found=false
    IFS=',' read -ra codes <<< "$exits"
    for code in "${codes[@]}"; do
        if [ "$actual_exit" -eq "$code" ]; then
            found=true
            break
        fi
    done

    if [ "$found" = false ]; then
        fail "$name" "exit code $actual_exit (expected one of: $exits). Output: $(echo "$output" | head -3)"
        return
    fi

    if [ -z "$pattern" ]; then
        pass "$name"
        return
    fi

    if echo "$output" | grep -qF "$pattern"; then
        pass "$name"
    elif echo "$output" | grep -qi "$pattern"; then
        pass "$name"
    else
        fail "$name" "output missing '$pattern'. Got: $(echo "$output" | head -3)"
    fi
}

# ---------------------------------------------------------------------------
# Setup test workspace
# ---------------------------------------------------------------------------

WORKDIR=$(mktemp -d)
cd "$WORKDIR"
git init -q .
git config user.email "test@test.com"
git config user.name "Test User"

# Enable experimental features for agent/fleet
export PREFLIGHT_EXPERIMENTAL=1

printf "\n${CYAN}Preflight CLI Smoke Tests${NC}\n"
printf "Binary: %s\n" "$PREFLIGHT"
printf "Workdir: %s\n" "$WORKDIR"
$PREFLIGHT version 2>/dev/null || true

# =========================================================================
section "Core: version & help"
# =========================================================================

run_test "version" 0 "preflight" \
    $PREFLIGHT version

run_test "help" 0 "Usage" \
    $PREFLIGHT --help

run_test "help for plan" 0 "Usage" \
    $PREFLIGHT plan --help

run_test "help for apply" 0 "Usage" \
    $PREFLIGHT apply --help

run_test "unknown command help" 0 "Unknown help topic" \
    $PREFLIGHT help notarealcommand

# =========================================================================
section "Completion"
# =========================================================================

run_test_exit "completion bash" 0 \
    $PREFLIGHT completion bash

run_test_exit "completion zsh" 0 \
    $PREFLIGHT completion zsh

run_test_exit "completion fish" 0 \
    $PREFLIGHT completion fish

run_test_exit "completion powershell" 0 \
    $PREFLIGHT completion powershell

# =========================================================================
section "Init"
# =========================================================================

INIT_DIR=$(mktemp -d)
cd "$INIT_DIR"
git init -q .
git config user.email "test@test.com"
git config user.name "Test User"

run_test "init --preset minimal" 0 "preflight.yaml" \
    $PREFLIGHT init --preset minimal --non-interactive

if [ -f "$INIT_DIR/preflight.yaml" ]; then
    pass "init creates preflight.yaml"
else
    fail "init creates preflight.yaml" "file not found"
fi

if [ -d "$INIT_DIR/layers" ]; then
    pass "init creates layers directory"
else
    fail "init creates layers directory" "directory not found"
fi

# Stay in INIT_DIR for config-dependent tests
# =========================================================================
section "Validate"
# =========================================================================

run_test_exit "validate (valid config)" 0 \
    $PREFLIGHT validate

# Invalid config returns exit 2
BAD_DIR=$(mktemp -d)
echo "invalid: [yaml: {{broken" > "$BAD_DIR/preflight.yaml"
cd "$BAD_DIR"
run_test "validate (invalid config)" 2 "YAML" \
    $PREFLIGHT validate
cd "$INIT_DIR"

# =========================================================================
section "Plan"
# =========================================================================

run_test_exit "plan" 0 \
    $PREFLIGHT plan

run_test "plan --help" 0 "Usage" \
    $PREFLIGHT plan --help

# =========================================================================
section "Doctor"
# =========================================================================

# Doctor without --quiet needs TTY, so only test --quiet in Docker
run_test_exit "doctor --quiet" 0 \
    $PREFLIGHT doctor --quiet

# =========================================================================
section "Export"
# =========================================================================

run_test_exit "export" 0 \
    $PREFLIGHT export

run_test_exit "export --format json" 0 \
    $PREFLIGHT export --format json

run_test_exit "export --format yaml" 0 \
    $PREFLIGHT export --format yaml

# =========================================================================
section "Diff"
# =========================================================================

run_test_exit "diff" 0 \
    $PREFLIGHT diff

# =========================================================================
section "Outdated (requires brew)"
# =========================================================================

# Outdated exits 2 when brew unavailable — that's expected
run_test_any_exit "outdated" "0,1,2" "" \
    $PREFLIGHT outdated

run_test_any_exit "outdated --json" "0,1,2" "" \
    $PREFLIGHT outdated --json

# =========================================================================
section "Deprecated (requires brew)"
# =========================================================================

run_test_any_exit "deprecated" "0,1,2" "" \
    $PREFLIGHT deprecated

run_test_any_exit "deprecated --json" "0,1,2" "" \
    $PREFLIGHT deprecated --json

# =========================================================================
section "Security (requires scanners)"
# =========================================================================

run_test_any_exit "security" "0,1,2" "" \
    $PREFLIGHT security

run_test_any_exit "security --json" "0,1,2" "" \
    $PREFLIGHT security --json

# =========================================================================
section "Cleanup (requires brew)"
# =========================================================================

run_test_any_exit "cleanup --dry-run" "0,1,2" "" \
    $PREFLIGHT cleanup --dry-run

run_test_any_exit "cleanup --json --dry-run" "0,1,2" "" \
    $PREFLIGHT cleanup --json --dry-run

# =========================================================================
section "Clean"
# =========================================================================

run_test "clean --help" 0 "Usage" \
    $PREFLIGHT clean --help

run_test_exit "clean -y" 0 \
    $PREFLIGHT clean -y

# =========================================================================
section "Audit"
# =========================================================================

run_test_exit "audit" 0 \
    $PREFLIGHT audit

run_test_exit "audit --json" 0 \
    $PREFLIGHT audit --json

run_test_exit "audit show" 0 \
    $PREFLIGHT audit show

# =========================================================================
section "History"
# =========================================================================

run_test_exit "history" 0 \
    $PREFLIGHT history

run_test_exit "history --json" 0 \
    $PREFLIGHT history --json

run_test_exit "history clear --yes" 0 \
    $PREFLIGHT history clear --yes

# =========================================================================
section "Discover (requires gh CLI)"
# =========================================================================

# Discover needs gh CLI and network
if command -v gh >/dev/null 2>&1; then
    run_test_exit "discover" 0 \
        $PREFLIGHT discover
else
    skip "discover" "gh CLI not available"
fi

# =========================================================================
section "Tour (requires TTY)"
# =========================================================================

# Tour needs TTY, skip in Docker
skip "tour" "requires TTY (no /dev/tty in Docker)"

# =========================================================================
section "Profile"
# =========================================================================

run_test_exit "profile list" 0 \
    $PREFLIGHT profile list

run_test_exit "profile current" 0 \
    $PREFLIGHT profile current

# =========================================================================
section "Env"
# =========================================================================

run_test_exit "env list" 0 \
    $PREFLIGHT env list

run_test_exit "env set TEST_VAR hello" 0 \
    $PREFLIGHT env set TEST_VAR hello

run_test_exit "env export" 0 \
    $PREFLIGHT env export

run_test_exit "env unset TEST_VAR" 0 \
    $PREFLIGHT env unset TEST_VAR

# =========================================================================
section "Catalog"
# =========================================================================

run_test_exit "catalog list" 0 \
    $PREFLIGHT catalog list

# =========================================================================
section "Plugin"
# =========================================================================

run_test_exit "plugin list" 0 \
    $PREFLIGHT plugin list

# =========================================================================
section "Trust"
# =========================================================================

run_test_exit "trust list" 0 \
    $PREFLIGHT trust list

# =========================================================================
section "Lock"
# =========================================================================

run_test_exit "lock update" 0 \
    $PREFLIGHT lock update

# =========================================================================
section "Rollback"
# =========================================================================

run_test_exit "rollback (no snapshots)" 0 \
    $PREFLIGHT rollback

# =========================================================================
section "Secrets"
# =========================================================================

run_test_exit "secrets" 0 \
    $PREFLIGHT secrets

run_test_exit "secrets --json" 0 \
    $PREFLIGHT secrets --json

# =========================================================================
section "Compliance"
# =========================================================================

run_test_exit "compliance" 0 \
    $PREFLIGHT compliance

run_test_exit "compliance --json" 0 \
    $PREFLIGHT compliance --json

# =========================================================================
section "Compare"
# =========================================================================

# Compare needs two targets — may fail if config doesn't define them
run_test_any_exit "compare default default" "0,1" "" \
    $PREFLIGHT compare default default

# =========================================================================
section "Fleet (experimental)"
# =========================================================================

# Fleet needs inventory file - exit 1 without one is expected
run_test_any_exit "fleet list" "0,1" "" \
    $PREFLIGHT fleet list

run_test_any_exit "fleet list --json" "0,1" "" \
    $PREFLIGHT fleet list --json

# =========================================================================
section "Marketplace (requires network)"
# =========================================================================

# Marketplace needs network access to fetch index
run_test_any_exit "marketplace search test" "0,1" "" \
    $PREFLIGHT marketplace search test

# =========================================================================
section "Repo"
# =========================================================================

run_test_exit "repo status" 0 \
    $PREFLIGHT repo status

# =========================================================================
section "Analyze (no AI)"
# =========================================================================

run_test_exit "analyze --no-ai" 0 \
    $PREFLIGHT analyze --no-ai

run_test_exit "analyze --json --no-ai" 0 \
    $PREFLIGHT analyze --json --no-ai

# =========================================================================
section "Apply (dry-run)"
# =========================================================================

run_test_exit "apply --dry-run" 0 \
    $PREFLIGHT apply --dry-run

# =========================================================================
section "Agent (experimental)"
# =========================================================================

run_test_exit "agent status" 0 \
    $PREFLIGHT agent status

run_test "agent --help" 0 "Usage" \
    $PREFLIGHT agent --help

# =========================================================================
section "MCP"
# =========================================================================

run_test "mcp --help" 0 "Usage" \
    $PREFLIGHT mcp --help

# =========================================================================
section "Capture"
# =========================================================================

run_test "capture --help" 0 "Usage" \
    $PREFLIGHT capture --help

# =========================================================================
section "Watch"
# =========================================================================

run_test "watch --help" 0 "Usage" \
    $PREFLIGHT watch --help

# =========================================================================
section "Sync"
# =========================================================================

run_test "sync --help" 0 "Usage" \
    $PREFLIGHT sync --help

# =========================================================================
section "No config error handling"
# =========================================================================

EMPTY_DIR=$(mktemp -d)
cd "$EMPTY_DIR"
git init -q .
git config user.email "test@test.com"
git config user.name "Test User"

run_test_exit "plan without config" 1 \
    $PREFLIGHT plan

run_test_exit "validate without config" 2 \
    $PREFLIGHT validate

# Doctor --quiet without config should fail
run_test_any_exit "doctor --quiet without config" "1,2" "" \
    $PREFLIGHT doctor --quiet

# version always works
run_test "version without config" 0 "preflight" \
    $PREFLIGHT version

# =========================================================================
section "Flag validation"
# =========================================================================

cd "$INIT_DIR"

# Invalid mode
run_test "invalid --mode" 1 "" \
    $PREFLIGHT plan --mode invalid_mode

# Verbose flag
run_test_exit "plan --verbose" 0 \
    $PREFLIGHT plan --verbose

# =========================================================================
# Summary
# =========================================================================

printf "\n${CYAN}=====================================${NC}\n"
printf "${CYAN}  Test Summary${NC}\n"
printf "${CYAN}=====================================${NC}\n"
printf "  ${GREEN}Passed:  %d${NC}\n" "$PASSED"
printf "  ${RED}Failed:  %d${NC}\n" "$FAILED"
printf "  ${YELLOW}Skipped: %d${NC}\n" "$SKIPPED"
printf "  Total:   %d\n" "$((PASSED + FAILED + SKIPPED))"

if [ "$FAILED" -gt 0 ]; then
    printf "\n${RED}Failures:${NC}"
    printf "$FAILURES\n"
    exit 1
fi

printf "\n${GREEN}All tests passed!${NC}\n"

# Cleanup
rm -rf "$WORKDIR" "$INIT_DIR" "$BAD_DIR" "$EMPTY_DIR" 2>/dev/null || true
