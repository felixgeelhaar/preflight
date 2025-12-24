---
title: Configuration
description: Complete guide to Preflight configuration files and syntax.
---

Preflight uses a layered YAML configuration system that enables modular, composable workstation definitions.

## Configuration Files

| File | Purpose |
|------|---------|
| `preflight.yaml` | Root manifest with targets and defaults |
| `layers/*.yaml` | Composable configuration overlays |
| `preflight.lock` | Resolved versions and integrity hashes |
| `dotfiles/` | Generated/templated configuration files |

## Manifest Structure

```yaml
# preflight.yaml
name: my-workstation
version: "1.0"

# Target platform
target: macos  # or linux

# Layers to compose (in order)
layers:
  - base
  - identity.work
  - role.go
  - device.laptop

# Inline configuration (merged with layers)
packages:
  brew:
    taps: []
    formulae: []
    casks: []

git:
  user:
    name: ""
    email: ""

shell:
  default: zsh
  framework: oh-my-zsh
  plugins: []

files:
  links: []
  templates: []

# AI advisor configuration (optional)
advisor:
  provider: none  # openai | anthropic | ollama | none
```

## Section Reference

### packages

Package manager configuration:

```yaml
packages:
  brew:
    taps:
      - homebrew/cask-fonts
      - homebrew/cask-versions
    formulae:
      - git
      - gh
      - ripgrep
      - fzf
      - jq
    casks:
      - visual-studio-code
      - iterm2
      - docker

  apt:  # Linux only
    packages:
      - git
      - curl
      - build-essential
```

### git

Git configuration:

```yaml
git:
  user:
    name: Your Name
    email: you@example.com
    signingkey: ABC123  # GPG key ID

  core:
    editor: nvim
    autocrlf: input

  aliases:
    co: checkout
    br: branch
    ci: commit
    st: status

  includes:
    - path: ~/.gitconfig.local
```

### ssh

SSH configuration:

```yaml
ssh:
  hosts:
    - name: github.com
      user: git
      identity_file: ~/.ssh/github_ed25519

    - name: work
      hostname: work.example.com
      user: deploy
      identity_file: ~/.ssh/work_rsa
      port: 22

  defaults:
    add_keys_to_agent: yes
    identity_file: ~/.ssh/id_ed25519
```

### shell

Shell configuration:

```yaml
shell:
  default: zsh  # zsh | bash | fish

  # Oh-My-Zsh configuration
  framework: oh-my-zsh
  theme: robbyrussell
  plugins:
    - git
    - docker
    - kubectl
    - fzf

  # Starship prompt
  starship:
    enabled: true
    config: |
      [character]
      success_symbol = "[➜](bold green)"

  # Environment variables
  env:
    EDITOR: nvim
    PAGER: less

  # Aliases
  aliases:
    ll: "ls -la"
    g: git
```

### files

Dotfile management:

```yaml
files:
  # Symlinks
  links:
    - src: dotfiles/.zshrc
      dest: ~/.zshrc

    - src: dotfiles/nvim
      dest: ~/.config/nvim

  # Template rendering
  templates:
    - src: templates/gitconfig.tmpl
      dest: ~/.gitconfig

  # Copy (one-time)
  copies:
    - src: defaults/.vimrc
      dest: ~/.vimrc
      overwrite: false
```

### runtime

Tool version management (rtx/asdf):

```yaml
runtime:
  manager: rtx  # rtx | asdf

  tools:
    node: "20.10.0"
    python: "3.12.0"
    go: "1.23.0"
    rust: "1.75.0"
```

### nvim

Neovim configuration:

```yaml
nvim:
  install: true

  # Preset: minimal | balanced | pro | custom
  preset: balanced

  # Or custom config
  config:
    source: git
    repo: https://github.com/LazyVim/starter
    ref: main

  # Required external tools
  dependencies:
    - ripgrep
    - fd
    - lazygit
```

### vscode

VS Code configuration:

```yaml
vscode:
  extensions:
    - ms-python.python
    - golang.go
    - esbenp.prettier-vscode
    - dbaeumer.vscode-eslint

  settings:
    editor.fontSize: 14
    editor.tabSize: 2
    editor.formatOnSave: true
```

## Merge Semantics

When layers are merged:

| Type | Behavior |
|------|----------|
| Scalars | Last-wins |
| Maps | Deep merge |
| Lists | Set union with add/remove directives |

### List Directives

```yaml
# In layers/role.go.yaml
packages:
  brew:
    formulae:
      - +go        # Add to list
      - +gopls
      - -nodejs    # Remove from list
```

## Validation

Preflight validates configuration at multiple stages:

1. **Schema validation** — Structure and types
2. **Provider validation** — Required fields, valid values
3. **Constraint evaluation** — Cross-field dependencies

Run validation manually:

```bash
preflight plan --validate-only
```

## Environment Variables

Reference environment variables in config:

```yaml
git:
  user:
    email: ${EMAIL}
    signingkey: ${GPG_KEY_ID}
```

## Secrets

Never store secrets in configuration. Use references:

```yaml
ssh:
  hosts:
    - name: production
      identity_file: ~/.ssh/prod_key  # Key file, not content
```

Preflight will:
- Never export private keys
- Redact secrets during capture
- Use system keychain/1Password references where possible
