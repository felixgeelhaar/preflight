---
title: Installation
description: How to install Preflight on macOS and Linux.
---

Preflight can be installed via Go, Homebrew, or built from source.

## Prerequisites

- **macOS** or **Linux** (Windows WSL supported)
- **Go 1.23+** (for `go install` method)

## Installation Methods

### Using Go

```bash
go install github.com/felixgeelhaar/preflight@latest
```

### Using Homebrew

```bash
brew tap felixgeelhaar/tap
brew install preflight
```

### From Source

```bash
git clone https://github.com/felixgeelhaar/preflight.git
cd preflight
make build
./bin/preflight version
```

### Download Binary

Download pre-built binaries from the [GitHub Releases](https://github.com/felixgeelhaar/preflight/releases) page:

- `preflight-darwin-amd64.tar.gz` — macOS Intel
- `preflight-darwin-arm64.tar.gz` — macOS Apple Silicon
- `preflight-linux-amd64.tar.gz` — Linux x64
- `preflight-linux-arm64.tar.gz` — Linux ARM64

```bash
# Example for macOS Apple Silicon
curl -L https://github.com/felixgeelhaar/preflight/releases/latest/download/preflight-darwin-arm64.tar.gz | tar xz
sudo mv preflight /usr/local/bin/
```

## Verify Installation

```bash
preflight version
```

You should see output like:

```
preflight version 1.3.0
```

## Shell Completion

Generate shell completion scripts for enhanced CLI experience:

```bash
# Bash
preflight completion bash > /etc/bash_completion.d/preflight

# Zsh
preflight completion zsh > "${fpath[1]}/_preflight"

# Fish
preflight completion fish > ~/.config/fish/completions/preflight.fish

# PowerShell
preflight completion powershell > preflight.ps1
```

## What's Next?

- [Quick Start](/preflight/getting-started/quickstart/) — Create your first configuration
- [Configuration Guide](/preflight/guides/configuration/) — Learn the configuration model
