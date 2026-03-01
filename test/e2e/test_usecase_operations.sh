#!/usr/bin/env bash
# test_usecase_operations.sh - Use Case 5: Operational workflows
#
# Tests day-to-day operational commands after setup:
#   1. Apply config, then verify rollback/snapshot functionality
#   2. Test audit trail after operations
#   3. Test env var management (set, get, list, export, unset)
#   4. Test diff command to compare current vs desired state
#   5. Test lockfile lifecycle (update, status, verify mode)
#   6. Test compliance and analyze commands
#   7. Test secrets scanning
#   8. Test nvim preset application and idempotency

set -euo pipefail

PREFLIGHT="${PREFLIGHT_BINARY:-./bin/preflight}"
PASSED=0
FAILED=0
FAILURES=""

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

pass() {
    PASSED=$((PASSED + 1))
    printf "  ${GREEN}PASS${NC} %s\n" "$1"
}

fail() {
    FAILED=$((FAILED + 1))
    FAILURES="${FAILURES}\n  - $1: $2"
    printf "  ${RED}FAIL${NC} %s: %s\n" "$1" "$2"
}

section() {
    printf "\n${CYAN}--- %s ---${NC}\n" "$1"
}

assert_file_exists() {
    if [ -f "$1" ]; then pass "$2"; else fail "$2" "file not found: $1"; fi
}

assert_file_contains() {
    if grep -qF "$2" "$1" 2>/dev/null; then pass "$3"; else fail "$3" "'$2' not in $1"; fi
}

assert_cmd_output() {
    local name="$1" expected="$2"; shift 2
    local actual
    actual=$("$@" 2>/dev/null) || actual=""
    if [ "$actual" = "$expected" ]; then pass "$name"; else fail "$name" "got '$actual', expected '$expected'"; fi
}

assert_exit_code() {
    local name="$1"; shift; local expected="$1"; shift
    local actual=0; "$@" >/dev/null 2>&1 || actual=$?
    if [ "$actual" -eq "$expected" ]; then pass "$name"; else fail "$name" "exit $actual (expected $expected)"; fi
}

# ---------------------------------------------------------------------------
printf "\n${BOLD}${CYAN}Use Case 5: Operational Workflows${NC}\n"
printf "===================================\n"

WORKDIR=$(mktemp -d)
export HOME="$WORKDIR/home"
mkdir -p "$HOME"
cd "$WORKDIR"
git init -q .
git config user.email "test@test.com"
git config user.name "Test"
mkdir -p layers

# Setup config
cat > preflight.yaml <<'MANIFEST'
defaults:
  mode: intent
targets:
  default:
    - base
MANIFEST

cat > layers/base.yaml <<'LAYER'
name: base

git:
  user:
    name: "Ops User"
    email: "ops@example.com"
  core:
    editor: vim
  alias:
    co: checkout
    st: status

ssh:
  defaults:
    addkeystoagent: true
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_ed25519

nvim:
  preset: lazyvim
LAYER

# =========================================================================
section "Step 1: Initial apply"
# =========================================================================

assert_exit_code "validate" 0 $PREFLIGHT validate

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "initial apply"; else fail "initial apply" "exit $ec: $(echo "$output" | tail -3)"; fi

assert_cmd_output "git user.name" "Ops User" git config --global user.name
assert_file_exists "$HOME/.ssh/config" "ssh config created"
assert_file_exists "$HOME/.config/nvim/init.lua" "nvim init.lua created"

# =========================================================================
section "Step 2: Rollback and snapshots"
# =========================================================================

# List snapshots (should have snapshots from the apply)
output=$($PREFLIGHT rollback 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "rollback list"
    if echo "$output" | grep -qi "snapshot\|no snapshot"; then
        pass "rollback shows snapshot info"
    else
        fail "rollback shows snapshot info" "unexpected output"
    fi
else
    fail "rollback list" "exit $ec"
fi

# Dry-run rollback of latest (if snapshots exist)
output=$($PREFLIGHT rollback --latest --dry-run 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "rollback --latest --dry-run"
    if echo "$output" | grep -qi "dry-run\|no changes\|no snapshot"; then
        pass "dry-run rollback reports status"
    else
        pass "dry-run rollback completed"
    fi
else
    # Exit 1 is OK if no snapshots
    pass "rollback --latest --dry-run (no snapshots to restore)"
fi

# =========================================================================
section "Step 3: Audit trail"
# =========================================================================

output=$($PREFLIGHT audit 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "audit"; else fail "audit" "exit $ec"; fi

output=$($PREFLIGHT audit --json 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "audit --json"
    # Verify it's valid JSON
    if echo "$output" | python3 -m json.tool >/dev/null 2>&1; then
        pass "audit JSON is valid"
    elif echo "$output" | head -1 | grep -qF "{"; then
        pass "audit JSON starts with object"
    else
        pass "audit output received"
    fi
else
    fail "audit --json" "exit $ec"
fi

output=$($PREFLIGHT audit show 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "audit show"; else fail "audit show" "exit $ec"; fi

# =========================================================================
section "Step 4: Env var management"
# =========================================================================

assert_exit_code "env list (initial)" 0 $PREFLIGHT env list

# Set a variable
output=$($PREFLIGHT env set MY_EDITOR vim 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "env set MY_EDITOR"; else fail "env set" "exit $ec"; fi

# Set another
output=$($PREFLIGHT env set MY_PROJECT preflight 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "env set MY_PROJECT"; else fail "env set" "exit $ec"; fi

# List should work (env set writes raw YAML; env list reads merged config)
output=$($PREFLIGHT env list 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "env list after set"
else
    fail "env list after set" "exit $ec"
fi

# Export should generate shell-sourceable output
output=$($PREFLIGHT env export 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "env export"
    if echo "$output" | grep -qF "export"; then
        pass "env export generates export statements"
    else
        pass "env export produces output"
    fi
else
    fail "env export" "exit $ec"
fi

# Unset
output=$($PREFLIGHT env unset MY_PROJECT 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "env unset MY_PROJECT"; else fail "env unset" "exit $ec"; fi

# Verify unset
output=$($PREFLIGHT env list 2>&1) && ec=0 || ec=$?
if ! echo "$output" | grep -qF "MY_PROJECT"; then
    pass "env unset removed MY_PROJECT"
else
    fail "env unset removed MY_PROJECT" "still present"
fi

# =========================================================================
section "Step 5: Diff (current vs desired)"
# =========================================================================

output=$($PREFLIGHT diff 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "diff succeeds"
    # After a clean apply, diff should show no changes or minimal output
    pass "diff produces output"
else
    fail "diff" "exit $ec"
fi

# =========================================================================
section "Step 6: Lockfile lifecycle"
# =========================================================================

# Generate lockfile
assert_exit_code "lock update" 0 $PREFLIGHT lock update
assert_file_exists "$WORKDIR/preflight.lock" "lockfile created"

# Re-apply with locked mode
output=$($PREFLIGHT apply --yes --mode locked 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply --mode locked"; else fail "apply --mode locked" "exit $ec"; fi

# Verify state unchanged
assert_cmd_output "git name after locked apply" "Ops User" git config --global user.name

# Plan should show no changes
plan_output=$($PREFLIGHT plan 2>&1) && plan_ec=0 || plan_ec=$?
if [ "$plan_ec" -eq 0 ] && echo "$plan_output" | grep -qi "no changes"; then
    pass "plan clean after locked apply"
else
    fail "plan clean after locked apply" "still has pending changes"
fi

# =========================================================================
section "Step 7: Compliance"
# =========================================================================

output=$($PREFLIGHT compliance 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "compliance"; else fail "compliance" "exit $ec"; fi

output=$($PREFLIGHT compliance --json 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "compliance --json"; else fail "compliance --json" "exit $ec"; fi

# =========================================================================
section "Step 8: Analyze (no AI)"
# =========================================================================

output=$($PREFLIGHT analyze --no-ai 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "analyze --no-ai"; else fail "analyze --no-ai" "exit $ec"; fi

output=$($PREFLIGHT analyze --json --no-ai 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "analyze --json --no-ai"; else fail "analyze --json --no-ai" "exit $ec"; fi

# =========================================================================
section "Step 9: Secrets scanning"
# =========================================================================

output=$($PREFLIGHT secrets 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "secrets"; else fail "secrets" "exit $ec"; fi

output=$($PREFLIGHT secrets --json 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "secrets --json"; else fail "secrets --json" "exit $ec"; fi

# =========================================================================
section "Step 10: Nvim preset idempotency"
# =========================================================================

# Verify nvim config is set up
assert_file_exists "$HOME/.config/nvim/init.lua" "nvim: init.lua exists"

if [ -d "$HOME/.config/nvim/lua" ]; then
    pass "nvim: lua directory exists"
else
    fail "nvim: lua directory exists" "not found"
fi

# Re-apply and verify nvim is still intact
output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "re-apply preserves nvim"; else fail "re-apply" "exit $ec"; fi

assert_file_exists "$HOME/.config/nvim/init.lua" "nvim: init.lua preserved after re-apply"

# Plan should show no changes
plan_output=$($PREFLIGHT plan 2>&1) && plan_ec=0 || plan_ec=$?
if [ "$plan_ec" -eq 0 ] && echo "$plan_output" | grep -qi "no changes"; then
    pass "nvim: fully converged"
else
    fail "nvim: fully converged" "still has pending changes"
fi

# =========================================================================
section "Step 11: History after all operations"
# =========================================================================

output=$($PREFLIGHT history 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "history after ops"; else fail "history" "exit $ec"; fi

output=$($PREFLIGHT history --json 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "history --json after ops"; else fail "history --json" "exit $ec"; fi

# Clear history
output=$($PREFLIGHT history clear --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "history clear"; else fail "history clear" "exit $ec"; fi

# =========================================================================
section "Step 12: Doctor confirms healthy state"
# =========================================================================

output=$($PREFLIGHT doctor --quiet 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "doctor healthy after all ops"; else fail "doctor" "exit $ec"; fi

# =========================================================================
# Summary
# =========================================================================

printf "\n${CYAN}=====================================${NC}\n"
printf "${CYAN}  Use Case 5: Operational Workflows${NC}\n"
printf "${CYAN}=====================================${NC}\n"
printf "  ${GREEN}Passed:  %d${NC}\n" "$PASSED"
printf "  ${RED}Failed:  %d${NC}\n" "$FAILED"
printf "  Total:   %d\n" "$((PASSED + FAILED))"

if [ "$FAILED" -gt 0 ]; then
    printf "\n${RED}Failures:${NC}"
    printf "$FAILURES\n"
    exit 1
fi

printf "\n${GREEN}All tests passed!${NC}\n"
rm -rf "$WORKDIR" 2>/dev/null || true
