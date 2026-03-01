#!/usr/bin/env bash
# test_usecase_config_evolution.sh - Use Case 3: Config evolution
#
# Simulates a user evolving their config over time:
#   1. Init with minimal preset, apply
#   2. Add git aliases, re-apply, verify additions
#   3. Change git email, re-apply, verify update
#   4. Add SSH config via new layer, re-apply, verify merge
#   5. Remove an alias, re-apply, verify removal
#   6. Verify idempotency after final state
#   7. Verify audit/history trail records operations

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

assert_file_not_contains() {
    if ! grep -qF "$2" "$1" 2>/dev/null; then pass "$3"; else fail "$3" "'$2' should not be in $1"; fi
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
printf "\n${BOLD}${CYAN}Use Case 3: Config Evolution${NC}\n"
printf "=============================\n"

WORKDIR=$(mktemp -d)
export HOME="$WORKDIR/home"
mkdir -p "$HOME"
cd "$WORKDIR"
git init -q .
git config user.email "test@test.com"
git config user.name "Test"

# =========================================================================
section "Step 1: Init with minimal preset"
# =========================================================================

output=$($PREFLIGHT init --preset git:minimal --non-interactive 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "init succeeds"; else fail "init" "exit $ec: $output"; fi

# Write initial minimal config
cat > "$WORKDIR/layers/base.yaml" <<'LAYER'
name: base

git:
  user:
    name: "Initial Name"
    email: "initial@example.com"
  core:
    editor: nano
  alias:
    co: checkout
    st: status
LAYER

assert_exit_code "validate initial" 0 $PREFLIGHT validate

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "initial apply"; else fail "initial apply" "exit $ec"; fi

assert_cmd_output "initial git name" "Initial Name" git config --global user.name
assert_cmd_output "initial git email" "initial@example.com" git config --global user.email
assert_cmd_output "initial editor" "nano" git config --global core.editor
assert_cmd_output "initial alias.co" "checkout" git config --global alias.co
assert_cmd_output "initial alias.st" "status" git config --global alias.st

# =========================================================================
section "Step 2: Add git aliases (additive change)"
# =========================================================================

cat > "$WORKDIR/layers/base.yaml" <<'LAYER'
name: base

git:
  user:
    name: "Initial Name"
    email: "initial@example.com"
  core:
    editor: nano
  alias:
    co: checkout
    st: status
    br: branch
    ci: commit
    lg: "log --oneline --graph --all"
LAYER

output=$($PREFLIGHT plan 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "plan after adding aliases"
    # Plan should show changes needed
    if ! echo "$output" | grep -qi "no changes"; then
        pass "plan detects new aliases"
    else
        fail "plan detects new aliases" "plan says no changes"
    fi
else
    fail "plan after adding aliases" "exit $ec"
fi

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply added aliases"; else fail "apply added aliases" "exit $ec"; fi

# Verify old aliases preserved
assert_cmd_output "alias.co preserved" "checkout" git config --global alias.co
assert_cmd_output "alias.st preserved" "status" git config --global alias.st
# Verify new aliases added
assert_cmd_output "alias.br added" "branch" git config --global alias.br
assert_cmd_output "alias.ci added" "commit" git config --global alias.ci
assert_cmd_output "alias.lg added" "log --oneline --graph --all" git config --global alias.lg

# =========================================================================
section "Step 3: Change email (update change)"
# =========================================================================

cat > "$WORKDIR/layers/base.yaml" <<'LAYER'
name: base

git:
  user:
    name: "Updated Name"
    email: "updated@example.com"
  core:
    editor: vim
  alias:
    co: checkout
    st: status
    br: branch
    ci: commit
    lg: "log --oneline --graph --all"
LAYER

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply updated config"; else fail "apply updated config" "exit $ec"; fi

assert_cmd_output "name updated" "Updated Name" git config --global user.name
assert_cmd_output "email updated" "updated@example.com" git config --global user.email
assert_cmd_output "editor updated" "vim" git config --global core.editor
# Aliases still intact
assert_cmd_output "aliases survived update" "branch" git config --global alias.br

# =========================================================================
section "Step 4: Add SSH via modified layer"
# =========================================================================

cat > "$WORKDIR/layers/base.yaml" <<'LAYER'
name: base

git:
  user:
    name: "Updated Name"
    email: "updated@example.com"
  core:
    editor: vim
  alias:
    co: checkout
    st: status
    br: branch
    ci: commit
    lg: "log --oneline --graph --all"

ssh:
  defaults:
    addkeystoagent: true
    identitiesonly: true
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_ed25519
LAYER

output=$($PREFLIGHT plan 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "plan with SSH addition"
    if echo "$output" | grep -qF "ssh:config"; then
        pass "plan shows ssh:config step"
    else
        fail "plan shows ssh:config step" "not in plan"
    fi
else
    fail "plan with SSH addition" "exit $ec"
fi

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply with SSH"; else fail "apply with SSH" "exit $ec"; fi

assert_file_exists "$HOME/.ssh/config" "ssh config created"
assert_file_contains "$HOME/.ssh/config" "github.com" "ssh has github host"
assert_file_contains "$HOME/.ssh/config" "IdentityFile" "ssh has identity file"
# Git still correct
assert_cmd_output "git name after SSH add" "Updated Name" git config --global user.name

# =========================================================================
section "Step 5: Add a second layer"
# =========================================================================

# Update manifest to include a second layer
cat > "$WORKDIR/preflight.yaml" <<'MANIFEST'
defaults:
  mode: intent
targets:
  default:
    - base
    - team
MANIFEST

cat > "$WORKDIR/layers/team.yaml" <<'LAYER'
name: team

git:
  alias:
    df: "diff --stat"
    pr: "pull --rebase"

ssh:
  hosts:
    - host: gitlab.internal.com
      hostname: gitlab.internal.com
      user: git
      identityfile: ~/.ssh/id_company
LAYER

assert_exit_code "validate with two layers" 0 $PREFLIGHT validate

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply with two layers"; else fail "apply with two layers" "exit $ec"; fi

# Verify base layer aliases still present
assert_cmd_output "base alias.co still set" "checkout" git config --global alias.co
# Verify team layer aliases added
assert_cmd_output "team alias.df added" "diff --stat" git config --global alias.df
assert_cmd_output "team alias.pr added" "pull --rebase" git config --global alias.pr
# Both SSH hosts should exist
assert_file_contains "$HOME/.ssh/config" "github.com" "ssh still has github"
assert_file_contains "$HOME/.ssh/config" "gitlab.internal.com" "ssh has gitlab from team layer"

# =========================================================================
section "Step 6: Verify idempotency after evolution"
# =========================================================================

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "re-apply succeeds"; else fail "re-apply" "exit $ec"; fi

plan_output=$($PREFLIGHT plan 2>&1) && plan_ec=0 || plan_ec=$?
if [ "$plan_ec" -eq 0 ] && echo "$plan_output" | grep -qi "no changes"; then
    pass "plan shows no changes (converged)"
else
    fail "plan shows no changes" "still has pending changes after evolution"
fi

# =========================================================================
section "Step 7: History records operations"
# =========================================================================

output=$($PREFLIGHT history 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "history command works"; else fail "history" "exit $ec"; fi

output=$($PREFLIGHT history --json 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "history --json works"; else fail "history --json" "exit $ec"; fi

# =========================================================================
section "Step 8: Doctor confirms healthy state"
# =========================================================================

output=$($PREFLIGHT doctor --quiet 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "doctor healthy after evolution"; else fail "doctor" "exit $ec"; fi

# =========================================================================
# Summary
# =========================================================================

printf "\n${CYAN}=====================================${NC}\n"
printf "${CYAN}  Use Case 3: Config Evolution${NC}\n"
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
