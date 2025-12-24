---
title: Plugins
description: Extend Preflight with community plugins.
---

Preflight supports plugins that add new providers, presets, and capability packs. This guide covers finding, installing, and developing plugins.

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

### Directory Structure

A plugin has this structure:

```
my-plugin/
├── plugin.yaml    # Required: manifest file
├── presets/       # Optional: preset definitions
│   ├── basic.yaml
│   └── advanced.yaml
└── README.md      # Optional: documentation
```

### Plugin Manifest

The `plugin.yaml` manifest describes your plugin:

```yaml
apiVersion: v1
name: my-plugin
version: 1.0.0
description: My custom Preflight plugin
author: Your Name
license: MIT
homepage: https://github.com/you/preflight-my-plugin
repository: https://github.com/you/preflight-my-plugin

# What this plugin provides
provides:
  providers:
    - name: my-tool
      configKey: my_tool
      description: Manage my-tool installation
  presets:
    - my-plugin:basic
    - my-plugin:advanced
  capabilityPacks:
    - my-developer

# Dependencies on other plugins
requires:
  - name: docker
    version: ">=1.0.0"

# Minimum Preflight version required
minPreflightVersion: "2.0.0"
```

### Manifest Fields

| Field | Required | Description |
|-------|----------|-------------|
| `apiVersion` | Yes | Must be `v1` |
| `name` | Yes | Plugin identifier (e.g., `docker`, `kubernetes`) |
| `version` | Yes | Semantic version (e.g., `1.0.0`) |
| `description` | No | Brief description |
| `author` | No | Plugin author name |
| `license` | No | License identifier (e.g., `MIT`, `Apache-2.0`) |
| `homepage` | No | Plugin homepage URL |
| `repository` | No | Source repository URL |
| `keywords` | No | Searchable tags |
| `provides` | No | Capabilities this plugin offers |
| `requires` | No | Plugin dependencies |
| `minPreflightVersion` | No | Minimum Preflight version |

### Provider Specifications

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

## Example: Docker Plugin

Here's a complete example plugin:

```
preflight-docker/
├── plugin.yaml
├── presets/
│   ├── basic.yaml
│   └── kubernetes.yaml
└── README.md
```

**plugin.yaml:**

```yaml
apiVersion: v1
name: docker
version: 1.0.0
description: Docker Desktop management for Preflight
author: Docker Community
license: MIT
homepage: https://github.com/preflight-plugins/docker

provides:
  providers:
    - name: docker
      configKey: docker
      description: Install and configure Docker Desktop
  presets:
    - docker:basic
    - docker:kubernetes
  capabilityPacks:
    - container-developer
```

**presets/basic.yaml:**

```yaml
id: "docker:basic"
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
id: "docker:kubernetes"
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

## Publishing Plugins

To share your plugin with the community:

1. **Create a GitHub repository** with your plugin code
2. **Tag releases** using semantic versioning (e.g., `v1.0.0`)
3. **Document installation** in your README:

```markdown
## Installation

```bash
# Clone to plugins directory
git clone https://github.com/you/preflight-my-plugin \
    ~/.preflight/plugins/my-plugin

# Verify installation
preflight plugin info my-plugin
```

---

## What's Next?

- [Providers](/preflight/guides/providers/) — Built-in provider reference
- [Configuration](/preflight/guides/configuration/) — Configuration guide
- [CLI Commands](/preflight/cli/commands/) — Full CLI reference
