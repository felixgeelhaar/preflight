---
title: Changelog
description: All notable changes to Preflight.
---

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2024-12-24

### Added

- **Three-Way Merge Domain**: New `internal/domain/merge/` package for conflict detection
  - `ThreeWayMerge`: Performs merge with automatic conflict detection
  - `DetectChangeType`: Classifies changes as none/ours/theirs/both/same
  - `HasConflictMarkers`: Detects conflict markers in content
  - `ParseConflictRegions`: Extracts conflict sections from marked content
  - `ResolveAllConflicts`: Programmatic resolution using ours/theirs/base
  - Git-style and diff3-style conflict markers with descriptive labels

### Changed

- Updated PRD to mark v1.3 Three-Way Merge as complete

---

## [1.2.0] - 2024-12-24

### Added

- **Snapshot Domain**: Automatic backup before file modifications
  - `Snapshot`: Individual file backup with hash and metadata
  - `Set`: Group of snapshots created together with reason tracking
  - `FileStore`: Persistent storage in `~/.preflight/snapshots/`
  - `Manager`: Orchestrates snapshot creation and restoration

- **Drift Detection Domain**: Hash-based detection of external file changes
  - `Detector`: Checks tracked files for modifications
  - `AppliedState`: Tracks file state after apply operations
  - `StateStore`: Persistent storage of applied state

- **Doctor Enhancements**:
  - `--update-config` flag to merge drift back into layer files
  - `--dry-run` flag to preview config changes without writing
  - Config patch generation from detected drift

- **Lifecycle Integration**:
  - `FileLifecycle` port for snapshot and drift tracking
  - `LifecycleManager` combines snapshot and drift services
  - Automatic state recording after apply operations

### Changed

- Files provider now snapshots before modifications
- Updated PRD to mark v1.2 Full Dotfile Lifecycle as complete

---

## [1.1.0] - 2024-12-24

### Added

- **TUI Layer Reassignment**: Move items between configuration layers during capture review
  - `l` key enters layer selection mode for current item
  - Navigate layers with `j`/`k` or arrow keys
  - Quick selection with number keys 1-9
  - Enter confirms selection, Esc cancels
  - Layer shown in item list with arrow notation (→ layer_name)
  - Full undo/redo support for layer changes

- **Enhanced Capture Review Options**:
  - `AvailableLayers` field for custom layer list
  - `DefaultLayers()` provides standard layers: base, identity.work, identity.personal, role.dev, device.laptop, captured
  - `WithAvailableLayers()` builder method

- **Rich Capture Results**:
  - `CaptureItemResult` with full item metadata
  - `ToCaptureItemResult()` and `ToCaptureItemResults()` converters
  - Layer information preserved in results

### Changed

- Updated PRD to mark v1.1 Enhanced Capture TUI as complete

---

## [1.0.0] - 2024-12-24

### Added

- First stable release with all Horizon 1 features complete
- Production-ready CLI with init, capture, plan, apply, doctor commands
- Full provider suite: brew, apt, files, git, ssh, shell, runtime, nvim, vscode
- Interactive TUI for init wizard, plan review, apply progress, doctor report
- Lockfile support with intent/locked/frozen modes
- AI advisor integration (OpenAI, Anthropic, Ollama) with BYOK support

### Changed

- Stabilized all public APIs
- Coverage thresholds enforced per domain

---

## [0.1.1] - 2025-12-23

### Fixed

- Fix all golangci-lint warnings (errorlint, revive, testifylint)
- Rename `tui/common` to `tui/ui` to satisfy package naming requirements

### Improved

- Config domain test coverage improved from 65.8% to 86.7%
- Add comprehensive tests for `Raw()` method covering SSH, Runtime, Shell, Nvim, VSCode, APT configurations

---

## [0.1.0] - 2025-12-23

### Added

#### Core Domains

- **Config Domain**: Manifest, Layer, and Target value objects with deep merge semantics and provenance tracking
- **Compiler Domain**: StepGraph with DAG compilation and topological ordering
- **Execution Domain**: Step interface with Check/Plan/Apply lifecycle and DAG scheduler
- **Lock Domain**: Lockfile management with integrity verification (SHA256/SHA512) and resolution modes
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
- `preflight completion` - Shell completion generation

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

### Technical Details

- Built with Go 1.23
- Domain-Driven Design architecture
- Test-Driven Development with >80% coverage requirement per domain

---

[1.3.0]: https://github.com/felixgeelhaar/preflight/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/felixgeelhaar/preflight/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/felixgeelhaar/preflight/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/felixgeelhaar/preflight/compare/v0.1.1...v1.0.0
[0.1.1]: https://github.com/felixgeelhaar/preflight/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/felixgeelhaar/preflight/releases/tag/v0.1.0
