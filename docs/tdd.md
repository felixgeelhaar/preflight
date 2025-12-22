Preflight — Technical Design Document (TDD) v1

1. Architecture Overview
   Preflight is a local-first Go CLI with a TUI frontend and a deterministic compilation + execution engine.
   High-level flow:
1. Load config (manifest + layers)
1. Merge into a target config
1. Compile providers into a step graph (DAG)
1. Plan (diff + explanations)
1. Apply (idempotent)
1. Verify (doctor / drift)
1. Update lock (mode-dependent)

---

2. Repository Structure (recommended)
   cmd/preflight/ # main entry
   internal/config/ # schema, merge, validation
   internal/compiler/ # compile target → step graph
   internal/steps/ # step types + execution runtime
   internal/providers/ # brew, files, nvim, editor, ssh, git, runtime
   internal/lock/ # lock model + resolver helpers
   internal/tui/ # Bubble Tea views + interactions
   internal/ai/ # advisor interface (BYOK)
   internal/catalog/ # presets, capability packs, explain data

---

3. Config Model
   3.1 Files
   • preflight.yaml — root manifest (targets, defaults)
   • layers/\*.yaml — overlays
   • preflight.lock — resolved versions and integrity hashes
   • dotfiles/ — generated/templates/user-owned (optional)
   3.2 Merge Semantics
   • Scalars: last-wins
   • Maps: deep merge
   • Lists: set union with explicit add/remove directives
   • Each merge emits provenance metadata (layer origin) for explainability in TUI.
   3.3 Validation
   • JSON schema (or go struct validation) for all config
   • Provider-level validation: missing required fields, invalid values
   • Constraints evaluated during compile and surfaced in plan.

---

4. Compilation Engine
   4.1 Providers → Steps
   Each provider produces a set of steps for a compiled target:
   type Provider interface {
   Name() string
   Compile(ctx CompileContext) ([]Step, error)
   }
   Steps are idempotent units:
   type Step interface {
   ID() string
   DependsOn() []string
   Check(ctx RunContext) (Status, error) // satisfied / needs-apply / unknown
   Plan(ctx RunContext) (Diff, error) // human-readable + structured diff
   Apply(ctx RunContext) error
   Explain(ctx ExplainContext) Explanation // why/tradeoffs/docs
   }
   4.2 Step Graph (DAG)
   • Provider steps are topologically sorted
   • Parallel execution is allowed where safe (future)
   • On failure: stop, surface error, optionally rollback.

---

5. Providers (v1)
   5.1 brew provider (macOS)
   • Detect/install Homebrew if missing (explicit consent)
   • Taps, formulae, casks
   • Capture distinguishes leaves vs deps (default: leaves)
   • Lock:
   ◦ taps + commits (best-effort)
   ◦ resolved versions (best-effort)
   5.2 apt provider (Linux, optional v1)
   • Install packages
   • Lock versions (best-effort)
   5.3 files provider (dotfiles)
   Supports dotfile modes:
   • Generated: renderer owns file
   • Template+override: generate base + stable include mechanism
   • BYO: symlink/copy + validation only
   Key features:
   • snapshot before modification
   • diff rendering in plan
   • drift hashing in doctor
   5.4 git provider
   • Render .gitconfig (generated/templated) or manage include blocks
   • Identity separation via targets
   5.5 ssh provider
   • Render ~/.ssh/config with per-host identity mapping
   • Never export private keys
   • Secret references only (keychain/1password/etc.)
   5.6 runtime provider (rtx/asdf)
   • Install runtime manager (optional)
   • Install tool versions by intent; lock resolved versions
   5.7 editor providers
   Neovim provider
   • Ensure nvim installed
   • Config source:
   ◦ template-generated OR git repo clone
   • Bootstrap:
   ◦ nvim --headless '+Lazy sync' +qa
   • Doctor:
   ◦ lazy lock exists
   ◦ external binaries exist
   ◦ health checks (best-effort)
   VS Code / Cursor provider (v1-lite)
   • Install extensions by ID
   • Apply settings
   • Lock installed extension versions (best-effort)

---

6. Lockfile Design
   preflight.lock records:
   • machine (os, arch)
   • provider resolution (versions, commits, IDs)
   • integrity hashes:
   ◦ merged manifest hash
   ◦ per-layer hash
   ◦ key dotfile hashes
   Modes:
   • intent: lock optional
   • locked: lock preferred; --update-lock updates
   • frozen: lock required; mismatches error

---

7. TUI Design (Bubble Tea)
   Key screens:
   • Init wizard
   • Capture review (git-add -p style)
   • Plan review with explanation panel
   • Apply progress + logs
   • Doctor report with fix/update options
   • Tour mode (interactive learning)
   Explain panel content is data-driven from catalog + provider metadata.

---

8. AI (BYOK) Architecture
   AI is an advisor, not an executor.
   type Advisor interface {
   Suggest(ctx SuggestContext) ([]Recommendation, error)
   Explain(ctx ExplainContext) (Explanation, error)
   }
   Providers supported:
   • OpenAI
   • Anthropic
   • Ollama
   • None
   AI output must map to:
   • known presets
   • known capability packs
   • or user-confirmed custom additions

---

9. Security & Privacy
   • Local-first by default
   • Never store secrets in config or lock
   • Redact tokens from capture
   • Provide --no-ai and offline modes
   • Explicit consent for installations that change system state (e.g., installing Homebrew)

---

10. Future Extensions (explicitly out of scope v1)
    • Marketplace / signed community packs
    • Org baselines and policy enforcement
    • Remote execution and fleet mgmt
    • Windows native support
