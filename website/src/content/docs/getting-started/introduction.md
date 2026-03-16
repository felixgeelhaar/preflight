---
title: Introduction
description: Learn what Preflight is and why you should use it for workstation setup.
---

Preflight is a **deterministic workstation compiler** — a Go CLI/TUI tool that compiles declarative configuration into reproducible machine setups.

## What is Preflight?

Preflight treats workstation setup as a compilation problem:

- **Intent** → Define what you want in YAML
- **Merge** → Combine layers into a target config
- **Plan** → See exactly what will change
- **Apply** → Execute idempotently
- **Verify** → Detect drift with doctor

## Why Preflight?

### The Problem

Setting up a new machine is typically:
- **Slow** — Hours of manual installation and configuration
- **Error-prone** — Forgotten settings, missing dependencies
- **Undocumented** — "How did I set this up again?"
- **Fragile** — Scripts break, dotfiles drift

### Existing Solutions

| Solution | Gap |
|----------|-----|
| Dotfiles | Not declarative, not explainable |
| Brewfiles | No structure, no profiles |
| Nix | Powerful, but inaccessible |
| MDMs | Heavy, centralized, enterprise-only |
| IDE sync | Editor-only, not system-wide |

### The Preflight Approach

Preflight fills the gap between raw scripts and heavy infrastructure tools:

- **Declarative** — Define intent, not steps
- **Explainable** — Every action has context
- **Safe** — Plan before apply, always
- **Portable** — Git-native configuration
- **Accessible** — TUI for non-engineers too
- **Secure** — Defense-in-depth plugin security

## Core Concepts

### Layers

Composable units of configuration:

```
layers/
  base.yaml           # Common settings
  identity.work.yaml  # Work identity
  identity.personal.yaml
  role.go.yaml        # Go developer tools
  device.laptop.yaml  # Device-specific
```

### Targets

An ordered list of layers to apply:

```yaml
targets:
  work: [base, identity.work, role.go, device.laptop]
  personal: [base, identity.personal]
```

### Providers

System integration adapters that compile config into executable steps:

- **brew** — Homebrew packages (macOS)
- **apt** — Package installation (Linux)
- **files** — Dotfile management
- **git** — .gitconfig generation
- **ssh** — SSH config management
- **shell** — Shell configuration
- **nvim** — Neovim setup
- **vscode** — VSCode configuration

## Enterprise Features

Starting with v5.0, Preflight includes enterprise-grade capabilities for organizations managing fleets of workstations:

- **Enterprise Identity (OIDC)** — Authenticate with corporate identity providers using device authorization flow, with tokens stored securely in the OS keychain
- **SLSA Attestation** — Verify that locked packages were built by trusted builders with reproducible processes, using Sigstore keyless signing
- **Cloud Fleet Discovery** — Auto-discover hosts from AWS EC2, Azure VMs, and GCP Compute Engine instead of maintaining static inventory files
- **Compliance Attestation** — Generate cryptographic proof that machines met policy requirements at a point in time
- **Provisioner Plugins** — Extend Preflight with infrastructure provisioning via WASM plugins (Terraform, Pulumi, etc.)
- **Marketplace Security Scanning** — Automated vulnerability scanning of marketplace packages during installation

## What's Next?

- [Installation](/preflight/getting-started/installation/) — Get Preflight on your machine
- [Quick Start](/preflight/getting-started/quickstart/) — Set up your first configuration
- [Configuration Guide](/preflight/guides/configuration/) — Deep dive into preflight.yaml
