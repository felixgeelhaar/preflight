# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Split Strategies for Capture**: New `--split-by` flag for flexible layer organization
  - `category` (default): Fine-grained categories (base, dev-go, security, containers)
  - `language`: By programming language (go, node, python, rust, java)
  - `stack`: By tech stack role (frontend, backend, devops, data, security)
  - `provider`: By provider name (brew, git, shell, vscode)
  - Example: `preflight capture --all --split-by language`

- **AI-Assisted Package Categorization**: Optional AI enhancement for uncategorized packages
  - Uses configured AI provider (OpenAI, Anthropic, Ollama) when available
  - Graceful degradation: falls back to "misc" layer if AI unavailable
  - AI suggests layer assignments with reasoning for unknown packages

### Fixed

- **Step ID Validation**: Allow `@` symbol in step IDs for versioned packages
  - Fixes panic when capturing packages like `go@1.24`, `python@3.12`, `openssl@3`
  - Updated regex pattern to include `@` in valid characters

## [4.0.1] - 2025-12-30

### Added

- **Fleet Management Guide**: Comprehensive documentation for SSH-based fleet operations
  - Quick start, inventory configuration, targeting syntax
  - Commands reference (list, ping, plan, apply, status, diff)
  - Execution strategies, policies, and maintenance windows
  - SSH configuration and troubleshooting

- **Demo Recordings**: VHS tape files and GIF recordings for v4 commands
  - `sync.gif`: Multi-machine synchronization workflow
  - `agent.gif`: Background agent management
  - `fleet.gif`: Fleet management over SSH

- **SSH Integration Tests**: Comprehensive test coverage for fleet operations
  - Transport tests: ping, connect, command execution, file transfer
  - Executor tests: parallel/rolling strategies, step dependencies
  - Run with: `PREFLIGHT_SSH_TEST=1 go test -tags=integration ./internal/domain/fleet/...`

- **CLI Test Coverage**: Additional tests for v4 commands
  - sync, agent, fleet, conflicts command tests
  - Flag defaults, subcommands, and shorthand validation

## [4.0.0] - 2025-12-30

### Added

#### Phase 1: Multi-Machine Sync
- **Version Vectors**: Causal ordering for distributed configuration changes
  - Vector clocks track changes across machines
  - Compare() returns Before, After, Concurrent, or Equal relations
  - Merge() for combining vectors during sync

- **Machine Identity**: Stable UUID per machine
  - Auto-generated machine ID stored in `~/.preflight/machine-id`
  - Hostname and metadata tracking
  - Lineage tracking for machine history

- **Conflict Detection**: Three-way merge for lockfile conflicts
  - Detect concurrent changes to the same package
  - Conflict types: VersionMismatch, LocalOnly, RemoteOnly, BothModified
  - Base, local, and remote state comparison

- **Conflict Resolution**: Multiple resolution strategies
  - Manual: User selects resolution
  - Newest: Use most recently modified version
  - Local: Prefer local changes
  - Remote: Prefer remote changes
  - Auto: Automatic resolution for non-conflicting changes

- **Lockfile v2 Format**: Extended lockfile with sync metadata
  - Version vector embedded in lockfile
  - Machine lineage tracking
  - Per-package provenance (modified_by, vector_at_change)

- **CLI Commands**:
  - `preflight sync` - Synchronize configuration across machines
  - `preflight conflicts` - View and resolve sync conflicts
  - `preflight conflicts resolve` - Interactive conflict resolution

#### Phase 2: Background Agent
- **Agent Domain**: Scheduled reconciliation and drift monitoring
  - Agent state machine (idle, running, paused, stopped)
  - Health status tracking with last check timestamps
  - Reconciliation result history

- **Scheduling**: Flexible schedule configuration
  - Interval-based: `30m`, `1h`, `6h`
  - Cron expressions: `0 */30 * * *`
  - On-demand trigger via IPC

- **Remediation Policies**: Configurable remediation behavior
  - `notify`: Alert user about drift (default)
  - `auto`: Automatically remediate all drift
  - `approved`: Require approval for remediation
  - `safe`: Only remediate safe changes

- **Service Integration**: Platform-specific daemon support
  - macOS: launchd plist (`~/Library/LaunchAgents/com.preflight.agent.plist`)
  - Linux: systemd user service (`~/.config/systemd/user/preflight-agent.service`)

- **IPC Communication**: Unix socket for agent control
  - Status queries
  - Stop/pause/resume commands
  - Approval requests

- **CLI Commands**:
  - `preflight agent start [--foreground] [--schedule 30m]`
  - `preflight agent stop [--force]`
  - `preflight agent status [--json] [--watch]`
  - `preflight agent install` - Install system service
  - `preflight agent uninstall` - Remove system service
  - `preflight agent approve <request-id>` - Approve remediation

#### Phase 3: Fleet Management
- **Fleet Domain**: Multi-machine configuration management over SSH
  - Host entity with SSH config, tags, groups, status
  - Group value object with host patterns and policies
  - Tag value object for categorization
  - Inventory aggregate root

- **Targeting System**: Flexible host selection
  - `@all` or `*` - Select all hosts
  - `@groupname` - Select by group
  - `tag:tagname` - Select by tag
  - `host-*` - Glob pattern matching
  - `~regex~` - Regex pattern matching
  - `!pattern` - Exclude matching hosts

- **SSH Transport**: Secure remote execution
  - Connection pooling for efficiency
  - Configurable timeouts
  - Proxy jump support
  - Identity file selection

- **Execution Strategies**: Multiple execution patterns
  - `parallel` - Execute on all hosts concurrently (up to maxParallel)
  - `rolling` - Execute in batches with health checks
  - `canary` - Test on canary host before rolling out

- **Remote Steps**: Idempotent remote execution
  - Check/Apply pattern for all operations
  - Package installation helpers (apt, brew, dnf, yum)
  - File write and symlink steps
  - Custom command execution

- **Fleet Inventory** (`fleet.yaml`):
  ```yaml
  version: 1
  hosts:
    workstation-01:
      hostname: dev-ws-01.internal
      user: admin
      port: 22
      tags: [workstation, darwin]
      groups: [dev-team]
  groups:
    production:
      hosts: [server-prod-*]
      policies: [require-approval]
  defaults:
    ssh_timeout: 30s
    max_parallel: 10
  ```

- **CLI Commands**:
  - `preflight fleet list [--target TARGET] [--json]`
  - `preflight fleet ping [--target TARGET] [--timeout 30s]`
  - `preflight fleet plan [--target TARGET]`
  - `preflight fleet apply [--target TARGET] [--strategy rolling]`
  - `preflight fleet status`

### New Domains
- **sync domain** (`internal/domain/sync/`): Version vectors, conflict detection, resolution
- **agent domain** (`internal/domain/agent/`): Agent lifecycle, scheduling, remediation
- **fleet domain** (`internal/domain/fleet/`): Host, group, tag, inventory management
- **fleet/targeting** (`internal/domain/fleet/targeting/`): Pattern matching, selectors
- **fleet/transport** (`internal/domain/fleet/transport/`): SSH and local transports
- **fleet/execution** (`internal/domain/fleet/execution/`): Remote steps, fleet executor

### Breaking Changes
- Lockfile format upgraded to v2 (automatic migration on first sync)
- New required configuration section for multi-machine sync

### Test Coverage
- sync domain: 95%+
- agent domain: 90%+
- fleet domain: 99%
- fleet/targeting: 99.3%
- fleet/execution: 91.2%

---

## [3.4.0] - 2025-12-30

### Added
- **Security Audit Logging (PRD 15.5)**: Complete audit trail for plugin operations
  - Event types: catalog_installed/removed/verified, plugin_installed/executed, trust_added/removed, signature_verified/failed, capability_granted/denied, sandbox_violation, security_audit
  - Severity levels: info, warning, error, critical
  - File-based JSON logging with rotation and cleanup
  - Query builder with fluent API for filtering by type, severity, catalog, plugin, user, time range
  - CLI commands: `preflight audit`, `preflight audit summary`, `preflight audit security`, `preflight audit clean`

- **Phase 1-4 Enterprise Features**:
  - CLI commands for configuration management (`preflight config show/set/list`)
  - Compliance command (`preflight compliance check`)
  - Marketplace recommendations (`preflight marketplace recommend`)
  - Catalog verification (`preflight catalog verify`)
  - 8 new providers for expanded platform support
  - Phase 4 capability packs for enterprise workflows

- **TUI Enhancements**: Extended editor list with installed detection

- **Documentation**: Demo recordings and GIFs for all CLI commands

### Fixed
- Race conditions in CLI and config tests (mutex synchronization for stdout/stderr capture)
- golangci-lint warnings across test files
- Test failures in CLI commands

### Changed
- **Test Coverage Improvements**:
  - Audit domain: 56.1% (isolated: 90%+)
  - CLI domain: 40.6%
  - Provider tests expanded for comprehensive coverage

## [3.3.2] - 2025-12-26

### Fixed
- Plugin validation test failures
- Coverage threshold adjustments for CI compatibility

## [3.3.1] - 2025-12-25

### Changed
- **Test Coverage Improvements**: Improved platform and marketplace domain coverage
  - Platform domain: 39% → 47.6% with WSL path translation tests
  - Marketplace domain: 76% → 83.9% with Update/UpdateAll/CheckUpdates tests

## [3.3.0] - 2025-12-25

### Added
- **Complete Plugin Ecosystem**: A+ product grade with full feature set
  - Git clone installer for remote plugin installation
  - Dependency resolution with topological sorting
  - Plugin validation CLI (`preflight plugin validate`)
  - Trust signals in search results (stars, activity, signature status)
  - Upgrade mechanism (`preflight plugin upgrade`)

- **WASM Plugin Sandbox**: Complete isolation for untrusted plugins using WebAssembly
  - Wazero runtime (pure Go, no CGO) for deterministic plugin execution
  - Plugin runs in isolated VM with no direct system access
  - SHA256 checksum verification of plugin modules

- **Sandbox Modes**: Three isolation levels for different trust scenarios
  - `ModeFull`: Complete isolation, no side effects (preview/audit unknown plugins)
  - `ModeRestricted`: Limited to declared capabilities (normal operation)
  - `ModeTrusted`: Full access like builtin (verified publishers only)

- **Resource Limits**: Prevent plugin resource abuse
  - `MaxMemoryBytes`: Memory allocation cap for plugins
  - `MaxCPUTime`: CPU time limit for execution
  - `MaxFileDescriptors`: File descriptor limit
  - `MaxOutputBytes`: Output size limit

- **Host Function Bindings**: Controlled access to host services
  - Logging functions: `log_info`, `log_warn`, `log_error`
  - File operations (requires `files:read`/`files:write` capabilities)
  - Shell execution (requires `shell:execute` capability)
  - Network access (requires `network:fetch` capability)
  - Host services interface for extensible bindings

- **Plugin Manifest**: YAML-based plugin declarations
  - Plugin metadata: ID, name, version, description, author
  - Module path with SHA256 checksum verification
  - Capability declarations with justifications
  - Optional capability support

- **Plugin Loader & Executor**: Load and run plugins safely
  - `Loader`: Load plugin manifests and verify checksums
  - `Executor`: Create sandbox, validate plugin, and execute
  - `ValidatePlugin`: Validate without execution
  - List available plugins in a directory

### New Domain
- **sandbox domain**: Complete WASM isolation infrastructure
  - `Plugin`, `PluginManifest`, `ManifestCapability` types
  - `Sandbox`, `Runtime` interfaces for runtime abstraction
  - `WazeroRuntime`, `WazeroSandbox` implementations
  - `HostServices`, `HostFunction`, `HostFS`, `HostShell`, `HostHTTP`, `HostLogger` interfaces
  - `Config`, `ResourceLimits`, `Mode` for sandbox configuration
  - `Loader`, `Executor` for plugin lifecycle management
  - `NullFileSystem`, `NullShell`, `NullHTTP` for full isolation
  - 80%+ test coverage

### Example Plugin Manifest
```yaml
# plugin.yaml
id: my-plugin
name: My Plugin
version: 1.0.0
description: A custom plugin for Preflight
author: Developer Name
module: plugin.wasm
checksum: sha256:abc123...

capabilities:
  - name: files:read
    justification: Read configuration files
  - name: shell:execute
    justification: Run validation commands
    optional: true
```

## [3.2.0] - 2024-12-25

### Added
- **Capability-Based Permissions**: Fine-grained permission system for plugins
  - Capability types: `files:read`, `files:write`, `packages:brew`, `packages:apt`, `shell:execute`, `network:fetch`, `secrets:read`, `secrets:write`, `system:modify`
  - Dangerous capability detection requiring explicit approval
  - Wildcard matching (`files:*` matches all file operations)

- **Policy Enforcement**: Control which capabilities are allowed
  - Grant, block, and approve capabilities
  - Default, full-access, and restricted policy presets
  - Policy validation with violation reporting
  - Pending approval tracking for dangerous capabilities

- **Plugin Requirements**: Capability declarations for plugins
  - Required and optional capability support
  - Justification for capability requests
  - Validation against policy with detailed results
  - Summary reporting for requirement validation

- **Content Security Policy (CSP)**: Pattern-based command validation
  - Deny rules for dangerous patterns (curl|sh, sudo, rm -rf /)
  - Warning rules for potentially problematic patterns (eval, command substitution)
  - Default CSP with common security rules
  - Strict CSP mode for untrusted content

- **Security Configuration**: YAML-based security settings
  - `blocked_capabilities`: List of denied capabilities
  - `csp_deny`: Custom deny patterns with reasons
  - `csp_warn`: Custom warning patterns
  - `require_approval`: Toggle for dangerous capability approval

### New Domain
- **capability domain**: Complete capability-based permission system
  - `Capability`, `Category`, `Action` types with parsing
  - `Set` for capability collections with set operations
  - `Policy` for enforcement with grant/block/approve
  - `Requirements` for plugin capability declarations
  - `CSP` for content security policy rules
  - `SecurityConfig` for YAML configuration
  - 97.5% test coverage

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

[Unreleased]: https://github.com/felixgeelhaar/preflight/compare/v4.0.0...HEAD
[4.0.0]: https://github.com/felixgeelhaar/preflight/compare/v3.4.0...v4.0.0
[3.4.0]: https://github.com/felixgeelhaar/preflight/compare/v3.3.2...v3.4.0
[3.3.2]: https://github.com/felixgeelhaar/preflight/compare/v3.3.1...v3.3.2
[3.3.1]: https://github.com/felixgeelhaar/preflight/compare/v3.3.0...v3.3.1
[3.3.0]: https://github.com/felixgeelhaar/preflight/compare/v3.2.0...v3.3.0
[3.2.0]: https://github.com/felixgeelhaar/preflight/compare/v3.1.0...v3.2.0
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
