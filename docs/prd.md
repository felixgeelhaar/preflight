Preflight — Product Requirements Document (PRD) v1

1. Product Summary
   Preflight is a CLI/TUI tool that compiles a workstation from declarative configuration into a reproducible machine setup, with optional AI-guided discovery and explainability.

---

2. Goals
   2.1 Primary Goals (v1)
   • Deterministic workstation setup: plan → apply → verify
   • Safe, explainable execution (trust-first)
   • Dotfile creation and lifecycle management
   • Profile/layer composition (work, personal, roles, device)
   • Lockfile-backed reproducibility (intent/locked/frozen)
   • Pure terminal experience (TUI-first)
   • Optional AI guidance (BYOK: cloud + local/Ollama)
   2.2 Non-Goals (v1)
   • SaaS features or account requirement
   • Full MDM/policy enforcement (beyond basic constraints)
   • Background agent / continuous automation
   • Windows native support (WSL is a future target)
   • Plugin marketplace (design hooks only)

---

3. Target Users
   Primary
   • Developers / platform engineers
   • Users currently using dotfiles, bootstrap scripts, Brewfiles, partial config managers
   Secondary (explicitly supported)
   • Designers
   • Product managers
   • Researchers / writers
   • Non-technical users via guided TUI + presets

---

4. Core Concepts
   4.1 Layers & Targets
   • Layer: composable unit of intent (base, identity.work, role.go, device.laptop)
   • Target: an ordered list of layers (e.g., work, personal)
   4.2 Modules
   Providers that compile into executable steps, e.g.:
   • brew / apt
   • files (dotfiles render/link)
   • git, ssh
   • runtime (rtx/asdf)
   • editor (nvim, vscode-like/cursor)
   • ai (advisor only)
   4.3 Reproducibility Modes
   • intent: install latest compatible versions
   • locked: prefer lockfile; update lock intentionally
   • frozen: fail if resolution differs from lock

---

5. User Journeys
   5.1 Init — Build mode (new user)
   User story: “I’m new to Neovim and want a great setup.”
   Flow:
1. preflight init opens guided TUI
1. User selects personas/goals (e.g., “Beginner”, “Balanced”, languages)
1. Preflight proposes presets/capabilities with explanations + links
1. User reviews in TUI (toggle include/exclude)
1. Preflight writes config (preflight.yaml, layers/, dotfiles scaffolding)
1. Optional: preflight repo init --github creates a private GitHub repo
1. preflight apply installs tools, bootstraps editor, writes lock(s)
   Acceptance criteria:
   • No config editing required
   • Every suggested tool has “why”, tradeoffs, demo, docs links
   • preflight doctor passes after apply
   5.2 Init — Capture mode (existing machine)
   Flow:
1. preflight capture
1. Detect packages/configs/editors
1. Infer layers (base + identities + roles)
1. TUI review: keep leaves only, move items across layers, accept
1. Write layers + lock snapshot
   Acceptance criteria:
   • Produces readable layers (not a giant dump)
   • Never exports secrets
   • Can reproduce on a new machine
   5.3 Plan → Apply
   • preflight plan shows actions, diffs, and explanations
   • preflight apply executes deterministically; prompts unless --yes
   • Updates lock based on mode and flags
   5.4 Doctor / Drift
   • preflight doctor checks packages, dotfiles, editor plugins/extensions, missing secrets
   • Offers:
   ◦ --fix (converge machine to config)
   ◦ --update-config (capture delta into layers)

---

6. Feature Requirements
   6.1 TUI Requirements
   • Full workflow in terminal (no browser required)
   • Search/filter lists
   • Explain panel for any item (why, tradeoffs, docs)
   • “Patch-like” acceptance for capture results (include/exclude/move)
   • Clear destructive-step labeling
   6.2 Dotfiles Requirements
   Dotfile modes:
1. Generated
1. Template + user overrides
1. Bring-your-own (link/validate only)
   Capabilities:
   • Render structured config into files (~/.gitconfig, ~/.ssh/config, shell config)
   • Annotate managed sections
   • Provide diffs in plan
   • Detect drift in doctor
   • Snapshot before applying changes
   6.3 Editor Requirements (v1)
   Neovim (first-class)
   • Install nvim
   • Apply a preset (minimal/balanced/pro)
   • Bootstrap plugins headlessly
   • Lock via lazy-lock.json (and record in preflight.lock)
   • Doctor checks:
   ◦ lock present
   ◦ required binaries present (rg, fd, formatters, LSP tools)
   VS Code / Cursor (v1-lite)
   • Install extensions by ID
   • Apply settings
   • Record installed versions in lock (best-effort)
   • Doctor checks extension presence and settings drift
   6.4 AI Requirements (BYOK)
   • Providers: OpenAI, Anthropic, Ollama, None
   • AI outputs must be:
   ◦ explainable
   ◦ mapped to a versioned catalog/preset when possible
   ◦ never executed directly
   6.5 Repo Requirements
   • preflight repo init --github creates private repo using gh
   • Repo contains:
   ◦ config + layers
   ◦ lockfile
   ◦ dotfiles (generated/templates/user)
   ◦ README with bootstrap instructions
   • preflight repo pull supports new-machine bootstrap

---

7. Success Metrics (qualitative for v1)
   • A new user can go from zero → working setup in < 30 minutes
   • Re-running apply causes no surprises
   • Non-engineers can complete init without editing YAML
   • Captured config can reproduce on a fresh machine with minimal edits

---

8. Out of Scope (explicit)
   • Marketplace (future)
   • Org policy and compliance engine (future)
   • Remote execution and fleet management
   • Continuous background reconciliation
