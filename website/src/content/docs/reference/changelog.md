---
title: Changelog
description: All notable changes to Preflight.
---

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.8.0] - 2024-12-24

### Added

- **Hands-On Tutorials**: Three interactive tutorials with practice commands
  - `nvim-basics`: Learn Neovim fundamentals with hands-on exercises
    - Modes (Normal, Insert, Visual, Command)
    - Essential movements and editing commands
    - Practice files and verify commands
  - `git-workflow`: Practice Git commands and conventional commits
    - Repository initialization and configuration
    - Commit messages and branch workflow
    - Preflight Git integration
  - `shell-customization`: Configure your shell environment
    - Aliases, functions, and environment variables
    - Oh-My-Zsh and Starship prompt setup
    - Preflight shell provider integration

- **Enhanced Tour Display**: Visual distinction for hands-on content
  - ⌨️ indicator for hands-on sections
  - Styled command blocks with "Try this command" header
  - 💡 Hint sections for guidance
  - ✓ Verify commands to check completion
  - 🛠️ indicator for hands-on topics in menu

---

## [1.7.0] - 2024-12-24

### Added

- **New Presets**: 10 new presets across 5 categories
  - Fonts: `fonts:nerd-essential`, `fonts:nerd-complete`
  - SSH: `ssh:basic`, `ssh:github`, `ssh:multi-identity`
  - Docker: `docker:basic`, `docker:kubernetes`
  - Runtime: `runtime:mise-node`, `runtime:mise-polyglot`
  - Brew: `brew:cli-essentials`, `brew:dev-tools`

- **New Capability Packs**: 6 new role-based packs
  - `mobile-developer`: React Native and Flutter development
  - `qa-engineer`: Testing frameworks and automation tools
  - `cloud-architect`: Multi-cloud infrastructure tooling
  - `java-developer`: JVM ecosystem development
  - `security-engineer`: Security testing and compliance
  - `technical-writer`: Documentation and static site generation

### Changed

- Catalog expanded from 12 to 22 presets, 8 to 14 capability packs

---

## [1.6.0] - 2024-12-24

### Added

- **Interactive Tour System**: Full TUI for guided learning
  - `preflight tour` opens topic selection menu
  - `preflight tour <topic>` starts specific topic
  - `preflight tour --list` lists available topics
  - 6 comprehensive topics: basics, config, layers, providers, presets, workflow
  - Section-based navigation with keyboard controls (n/p, h/l, 1-9, g/G)
  - Code examples with syntax highlighting
  - Next topic suggestions for learning path

- **Tour Completion Tracking**: Persistent progress tracking
  - Progress saved to `~/.preflight/tour-progress.json`
  - Per-topic and per-section completion tracking
  - Progress indicators in topic list (✓ completed, % in progress)
  - Overall progress display in tour menu

### Changed

- Tour command now launches interactive TUI instead of static output
- Updated CLI commands documentation with tour navigation keys

---

## [1.5.0] - 2024-12-24

### Added

- **Integration Tests**: Comprehensive Go-based test suite
  - Init tests for configuration initialization and parsing
  - Capture tests for system capture and review workflows
  - Doctor tests for drift detection and fix scenarios
  - Rollback tests for snapshot creation and restoration
  - Discover tests for pattern detection in dotfile repositories
  - Full workflow tests for end-to-end plan/apply cycles

- **Performance Benchmarks**: New benchmark suite for critical paths
  - `loader_bench_test.go`: Config loading benchmarks (LoadManifest ~21µs)
  - `merger_bench_test.go`: Layer merging benchmarks (10 layers ~259µs)
  - `step_graph_bench_test.go`: Step graph operations (TopologicalSort ~16µs, Get ~7.6ns)

- **Migration Guide**: Step-by-step migration documentation
  - From manual dotfiles with git and symlinks
  - From chezmoi with template mapping
  - From yadm with class-to-layer conversion
  - From GNU stow with package mapping
  - From Ansible/shell scripts with task conversion

- **Troubleshooting Guide**: Comprehensive debugging documentation
  - Common configuration errors and solutions
  - Apply and doctor error resolution
  - Debugging techniques (verbose, dry-run, explain modes)
  - Recovery procedures (snapshots, manual recovery, reset)
  - Provider-specific issues (Homebrew, Git, SSH, Neovim)

- **Terminal Demo GIFs**: Animated demos for documentation
  - `init-wizard.gif`: Interactive init flow
  - `capture-review.gif`: Capture with TUI review
  - `plan-apply.gif`: Plan preview and apply
  - `doctor-fix.gif`: Doctor detection and fix
  - `rollback.gif`: Snapshot rollback
  - Demo recording script (`scripts/record-demos.sh`)
  - Asciinema cast files for reproducible recordings

### Changed

- README now includes demo GIF in hero section
- Quick Start guide includes inline demo GIFs for each step
- CLI Commands page includes rollback demo GIF
- Sidebar navigation includes Migration and Troubleshooting guides

---

## [1.4.0] - 2024-12-24

### Added

- **Rollback Command**: Restore files from automatic snapshots
  - `preflight rollback` lists available snapshots with ID, date, age, file count
  - `preflight rollback --to <id>` restores specific snapshot (supports partial ID matching)
  - `preflight rollback --latest` restores most recent snapshot
  - `preflight rollback --dry-run` previews restoration without applying
  - Human-readable age formatting (mins, hours, days, weeks)
  - Confirmation prompt before restoration

- **Layer Preview Before Commit**: Preview generated YAML before writing to disk
  - New preview step in init wizard between confirm and complete
  - File tabs for navigating multiple layer files (h/l or ←/→)
  - YAML syntax highlighting for keys, values, booleans, numbers, comments
  - Scrollable content with j/k or ↑/↓
  - Quick file selection with number keys 1-9
  - `RunLayerPreview()` public API for standalone usage

- **TUI Conflict Resolution**: Interactive resolution for three-way merge conflicts
  - Side-by-side diff view showing ours (config) vs theirs (file)
  - Color-coded differences between versions
  - Navigate conflicts with n/p (next/previous)
  - Resolve with o/t/b (ours/theirs/base) for current conflict
  - Bulk resolve with O/T/B (uppercase) for all conflicts
  - Scroll content with j/k
  - Optional base content display
  - `RunConflictResolution()` public API

- **Enhanced Shell Completions**: Custom flag completions with descriptions
  - `--config` completes with .yaml/.yml files
  - `--ai-provider` completes with openai/anthropic/ollama
  - `--mode` completes with intent/locked/frozen

- **New Catalog Presets and Capability Packs**:
  - Presets: vscode:minimal, vscode:full, terminal:tmux, terminal:alacritty
  - Packs: python-developer, rust-developer, data-scientist, writer, full-stack

### Changed

- Init wizard now shows layer preview before writing config files
- Updated PRD to mark v1.4 UX Polish & Rollback as complete

---

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

[1.8.0]: https://github.com/felixgeelhaar/preflight/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/felixgeelhaar/preflight/compare/v1.6.0...v1.7.0
[1.6.0]: https://github.com/felixgeelhaar/preflight/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/felixgeelhaar/preflight/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/felixgeelhaar/preflight/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/felixgeelhaar/preflight/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/felixgeelhaar/preflight/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/felixgeelhaar/preflight/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/felixgeelhaar/preflight/compare/v0.1.1...v1.0.0
[0.1.1]: https://github.com/felixgeelhaar/preflight/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/felixgeelhaar/preflight/releases/tag/v0.1.0
