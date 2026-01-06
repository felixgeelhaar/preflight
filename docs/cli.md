Preflight — CLI Contract (v0.1)
This document is the canonical UX contract for the Preflight CLI. Treat it as a “public API” for users.

---

preflight
Compile, apply, and verify a reproducible workstation.
Preflight is a deterministic workstation compiler.
It turns intent (targets, layers, capabilities) into a reproducible,
explainable local setup.

Usage:
preflight <command> [flags]

Core Commands:
init Design or discover a workstation configuration
capture Detect current system and generate config
plan Show what would change (no execution)
apply Apply the compiled plan to this machine
doctor Verify state and detect drift
tour Guided walkthrough of installed tools

Utility Commands:
diff Show differences between config and machine
lock Manage lockfile (update, freeze)
repo Manage config repository (git/GitHub)
completion Generate shell completion
version Show version information

Global Flags:
--config <path> Path to config (default: ./preflight.yaml)
--target <name> Target/profile to apply (e.g. work, personal)
--mode <mode> intent | locked | frozen (default: intent)
--no-ai Disable AI guidance
--ai-provider <name> openai | anthropic | ollama | none
--dry-run Never modify the system
--verbose Show detailed execution output
--yes Skip confirmation prompts (including bootstrap)
--allow-bootstrap Skip confirmation for bootstrap steps

## Run 'preflight <command> --help' for details.

preflight init
Create a new Preflight configuration (guided or minimal).
Usage:
preflight init [flags]

Description:
Initializes a new Preflight setup.
Can guide you from scratch or scaffold minimal config.

Modes:
• Build mode – design a setup with guidance (AI optional)
• Capture mode – start from existing machine state

Flags:
--guided Interactive TUI (default)
--minimal Generate minimal config, no prompts
--editor <name> nvim | vscode | cursor | none
--languages <list> go,ts,python,rust,...
--repo Create a Git repository for config
--github Create private GitHub repo (requires gh)

Outputs:
• preflight.yaml
• layers/
• dotfiles/ (optional)
• preflight.lock (created on first apply)

Examples:
preflight init
preflight init --minimal
preflight init --editor nvim --languages go,ts

---

preflight capture
Reverse-engineer the current machine into config.
Usage:
preflight capture [flags]

Description:
Detects installed tools, configs, and identities,
then generates clean, deterministic layers.

Flags:
--include <module> brew,files,nvim,git,ssh,editor
--exclude <module> Skip specific detectors
--infer-profiles Infer work/personal/roles (default)
--redact-secrets Always enabled; never exports secrets
--review Open TUI to accept/reject findings

Outputs:
• layers/base.yaml
• layers/identity._.yaml
• layers/role._.yaml
• preflight.lock

Examples:
preflight capture
preflight capture --include nvim,git

---

preflight plan
Compile configuration into an executable plan.
Usage:
preflight plan [flags]

Description:
Shows exactly what would change without applying anything.
Every action is explained.

Flags:
--target <name> Profile/target to plan
--diff Show file diffs
--explain Explain why each action exists
--json Output machine-readable plan

Examples:
preflight plan
preflight plan --target work --explain

---

preflight apply
Apply the compiled plan to this machine.
Usage:
preflight apply [flags]

Description:
Applies the plan deterministically.
Requires confirmation unless --yes is used (including bootstrap).

Flags:
--target <name> Profile/target to apply
--yes Skip confirmation (including bootstrap)
--update-lock Update lockfile after apply
--rollback-on-error Attempt rollback on failure

Safety:
• No execution without a plan
• Destructive steps are flagged
• Idempotent by default

Examples:
preflight apply
preflight apply --target personal --yes

---

preflight doctor
Verify system state and detect drift.
Usage:
preflight doctor [flags]

Description:
Checks whether the machine matches the compiled config.

Detects:
• Missing packages
• Drifted dotfiles
• Editor/plugin mismatch
• Missing secrets
• Lock inconsistencies

Flags:
--fix Fix machine to match config
--update-config Update config to match machine
--report Output report (json/markdown)

Examples:
preflight doctor
preflight doctor --fix

---

preflight tour
Learn how your setup works.
Usage:
preflight tour [topic]

Topics:
nvim Learn Neovim basics + installed features
editor Learn VS Code / Cursor setup
git Learn Git config & workflows
shell Learn shell environment

Description:
Interactive, terminal-based walkthroughs that:
• explain why tools were installed
• demonstrate how to use them
• link to relevant docs

Examples:
preflight tour nvim
preflight tour git

---

preflight repo
Manage configuration repositories.
Usage:
preflight repo <command>

Commands:
init Initialize git repo
push Commit & push changes
pull Pull config on new machine
status Show repo status

Examples:
preflight repo init --github
preflight repo pull
