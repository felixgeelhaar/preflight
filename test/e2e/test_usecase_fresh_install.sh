#!/usr/bin/env bash
# test_usecase_fresh_install.sh - Use Case 1: Fresh install with wizard
#
# Simulates a user setting up a fresh workstation:
#   1. Run preflight init with a preset
#   2. Customize the layer with git/ssh config
#   3. Plan and review changes
#   4. Apply configuration
#   5. Verify with doctor
#   6. Re-apply to verify idempotency
#   7. Generate lockfile

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
printf "\n${BOLD}${CYAN}Use Case 1: Fresh Install with Wizard${NC}\n"
printf "======================================\n"

# Setup isolated home
WORKDIR=$(mktemp -d)
export HOME="$WORKDIR/home"
mkdir -p "$HOME"
cd "$WORKDIR"
git init -q .
git config user.email "test@test.com"
git config user.name "Test"

# =========================================================================
section "Step 1: Init with preset"
# =========================================================================

output=$($PREFLIGHT init --preset git:minimal --non-interactive 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "preflight init"; else fail "preflight init" "exit $ec: $output"; fi

assert_file_exists "$WORKDIR/preflight.yaml" "manifest created"
assert_file_exists "$WORKDIR/layers/base.yaml" "base layer created"
assert_file_contains "$WORKDIR/preflight.yaml" "intent" "manifest has intent mode"

# =========================================================================
section "Step 2: Enrich config with git + ssh"
# =========================================================================

cat > "$WORKDIR/layers/base.yaml" <<'LAYER'
name: base

git:
  user:
    name: "Fresh Dev"
    email: "freshdev@example.com"
  core:
    editor: vim
    autocrlf: input
  alias:
    co: checkout
    br: branch
    ci: commit
    st: status
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
    - host: gitlab.com
      hostname: gitlab.com
      user: git
      identityfile: ~/.ssh/id_gitlab
LAYER

pass "enriched base layer"

# =========================================================================
section "Step 3: Validate"
# =========================================================================

assert_exit_code "validate" 0 $PREFLIGHT validate

# =========================================================================
section "Step 4: Plan"
# =========================================================================

output=$($PREFLIGHT plan 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "plan succeeds"
    if echo "$output" | grep -qF "git:config"; then
        pass "plan includes git:config step"
    else
        fail "plan includes git:config step" "not found in plan output"
    fi
    if echo "$output" | grep -qF "ssh:config"; then
        pass "plan includes ssh:config step"
    else
        fail "plan includes ssh:config step" "not found in plan output"
    fi
else
    fail "plan" "exit $ec"
fi

# =========================================================================
section "Step 5: Apply"
# =========================================================================

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "apply succeeds"
    if echo "$output" | grep -qF "succeeded"; then
        pass "apply reports success"
    fi
else
    fail "apply" "exit $ec: $(echo "$output" | tail -3)"
fi

# =========================================================================
section "Step 6: Verify applied state"
# =========================================================================

# Git config
assert_cmd_output "git user.name" "Fresh Dev" git config --global user.name
assert_cmd_output "git user.email" "freshdev@example.com" git config --global user.email
assert_cmd_output "git core.editor" "vim" git config --global core.editor
assert_cmd_output "git alias.co" "checkout" git config --global alias.co
assert_cmd_output "git alias.st" "status" git config --global alias.st

# SSH config
assert_file_exists "$HOME/.ssh/config" "ssh config created"
assert_file_contains "$HOME/.ssh/config" "github.com" "ssh has github.com"
assert_file_contains "$HOME/.ssh/config" "gitlab.com" "ssh has gitlab.com"
assert_file_contains "$HOME/.ssh/config" "IdentityFile" "ssh has identity file"

# =========================================================================
section "Step 7: Doctor"
# =========================================================================

output=$($PREFLIGHT doctor --quiet 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "doctor reports no issues"
else
    fail "doctor" "exit $ec: $(echo "$output" | tail -3)"
fi

# =========================================================================
section "Step 8: Idempotency"
# =========================================================================

# Capture state
state_before=$(git config --global --list 2>/dev/null | sort)
ssh_md5_before=$(md5sum "$HOME/.ssh/config" 2>/dev/null | cut -d' ' -f1)

# Re-apply
output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "re-apply succeeds"
else
    fail "re-apply" "exit $ec"
fi

# Verify state unchanged
state_after=$(git config --global --list 2>/dev/null | sort)
ssh_md5_after=$(md5sum "$HOME/.ssh/config" 2>/dev/null | cut -d' ' -f1)

if [ "$state_before" = "$state_after" ]; then
    pass "git config unchanged after re-apply"
else
    fail "git config unchanged" "state changed"
fi

if [ "$ssh_md5_before" = "$ssh_md5_after" ]; then
    pass "ssh config unchanged after re-apply"
else
    fail "ssh config unchanged" "content changed"
fi

# =========================================================================
section "Step 9: Lockfile"
# =========================================================================

assert_exit_code "lock update" 0 $PREFLIGHT lock update
assert_file_exists "$WORKDIR/preflight.lock" "lockfile created"

# =========================================================================
# Summary
# =========================================================================

printf "\n${CYAN}=====================================${NC}\n"
printf "${CYAN}  Use Case 1: Fresh Install${NC}\n"
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
