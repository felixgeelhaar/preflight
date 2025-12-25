# Preflight — Full Product Requirements Document

---

## 1. Product Overview

### 1.1 Product Vision

Preflight enables anyone to compile a reproducible, explainable, and portable workstation.

A workstation should not be:
- a one-off snowflake
- a pile of scripts
- or an opaque installer

Preflight treats workstation setup as a deterministic compilation problem, producing outcomes that are:
- repeatable
- inspectable
- safe
- and fully owned by the user.

### 1.2 Product Mission

Make workstation setup boringly reliable, easy to reason about, and safe for both personal and work machines — without requiring deep technical knowledge.

### 1.3 Product Principles

| Principle | Description |
|-----------|-------------|
| Compiler first, advisor second | Core value is deterministic compilation, AI is optional enhancement |
| Local-first, offline-capable | No cloud dependency, works without internet after initial setup |
| No execution without a plan | Always show what will change before making changes |
| Explain everything | Every action has "why", tradeoffs, and documentation links |
| User owns all outputs | Config is portable, inspectable, git-native |
| AI is optional and advisory | AI never executes, only suggests |
| Non-engineers are first-class users | Guided experiences for all skill levels |

---

## 2. Problem Space

### 2.1 User Problems

| Problem | Who |
|---------|-----|
| New machines are slow and error-prone to set up | Everyone |
| Dotfiles are fragile and undocumented | Engineers |
| Work vs personal separation is manual and unsafe | Professionals |
| Existing tools are too complex (Nix, Ansible) | Non-experts |
| Existing tools are too shallow (scripts, installers) | Power users |
| No explainability or learning | Newcomers |
| No safe iteration or rollback | Teams |

### 2.2 Existing Solutions & Gaps

| Solution | Gap |
|----------|-----|
| Dotfiles | Not declarative, not explainable |
| Brewfiles | No structure, no profiles |
| Nix | Powerful, but inaccessible |
| MDMs | Heavy, centralized, enterprise-only |
| IDE sync | Editor-only, not system-wide |

**Preflight fills the gap between raw scripts and heavy infra tools.**

---

## 3. Target Users & Personas

### 3.1 Primary Personas

#### A. Technical Builder
- Uses dotfiles, Brew, custom configs
- Wants determinism and reproducibility
- Pain: complexity, drift, undocumented state

#### B. Knowledge Worker (Non-Engineer)
- Designer, PM, researcher
- Wants "a good setup" without YAML or scripts
- Pain: lack of guidance and confidence

### 3.2 Secondary Personas (Later Horizons)
- Contractors / freelancers
- OSS maintainers
- Small teams without IT
- Security-conscious professionals

---

## 4. Product Scope by Horizon

### Horizon 1 — Compiler MVP (v0.1–v1)

**Goal:** Make Preflight real and trusted.

#### Core Capabilities
- CLI + TUI
- Config model (manifest + layers)
- Targets (work / personal / roles)
- `init`, `capture`, `plan`, `apply`, `doctor`
- Deterministic execution
- Lockfile (intent | locked | frozen)
- Dotfile generation & management
- Brew + Files + Neovim providers
- BYOK AI (advisory only)
- GitHub repo bootstrap

#### Success Criteria
- Fresh machine → working setup in < 30 minutes
- Re-running apply causes no changes
- Safe enough to use on a work laptop
- Config remains readable after months

---

### Horizon 2 — Discovery & Learning (v1.1–v2)

**Goal:** Help users find their ideal setup, not just reproduce one.

#### Added Capabilities
- Capability packs ("Go Dev", "Writer", "Designer")
- Editor-agnostic recommendations (Neovim, Cursor, VS Code)
- Explainability for all suggestions
- Guided "tour" mode
- Profile inference improvements
- More providers (apt, runtime tools, shell)

#### Success Criteria
- New users can choose tools confidently
- Users understand why something is installed
- Non-engineers can iterate without fear

---

### Horizon 3 — Ecosystem & Scale (v2+)

**Goal:** Enable community and organizational reuse.

#### Future Capabilities
- Plugin / capability marketplace
- Signed community packs
- Org baselines (still local-first)
- CI validation of configs
- Windows / WSL support
- Policy-lite constraints ("forbid X on work machines")

> **Note:** Explicitly not an MDM or SaaS control plane

---

## 5. Core Concepts

### 5.1 Layers & Targets

| Concept | Description |
|---------|-------------|
| Layer | Composable unit of intent (base, identity.work, role.go, device.laptop) |
| Target | An ordered list of layers (e.g., work, personal) |

### 5.2 Providers (Modules)

Providers compile config sections into executable steps:

| Provider | Responsibility |
|----------|----------------|
| brew | Homebrew taps, formulae, casks (macOS) |
| apt | Package installation (Linux) |
| files | Dotfile rendering, linking, drift detection |
| git | .gitconfig generation, identity separation |
| ssh | ~/.ssh/config rendering (never exports keys) |
| shell | Shell framework, plugins, themes |
| runtime | rtx/asdf tool version management |
| nvim | Neovim install, preset bootstrap, lazy-lock |
| vscode | Extension install, settings management |
| ai | Advisory only (BYOK) |

### 5.3 Reproducibility Modes

| Mode | Behavior |
|------|----------|
| intent | Install latest compatible versions |
| locked | Prefer lockfile; update lock intentionally |
| frozen | Fail if resolution differs from lock |

---

## 6. User Journeys

### 6.1 Init — Build Mode (New User)

**User story:** "I'm new to Neovim and want a great setup."

**Flow:**
1. `preflight init` opens guided TUI
2. User selects personas/goals (e.g., "Beginner", "Balanced", languages)
3. Preflight proposes presets/capabilities with explanations + links
4. User reviews in TUI (toggle include/exclude)
5. Preflight writes config (preflight.yaml, layers/, dotfiles scaffolding)
6. Optional: `preflight repo init --github` creates a private GitHub repo
7. `preflight apply` installs tools, bootstraps editor, writes lock(s)

**Acceptance Criteria:**
- No config editing required
- Every suggested tool has "why", tradeoffs, demo, docs links
- `preflight doctor` passes after apply

### 6.2 Init — Capture Mode (Existing Machine)

**Flow:**
1. `preflight capture`
2. Detect packages/configs/editors
3. Infer layers (base + identities + roles)
4. TUI review: keep leaves only, move items across layers, accept
5. Write layers + lock snapshot

**Acceptance Criteria:**
- Produces readable layers (not a giant dump)
- Never exports secrets
- Can reproduce on a new machine

### 6.3 Plan → Apply

- `preflight plan` shows actions, diffs, and explanations
- `preflight apply` executes deterministically; prompts unless `--yes`
- Updates lock based on mode and flags

### 6.4 Doctor / Drift

- `preflight doctor` checks packages, dotfiles, editor plugins/extensions, missing secrets
- Offers:
  - `--fix` (converge machine to config)
  - `--update-config` (capture delta into layers)
  - `--dry-run` (preview changes without writing)

---

## 7. Functional Requirements

### 7.1 Configuration
- Declarative YAML
- Layer-based composition
- Deterministic merge rules
- Machine-independent by default

### 7.2 Execution
- Step DAG
- Idempotent operations
- Dry-run everywhere
- Explicit destructive step labeling

### 7.3 Dotfiles

**Modes:**
1. Generated
2. Template + user overrides
3. Bring-your-own (link/validate only)

**Capabilities:**
- Render structured config into files (~/.gitconfig, ~/.ssh/config, shell config)
- Annotate managed sections
- Provide diffs in plan
- Detect drift in doctor
- Snapshot before applying changes
- Three-way merge for conflict resolution

### 7.4 Editors

**Neovim (first-class):**
- Install nvim
- Apply a preset (minimal/balanced/pro)
- Bootstrap plugins headlessly
- Lock via lazy-lock.json (and record in preflight.lock)
- Doctor checks: lock present, required binaries present (rg, fd, formatters, LSP tools)

**VS Code / Cursor (v1-lite):**
- Install extensions by ID
- Apply settings
- Record installed versions in lock (best-effort)
- Doctor checks extension presence and settings drift

### 7.5 AI (BYOK)
- Providers: OpenAI, Anthropic, Ollama, None
- AI outputs must be:
  - Explainable
  - Mapped to a versioned catalog/preset when possible
  - Never executed directly

### 7.6 Repository
- `preflight repo init --github` creates private repo using gh
- Repo contains: config + layers, lockfile, dotfiles, README with bootstrap instructions
- `preflight repo pull` supports new-machine bootstrap

---

## 8. Non-Functional Requirements

### 8.1 Security
- No secrets in config or lock
- Redaction on capture
- Explicit consent for system changes

### 8.2 Reliability
- Safe re-runs
- Partial failure handling
- Clear error states

### 8.3 Performance
- Plan under seconds
- Apply fast, but correctness > speed

### 8.4 Portability
- macOS first
- Linux second
- Windows later (WSL)

---

## 9. UX Requirements

### 9.1 TUI
- Fully usable without mouse
- Full workflow in terminal (no browser required)
- Clear navigation
- Search/filter lists
- Patch-like capture review (include/exclude/move)
- Explanation panel for any item (why, tradeoffs, docs)
- Clear destructive-step labeling

### 9.2 CLI
- Predictable verbs
- Stable interface
- Scriptable output (JSON where relevant)

---

## 10. v1.x Roadmap — Remaining Features

The following v1 PRD requirements are planned for v1.x releases:

### 10.1 Enhanced Capture TUI (v1.1) ✓

Completes PRD 6.2 and 9.1 requirements for capture workflow:
- Search/filter items by name, provider, category ✓
- Layer reassignment — move items between layers in TUI ✓
- Undo/redo for review decisions ✓
- Keyboard shortcuts for power users ✓

### 10.2 Full Dotfile Lifecycle (v1.2) ✓

Completes PRD 7.3 requirements for dotfile management:
- Snapshot before applying changes (automatic backup) ✓
- Hash-based drift detection ✓
- Doctor `--update-config` flag ✓
- Config patch generation from drift ✓
- VS Code settings drift detection ✓

### 10.3 Three-Way Merge (v1.3) ✓

Completes PRD 7.3 requirements for conflict resolution:
- DetectChangeType: Classifies changes as none/ours/theirs/both/same ✓
- ThreeWayMerge: Automatic merge when possible ✓
- Conflict markers with descriptive labels (git/diff3 style) ✓
- ParseConflictRegions: Extract conflicts from marked content ✓
- ResolveAllConflicts: Programmatic conflict resolution ✓

### 10.4 UX Polish & Rollback (v1.4) ✓

Completes PRD 9.1 TUI requirements and adds rollback capability:

#### Layer Preview Before Commit ✓
- Preview generated YAML structure before writing to disk ✓
- TUI screen showing layer files with syntax highlighting ✓
- Confirm/cancel options before finalizing ✓
- Applies to `init` workflow ✓

#### TUI Conflict Resolution ✓
- Interactive conflict resolution when three-way merge has conflicts ✓
- Side-by-side diff view of ours/theirs versions ✓
- Actions: pick ours, pick theirs, pick base ✓
- Navigate between conflict regions with keyboard (n/p) ✓
- Resolve all conflicts at once (O/T/B) ✓
- Scrollable content view for large diffs ✓

#### Rollback Command ✓
- `preflight rollback` — list available snapshots ✓
- `preflight rollback --to <snapshot-id>` — restore specific snapshot ✓
- `preflight rollback --latest` — restore most recent snapshot ✓
- Snapshot metadata display (date, files affected) ✓
- Dry-run mode to preview restoration before applying ✓

---

## 11. Out of Scope (Explicit)

### Never
- Centralized SaaS management
- Mandatory accounts
- Proprietary lock-in

### Deferred (v2+)
- Plugin marketplace
- Org policy and compliance engine
- Remote execution and fleet management
- Continuous background reconciliation

---

## 12. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Becoming "Nix but worse" | Keep scope tight, UX-first |
| Too many providers | Start with few, extensible |
| AI trust issues | Advisory-only, BYOK |
| Complexity creep | Compiler model discipline |

---

## 13. Success Metrics

### v1 (Qualitative)
- A new user can go from zero → working setup in < 30 minutes
- Re-running apply causes no surprises
- Non-engineers can complete init without editing YAML
- Captured config can reproduce on a fresh machine with minimal edits
- Users trust it on their work machines

### v2 (Quantitative)
- Marketplace has ≥50 community-contributed presets
- WSL support enables full workflow on Windows machines
- Org policy adoption by ≥3 teams in beta
- Learning tour completion rate ≥70%
- Time to onboard new user reduced by 50% vs v1

---

## 14. v2 Feature Requirements

### 14.1 Plugin Marketplace ✓
- Registry of community presets, capability packs, and layer templates
- Versioned with integrity verification (SHA256)
- `preflight marketplace search <query>`
- `preflight marketplace install <pack>`
- Local cache for offline use
- Provenance tracking (author, source repo, license)

*Status: Complete (v2.6.0) — Implemented:*
- *Marketplace domain in `internal/domain/marketplace/` with Package, Index, Cache, Client, Service types*
- *Package types: preset, capability-pack, layer-template with validation*
- *SHA256 checksum verification for all downloaded packages*
- *Local cache with configurable TTL for offline support*
- *Provenance tracking (author, repository, license, verification status)*
- *CLI commands: search, install, uninstall, update, list, info (aliases: mp, market)*
- *HTTP client with error handling (rate limiting, auth, server errors)*

### 14.2 Org Policy Engine ✓
- Define constraints in `org-policy.yaml`
- Policies: required packages, forbidden packages, required layers
- Enforcement: warn or block on plan
- No central server — policies distributed via git
- Override mechanism for exceptions with justification

*Status: Complete (v2.3.0) — Implemented in `internal/domain/policy/org.go`, `org_loader.go` with full test coverage. Integrated with `preflight validate --org-policy` command.*

### 14.3 WSL/Windows Support
- Detect WSL environment
- Windows-native package managers: winget, scoop, chocolatey
- Path translation between Windows and WSL
- Dotfile symlink compatibility (Windows junctions)
- VS Code Remote-WSL integration

*Status: Complete (v2.4.0 + v2.5.0) — Implemented:*
- *Platform detection domain in `internal/domain/platform/` with OS, architecture, and environment detection*
- *WSL detection (WSL1/WSL2) with distro and mount path identification*
- *Path translation utilities between Windows and WSL formats*
- *winget provider in `internal/provider/winget/` with WSL interop via `winget.exe`*
- *scoop provider in `internal/provider/scoop/` with bucket management and WSL interop via `scoop.cmd`*
- *Windows junctions for directories (no admin required), symlinks for files*
- *Files provider updated to use `CreateLink()` for platform-aware linking*
- *chocolatey provider in `internal/provider/chocolatey/` with version pinning and WSL interop via `choco.exe` (v2.5.0)*
- *VS Code Remote-WSL integration with RemoteWSLSetupStep, RemoteWSLExtensionStep, RemoteWSLSettingsStep (v2.5.0)*

### 14.4 Learning Tours ✓ COMPLETE
- Interactive mode: `preflight tour <topic>`
- Topics: nvim-basics, git-workflow, shell-customization
- AI-powered personalization (optional)
- Step-by-step with checkpoints
- Completion tracking

**Implementation Notes:**
- *Full TUI tour experience with topic menu and section navigation*
- *Conceptual tours: basics, config, layers, providers, presets, workflow*
- *Hands-on tutorials: nvim-basics, git-workflow, shell-customization*
- *Progress tracking persisted to ~/.preflight/tour-progress.json*
- *Keyboard navigation: Enter to advance, Esc to go back, q to quit*

---

## 15. v3 Feature Requirements — Plugin Ecosystem & Security

v3 introduces external plugin support with a defense-in-depth security model. Each phase builds on the previous to create a layered security stack.

### 15.1 Plugin Foundation (v3.0) ✅ COMPLETE

**Goal:** Enable external catalogs with basic security guarantees.

#### External Catalog Support
- Load catalog YAML from URLs or local paths
- Catalog manifest with integrity hashes for all files
- User approval required for new catalog sources
- Local caching for offline use

#### CLI Commands
```bash
preflight catalog add <url>       # Add external catalog
preflight catalog list            # List installed catalogs
preflight catalog remove <name>   # Remove catalog
preflight catalog verify          # Verify all catalog integrity
preflight catalog audit <name>    # Security audit of catalog
```

#### Integrity Verification
- SHA256 hashes for all catalog files
- Manifest validation before loading
- Hash mismatch fails immediately

```yaml
# catalog-manifest.yaml
version: "1.0"
integrity:
  algorithm: sha256
  files:
    catalog.yaml: "sha256:abc123..."
    presets/base.yaml: "sha256:def456..."
```

### 15.2 Signature Verification (v3.1)

**Goal:** Verify publisher identity and establish trust chains.

#### Cryptographic Signatures
- GPG signature verification on catalog manifests
- Sigstore integration for keyless signing (OIDC-based)
- SSH key signature support

#### Trust Management
```bash
preflight trust list              # List trusted publishers
preflight trust add <keyid>       # Trust a GPG key
preflight trust remove <keyid>    # Untrust a key
preflight trust show <keyid>      # Show key details
```

#### Trust Levels

| Level | Description | Verification |
|-------|-------------|--------------|
| `builtin` | Embedded in binary | Compiled-in |
| `verified` | Signed by known publisher | GPG/Sigstore signature |
| `community` | Hash-verified, user-reviewed | SHA256 + user approval |
| `untrusted` | No verification | Explicit `--allow-untrusted` |

#### Configuration
```yaml
# preflight.yaml
security:
  min_trust_level: community
  require_signatures: false
  auto_approve_publishers:
    - "Preflight Official <team@preflight.dev>"
```

### 15.3 Capability-Based Permissions (v3.2)

**Goal:** Limit what plugins can do through declared capabilities.

#### Capability Declaration
Plugins declare required capabilities upfront:

```yaml
capabilities:
  - files:read      # Read dotfiles
  - files:write     # Write dotfiles
  - packages:brew   # Install Homebrew packages
  - packages:apt    # Install APT packages
  - shell:execute   # Run shell commands
  - network:fetch   # Fetch from network
  - secrets:read    # Access secrets (SSH keys, etc.)
```

#### Enforcement
- Plugins only get capabilities they declare
- User must approve capabilities on first install
- Dangerous capabilities (`shell:execute`, `secrets:read`) require explicit confirmation
- Runtime enforcement prevents capability escalation

#### Blocked Capabilities
```yaml
# preflight.yaml
security:
  blocked_capabilities:
    - secrets:read
    - shell:execute
```

#### Content Security Policy
Pattern-based rules to block dangerous operations:

```yaml
# security-policy.yaml
deny:
  - pattern: "curl.*|.*sh"
    reason: "Piped curl to shell is dangerous"
  - pattern: "chmod.*777"
    reason: "World-writable permissions"
  - pattern: "sudo.*"
    reason: "Sudo commands not allowed in presets"
  - pattern: "rm.*-rf.*/"
    reason: "Recursive delete of root paths"

warn:
  - pattern: ".*eval.*"
    reason: "Eval can execute arbitrary code"
```

### 15.4 Plugin Sandbox (v3.3)

**Goal:** Complete isolation for untrusted plugins.

#### WASM Plugin Runtime
- Plugins compile to WebAssembly
- Run in isolated VM with no direct system access
- Deterministic execution

#### Resource Limits
- CPU throttling
- Memory caps
- File descriptor limits
- Network policy enforcement

#### Sandbox Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `full` | Complete isolation, no side effects | Preview/audit unknown plugins |
| `restricted` | Limited to declared capabilities | Normal operation |
| `trusted` | Full access (like builtin) | Verified publishers only |

### 15.5 Audit Logging

All plugin operations are logged for security review:

```json
{
  "timestamp": "2024-12-24T12:00:00Z",
  "event": "catalog_installed",
  "catalog": "company-devtools",
  "source": "https://company.com/catalog.yaml",
  "integrity": "sha256:abc123...",
  "signature_verified": true,
  "signer": "devops@company.com",
  "capabilities_granted": ["files:write", "packages:brew"],
  "user": "jane"
}
```

### 15.6 Security Stack Summary

Each layer catches different attack vectors:

```
┌─────────────────────────────────────┐
│  v3.3: Sandbox (WASM isolation)     │  ← Can't escape even if malicious
├─────────────────────────────────────┤
│  v3.2: Capabilities (permissions)   │  ← Can only do what's declared
├─────────────────────────────────────┤
│  v3.1: Signatures (identity)        │  ← Know who published it
├─────────────────────────────────────┤
│  v3.0: Integrity (hashes)           │  ← Know it wasn't tampered with
└─────────────────────────────────────┘
```

---

## 16. Future Considerations (v4+)

- Remote execution and fleet management
- Background agent with scheduled reconciliation
- Integration with enterprise identity providers
- Multi-machine sync and conflict resolution
- Plugin marketplace with automated security scanning
- Reproducible builds verification for catalogs

---

## 17. Final Positioning Statement

> **Preflight is a deterministic workstation compiler that helps anyone design, reproduce, and understand their setup — safely, locally, and without lock-in.**
