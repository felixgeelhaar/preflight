---
title: Design Principles
description: Core guarantees and design principles that guide Preflight.
---

Preflight is built on a set of core principles that ensure safety, transparency, and user control.

## Core Guarantees

### 1. No Execution Without a Plan

Every change is previewed before execution.

```bash
# Always see what will change first
preflight plan

# Then apply
preflight apply
```

**Implementation:**
- `plan` command is read-only
- `apply` requires explicit confirmation (unless `--yes`)
- Destructive steps are clearly labeled

### 2. Idempotent Operations

Re-running is always safe.

```bash
# Running apply multiple times is safe
preflight apply
preflight apply  # No changes if already applied
```

**Implementation:**
- Every step implements `Check()` to determine if action is needed
- `Apply()` only runs when `Check()` returns `NeedsApply`
- Same input always produces same output

### 3. Explainability

Every action has context and documentation.

```bash
preflight plan --explain
```

```
[brew] Install ripgrep
  Why: Required for Neovim Telescope plugin
  Docs: https://github.com/BurntSushi/ripgrep
  Layer: layers/role.go.yaml:12
```

**Implementation:**
- Steps implement `Explain()` method
- Provenance tracking shows which layer defined each setting
- Documentation links for all tools

### 4. Secrets Never Leave the Machine

Configuration uses references, not values.

```yaml
# Good: Reference to file
ssh:
  hosts:
    - name: github
      identity_file: ~/.ssh/github_key

# Never: Actual secret content
ssh:
  private_key: "-----BEGIN RSA..."  # NEVER!
```

**Implementation:**
- Capture automatically redacts secrets
- Secrets are referenced by path or keychain ID
- Private keys are never exported

### 5. AI Advises, Never Executes

AI is optional and advisory only.

```
AI may:
✓ Suggest tools and configurations
✓ Explain tradeoffs
✓ Infer profiles from machine state
✓ Link to documentation

AI may never:
✗ Execute commands
✗ Modify files directly
✗ Access secrets
✗ Override user approval
```

**Implementation:**
- AI outputs map to known presets or require confirmation
- BYOK (Bring Your Own Key) model
- Works offline without AI

### 6. User Ownership

Configuration is portable, inspectable, and git-native.

```bash
# Your config is plain YAML
cat preflight.yaml

# Version controlled
git add preflight.yaml layers/
git commit -m "Add Go developer setup"

# Portable across machines
git clone your-dotfiles new-machine
cd new-machine && preflight apply
```

## Design Philosophy

### Compiler Model

Preflight treats workstation setup as a compilation problem:

| Concept | Compiler Analogy |
|---------|------------------|
| Layers | Source modules |
| Merge | Linking |
| Plan | Compile output |
| Apply | Execution |
| Lock | Binary version |

### Determinism

Same input always produces same output:

```
Config (YAML) + Mode (locked) → Identical System State
```

Controlled via modes:
- **Intent** — Latest compatible versions
- **Locked** — Prefer lockfile
- **Frozen** — Fail on mismatch

### Layered Architecture

Clean separation of concerns:

```
┌──────────────┐
│   CLI/TUI    │  ← User interface
├──────────────┤
│  Application │  ← Use cases
├──────────────┤
│    Domain    │  ← Business logic
├──────────────┤
│   Adapters   │  ← External systems
└──────────────┘
```

### Ports & Adapters

External dependencies are abstracted:

```go
// Port (interface)
type FileSystem interface {
    Read(path string) ([]byte, error)
    Write(path string, data []byte) error
}

// Adapter (implementation)
type OSFileSystem struct{}

// Test double
type MockFileSystem struct{}
```

Benefits:
- Testability with mocks
- Swappable implementations
- Clear system boundaries

## What Preflight Is Not

### Not a SaaS

- No cloud dependency
- Works fully offline
- No accounts required

### Not an MDM

- No remote control
- No centralized management
- User controls everything

### Not a Background Agent

- Runs on demand
- No daemon process
- Explicit user invocation

### Not a Nix Replacement

- Simpler mental model
- Accessible to non-engineers
- Pragmatic, not ideological

## Safety Measures

### Before Apply

1. **Validation** — Config schema and constraints
2. **Snapshot** — Backup existing files
3. **Plan** — Show all changes
4. **Confirm** — Require user approval

### During Apply

1. **Check First** — Only apply if needed
2. **Idempotent** — Safe to retry
3. **Ordered** — Respect dependencies
4. **Logged** — Record all actions

### After Apply

1. **Verify** — Doctor checks state
2. **Lock** — Record resolved versions
3. **Report** — Show what changed

## Error Philosophy

### Fail Fast

Detect problems early in the pipeline:

```
Load → Validate → Merge → Compile → Plan → Apply
  ↓        ↓         ↓        ↓        ↓       ↓
Error   Error     Error    Error    Error   Error
```

### Fail Safe

When errors occur:
- Stop execution
- Report clearly
- Leave system in known state
- Offer recovery options

### User Control

Users decide how to handle issues:

```bash
# Continue on error
preflight apply --continue-on-error

# Rollback on error
preflight apply --rollback-on-error

# Preview only
preflight apply --dry-run
```

## Evolution Strategy

### v1: Compiler MVP

- Core compilation pipeline
- Essential providers (brew, files, git, ssh, shell, nvim, vscode)
- TUI for init, capture, plan, apply, doctor
- Lockfile with integrity verification

### v2+: Discovery & Ecosystem

- Capability marketplace
- Community presets
- Enhanced learning tours
- Org baselines (still local-first)

### Never

- Centralized SaaS control
- Mandatory accounts
- Proprietary lock-in
- Remote execution

## What's Next?

- [Architecture Overview](/preflight/architecture/overview/) — System design
- [Domains](/preflight/architecture/domains/) — Bounded contexts
- [TDD Workflow](/preflight/development/tdd/) — Development practices
