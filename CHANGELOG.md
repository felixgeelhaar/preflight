# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2024-12-23

### Added

#### Core Domains
- **Config Domain**: Manifest, Layer, and Target value objects with deep merge semantics and provenance tracking
- **Compiler Domain**: StepGraph with DAG compilation and topological ordering
- **Execution Domain**: Step interface with Check/Plan/Apply lifecycle and DAG scheduler
- **Lock Domain**: Lockfile management with integrity verification (SHA256/SHA512) and resolution modes (intent, locked, frozen)
- **Catalog Domain**: Presets and capability packs for quick setup with difficulty levels
- **Advisor Domain**: AI provider interfaces (OpenAI, Anthropic, Ollama) with BYOK support

#### CLI Commands
- `preflight plan` - Generate execution plan from configuration
- `preflight apply` - Apply configuration to the system
- `preflight version` - Display version information
- `preflight init` - Interactive configuration wizard (TUI)
- `preflight doctor` - Verify system state and detect drift
- `preflight capture` - Reverse-engineer current machine configuration
- `preflight diff` - Show configuration vs machine differences
- `preflight lock` - Lockfile management (update, freeze, status)
- `preflight tour` - Interactive guided walkthroughs
- `preflight repo` - Configuration repository management
- `preflight completion` - Shell completion generation (bash, zsh, fish, powershell)

#### Providers
- **brew**: Homebrew package management (taps, formulae, casks)
- **apt**: Debian/Ubuntu package management with PPAs
- **files**: Dotfile management (links, templates, copies)
- **git**: .gitconfig generation with aliases and includes
- **ssh**: ~/.ssh/config generation with host and match blocks
- **runtime**: Tool version management (rtx/asdf compatible)
- **shell**: Shell configuration (zsh/bash/fish, oh-my-zsh, starship)
- **nvim**: Neovim configuration (LazyVim, NvChad, AstroNvim, Kickstart)
- **vscode**: VSCode configuration (extensions, settings, keybindings)

#### TUI Infrastructure
- Bubble Tea-based terminal UI with Catppuccin theming
- Reusable components: List, Panel, Progress, Confirm, Explain, DiffView, Search
- Init wizard with step-by-step configuration
- Plan review with explanation panels
- Apply progress with real-time status

#### Testing Infrastructure
- Comprehensive test utilities and helpers
- Builder patterns for test data
- Custom assertions for file and YAML operations
- Fixture loading from embedded filesystem

#### Distribution
- GitHub Actions release workflow for multi-platform builds
- Homebrew formula for macOS/Linux installation
- Cross-compilation for darwin/linux on amd64/arm64

#### Infrastructure
- GitHub Actions CI pipeline
- golangci-lint configuration with 23+ linters
- coverctl integration for domain-aware coverage enforcement
- TDD workflow with 80%+ coverage requirement

### Technical Details
- Built with Go 1.23
- Domain-Driven Design architecture
- Test-Driven Development with >80% coverage requirement per domain

[Unreleased]: https://github.com/felixgeelhaar/preflight/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/felixgeelhaar/preflight/releases/tag/v0.1.0
