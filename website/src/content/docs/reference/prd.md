---
title: Product Requirements
description: Full Product Requirements Document for Preflight.
---

This document contains the complete product requirements for Preflight.

## Product Vision

Preflight enables anyone to compile a reproducible, explainable, and portable workstation.

A workstation should not be:
- a one-off snowflake
- a pile of scripts
- or an opaque installer

Preflight treats workstation setup as a deterministic compilation problem, producing outcomes that are:
- repeatable
- inspectable
- safe
- and fully owned by the user.

## Product Mission

Make workstation setup boringly reliable, easy to reason about, and safe for both personal and work machines — without requiring deep technical knowledge.

## Product Principles

| Principle | Description |
|-----------|-------------|
| Compiler first, advisor second | Core value is deterministic compilation, AI is optional enhancement |
| Local-first, offline-capable | No cloud dependency, works without internet after initial setup |
| No execution without a plan | Always show what will change before making changes |
| Explain everything | Every action has "why", tradeoffs, and documentation links |
| User owns all outputs | Config is portable, inspectable, git-native |
| AI is optional and advisory | AI never executes, only suggests |
| Non-engineers are first-class users | Guided experiences for all skill levels |

## Target Users

### Primary Personas

#### A. Technical Builder
- Uses dotfiles, Brew, custom configs
- Wants determinism and reproducibility
- Pain: complexity, drift, undocumented state

#### B. Knowledge Worker (Non-Engineer)
- Designer, PM, researcher
- Wants "a good setup" without YAML or scripts
- Pain: lack of guidance and confidence

## Scope by Horizon

### Horizon 1 — Compiler MVP (v0.1–v1) ✓

**Goal:** Make Preflight real and trusted.

- CLI + TUI
- Config model (manifest + layers)
- Targets (work / personal / roles)
- `init`, `capture`, `plan`, `apply`, `doctor`
- Deterministic execution
- Lockfile (intent | locked | frozen)
- Dotfile generation & management
- Brew + Files + Neovim providers
- BYOK AI (advisory only)
- GitHub repo bootstrap

### Horizon 2 — Discovery & Learning (v1.1–v2)

**Goal:** Help users find their ideal setup.

- Capability packs
- Editor-agnostic recommendations
- Explainability for all suggestions
- Guided "tour" mode
- Profile inference improvements
- More providers (apt, runtime, shell)

### Horizon 3 — Ecosystem & Scale (v2+)

**Goal:** Enable community and organizational reuse.

- Plugin / capability marketplace
- Signed community packs
- Org baselines (still local-first)
- CI validation of configs
- Windows / WSL support

## Core Concepts

### Layers & Targets

| Concept | Description |
|---------|-------------|
| Layer | Composable unit of intent (base, identity.work, role.go, device.laptop) |
| Target | An ordered list of layers (e.g., work, personal) |

### Providers

| Provider | Responsibility |
|----------|----------------|
| brew | Homebrew taps, formulae, casks (macOS) |
| apt | Package installation (Linux) |
| files | Dotfile rendering, linking, drift detection |
| git | .gitconfig generation, identity separation |
| ssh | ~/.ssh/config rendering (never exports keys) |
| shell | Shell framework, plugins, themes |
| runtime | rtx/asdf tool version management |
| nvim | Neovim install, preset bootstrap, lazy-lock |
| vscode | Extension install, settings management |

### Reproducibility Modes

| Mode | Behavior |
|------|----------|
| intent | Install latest compatible versions |
| locked | Prefer lockfile; update lock intentionally |
| frozen | Fail if resolution differs from lock |

## Success Criteria

### v1 (Qualitative)
- New user can go from zero → working setup in < 30 minutes
- Re-running apply causes no surprises
- Non-engineers can complete init without editing YAML
- Captured config can reproduce on a fresh machine
- Users trust it on their work machines

### v2 (Quantitative)
- Marketplace has ≥50 community-contributed presets
- WSL support enables full workflow on Windows
- Org policy adoption by ≥3 teams in beta
- Learning tour completion rate ≥70%
- Time to onboard new user reduced by 50% vs v1

## v1.x Features Status

### v1.1 Enhanced Capture TUI ✓

- Search/filter items by name, provider, category
- Layer reassignment — move items between layers in TUI
- Undo/redo for review decisions
- Keyboard shortcuts for power users

### v1.2 Full Dotfile Lifecycle ✓

- Snapshot before applying changes (automatic backup)
- Hash-based drift detection
- Doctor `--update-config` flag
- Config patch generation from drift
- VS Code settings drift detection

### v1.3 Three-Way Merge ✓

- DetectChangeType: Classifies changes as none/ours/theirs/both/same
- ThreeWayMerge: Automatic merge when possible
- Conflict markers with descriptive labels (git/diff3 style)
- ParseConflictRegions: Extract conflicts from marked content
- ResolveAllConflicts: Programmatic conflict resolution

## Out of Scope

### Never
- Centralized SaaS management
- Mandatory accounts
- Proprietary lock-in

### Deferred (v2+)
- Plugin marketplace
- Org policy and compliance engine
- Remote execution and fleet management
- Continuous background reconciliation

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Becoming "Nix but worse" | Keep scope tight, UX-first |
| Too many providers | Start with few, extensible |
| AI trust issues | Advisory-only, BYOK |
| Complexity creep | Compiler model discipline |
