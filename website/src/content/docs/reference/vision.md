---
title: Vision
description: Preflight's vision, principles, and long-term direction.
---

## Vision Statement

Preflight enables anyone to compile a reproducible, explainable, and portable workstation.

A workstation should be:
- **Deterministic** — Same input produces same output
- **Transparent** — Every action is explainable
- **Safe** — Changes are previewed and reversible
- **Reproducible** — Easy to recreate on any machine
- **User-owned** — Configuration is portable and inspectable

Preflight treats workstation setup as a compilation problem, not a collection of scripts and ad-hoc installers.

## What Preflight Is

Preflight is:

- A **deterministic workstation compiler**
- With an **optional guided discovery layer**
- Producing **plain, human-owned configuration**
- Converging machines via **plan → apply → verify**

Preflight works for:
- Engineers and non-engineers
- Personal and work machines
- Online or fully offline environments

## What Preflight Is Not

Preflight is not:

- ❌ A SaaS-first product
- ❌ An MDM or remote control system
- ❌ A background agent that mutates your machine
- ❌ A replacement for your creative dotfiles workflows
- ❌ A Nix replacement (Preflight borrows ideas, not ideology)

## Core Guarantees

Preflight always guarantees:

### 1. No Execution Without a Plan

Every change is previewed before execution. Users always see what will change.

### 2. Every Change is Explainable

Actions include "why", tradeoffs, and documentation links. Provenance tracking shows which layer defined each setting.

### 3. Re-running is Safe and Idempotent

Applying the same configuration multiple times produces the same result. Safe to run repeatedly.

### 4. Configuration is Portable and Inspectable

Plain YAML files that can be version controlled, shared, and read by humans.

### 5. Secrets Never Leave the Machine

Capture automatically redacts sensitive data. Configuration uses references, not values.

### 6. AI Never Executes Actions

AI is advisory only. It may suggest, explain, and recommend, but never runs commands.

### 7. User Ownership Over All Outputs

No proprietary formats. No lock-in. Everything is git-native and portable.

## Compiler Model

Preflight operates like a compiler:

```
Intent (layers, profiles, capabilities)
         ↓
    Merge & normalize
         ↓
    Plan (diff + explanation)
         ↓
    Apply (deterministic)
         ↓
    Verify (doctor / drift)
```

### Execution Modes

Execution determinism is controlled via modes:

| Mode | Description |
|------|-------------|
| **intent** | Install latest compatible versions |
| **locked** | Prefer lockfile; update intentionally |
| **frozen** | Fail on lock mismatches |

## AI Philosophy

AI in Preflight is:

- **Optional** — Works completely without AI
- **BYOK** — Bring your own key/provider
- **Advisory only** — Never executes commands
- **Flexible** — Works with cloud providers or local models (e.g., Ollama)

### AI May

- ✓ Guide onboarding interviews
- ✓ Suggest tools, presets, and capability packs
- ✓ Explain why something is selected (with tradeoffs)
- ✓ Infer profiles/layers from an existing machine
- ✓ Link to relevant docs and provide quick demos/tours

### AI May Never

- ✗ Execute commands
- ✗ Mutate the system
- ✗ Access secrets
- ✗ Override user approval

## Dotfiles Philosophy

Dotfiles are first-class artifacts produced and managed by the compiler.

### Three Modes

1. **Generated** — Preflight owns the file (best for beginners/non-engineers)
2. **Template + user overrides** — Preflight manages a base; users extend safely
3. **Bring-your-own** — Preflight links/validates; never rewrites

### Guarantees

- Preflight never silently overwrites user changes
- Dotfile diffs appear in `preflight plan`
- Drift is detected by `preflight doctor`
- Snapshots are created before modifications

## Long-Term Direction

Preflight starts as a compiler. Over time it may grow:

- A plugin/capability marketplace
- Curated presets and packs for different personas
- Org baselines (still local-first)
- Richer discovery guidance and learning tours
- Windows/WSL support

But it will **always remain**:

- ✓ Local-first
- ✓ Transparent and explainable
- ✓ Deterministic
- ✓ Git-native
- ✓ BYOK for AI

## Final Positioning

> **Preflight is a deterministic workstation compiler that helps anyone design, reproduce, and understand their setup — safely, locally, and without lock-in.**
