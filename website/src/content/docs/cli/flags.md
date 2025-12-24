---
title: Flags & Options
description: Detailed reference for all CLI flags and options.
---

This page provides detailed information about all available flags and options.

## Global Flags

These flags are available on all commands.

### --config

Specify path to configuration file.

```bash
preflight plan --config ~/dotfiles/preflight.yaml
preflight apply --config /path/to/config.yaml
```

**Default:** `./preflight.yaml`

### --target

Select which target/profile to use.

```bash
preflight apply --target work
preflight apply --target personal
```

Targets are defined in your manifest:

```yaml
targets:
  work: [base, identity.work, role.go]
  personal: [base, identity.personal]
```

### --mode

Control version resolution behavior.

| Mode | Behavior |
|------|----------|
| `intent` | Install latest compatible versions |
| `locked` | Prefer lockfile; update with `--update-lock` |
| `frozen` | Fail if resolution differs from lock |

```bash
preflight apply --mode frozen
preflight plan --mode intent
```

**Default:** `locked`

### --dry-run

Preview changes without modifying the system.

```bash
preflight apply --dry-run
preflight doctor --fix --dry-run
```

### --verbose

Show detailed execution output.

```bash
preflight apply --verbose
```

### --yes

Skip all confirmation prompts.

```bash
preflight apply --yes
```

:::caution
Use with care — bypasses safety confirmations.
:::

## AI Configuration

### --no-ai

Disable AI guidance completely.

```bash
preflight init --no-ai
```

### --ai-provider

Select AI provider for guidance.

| Provider | Description |
|----------|-------------|
| `openai` | OpenAI API |
| `anthropic` | Anthropic Claude API |
| `ollama` | Local Ollama instance |
| `none` | Disable AI |

```bash
preflight init --ai-provider anthropic
```

Requires corresponding API key in environment:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

## Command-Specific Flags

### init Flags

| Flag | Description |
|------|-------------|
| `--guided` | Interactive TUI wizard |
| `--minimal` | Generate minimal config |
| `--editor <name>` | Editor preset |
| `--languages <list>` | Comma-separated languages |
| `--repo` | Initialize git repository |
| `--github` | Create GitHub repository |

### capture Flags

| Flag | Description |
|------|-------------|
| `--include <modules>` | Only capture these modules |
| `--exclude <modules>` | Skip these modules |
| `--infer-profiles` | Auto-detect work/personal |
| `--review` | Open review TUI |

**Modules:** brew, apt, files, git, ssh, shell, nvim, vscode

### plan Flags

| Flag | Description |
|------|-------------|
| `--diff` | Show file content diffs |
| `--explain` | Include explanations |
| `--json` | Machine-readable output |
| `--validate-only` | Only validate config |

### apply Flags

| Flag | Description |
|------|-------------|
| `--update-lock` | Update lockfile after apply |

### doctor Flags

| Flag | Description |
|------|-------------|
| `--fix` | Converge machine to config |
| `--update-config` | Update config from machine |
| `--report <format>` | Output format: json, markdown |

## Environment Variables

Configure Preflight via environment:

| Variable | Description |
|----------|-------------|
| `PREFLIGHT_CONFIG` | Default config path |
| `PREFLIGHT_TARGET` | Default target |
| `PREFLIGHT_MODE` | Default mode |
| `PREFLIGHT_NO_AI` | Disable AI (set to "1") |
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OLLAMA_HOST` | Ollama server URL |

```bash
export PREFLIGHT_TARGET=work
export PREFLIGHT_MODE=frozen
preflight apply  # Uses work target in frozen mode
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Lock mismatch (frozen mode) |
| 4 | User cancelled |

## Configuration File Precedence

1. Command-line flags (highest)
2. Environment variables
3. Config file settings
4. Built-in defaults (lowest)

## Examples

### Development Workflow

```bash
# Quick iteration
preflight plan --verbose
preflight apply --yes

# Safe production
preflight plan --mode frozen
preflight apply --mode frozen
```

### Scripting

```bash
# CI/CD usage
preflight plan --json > plan.json
preflight apply --yes --mode frozen
```

### Debugging

```bash
# Verbose with dry-run
preflight apply --verbose --dry-run

# Validate only
preflight plan --validate-only
```
