---
title: Quick Start
description: Get up and running with Preflight in 5 minutes.
---

This guide will help you create your first Preflight configuration and apply it to your machine.

## Step 1: Initialize Configuration

Start the interactive wizard:

```bash
preflight init
```

![Init Wizard Demo](/preflight/demos/gif/init-wizard.gif)

Or create a minimal configuration:

```bash
preflight init --minimal
```

This creates:
- `preflight.yaml` — Root manifest
- `layers/` — Configuration overlays

## Step 2: Review Your Configuration

```yaml
# preflight.yaml
name: my-workstation
target: macos

layers:
  - base

packages:
  brew:
    taps:
      - homebrew/cask-fonts
    formulae:
      - git
      - gh
      - ripgrep
      - fzf
    casks:
      - visual-studio-code

git:
  user:
    name: Your Name
    email: you@example.com

shell:
  default: zsh
  starship:
    enabled: true
```

## Step 3: Preview Changes

Always see what will change before applying:

```bash
preflight plan
```

![Plan and Apply Demo](/preflight/demos/gif/plan-apply.gif)

Example output:

```
Plan: 8 actions

[brew] Install formula: git
  → Already installed (satisfied)

[brew] Install formula: ripgrep
  → Will install ripgrep 14.1.0

[git] Generate ~/.gitconfig
  → Will create new file
  + [user]
  +   name = Your Name
  +   email = you@example.com

[shell] Configure starship prompt
  → Will install starship
```

## Step 4: Apply Configuration

Apply the plan to your machine:

```bash
preflight apply
```

Or skip confirmation prompts:

```bash
preflight apply --yes
```

## Step 5: Verify State

Check that everything is correctly configured:

```bash
preflight doctor
```

![Doctor Fix Demo](/preflight/demos/gif/doctor-fix.gif)

Example output:

```
Doctor Report

✓ brew: All packages installed
✓ git: Configuration valid
✓ shell: Starship configured
✓ files: No drift detected

All checks passed!
```

## Capture Existing Machine

Already have a configured machine? Capture it:

```bash
preflight capture
```

![Capture Review Demo](/preflight/demos/gif/capture-review.gif)

This reverse-engineers your current setup into Preflight configuration:

1. Detects installed packages
2. Finds dotfiles and configurations
3. Infers layers (base, identity, roles)
4. Opens TUI for review

## Common Workflows

### Adding a New Package

Edit your configuration:

```yaml
packages:
  brew:
    formulae:
      - git
      - gh
      - jq  # Add new package
```

Then apply:

```bash
preflight plan  # Preview
preflight apply # Execute
```

### Switching Targets

Work vs personal machine:

```bash
# Apply work configuration
preflight apply --target work

# Apply personal configuration
preflight apply --target personal
```

### Detecting Drift

Check if your machine has drifted from config:

```bash
preflight doctor
```

Fix drift by converging to config:

```bash
preflight doctor --fix
```

Or update config to match machine:

```bash
preflight doctor --update-config
```

## What's Next?

- [Configuration Guide](/preflight/guides/configuration/) — Deep dive into preflight.yaml
- [Layers & Targets](/preflight/guides/layers/) — Organize configuration with layers
- [CLI Reference](/preflight/cli/commands/) — All available commands
