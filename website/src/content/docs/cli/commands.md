---
title: Commands
description: Complete reference for all Preflight CLI commands.
---

Preflight provides a comprehensive CLI for workstation configuration management.

## Core Commands

### preflight init

Create a new Preflight configuration.

```bash
preflight init [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--guided` | Interactive TUI wizard (default) |
| `--minimal` | Generate minimal config, no prompts |
| `--editor <name>` | Editor preset: nvim, vscode, cursor, none |
| `--languages <list>` | Languages: go,ts,python,rust,... |
| `--repo` | Initialize Git repository |
| `--github` | Create private GitHub repo (requires gh) |

**Examples:**

```bash
# Interactive wizard
preflight init

# Minimal configuration
preflight init --minimal

# With specific presets
preflight init --editor nvim --languages go,ts
```

**Outputs:**
- `preflight.yaml` — Root manifest
- `layers/` — Configuration overlays
- `dotfiles/` — Optional dotfile templates

---

### preflight capture

Reverse-engineer current machine into configuration.

```bash
preflight capture [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--include <modules>` | Include specific modules: brew,files,nvim,git,ssh |
| `--exclude <modules>` | Exclude specific modules |
| `--infer-profiles` | Infer work/personal/role layers (default: true) |
| `--review` | Open TUI to accept/reject findings |

**Examples:**

```bash
# Capture everything
preflight capture

# Capture specific modules
preflight capture --include nvim,git

# Skip review TUI
preflight capture --no-review
```

**Outputs:**
- `layers/base.yaml`
- `layers/identity.*.yaml`
- `layers/role.*.yaml`

---

### preflight plan

Show what would change without applying.

```bash
preflight plan [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--target <name>` | Target/profile to plan |
| `--diff` | Show file diffs |
| `--explain` | Explain why each action exists |
| `--json` | Output machine-readable plan |

**Examples:**

```bash
# Basic plan
preflight plan

# With explanations
preflight plan --explain

# For specific target
preflight plan --target work

# JSON output for scripting
preflight plan --json
```

---

### preflight apply

Apply the compiled plan to this machine.

```bash
preflight apply [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--target <name>` | Target/profile to apply |
| `--yes` | Skip confirmation prompts |
| `--update-lock` | Update lockfile after apply |

**Examples:**

```bash
# Apply with confirmation
preflight apply

# Skip confirmation
preflight apply --yes

# Apply specific target
preflight apply --target personal
```

**Safety guarantees:**
- No execution without a plan
- Destructive steps are flagged
- All operations are idempotent

---

### preflight doctor

Verify system state and detect drift.

```bash
preflight doctor [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--fix` | Fix machine to match config |
| `--update-config` | Update config to match machine |
| `--dry-run` | Preview changes without writing |
| `--report <format>` | Output format: json, markdown |

**Examples:**

```bash
# Check for issues
preflight doctor

# Fix drift
preflight doctor --fix

# Update config from machine state
preflight doctor --update-config

# Preview config updates
preflight doctor --update-config --dry-run
```

**Checks:**
- Missing packages
- Drifted dotfiles
- Editor/plugin mismatches
- Missing secrets
- Lock inconsistencies

---

## Utility Commands

### preflight validate

Validate configuration for CI/CD pipelines.

```bash
preflight validate [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--config <path>` | Path to preflight.yaml (default: preflight.yaml) |
| `--target <name>` | Target to validate (default: default) |
| `--json` | Output results as JSON |
| `--strict` | Treat warnings as errors |
| `--policy <path>` | Path to policy YAML file (allow/deny rules) |
| `--org-policy <path>` | Path to org policy YAML file (required/forbidden) |

**Exit Codes:**

| Code | Meaning |
|------|---------|
| `0` | Valid configuration |
| `1` | Validation errors or policy violations |
| `2` | Could not read configuration |

![Validate Demo](/preflight/demos/gif/validate.gif)

**Examples:**

```bash
# Basic validation
preflight validate

# JSON output for CI
preflight validate --json

# With strict mode (warnings = errors)
preflight validate --strict

# With allow/deny policy file
preflight validate --policy policy.yaml

# With org policy file (required/forbidden patterns)
preflight validate --org-policy org-policy.yaml

# Validate specific target
preflight validate --target work
```

**Policy File Example (allow/deny rules):**

```yaml
# policy.yaml
policies:
  - name: security-baseline
    description: Block insecure tools
    rules:
      - pattern: "*:telnet"
        action: deny
        message: use SSH instead
      - pattern: "*:ftp"
        action: deny
        message: use sftp instead
      - pattern: "*"
        action: allow
```

**Org Policy File Example (required/forbidden patterns):**

```yaml
# org-policy.yaml
version: "1"
policy:
  name: acme-corp
  description: ACME Corporation workstation policy
  enforcement: block  # or "warn"

  required:
    - pattern: "git:*"
      message: "Git configuration is required"
    - pattern: "ssh:*"
      message: "SSH must be configured"

  forbidden:
    - pattern: "brew:*-nightly"
      message: "Nightly packages not allowed"
    - pattern: "vscode:extension:*-unofficial"
      message: "Only official extensions allowed"

  overrides:
    - pattern: "brew:rust-nightly"
      justification: "Needed for testing new Rust features"
      approved_by: "security@acme.com"
      expires_at: "2025-12-31T23:59:59Z"
```

**Output (text):**

```
✓ Configuration is valid
  • Loaded config from preflight.yaml
  • Target: default
  • Compiled 15 steps
```

**Output (with violations):**

```
✗ Validation errors:
  ✗ Compilation failed: missing provider

⛔ Policy violations:
  ⛔ policy violation: brew:install:telnet is denied by rule "*:telnet" (use SSH instead)

⚠ Warnings:
  ⚠ No steps generated - configuration may be empty

ℹ Info:
  • Loaded config from preflight.yaml
  • Target: default
```

---

### preflight rollback

Restore files from automatic snapshots.

```bash
preflight rollback [flags]
```

![Rollback Demo](/preflight/demos/gif/rollback.gif)

**Flags:**

| Flag | Description |
|------|-------------|
| `--to <id>` | Restore specific snapshot by ID |
| `--latest` | Restore most recent snapshot |
| `--dry-run` | Preview restoration without applying |

**Examples:**

```bash
# List available snapshots
preflight rollback

# Restore specific snapshot
preflight rollback --to abc123

# Restore most recent snapshot
preflight rollback --latest

# Preview what would be restored
preflight rollback --to abc123 --dry-run
```

**Output (listing):**

```
Available snapshots:

ID         DATE                 AGE        FILES   REASON
────────────────────────────────────────────────────────────
a1b2c3d4   2024-12-24 14:30:00  2 hours    3       pre-apply
e5f6g7h8   2024-12-24 10:15:00  6 hours    5       doctor-fix
```

---

### preflight tour

Interactive guided walkthroughs for learning Preflight with progress tracking.

```bash
preflight tour [topic] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--list` | List all available topics |

**Topics:**

| Topic | Description |
|-------|-------------|
| `basics` | Preflight fundamentals and compiler model |
| `config` | Configuration structure and YAML syntax |
| `layers` | Layer composition and merge semantics |
| `providers` | Provider overview (brew, git, shell, nvim, vscode) |
| `presets` | Using presets and capability packs |
| `workflow` | Daily workflow: plan, apply, doctor cycle |

**Hands-On Tutorials:** (🛠️)

| Topic | Description |
|-------|-------------|
| `nvim-basics` | Interactive Neovim tutorial with practice commands |
| `git-workflow` | Git commands and conventional commits practice |
| `shell-customization` | Shell aliases, functions, and prompt setup |

**Examples:**

```bash
# Open interactive topic menu
preflight tour

# Start specific topic
preflight tour basics
preflight tour workflow

# List all available topics
preflight tour --list
```

**Navigation:**

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate topics/scroll |
| `Enter` | Select topic / advance |
| `n` or `→` | Next section |
| `p` or `←` | Previous section |
| `1-9` | Jump to section number |
| `g` / `G` | Go to first / last section |
| `Esc` | Go back / exit |
| `q` | Quit tour |

**Progress Tracking:**

Your progress is automatically saved to `~/.preflight/tour-progress.json`:
- ✓ indicates completed topics
- (%) shows partial completion
- Progress persists between sessions

![Tour Demo](/preflight/demos/gif/tour.gif)

---

### preflight diff

Show differences between config and machine.

```bash
preflight diff [flags]
```

**Examples:**

```bash
# Show all differences
preflight diff

# Specific provider
preflight diff --provider brew
```

---

### preflight lock

Manage lockfile operations.

```bash
preflight lock <command>
```

**Commands:**

| Command | Description |
|---------|-------------|
| `update` | Update lock to current resolved versions |
| `freeze` | Set mode to frozen |
| `status` | Show lock status |

**Examples:**

```bash
# Update lockfile
preflight lock update

# Check status
preflight lock status
```

---

### preflight repo

Manage configuration repository.

```bash
preflight repo <command>
```

**Commands:**

| Command | Description |
|---------|-------------|
| `init` | Initialize git repository |
| `push` | Commit and push changes |
| `pull` | Pull config on new machine |
| `status` | Show repository status |

**Examples:**

```bash
# Initialize with GitHub
preflight repo init --github

# Pull on new machine
preflight repo pull
```

---

### preflight completion

Generate shell completion scripts.

```bash
preflight completion <shell>
```

**Shells:** bash, zsh, fish, powershell

**Examples:**

```bash
# Bash
preflight completion bash > /etc/bash_completion.d/preflight

# Zsh
preflight completion zsh > "${fpath[1]}/_preflight"

# Fish
preflight completion fish > ~/.config/fish/completions/preflight.fish
```

---

### preflight plugin

Manage Preflight plugins that extend functionality.

```bash
preflight plugin <command> [flags]
```

**Subcommands:**

| Command | Description |
|---------|-------------|
| `list` | List all installed plugins |
| `search [query]` | Search for plugins on GitHub |
| `install <source>` | Install a plugin from path or Git URL |
| `remove <name>` | Remove an installed plugin |
| `info <name>` | Show detailed plugin information |
| `validate [path]` | Validate plugin manifest |
| `upgrade [name]` | Upgrade plugins to latest version |

**Search Flags:**

| Flag | Description |
|------|-------------|
| `--type <type>` | Filter by plugin type: `config`, `provider` |
| `--min-stars <n>` | Minimum number of GitHub stars |
| `--limit <n>` | Maximum results (default: 20) |
| `--sort <field>` | Sort by: `stars`, `updated`, `best-match` |

**Examples:**

```bash
# List installed plugins
preflight plugin list

# Search for plugins
preflight plugin search docker
preflight plugin search --type provider kubernetes
preflight plugin search --min-stars 10 terminal
preflight plugin search --sort updated

# Install from local path
preflight plugin install /path/to/plugin

# Install from Git repository
preflight plugin install https://github.com/example/preflight-docker.git

# View plugin details
preflight plugin info docker

# Remove a plugin
preflight plugin remove docker

# Validate a plugin
preflight plugin validate /path/to/plugin
preflight plugin validate --strict --json

# Check for updates
preflight plugin upgrade --check

# Upgrade all plugins
preflight plugin upgrade

# Upgrade specific plugin
preflight plugin upgrade kubernetes --dry-run
```

**Output (list):**

```
NAME          VERSION   STATUS    DESCRIPTION
────          ───────   ──────    ───────────
docker        1.0.0     enabled   Docker provider for Preflight
kubernetes    2.0.0     enabled   Kubernetes tooling
```

**Output (search):**

```
Found 3 plugin(s) matching "docker":

REPOSITORY                        TYPE      STARS  DESCRIPTION
──────────                        ────      ─────  ───────────
example/preflight-docker          provider  42     Docker provider for Preflight
user/docker-presets               config    15     Docker configuration presets
team/container-tools              config    8      Container development tools

Install a plugin with:
  preflight plugin install https://github.com/<repository>
```

**Output (info):**

```
Name:        kubernetes
Version:     2.0.0
API Version: v1
Description: Kubernetes tooling for Preflight
Author:      K8s Team
License:     Apache-2.0

Providers:
  • kubectl (kubernetes.kubectl)
    kubectl installation
  • helm (kubernetes.helm)
    Helm chart management

Presets:
  • k8s:dev
  • k8s:prod

Dependencies:
  • docker >=1.0.0
```

![Plugin Demo](/preflight/demos/gif/plugin.gif)

---

### preflight marketplace

Browse, search, and install community presets, capability packs, and layer templates.

```bash
preflight marketplace <command> [flags]
```

**Subcommands:**

| Command | Description |
|---------|-------------|
| `search [query]` | Search for packages |
| `install <name>` | Install a package |
| `list` | List installed packages |
| `info <name>` | Show package details |
| `uninstall <name>` | Uninstall a package |
| `update` | Update all packages |

**Flags:**

| Flag | Description |
|------|-------------|
| `--offline` | Use cached data only |
| `--refresh` | Force refresh of package index |

**Examples:**

```bash
# Search for packages
preflight marketplace search nvim

# Install a package
preflight marketplace install nvim-pro

# List installed packages
preflight marketplace list

# Show package details
preflight marketplace info nvim-pro

# Update all packages
preflight marketplace update
```

![Marketplace Demo](/preflight/demos/gif/marketplace.gif)

---

### preflight version

Display version information.

```bash
preflight version
```

**Output:**

```
preflight version 3.3.1
```

---

## Global Flags

Available on all commands:

| Flag | Description |
|------|-------------|
| `--config <path>` | Path to config (default: ./preflight.yaml) |
| `--target <name>` | Target/profile to use |
| `--mode <mode>` | intent, locked, or frozen |
| `--no-ai` | Disable AI guidance |
| `--ai-provider <name>` | openai, anthropic, ollama, none |
| `--dry-run` | Never modify the system |
| `--verbose` | Show detailed output |
| `--yes` | Skip confirmation prompts |

## What's Next?

- [Flags & Options](/preflight/cli/flags/) — Detailed flag reference
- [Configuration](/preflight/guides/configuration/) — Configuration guide
