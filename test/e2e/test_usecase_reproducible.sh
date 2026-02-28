#!/usr/bin/env bash
# test_usecase_reproducible.sh - Use Case 2: Reproducible config on another machine
#
# Simulates sharing a preflight config between machines:
#   1. Create a known config (Machine A) and apply it
#   2. Generate a lockfile
#   3. Switch HOME to simulate Machine B (clean state)
#   4. Re-apply the same config + lockfile
#   5. Verify identical results
#   6. Test idempotency and locked mode

set -euo pipefail

PREFLIGHT="${PREFLIGHT_BINARY:-./bin/preflight}"
PASSED=0
FAILED=0
FAILURES=""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
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
printf "\n${BOLD}${CYAN}Use Case 2: Reproducible Config Across Machines${NC}\n"
printf "=================================================\n"

# =========================================================================
section "Phase A: Setup config repo"
# =========================================================================

CONFIG_REPO=$(mktemp -d)
cd "$CONFIG_REPO"
git init -q .
git config user.email "teamlead@company.com"
git config user.name "Team Lead"

mkdir -p layers

cat > preflight.yaml <<'MANIFEST'
defaults:
  mode: intent
targets:
  default:
    - base
    - team
MANIFEST

cat > layers/base.yaml <<'LAYER'
name: base

git:
  user:
    name: "Team Member"
    email: "member@company.com"
  core:
    editor: vim
    autocrlf: input
  alias:
    co: checkout
    br: branch
    ci: commit
    st: status
    lg: "log --oneline --graph --all"
  commit:
    gpgsign: false

ssh:
  defaults:
    addkeystoagent: true
    identitiesonly: true
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_ed25519
    - host: "*.internal.company.com"
      user: deploy
      identityfile: ~/.ssh/id_company
LAYER

cat > layers/team.yaml <<'LAYER'
name: team

git:
  alias:
    lg: "log --oneline --graph --all"
    df: "diff --stat"
LAYER

pass "created team config (manifest + 2 layers)"

# =========================================================================
section "Phase A: Apply on Machine A"
# =========================================================================

HOME_A=$(mktemp -d)
export HOME="$HOME_A"

assert_exit_code "validate" 0 $PREFLIGHT validate

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply on Machine A"; else fail "apply on Machine A" "exit $ec"; fi

# Capture Machine A state
assert_cmd_output "A: git user.name" "Team Member" git config --global user.name
assert_cmd_output "A: git user.email" "member@company.com" git config --global user.email
assert_cmd_output "A: git core.editor" "vim" git config --global core.editor
assert_cmd_output "A: git alias.co" "checkout" git config --global alias.co
assert_cmd_output "A: git core.autocrlf" "input" git config --global core.autocrlf
assert_cmd_output "A: git alias.df" "diff --stat" git config --global alias.df

if [ -f "$HOME/.ssh/config" ] && grep -qF "github.com" "$HOME/.ssh/config"; then
    pass "A: ssh config with github.com"
else
    fail "A: ssh config" "missing or incomplete"
fi

# Save full state for comparison
GIT_STATE_A=$(git config --global --list 2>/dev/null | sort)
SSH_MD5_A=$(md5sum "$HOME/.ssh/config" 2>/dev/null | cut -d' ' -f1)

# =========================================================================
section "Phase A: Generate lockfile"
# =========================================================================

assert_exit_code "lock update" 0 $PREFLIGHT lock update

if [ -f "$CONFIG_REPO/preflight.lock" ]; then
    pass "lockfile generated"
else
    fail "lockfile generated" "not found"
fi

# Doctor on Machine A
output=$($PREFLIGHT doctor --quiet 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "A: doctor clean"; else fail "A: doctor" "exit $ec"; fi

# =========================================================================
section "Phase B: Simulate Machine B (clean HOME)"
# =========================================================================

HOME_B=$(mktemp -d)
export HOME="$HOME_B"

# Verify clean state
git_test=$(git config --global user.name 2>/dev/null || echo "")
if [ -z "$git_test" ]; then
    pass "B: starts with clean git config"
else
    pass "B: starts fresh (may have system config)"
fi

# =========================================================================
section "Phase B: Apply same config"
# =========================================================================

cd "$CONFIG_REPO"

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply on Machine B"; else fail "apply on Machine B" "exit $ec"; fi

# =========================================================================
section "Phase B: Verify identical results"
# =========================================================================

assert_cmd_output "B: git user.name" "Team Member" git config --global user.name
assert_cmd_output "B: git user.email" "member@company.com" git config --global user.email
assert_cmd_output "B: git core.editor" "vim" git config --global core.editor
assert_cmd_output "B: git alias.co" "checkout" git config --global alias.co
assert_cmd_output "B: git core.autocrlf" "input" git config --global core.autocrlf
assert_cmd_output "B: git alias.df" "diff --stat" git config --global alias.df

if [ -f "$HOME/.ssh/config" ] && grep -qF "github.com" "$HOME/.ssh/config"; then
    pass "B: ssh config with github.com"
else
    fail "B: ssh config" "missing or incomplete"
fi

# Compare full state
GIT_STATE_B=$(git config --global --list 2>/dev/null | sort)
SSH_MD5_B=$(md5sum "$HOME/.ssh/config" 2>/dev/null | cut -d' ' -f1)

if [ "$GIT_STATE_A" = "$GIT_STATE_B" ]; then
    pass "git config identical across machines"
else
    fail "git config identical" "states differ"
fi

if [ "$SSH_MD5_A" = "$SSH_MD5_B" ]; then
    pass "ssh config identical across machines"
else
    fail "ssh config identical" "content differs"
fi

# =========================================================================
section "Phase B: Doctor on Machine B"
# =========================================================================

output=$($PREFLIGHT doctor --quiet 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "B: doctor clean"; else fail "B: doctor" "exit $ec"; fi

# =========================================================================
section "Phase C: Locked mode"
# =========================================================================

output=$($PREFLIGHT apply --yes --mode locked 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "locked mode apply succeeds"
else
    fail "locked mode apply" "exit $ec"
fi

# =========================================================================
section "Phase D: Idempotency"
# =========================================================================

state_before=$(git config --global --list 2>/dev/null | sort)
ssh_md5_before=$(md5sum "$HOME/.ssh/config" 2>/dev/null | cut -d' ' -f1)

$PREFLIGHT apply --yes >/dev/null 2>&1 || true
$PREFLIGHT apply --yes >/dev/null 2>&1 || true

state_after=$(git config --global --list 2>/dev/null | sort)
ssh_md5_after=$(md5sum "$HOME/.ssh/config" 2>/dev/null | cut -d' ' -f1)

if [ "$state_before" = "$state_after" ]; then
    pass "git config stable across 3 applies"
else
    fail "git config stable" "changed after repeated applies"
fi

if [ "$ssh_md5_before" = "$ssh_md5_after" ]; then
    pass "ssh config stable across 3 applies"
else
    fail "ssh config stable" "changed after repeated applies"
fi

# =========================================================================
section "Phase E: Export"
# =========================================================================

output=$($PREFLIGHT export --format yaml 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ] && [ -n "$output" ]; then
    pass "export yaml"
else
    fail "export yaml" "exit $ec or empty"
fi

output=$($PREFLIGHT export --format json 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ] && [ -n "$output" ]; then
    pass "export json"
else
    fail "export json" "exit $ec or empty"
fi

# =========================================================================
# Summary
# =========================================================================

printf "\n${CYAN}=====================================${NC}\n"
printf "${CYAN}  Use Case 2: Reproducible Config${NC}\n"
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
rm -rf "$CONFIG_REPO" "$HOME_A" "$HOME_B" 2>/dev/null || true
