<p align="center">
  <img src="website/public/logo.svg" alt="Preflight Logo" width="280">
</p>

<h1 align="center">Preflight</h1>

<p align="center">
  <strong>Deterministic workstation compiler</strong><br>
  A Go CLI/TUI tool that compiles declarative configuration into reproducible machine setups.
</p>

<p align="center">
  <a href="https://github.com/felixgeelhaar/preflight/actions/workflows/ci.yml"><img src="https://github.com/felixgeelhaar/preflight/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://goreportcard.com/report/github.com/felixgeelhaar/preflight"><img src="https://goreportcard.com/badge/github.com/felixgeelhaar/preflight" alt="Go Report Card"></a>
  <img src="https://img.shields.io/badge/coverage-80%25%2B-brightgreen" alt="Coverage">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

## Demo

<p align="center">
  <img src="website/public/demos/gif/plan-apply.gif" alt="Preflight Plan and Apply Demo" width="720">
</p>

## Overview

Preflight follows a compiler model to transform declarative YAML configuration into executable, idempotent machine setups:

```
Intent → Merge → Plan → Apply → Verify
```

Define your workstation configuration once, apply it anywhere.

## Features

- **Declarative Configuration**: Define your setup in `preflight.yaml` with composable layer overlays
- **Deterministic Execution**: Same config always produces the same result
- **Plan Before Apply**: Always preview changes before executing
- **Provenance Tracking**: Know exactly which layer defined each setting
- **Multi-Platform**: macOS (Homebrew) and Linux (apt) package management
- **Editor Support**: Neovim presets (LazyVim, NvChad, AstroNvim) and VSCode configuration
- **Shell Configuration**: zsh/bash/fish with oh-my-zsh, starship, and custom plugins
- **Dotfile Management**: Template, generate, or link your dotfiles with drift detection
- **Git & SSH**: Managed .gitconfig and ~/.ssh/config with identity separation
- **Capture Review TUI**: Interactive review with search/filter, layer reassignment, and undo/redo
- **Three-Way Merge**: Automatic conflict detection with git-style conflict markers

## Installation

```bash
go install github.com/felixgeelhaar/preflight@latest
```

Or build from source:

```bash
git clone https://github.com/felixgeelhaar/preflight.git
cd preflight
make build
./bin/preflight version
```

## Quick Start

1. **Initialize configuration**:
```bash
preflight init
```

2. **Review the plan**:
```bash
preflight plan
```

3. **Apply changes**:
```bash
preflight apply
```

## Configuration

Preflight uses a layered configuration system:

```
preflight.yaml          # Root manifest
layers/
  base.yaml             # Common settings
  identity.work.yaml    # Work identity
  role.go.yaml          # Go developer tools
  device.laptop.yaml    # Device-specific settings
```

### Example preflight.yaml

```yaml
name: my-workstation
target: macos

layers:
  - base
  - identity.work
  - role.go

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
      - iterm2

git:
  user:
    name: Your Name
    email: you@example.com

shell:
  default: zsh
  starship:
    enabled: true
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `preflight init` | Initialize configuration with interactive wizard |
| `preflight capture` | Capture existing system configuration |
| `preflight plan` | Preview changes without applying |
| `preflight apply` | Apply the configuration |
| `preflight doctor` | Check system health and detect drift |
| `preflight doctor --update-config` | Merge drift back into config |
| `preflight version` | Show version information |

## Architecture

Preflight is built with Domain-Driven Design principles:

```
cmd/preflight/           # CLI entry point
internal/
  domain/
    config/              # Configuration loading and merging
    compiler/            # Step graph compilation
    execution/           # Step execution engine
    drift/               # File change detection
    snapshot/            # Automatic backups before changes
    merge/               # Three-way merge with conflict markers
  provider/              # System integration adapters
    brew/                # Homebrew packages
    apt/                 # Apt packages
    files/               # Dotfile management
    git/                 # Git configuration
    ssh/                 # SSH configuration
    runtime/             # Tool version management (rtx/asdf)
    shell/               # Shell configuration
    nvim/                # Neovim configuration
    vscode/              # VSCode configuration
  tui/                   # Bubble Tea interactive interfaces
```

## Design Principles

1. **No execution without a plan** - Always show what will change first
2. **Idempotent operations** - Re-running apply is always safe
3. **Explainability** - Every action has context and documentation
4. **Secrets never leave the machine** - Config uses references, not values
5. **User ownership** - Config is portable, inspectable, and git-native

## Development

```bash
# Run tests
make test

# Run tests with race detector
make test-race

# Check coverage (requires coverctl)
make coverage-check

# Run linter
make lint
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## Documentation

- [Product Requirements](docs/prd.md)
- [CLI Design](docs/cli.md)
- [TDD Workflow](docs/tdd.md)
- [Vision](docs/vision.md)

## License

MIT License - see [LICENSE](LICENSE) for details.
