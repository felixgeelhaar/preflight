#!/usr/bin/env bash
# test_usecase_machine_migration.sh - Use Case 6: Machine migration
#
# Simulates migrating from one machine to another:
#   Phase 1 (Machine A): Set up a working environment, capture it
#   Phase 2 (Machine B): Fresh machine, apply captured config
#   Phase 3: Verify Machine B matches Machine A
#   Phase 4: Drift detection and re-apply
#
# This tests the core promise: "switch your MacBook tomorrow without losing anything"

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

assert_file_not_exists() {
    if [ ! -f "$1" ]; then pass "$2"; else fail "$2" "file should not exist: $1"; fi
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

assert_dir_exists() {
    if [ -d "$1" ]; then pass "$2"; else fail "$2" "dir not found: $1"; fi
}

# ---------------------------------------------------------------------------
printf "\n${BOLD}${CYAN}Use Case 6: Machine Migration${NC}\n"
printf "==============================\n"

# ===========================================================================
# PHASE 1: Machine A — Set up a working environment
# ===========================================================================

section "Phase 1: Machine A — Existing environment"

MACHINE_A=$(mktemp -d)
export HOME="$MACHINE_A/home"
mkdir -p "$HOME"
REPO_DIR="$MACHINE_A/dotfiles"
mkdir -p "$REPO_DIR"
cd "$REPO_DIR"
git init -q .
git config user.email "dev@company.com"
git config user.name "Dev"

# Simulate an existing machine with git, ssh, shell config
# Set up git config (as if user configured it manually)
git config --global user.name "Alice Developer"
git config --global user.email "alice@company.com"
git config --global core.editor "nvim"
git config --global core.autocrlf "input"
git config --global alias.co "checkout"
git config --global alias.br "branch"
git config --global alias.ci "commit"
git config --global alias.st "status"
git config --global alias.lg "log --oneline --graph --decorate"
pass "Machine A git config set up"

# Set up SSH config
mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
cat > "$HOME/.ssh/config" <<'SSH'
Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519
    AddKeysToAgent yes

Host gitlab.company.com
    HostName gitlab.company.com
    User git
    IdentityFile ~/.ssh/id_work
    ProxyJump bastion.company.com

Host *
    AddKeysToAgent yes
    IdentitiesOnly yes
SSH
chmod 600 "$HOME/.ssh/config"
pass "Machine A SSH config set up"

# Set up a dotfile
mkdir -p "$HOME/.config/starship"
cat > "$HOME/.config/starship.toml" <<'STARSHIP'
[character]
success_symbol = "[>](bold green)"
error_symbol = "[x](bold red)"

[git_branch]
symbol = " "

[golang]
symbol = " "
STARSHIP
pass "Machine A starship config set up"

# ===========================================================================
section "Phase 1b: Capture existing config with preflight"
# ===========================================================================

# Init preflight in the repo
output=$($PREFLIGHT init --preset git:minimal --non-interactive 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then pass "preflight init on Machine A"; else fail "preflight init" "exit $ec: $output"; fi

assert_file_exists "$REPO_DIR/preflight.yaml" "manifest created"

# Write a rich layer that represents what capture would produce
cat > "$REPO_DIR/layers/base.yaml" <<'LAYER'
name: base

git:
  user:
    name: "Alice Developer"
    email: "alice@company.com"
  core:
    editor: nvim
    autocrlf: input
  alias:
    co: checkout
    br: branch
    ci: commit
    st: status
    lg: "log --oneline --graph --decorate"


ssh:
  defaults:
    addkeystoagent: true
    identitiesonly: true
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_ed25519
    - host: gitlab.company.com
      hostname: gitlab.company.com
      user: git
      identityfile: ~/.ssh/id_work
      proxyjump: bastion.company.com
LAYER
pass "captured config to layers/base.yaml"

# Validate it
assert_exit_code "validate captured config" 0 $PREFLIGHT validate

# Plan to verify capture
output=$($PREFLIGHT plan 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "plan on Machine A"
else
    fail "plan on Machine A" "exit $ec"
fi

# Create lockfile
assert_exit_code "lock update on Machine A" 0 $PREFLIGHT lock update
assert_file_exists "$REPO_DIR/preflight.lock" "lockfile created"

# Commit the config (simulating git push)
git add -A
git commit -q -m "Machine A: captured config"
MACHINE_A_COMMIT=$(git rev-parse HEAD)
pass "config committed: $(git log --oneline -1)"

# ===========================================================================
# PHASE 2: Machine B — Fresh machine, clone and apply
# ===========================================================================

section "Phase 2: Machine B — Fresh machine setup"

MACHINE_B=$(mktemp -d)
export HOME="$MACHINE_B/home"
mkdir -p "$HOME"

# Verify Machine B is clean — no git config, no ssh config
assert_file_not_exists "$HOME/.gitconfig" "Machine B has no .gitconfig"
assert_file_not_exists "$HOME/.ssh/config" "Machine B has no SSH config"

# Clone the repo (simulating git clone on new machine)
git clone -q "$REPO_DIR" "$MACHINE_B/dotfiles"
cd "$MACHINE_B/dotfiles"
git config user.email "dev@test.com"
git config user.name "Dev"
pass "cloned config repo to Machine B"

# Verify files are there
assert_file_exists "$MACHINE_B/dotfiles/preflight.yaml" "manifest present"
assert_file_exists "$MACHINE_B/dotfiles/layers/base.yaml" "base layer present"
assert_file_exists "$MACHINE_B/dotfiles/preflight.lock" "lockfile present"

# ===========================================================================
section "Phase 2b: Plan on Machine B"
# ===========================================================================

output=$($PREFLIGHT plan 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "plan on Machine B"
    # Plan should show changes needed (since Machine B is fresh)
    if echo "$output" | grep -qF "git:config"; then
        pass "plan shows git:config needed"
    else
        fail "plan shows git:config needed" "not in plan output"
    fi
    if echo "$output" | grep -qF "ssh:config"; then
        pass "plan shows ssh:config needed"
    else
        fail "plan shows ssh:config needed" "not in plan output"
    fi
else
    fail "plan on Machine B" "exit $ec"
fi

# ===========================================================================
section "Phase 2c: Apply on Machine B"
# ===========================================================================

output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "apply on Machine B"
else
    fail "apply on Machine B" "exit $ec: $(echo "$output" | tail -5)"
fi

# ===========================================================================
# PHASE 3: Verify Machine B matches Machine A
# ===========================================================================

section "Phase 3: Verify Machine B matches Machine A"

# Git config
assert_cmd_output "git user.name" "Alice Developer" git config --global user.name
assert_cmd_output "git user.email" "alice@company.com" git config --global user.email
assert_cmd_output "git core.editor" "nvim" git config --global core.editor
assert_cmd_output "git core.autocrlf" "input" git config --global core.autocrlf
assert_cmd_output "git alias.co" "checkout" git config --global alias.co
assert_cmd_output "git alias.br" "branch" git config --global alias.br
assert_cmd_output "git alias.ci" "commit" git config --global alias.ci
assert_cmd_output "git alias.st" "status" git config --global alias.st
assert_cmd_output "git alias.lg" "log --oneline --graph --decorate" git config --global alias.lg

# SSH config
assert_file_exists "$HOME/.ssh/config" "ssh config created on Machine B"
assert_file_contains "$HOME/.ssh/config" "github.com" "ssh has github.com"
assert_file_contains "$HOME/.ssh/config" "gitlab.company.com" "ssh has gitlab.company.com"
assert_file_contains "$HOME/.ssh/config" "id_ed25519" "ssh has personal key ref"
assert_file_contains "$HOME/.ssh/config" "id_work" "ssh has work key ref"
assert_file_contains "$HOME/.ssh/config" "bastion.company.com" "ssh has proxy jump"
assert_file_contains "$HOME/.ssh/config" "IdentitiesOnly" "ssh has identities only"

# ===========================================================================
section "Phase 3b: Doctor reports clean"
# ===========================================================================

output=$($PREFLIGHT doctor --quiet 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "doctor reports no drift on Machine B"
else
    fail "doctor" "exit $ec: $(echo "$output" | tail -3)"
fi

# ===========================================================================
# PHASE 4: Drift detection and repair
# ===========================================================================

section "Phase 4: Drift detection and repair"

# Simulate drift: someone manually edits .gitconfig
git config --global core.editor "vim"
pass "introduced drift: changed editor to vim"

# Doctor should detect drift
output=$($PREFLIGHT doctor 2>&1) && ec=0 || ec=$?
if echo "$output" | grep -qi "drift\|change\|mismatch\|differ"; then
    pass "doctor detects drift"
elif [ "$ec" -ne 0 ]; then
    pass "doctor reports issues (exit $ec)"
else
    fail "doctor detects drift" "doctor did not report any issues"
fi

# Plan should show fix needed
output=$($PREFLIGHT plan 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "plan after drift"
else
    fail "plan after drift" "exit $ec"
fi

# Re-apply to fix drift
output=$($PREFLIGHT apply --yes 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ]; then
    pass "re-apply fixes drift"
else
    fail "re-apply" "exit $ec"
fi

# Verify fixed
assert_cmd_output "editor restored to nvim" "nvim" git config --global core.editor

# ===========================================================================
section "Phase 4b: Idempotency after repair"
# ===========================================================================

output=$($PREFLIGHT plan 2>&1) && ec=0 || ec=$?
if [ "$ec" -eq 0 ] && echo "$output" | grep -qi "no changes"; then
    pass "plan shows no changes after repair"
else
    fail "plan shows no changes" "plan still has pending changes"
fi

# ===========================================================================
# Summary
# ===========================================================================

printf "\n${CYAN}=====================================${NC}\n"
printf "${CYAN}  Use Case 6: Machine Migration${NC}\n"
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
rm -rf "$MACHINE_A" "$MACHINE_B" 2>/dev/null || true
