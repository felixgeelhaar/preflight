---
title: Plugins
description: Extend Preflight with community plugins.
---

Preflight supports plugins that add new providers, presets, and capability packs. This guide covers finding, installing, and developing plugins.

## Plugin Types

Preflight has a **two-tier plugin architecture** designed for security and simplicity:

| Type | Description | Format | Trust Model |
|------|-------------|--------|-------------|
| **Config** | Presets, capability packs, YAML configs | Pure YAML | Safe by design |
| **Provider** | Custom providers with executable logic | WASM + YAML | Sandboxed execution |

**Config plugins** are the recommended choice for most use cases. They can share presets, capability packs, and provider configurations without any executable code.

**Provider plugins** are needed when you require custom logic that built-in providers don't support. They run in a WebAssembly sandbox with declared capabilities.

## Finding Plugins

Search for community plugins directly from the CLI:

```bash
# Search for plugins
preflight plugin search docker

# Filter by type
preflight plugin search --type config terminal
preflight plugin search --type provider kubernetes

# Filter by popularity
preflight plugin search --min-stars 10

# Sort by update time
preflight plugin search --sort updated
```

Plugins are discovered via GitHub topics:
- `preflight-plugin` — Config plugins (presets, capability packs)
- `preflight-provider` — Provider plugins (WASM)

## Plugin Locations

Preflight discovers plugins from two directories:

1. **User plugins**: `~/.preflight/plugins/`
2. **System plugins**: `/usr/local/share/preflight/plugins/`

Each plugin lives in its own subdirectory containing a `plugin.yaml` manifest.

## Managing Plugins

### List Installed Plugins

```bash
preflight plugin list
```

Output:

```
NAME          VERSION   STATUS    DESCRIPTION
────          ───────   ──────    ───────────
docker        1.0.0     enabled   Docker provider for Preflight
kubernetes    2.0.0     enabled   Kubernetes tooling
```

### Install a Plugin

From a local path:

```bash
preflight plugin install /path/to/plugin
```

From a Git repository (coming soon):

```bash
preflight plugin install https://github.com/example/preflight-docker.git
```

### View Plugin Details

```bash
preflight plugin info kubernetes
```

### Remove a Plugin

```bash
preflight plugin remove kubernetes
```

---

## Developing Plugins

### Choosing a Plugin Type

| When to use... | Choose |
|----------------|--------|
| Sharing presets and configs | Config plugin |
| Bundling capability packs | Config plugin |
| Team/org configuration | Config plugin |
| Custom installation logic | Provider plugin |
| Integrating new tools | Provider plugin |

**Start with a config plugin.** Only create a provider plugin if you need executable logic that doesn't exist in built-in providers.

---

## Config Plugins

Config plugins are pure YAML — they share presets, capability packs, and configurations without any executable code.

### Directory Structure

```
my-config-plugin/
├── plugin.yaml    # Required: manifest file
├── presets/       # Optional: preset definitions
│   ├── basic.yaml
│   └── advanced.yaml
└── README.md      # Optional: documentation
```

### Config Plugin Manifest

```yaml
apiVersion: v1
type: config          # Optional, defaults to "config"
name: my-team-config
version: 1.0.0
description: Team configuration presets
author: Your Team
license: MIT
homepage: https://github.com/your-team/preflight-config
repository: https://github.com/your-team/preflight-config

# What this plugin provides (at least one required)
provides:
  presets:
    - my-team:backend
    - my-team:frontend
  capabilityPacks:
    - team-developer

# Optional: Dependencies on other plugins
requires:
  - name: docker
    version: ">=1.0.0"

# Optional: Minimum Preflight version
minPreflightVersion: "2.0.0"
```

**Validation rules for config plugins:**
- Must have `apiVersion: v1`
- Must provide at least one preset, capability pack, or provider config
- Must NOT have a `wasm` section (that's for provider plugins)

### Manifest Fields

| Field | Required | Description |
|-------|----------|-------------|
| `apiVersion` | Yes | Must be `v1` |
| `type` | No | `config` (default) or `provider` |
| `name` | Yes | Plugin identifier (e.g., `docker`, `kubernetes`) |
| `version` | Yes | Semantic version (e.g., `1.0.0`) |
| `description` | No | Brief description |
| `author` | No | Plugin author name |
| `license` | No | License identifier (e.g., `MIT`, `Apache-2.0`) |
| `homepage` | No | Plugin homepage URL |
| `repository` | No | Source repository URL |
| `keywords` | No | Searchable tags |
| `provides` | Yes* | Capabilities this plugin offers |
| `requires` | No | Plugin dependencies |
| `minPreflightVersion` | No | Minimum Preflight version |
| `wasm` | Provider only | WASM module configuration |

\* Config plugins must provide at least one preset, capability pack, or provider.

### Provides Section

Each provider in `provides.providers` needs:

```yaml
provides:
  providers:
    - name: tool-name        # Provider identifier
      configKey: tool.config # Config section handled
      description: What it does
```

The `configKey` maps to a section in `preflight.yaml`:

```yaml
# preflight.yaml
tool:
  config:
    option: value
```

### Presets and Capability Packs

Plugins can contribute to the catalog:

```yaml
provides:
  presets:
    - my-plugin:minimal
    - my-plugin:full
  capabilityPacks:
    - my-developer
```

Define preset content in `presets/*.yaml`:

```yaml
# presets/minimal.yaml
id: "my-plugin:minimal"
metadata:
  title: "Minimal Setup"
  description: "Basic my-tool configuration"
config:
  my_tool:
    enabled: true
```

### Dependencies

Declare dependencies on other plugins:

```yaml
requires:
  - name: docker
    version: ">=1.0.0"
  - name: base-tools  # No version = any version
```

Version constraints follow semver:
- `>=1.0.0` - At least version 1.0.0
- `^1.0.0` - Compatible with 1.x.x
- `~1.0.0` - Compatible with 1.0.x
- `1.0.0` - Exact version

---

## Provider Plugins

Provider plugins contain executable logic that runs in a WebAssembly sandbox. Use these when you need custom behavior that built-in providers don't support.

### Directory Structure

```
my-provider-plugin/
├── plugin.yaml    # Required: manifest with WASM config
├── plugin.wasm    # Required: compiled WASM module
├── presets/       # Optional: preset definitions
└── README.md      # Optional: documentation
```

### Provider Plugin Manifest

```yaml
apiVersion: v1
type: provider        # Required for provider plugins
name: docker
version: 1.0.0
description: Docker provider for Preflight
author: Docker Team
license: MIT

# Providers this plugin implements
provides:
  providers:
    - name: docker
      configKey: docker
      description: Install and configure Docker Desktop
  presets:
    - docker:basic
    - docker:kubernetes

# WASM configuration (required for provider plugins)
wasm:
  module: plugin.wasm
  checksum: sha256:abc123def456...
  capabilities:
    - name: shell:execute
      justification: Run docker commands
    - name: files:write
      justification: Write Docker configuration files
    - name: network:fetch
      justification: Download Docker releases
      optional: true
```

**Validation rules for provider plugins:**
- Must have `type: provider`
- Must have a `wasm` section with `module` and `checksum`
- Each capability must have a `name` and `justification`

### WASM Section

| Field | Required | Description |
|-------|----------|-------------|
| `module` | Yes | Path to WASM file (e.g., `plugin.wasm`) |
| `checksum` | Yes | SHA256 hash for integrity verification |
| `capabilities` | No | Declared permissions the plugin needs |

### Capability Declarations

Provider plugins must declare what they need access to:

```yaml
wasm:
  capabilities:
    - name: shell:execute
      justification: Run docker CLI commands
    - name: files:read
      justification: Read existing Docker config
    - name: files:write
      justification: Write Docker daemon.json
    - name: network:fetch
      justification: Download Docker releases
      optional: true
```

Each capability requires:
- `name` — The capability identifier (see table below)
- `justification` — Human-readable explanation of why it's needed
- `optional` — If true, plugin works without this capability

### Available Capabilities

| Capability | Description |
|------------|-------------|
| `files:read` | Read files from disk |
| `files:write` | Write/modify files |
| `shell:execute` | Run shell commands |
| `network:fetch` | Make HTTP requests |
| `packages:brew` | Install Homebrew packages |
| `packages:apt` | Install APT packages |
| `secrets:read` | Access secret references |
| `system:modify` | Modify system settings |

---

## Example: Config Plugin

Here's a complete config plugin that shares Docker presets:

```
docker-presets/
├── plugin.yaml
├── presets/
│   ├── basic.yaml
│   └── kubernetes.yaml
└── README.md
```

**plugin.yaml:**

```yaml
apiVersion: v1
type: config
name: docker-presets
version: 1.0.0
description: Docker configuration presets for Preflight
author: Docker Community
license: MIT
homepage: https://github.com/preflight-plugins/docker-presets

provides:
  presets:
    - docker-presets:basic
    - docker-presets:kubernetes
  capabilityPacks:
    - container-developer
```

**presets/basic.yaml:**

```yaml
id: "docker-presets:basic"
metadata:
  title: "Basic Docker"
  description: "Docker Desktop with Docker Compose"
  difficulty: beginner
config:
  docker:
    install: true
    compose: true
```

**presets/kubernetes.yaml:**

```yaml
id: "docker-presets:kubernetes"
metadata:
  title: "Docker + Kubernetes"
  description: "Docker Desktop with local Kubernetes"
  difficulty: intermediate
config:
  docker:
    install: true
    compose: true
    kubernetes: true
```

---

## Example: Provider Plugin

Here's a provider plugin with WASM executable logic:

```
preflight-docker/
├── plugin.yaml
├── plugin.wasm
├── presets/
│   └── full.yaml
└── README.md
```

**plugin.yaml:**

```yaml
apiVersion: v1
type: provider
name: docker
version: 1.0.0
description: Docker provider for Preflight
author: Docker Team
license: Apache-2.0

provides:
  providers:
    - name: docker
      configKey: docker
      description: Install and configure Docker Desktop

wasm:
  module: plugin.wasm
  checksum: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
  capabilities:
    - name: shell:execute
      justification: Run docker CLI commands
    - name: files:write
      justification: Write Docker daemon.json config
```

---

## Publishing Plugins

To share your plugin with the community:

1. **Create a GitHub repository** with your plugin code
2. **Add GitHub topics** for discoverability:
   - `preflight-plugin` — For config plugins
   - `preflight-provider` — For provider plugins (WASM)
3. **Tag releases** using semantic versioning (e.g., `v1.0.0`)
4. **Document installation** in your README

**Example README:**

```markdown
## Installation

```bash
# Clone to plugins directory
git clone https://github.com/you/preflight-my-plugin \
    ~/.preflight/plugins/my-plugin

# Verify installation
preflight plugin info my-plugin
```

## Discovery

Your plugin will appear in search results once you add the GitHub topics:

```bash
# Users can find your plugin with:
preflight plugin search my-plugin
```

---

## Plugin Security

Preflight's two-tier architecture is designed with security in mind:

### Security by Type

| Plugin Type | Security Model |
|-------------|----------------|
| **Config** | Safe by design — pure YAML, no execution |
| **Provider** | Sandboxed — WASM with declared capabilities |

**Config plugins are inherently safe** because they contain no executable code. They can only define presets, capability packs, and configuration values.

**Provider plugins run in a WASM sandbox** with explicit capability declarations and resource limits.

### Defense-in-Depth (Provider Plugins)

```
┌─────────────────────────────────────┐
│  Sandbox (WASM isolation)           │  ← Can't escape even if malicious
├─────────────────────────────────────┤
│  Capabilities (permissions)         │  ← Can only do what's declared
├─────────────────────────────────────┤
│  Checksums (integrity)              │  ← Know it wasn't tampered with
├─────────────────────────────────────┤
│  Signatures (identity) [future]     │  ← Know who published it
└─────────────────────────────────────┘
```

### Resource Limits (Provider Plugins)

WASM provider plugins run with these constraints:

| Limit | Default | Purpose |
|-------|---------|---------|
| Memory | 64MB | Prevent memory exhaustion |
| CPU time | 30s | Prevent infinite loops |
| File descriptors | 32 | Limit open files |
| Output | 1MB | Limit stdout/stderr |

### Trust Levels

| Level | Description | Source |
|-------|-------------|--------|
| `builtin` | Core Preflight providers | Shipped with Preflight |
| `verified` | Reviewed by Preflight team | Future: signed plugins |
| `community` | Community contributions | GitHub with topic |
| `untrusted` | Unknown source | Local/unverified |

### Recommendations

1. **Prefer config plugins** when possible — they're safe by design
2. **Review capabilities** before installing provider plugins
3. **Check checksums** match what's documented
4. **Use trusted sources** like official GitHub topics

---

## What's Next?

- [Providers](/preflight/guides/providers/) — Built-in provider reference
- [Configuration](/preflight/guides/configuration/) — Configuration guide
- [CLI Commands](/preflight/cli/commands/) — Full CLI reference
