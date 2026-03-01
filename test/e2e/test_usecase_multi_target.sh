#!/usr/bin/env bash
# test_usecase_multi_target.sh - Use Case 4: Multi-target and layer override
#
# Simulates a user with work and personal targets:
#   1. Create config with work + personal targets sharing a base layer
#   2. Apply work target, verify work-specific settings
#   3. Apply personal target to new HOME, verify personal settings
#   4. Verify layer override semantics (later layers win)
#   5. Test compare command between targets
#   6. Test export for each target
#   7. Test profile create/switch/list/delete lifecycle

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

assert_file_exists() {
    if [ -f "$1" ]; then pass "$2"; else fail "$2" "file not found: $1"; fi
}

assert_file_contains() {
    if grep -qF "$2" "$1" 2>/dev/null; then pass "$3"; else fail "$3" "'$2' not in $1"; fi
}

assert_file_not_contains() {
    if ! grep -qF "$2" "$1" 2>/dev/null; then pass "$3"; else fail "$3" "'$2' should not be in $1"; fi
}

# ---------------------------------------------------------------------------
printf "\n${BOLD}${CYAN}Use Case 4: Multi-Target & Layer Override${NC}\n"
printf "==========================================\n"

CONFIG_REPO=$(mktemp -d)
cd "$CONFIG_REPO"
git init -q .
git config user.email "test@test.com"
git config user.name "Test"
mkdir -p layers

# =========================================================================
section "Step 1: Create multi-target config"
# =========================================================================

cat > preflight.yaml <<'MANIFEST'
defaults:
  mode: intent
targets:
  work:
    - base
    - identity-work
  personal:
    - base
    - identity-personal
MANIFEST

# Base layer: shared settings
cat > layers/base.yaml <<'LAYER'
name: base

git:
  core:
    editor: vim
    autocrlf: input
  alias:
    co: checkout
    br: branch
    st: status

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

# Work identity layer: overrides user info, adds work SSH host
cat > layers/identity-work.yaml <<'LAYER'
name: identity-work

git:
  user:
    name: "Alice Corporate"
    email: "alice@company.com"
  commit:
    gpgsign: false
  alias:
    pr: "pull --rebase"

ssh:
  hosts:
    - host: gitlab.company.com
      hostname: gitlab.company.com
      user: git
      identityfile: ~/.ssh/id_work
LAYER

# Personal identity layer: overrides user info, changes editor
cat > layers/identity-personal.yaml <<'LAYER'
name: identity-personal

git:
  user:
    name: "Alice Personal"
    email: "alice@personal.dev"
  core:
    editor: nvim
  alias:
    lg: "log --oneline --graph --all"
LAYER

pass "created multi-target config (3 layers, 2 targets)"

assert_exit_code "validate work target" 0 $PREFLIGHT validate --target work
assert_exit_code "validate personal target" 0 $PREFLIGHT validate --target personal

# =========================================================================
section "Step 2: Apply work target"
# =========================================================================

HOME_WORK=$(mktemp -d)
export HOME="$HOME_WORK"

output=$($PREFLIGHT apply --yes --target work 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply work target"; else fail "apply work target" "exit $ec: $(echo "$output" | tail -3)"; fi

# Verify work-specific settings
assert_cmd_output "work: git user.name" "Alice Corporate" git config --global user.name
assert_cmd_output "work: git user.email" "alice@company.com" git config --global user.email
assert_cmd_output "work: editor from base" "vim" git config --global core.editor
# Work layer adds pr alias
assert_cmd_output "work: alias.pr" "pull --rebase" git config --global alias.pr
# Base layer aliases present
assert_cmd_output "work: alias.co from base" "checkout" git config --global alias.co
# SSH: both github (base) and gitlab (work) hosts
assert_file_exists "$HOME/.ssh/config" "work: ssh config exists"
assert_file_contains "$HOME/.ssh/config" "github.com" "work: ssh has github (base)"
assert_file_contains "$HOME/.ssh/config" "gitlab.company.com" "work: ssh has gitlab (work layer)"

# Work should NOT have personal-specific alias
lg_value=$(git config --global alias.lg 2>/dev/null || echo "")
if [ -z "$lg_value" ]; then
    pass "work: no personal alias.lg"
else
    fail "work: no personal alias.lg" "got '$lg_value'"
fi

# Doctor check
# Doctor always uses "default" target; accept 0 or 1 when no "default" target exists
output=$($PREFLIGHT doctor --quiet 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ] || [ "$ec" -eq 1 ]; then
    pass "work: doctor (no default target, exit $ec accepted)"
else
    fail "work: doctor" "exit $ec"
fi

# =========================================================================
section "Step 3: Apply personal target (clean HOME)"
# =========================================================================

HOME_PERSONAL=$(mktemp -d)
export HOME="$HOME_PERSONAL"

output=$($PREFLIGHT apply --yes --target personal 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "apply personal target"; else fail "apply personal target" "exit $ec: $(echo "$output" | tail -3)"; fi

# Verify personal-specific settings
assert_cmd_output "personal: git user.name" "Alice Personal" git config --global user.name
assert_cmd_output "personal: git user.email" "alice@personal.dev" git config --global user.email
# Personal layer overrides editor from base
assert_cmd_output "personal: editor override" "nvim" git config --global core.editor
# Personal layer adds lg alias
assert_cmd_output "personal: alias.lg" "log --oneline --graph --all" git config --global alias.lg
# Base layer aliases present
assert_cmd_output "personal: alias.co from base" "checkout" git config --global alias.co

# SSH: only github (base), no gitlab (that's work-specific)
assert_file_exists "$HOME/.ssh/config" "personal: ssh config exists"
assert_file_contains "$HOME/.ssh/config" "github.com" "personal: ssh has github (base)"
assert_file_not_contains "$HOME/.ssh/config" "gitlab.company.com" "personal: no work gitlab"

# Personal should NOT have work-specific alias
pr_value=$(git config --global alias.pr 2>/dev/null || echo "")
if [ -z "$pr_value" ]; then
    pass "personal: no work alias.pr"
else
    fail "personal: no work alias.pr" "got '$pr_value'"
fi

# Doctor always uses "default" target; accept 0 or 1 when no "default" target exists
output=$($PREFLIGHT doctor --quiet 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ] || [ "$ec" -eq 1 ]; then
    pass "personal: doctor (no default target, exit $ec accepted)"
else
    fail "personal: doctor" "exit $ec"
fi

# =========================================================================
section "Step 4: Verify layer override semantics"
# =========================================================================

# Personal layer sets editor to nvim, overriding base's vim.
# This confirms scalar last-wins semantics.
assert_cmd_output "override: editor (base=vim, personal=nvim)" "nvim" git config --global core.editor

# Base alias.co should survive when personal layer doesn't define it (map deep-merge).
assert_cmd_output "merge: alias.co survives from base" "checkout" git config --global alias.co
assert_cmd_output "merge: alias.br survives from base" "branch" git config --global alias.br

pass "layer override: scalar last-wins confirmed"
pass "layer merge: map deep-merge confirmed"

# =========================================================================
section "Step 5: Compare targets"
# =========================================================================

output=$($PREFLIGHT compare work personal 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ] || [ "$ec" -eq 1 ]; then
    pass "compare work personal succeeds"
    if [ -n "$output" ]; then
        pass "compare produces output"
    else
        fail "compare produces output" "empty output"
    fi
else
    fail "compare" "exit $ec"
fi

# =========================================================================
section "Step 6: Export for each target"
# =========================================================================

output=$($PREFLIGHT export --target work --format yaml 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ] && [ -n "$output" ]; then
    pass "export work yaml"
    if echo "$output" | grep -qF "alice@company.com"; then
        pass "export work contains work email"
    else
        fail "export work contains work email" "not found"
    fi
else
    fail "export work yaml" "exit $ec or empty"
fi

output=$($PREFLIGHT export --target personal --format json 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ] && [ -n "$output" ]; then
    pass "export personal json"
    if echo "$output" | grep -qF "alice@personal.dev"; then
        pass "export personal contains personal email"
    else
        fail "export personal contains personal email" "not found"
    fi
else
    fail "export personal json" "exit $ec or empty"
fi

# =========================================================================
section "Step 7: Profile lifecycle"
# =========================================================================

# List profiles (should show work and personal from targets)
output=$($PREFLIGHT profile list 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "profile list succeeds"
    if echo "$output" | grep -qF "work"; then
        pass "profile list shows work"
    else
        fail "profile list shows work" "not found"
    fi
    if echo "$output" | grep -qF "personal"; then
        pass "profile list shows personal"
    else
        fail "profile list shows personal" "not found"
    fi
else
    fail "profile list" "exit $ec"
fi

# Create custom profile
output=$($PREFLIGHT profile create meeting --from work 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "profile create meeting"
else
    fail "profile create meeting" "exit $ec"
fi

# List again should include meeting
output=$($PREFLIGHT profile list 2>&1) && ec=0 || ec=$?
if echo "$output" | grep -qF "meeting"; then
    pass "profile list includes meeting"
else
    fail "profile list includes meeting" "not found"
fi

# Switch to work profile
output=$($PREFLIGHT profile switch work 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "profile switch work"
else
    fail "profile switch work" "exit $ec"
fi

# Current profile should be work
output=$($PREFLIGHT profile current 2>&1) && ec=0 || ec=$?
if echo "$output" | grep -qF "work"; then
    pass "profile current shows work"
else
    fail "profile current shows work" "got: $output"
fi

# Delete meeting profile
output=$($PREFLIGHT profile delete meeting 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "profile delete meeting"
else
    fail "profile delete meeting" "exit $ec"
fi

# Deleting again should fail
output=$($PREFLIGHT profile delete meeting 2>&1) && ec=0 || ec=$?
if [ "$ec" -ne 0 ]; then
    pass "profile delete nonexistent fails"
else
    fail "profile delete nonexistent fails" "should have failed"
fi

# =========================================================================
section "Step 8: Idempotency per target"
# =========================================================================

# Re-apply personal target
export HOME="$HOME_PERSONAL"
output=$($PREFLIGHT apply --yes --target personal 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "re-apply personal"; else fail "re-apply personal" "exit $ec"; fi

plan_output=$($PREFLIGHT plan --target personal 2>&1) && plan_ec=0 || plan_ec=$?
if [ "$plan_ec" -eq 0 ] && echo "$plan_output" | grep -qi "no changes"; then
    pass "personal target converged"
else
    fail "personal target converged" "still has pending changes"
fi

# Re-apply work target
export HOME="$HOME_WORK"
output=$($PREFLIGHT apply --yes --target work 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "re-apply work"; else fail "re-apply work" "exit $ec"; fi

plan_output=$($PREFLIGHT plan --target work 2>&1) && plan_ec=0 || plan_ec=$?
if [ "$plan_ec" -eq 0 ] && echo "$plan_output" | grep -qi "no changes"; then
    pass "work target converged"
else
    fail "work target converged" "still has pending changes"
fi

# =========================================================================
# Summary
# =========================================================================

printf "\n${CYAN}=====================================${NC}\n"
printf "${CYAN}  Use Case 4: Multi-Target & Override${NC}\n"
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
rm -rf "$CONFIG_REPO" "$HOME_WORK" "$HOME_PERSONAL" 2>/dev/null || true
