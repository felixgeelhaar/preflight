---
title: Layers & Targets
description: Organize configuration with composable layers and targets.
---

Layers are the foundation of Preflight's modular configuration system. They enable you to compose different aspects of your setup into reusable, maintainable units.

## Layer Concepts

### What is a Layer?

A layer is a YAML file containing a subset of configuration. Layers are merged together to create a complete configuration.

```
layers/
  base.yaml           # Common settings for all machines
  identity.work.yaml  # Work identity and credentials
  identity.personal.yaml
  role.go.yaml        # Go developer tools
  role.python.yaml    # Python developer tools
  device.laptop.yaml  # Laptop-specific settings
  device.desktop.yaml # Desktop-specific settings
```

### Layer Naming Convention

Use a hierarchical naming scheme:

| Prefix | Purpose | Examples |
|--------|---------|----------|
| `base` | Common to all configurations | `base.yaml` |
| `identity.*` | Identity/credential separation | `identity.work.yaml` |
| `role.*` | Role-based tools | `role.go.yaml`, `role.devops.yaml` |
| `device.*` | Device-specific settings | `device.macbook.yaml` |

## Creating Layers

### Base Layer

The foundation shared across all configurations:

```yaml
# layers/base.yaml
packages:
  brew:
    formulae:
      - git
      - gh
      - ripgrep
      - fzf
      - jq

shell:
  default: zsh
  plugins:
    - git
    - docker

files:
  links:
    - src: dotfiles/.zshrc
      dest: ~/.zshrc
```

### Identity Layers

Separate work and personal identities:

```yaml
# layers/identity.work.yaml
git:
  user:
    name: Your Name
    email: you@company.com
    signingkey: WORK_GPG_KEY

ssh:
  hosts:
    - name: github.com-work
      hostname: github.com
      user: git
      identity_file: ~/.ssh/work_github
```

```yaml
# layers/identity.personal.yaml
git:
  user:
    name: Your Name
    email: personal@email.com
    signingkey: PERSONAL_GPG_KEY

ssh:
  hosts:
    - name: github.com
      user: git
      identity_file: ~/.ssh/personal_github
```

### Role Layers

Tools for specific development roles:

```yaml
# layers/role.go.yaml
packages:
  brew:
    formulae:
      - go
      - gopls
      - golangci-lint
      - delve

runtime:
  tools:
    go: "1.23.0"

nvim:
  preset: balanced
  dependencies:
    - gopls
```

```yaml
# layers/role.python.yaml
packages:
  brew:
    formulae:
      - python@3.12
      - pipx

runtime:
  tools:
    python: "3.12.0"

vscode:
  extensions:
    - ms-python.python
    - ms-python.vscode-pylance
```

### Device Layers

Device-specific configuration:

```yaml
# layers/device.laptop.yaml
shell:
  env:
    # Optimize for battery
    DOCKER_BUILDKIT: "1"

packages:
  brew:
    casks:
      - rectangle  # Window management
      - amphetamine # Prevent sleep
```

## Targets

A target is an ordered list of layers that defines a complete configuration.

### Defining Targets

```yaml
# preflight.yaml
targets:
  work:
    - base
    - identity.work
    - role.go
    - device.laptop

  personal:
    - base
    - identity.personal
    - role.python
    - device.desktop

  minimal:
    - base
```

### Using Targets

```bash
# Apply work configuration
preflight apply --target work

# Apply personal configuration
preflight apply --target personal

# Plan for specific target
preflight plan --target minimal
```

### Default Target

Set a default target in the manifest:

```yaml
# preflight.yaml
default_target: work
```

## Merge Behavior

### Merge Order

Layers are merged in the order specified:

```yaml
layers:
  - base           # Applied first
  - identity.work  # Merged on top
  - role.go        # Merged on top
  - device.laptop  # Applied last (highest priority)
```

### Merge Rules

| Type | Rule | Example |
|------|------|---------|
| Scalar | Last wins | `default: zsh` overwrites `default: bash` |
| Map | Deep merge | Git settings from all layers combined |
| List | Union | Packages accumulated from all layers |

### List Modifiers

Control list merging explicitly:

```yaml
# layers/role.go.yaml
packages:
  brew:
    formulae:
      - +go          # Add to existing list
      - +gopls
      - -nodejs      # Remove if present
```

### Replace vs Merge

Force replacement instead of merge:

```yaml
# layers/device.minimal.yaml
packages:
  brew:
    formulae: !replace
      - git
      - curl
```

## Provenance Tracking

Preflight tracks which layer defined each setting. View provenance in the TUI:

```bash
preflight plan --explain
```

Example output:

```
[git] user.email = you@company.com
  └─ Defined in: layers/identity.work.yaml:3

[packages] brew.formulae includes: go
  └─ Added by: layers/role.go.yaml:5
```

## Best Practices

### 1. Keep Layers Focused

Each layer should have a single responsibility:

```
✓ role.go.yaml        # Go-specific tools
✓ role.frontend.yaml  # Frontend tools

✗ role.dev.yaml       # Too broad
```

### 2. Use Identity Separation

Never mix work and personal credentials:

```yaml
# Good: Separate layers
identity.work.yaml
identity.personal.yaml

# Bad: Mixed credentials
base.yaml  # Contains both work and personal
```

### 3. Device Layers for Hardware Differences

```yaml
# device.m1-macbook.yaml
packages:
  brew:
    # ARM-specific packages
    formulae:
      - libpq  # Native ARM build
```

### 4. Version Your Layers

Track layers in git for reproducibility:

```bash
git add layers/
git commit -m "feat: add Go developer role"
```

## Layer Discovery

Capture existing machine configuration into layers:

```bash
preflight capture
```

The capture TUI allows you to:
- Review detected configuration
- Assign items to layers
- Move items between layers
- Accept or reject findings

## What's Next?

- [Providers](/preflight/guides/providers/) — System integration adapters
- [Dotfile Management](/preflight/guides/dotfiles/) — Managing configuration files
- [CLI Commands](/preflight/cli/commands/) — Full command reference
