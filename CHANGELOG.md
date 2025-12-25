# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [3.1.0] - 2024-12-25

### Added
- **Signature Verification**: Cryptographic verification of catalog publishers
  - GPG signature support for catalog manifests
  - SSH key signature verification (ED25519)
  - Sigstore integration placeholder for keyless signing

- **Trust Management CLI**: Commands for managing trusted publishers
  - `preflight trust list`: List all trusted keys with stats
  - `preflight trust add <keyfile>`: Add SSH or GPG public key
  - `preflight trust remove <keyid>`: Remove key from trust store
  - `preflight trust show <keyid>`: Display detailed key information

- **Trust Levels**: Hierarchical trust system for catalogs
  - `builtin`: Embedded in binary (highest trust)
  - `verified`: Signed by known publisher (GPG/SSH/Sigstore)
  - `community`: Hash-verified with user approval
  - `untrusted`: No verification (requires explicit flag)

- **TrustStore**: Persistent storage for trusted keys
  - JSON-based storage in `~/.preflight/trust.json`
  - Key expiration support
  - Publisher metadata (name, email, key type)
  - Trust level per key

### New Domain Components
- **Signature**: Cryptographic signature with type, key ID, and publisher info
- **Publisher**: Publisher identity with name, email, and key reference
- **TrustedKey**: Public key with trust level, fingerprint, and expiration
- **TrustStore**: Trusted key storage with persistence
- **ED25519Verifier**: SSH key signature verification
- **VerificationResult**: Structured verification outcome

## [3.0.0] - 2024-12-25

### Added
- **External Catalog Support**: Load catalogs from URLs or local paths
  - `preflight catalog add <url-or-path>`: Add external catalog source
  - `preflight catalog add --local ./path`: Add local catalog directory
  - `preflight catalog add --name <name>`: Specify custom catalog name
  - `preflight catalog list`: List all registered catalogs with stats
  - `preflight catalog remove <name>`: Remove external catalog
  - `preflight catalog verify [name]`: Verify catalog integrity
  - `preflight catalog audit <name>`: Security audit for catalogs

- **Catalog Manifest & Integrity**:
  - SHA256 integrity hashes for all catalog files
  - Manifest with version, author, description, license, repository
  - Automatic integrity verification on load
  - Cache with offline support

- **Security Auditor**: Pattern-based security scanning for catalogs
  - Detects remote code execution patterns (curl|sh, wget|bash)
  - Flags privilege escalation (sudo, doas)
  - Warns about destructive operations (rm -rf /, chmod 777)
  - Scans for hardcoded secrets (API keys, tokens, passwords)
  - Finds unsafe shell patterns (eval, insecure permissions)
  - Severity levels: critical, high, medium, low, info

### New Domain Components
- **Source**: Catalog source types (builtin, url, local) with validation
- **Manifest**: Integrity manifest with SHA256 file hashes
- **Registry**: Multi-catalog management with preset/pack lookup
- **ExternalLoader**: URL and local catalog loading with caching
- **Auditor**: Security rule engine with pattern matching

## [2.6.0] - 2024-12-25

### Added
- **Plugin Marketplace**: Community package registry for presets, capability packs, and layer templates
  - `preflight marketplace search <query>`: Search for packages by name, keyword, or author
  - `preflight marketplace install <package>[@version]`: Download and install packages with integrity verification
  - `preflight marketplace uninstall <package>`: Remove installed packages
  - `preflight marketplace update [package]`: Update one or all packages to latest versions
  - `preflight marketplace list [--check-updates]`: List installed packages with update status
  - `preflight marketplace info <package>`: View detailed package information
  - SHA256 checksum verification for all downloaded packages
  - Local cache with configurable TTL for offline support
  - Provenance tracking (author, repository, license, verification status)
  - Package types: preset, capability-pack, layer-template

### New Domain
- **marketplace domain**: Complete marketplace implementation
  - `Package`, `PackageID`, `PackageVersion` types with validation
  - `Index` for registry search, filtering, and statistics
  - `Cache` for local caching with TTL and checksum verification
  - `Client` for HTTP registry access with error handling
  - `Service` for orchestrating install, update, and uninstall operations

## [2.5.0] - 2024-12-25

### Added
- **Chocolatey Provider**: Windows package manager integration
  - Package installation with version pinning support
  - WSL support via `choco.exe` interop from WSL environments
  - Upgrade and uninstall operations
  - Package validation against Chocolatey naming conventions
  - Automatic Chocolatey installation check

- **VS Code Remote-WSL Integration**: Development inside WSL from Windows
  - `RemoteWSLSetupStep`: Installs `ms-vscode-remote.remote-wsl` extension
  - `RemoteWSLExtensionStep`: Installs extensions in WSL remote context
  - `RemoteWSLSettingsStep`: Manages WSL-specific VS Code settings
  - Distro-specific targeting (e.g., `Ubuntu-22.04`)
  - Platform-aware command selection (`code` vs `code.exe` in WSL)

### Changed
- VSCode provider now accepts platform parameter for WSL-aware compilation
- Platform detection integrated into app initialization

## [2.4.0] - 2024-12-25

### Added
- **WSL/Windows Support**: Cross-platform support for Windows and WSL environments
  - New `platform` domain with OS, architecture, and environment detection
  - WSL detection (WSL1/WSL2) with distro and mount path identification
  - Path translation utilities between Windows and WSL formats

- **Windows Package Managers**:
  - **winget provider**: Windows Package Manager integration
    - Package installation with `Publisher.PackageName` format
    - Version pinning and source selection (winget, msstore)
    - WSL support via `winget.exe` interop
  - **scoop provider**: Scoop package manager integration
    - Bucket management (add, custom URL)
    - Package installation with bucket specification
    - WSL support via `scoop.cmd` interop

- **Windows Junction Support**: Enhanced dotfile linking for Windows
  - Junctions for directories (no admin privileges required)
  - Symlinks for files (may require admin)
  - `CreateLink()` method auto-selects appropriate link type
  - `IsJunction()` method for junction detection

### Changed
- Files provider now uses `CreateLink()` instead of `CreateSymlink()` for platform-aware linking
- Step ID pattern updated to allow dots for winget package IDs
- Updated PRD to mark v2.4 WSL/Windows Support as complete

## [2.3.0] - 2024-12-24

### Added
- **Organization Policy Engine**: Enterprise-grade policy enforcement
  - Policy rules with allow/deny/require actions
  - Pattern matching for providers and packages
  - Policy validation with detailed error messages
  - Policy inheritance and override support

## [2.2.0] - 2024-12-24

### Added
- **Docker Provider**: Container and compose management
  - Container creation and lifecycle management
  - Docker Compose file deployment
  - Image pulling and versioning
  - Volume and network configuration

## [2.1.0] - 2024-12-24

### Added
- **Plugin System**: Extensible provider architecture
  - Plugin discovery and loading
  - Plugin lifecycle management (init, compile, cleanup)
  - Plugin configuration schema validation
  - Security sandboxing for untrusted plugins

## [2.0.0] - 2024-12-24

### Added
- **Validate Command**: Configuration validation without applying
  - Schema validation for all config sections
  - Provider-specific validation rules
  - Detailed error messages with suggestions

- **Policy Constraints**: Declarative constraints for configuration
  - Package version constraints
  - Provider restrictions
  - Target-specific rules

### Changed
- Major version bump for breaking API changes
- Reorganized internal package structure

## [1.4.0] - 2024-12-24

### Added
- **Rollback Command**: Restore files from automatically created snapshots
  - `preflight rollback` lists available snapshots with ID, date, age, file count
  - `preflight rollback --to <id>` restores specific snapshot (supports partial ID matching)
  - `preflight rollback --latest` restores most recent snapshot
  - `preflight rollback --dry-run` previews restoration without applying
  - Human-readable age formatting (mins, hours, days, weeks)
  - Confirmation prompt before restoration

- **Layer Preview Before Commit**: Preview generated YAML before writing to disk
  - New `stepPreview` step in init wizard between confirm and complete
  - File tabs for navigating multiple layer files (h/l or ←/→)
  - YAML syntax highlighting for keys, values, booleans, numbers, comments
  - Scrollable content with j/k or ↑/↓
  - Quick file selection with number keys 1-9
  - Enter confirms, Esc cancels
  - `RunLayerPreview()` public API for standalone usage

- **TUI Conflict Resolution**: Interactive resolution for three-way merge conflicts
  - Side-by-side diff view showing ours (config) vs theirs (file)
  - Color-coded differences between versions
  - Navigate conflicts with n/p (next/previous)
  - Resolve with o/t/b (ours/theirs/base) for current conflict
  - Bulk resolve with O/T/B (uppercase) for all conflicts
  - Scroll content with j/k
  - Optional base content display
  - Progress indicator showing resolved count
  - Auto-advance to next unresolved conflict after resolution
  - `RunConflictResolution()` public API

### Changed
- Init wizard now shows layer preview before writing config files
- Updated PRD to mark v1.4 UX Polish & Rollback as complete

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

## [0.1.1] - 2025-12-23

### Fixed
- Fix all golangci-lint warnings (errorlint, revive, testifylint)
- Rename `tui/common` to `tui/ui` to satisfy package naming requirements

### Improved
- Config domain test coverage improved from 65.8% to 86.7%
- Add comprehensive tests for `Raw()` method covering SSH, Runtime, Shell, Nvim, VSCode, APT configurations

## [0.1.0] - 2025-12-23

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

[Unreleased]: https://github.com/felixgeelhaar/preflight/compare/v3.1.0...HEAD
[3.1.0]: https://github.com/felixgeelhaar/preflight/compare/v3.0.0...v3.1.0
[3.0.0]: https://github.com/felixgeelhaar/preflight/compare/v2.6.0...v3.0.0
[2.6.0]: https://github.com/felixgeelhaar/preflight/compare/v2.5.0...v2.6.0
[2.5.0]: https://github.com/felixgeelhaar/preflight/compare/v2.4.0...v2.5.0
[2.4.0]: https://github.com/felixgeelhaar/preflight/compare/v2.3.0...v2.4.0
[2.3.0]: https://github.com/felixgeelhaar/preflight/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/felixgeelhaar/preflight/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/felixgeelhaar/preflight/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/felixgeelhaar/preflight/compare/v1.4.0...v2.0.0
[1.4.0]: https://github.com/felixgeelhaar/preflight/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/felixgeelhaar/preflight/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/felixgeelhaar/preflight/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/felixgeelhaar/preflight/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/felixgeelhaar/preflight/compare/v0.1.1...v1.0.0
[0.1.1]: https://github.com/felixgeelhaar/preflight/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/felixgeelhaar/preflight/releases/tag/v0.1.0
