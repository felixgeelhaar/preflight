---
title: Providers
description: System integration adapters that compile configuration into executable steps.
---

Providers are the adapters that translate configuration into executable, idempotent steps. Each provider handles a specific aspect of system configuration.

## Provider Architecture

```
Configuration (YAML)
       ↓
    Provider
       ↓
  Step Graph (DAG)
       ↓
    Execution
```

Each provider implements:

```go
type Provider interface {
    Name() string
    Compile(ctx CompileContext) ([]Step, error)
}
```

## Available Providers

### brew (macOS)

Homebrew package management for macOS.

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
      - yq

    casks:
      - visual-studio-code
      - iterm2
      - docker
      - 1password
```

**Capabilities:**
- Automatic Homebrew installation (with consent)
- Tap management
- Formula installation with version locking
- Cask installation
- Leaf package detection during capture

**Lock behavior:**
- Records tap commits (best-effort)
- Records resolved versions

### apt (Linux)

Package management for Debian/Ubuntu systems.

```yaml
packages:
  apt:
    packages:
      - git
      - curl
      - build-essential
      - python3-pip

    ppas:
      - ppa:neovim-ppa/stable
```

**Capabilities:**
- Package installation
- PPA management
- Version locking (best-effort)

### docker

Docker Desktop and container configuration.

```yaml
docker:
  install: true
  compose: true
  buildkit: true
  kubernetes: false

  # Resource limits (Docker Desktop)
  resource_limits:
    cpus: 4
    memory: "8GB"
    swap: "2GB"
    disk: "100GB"

  # Remote contexts
  contexts:
    - name: production
      host: ssh://deploy@prod.example.com
      description: "Production Docker host"
      default: false

    - name: staging
      host: ssh://deploy@staging.example.com

  # Registry configuration
  registries:
    - ghcr.io
    - url: registry.example.com
      username: admin
      insecure: false
```

**Capabilities:**
- Docker Desktop installation (via Homebrew on macOS)
- Docker Compose support
- BuildKit enablement for optimized builds
- Kubernetes cluster enablement
- Multi-host context management
- Private registry configuration
- Resource limit configuration

**Steps produced:**
- `docker:install` — Install Docker Desktop
- `docker:buildkit` — Enable BuildKit builder
- `docker:kubernetes` — Enable Kubernetes cluster
- `docker:context:*` — Create named contexts

**Presets:**
- **basic** — Docker Desktop with Compose
- **kubernetes** — Docker with local Kubernetes
- **buildkit** — Docker with BuildKit optimizations
- **full** — All features enabled

### files

Dotfile and configuration file management.

```yaml
files:
  # Symbolic links
  links:
    - src: dotfiles/.zshrc
      dest: ~/.zshrc

    - src: dotfiles/nvim
      dest: ~/.config/nvim

  # Template rendering
  templates:
    - src: templates/gitconfig.tmpl
      dest: ~/.gitconfig
      vars:
        name: "{{ .Git.User.Name }}"
        email: "{{ .Git.User.Email }}"

  # One-time copies
  copies:
    - src: defaults/.vimrc
      dest: ~/.vimrc
      overwrite: false
```

**Capabilities:**
- Symbolic linking
- Template rendering (Go templates)
- Snapshot before modification
- Drift detection via hashing
- Three-way merge for conflicts

**Dotfile modes:**
1. **Generated** — Preflight owns the file
2. **Template + override** — Base with user extensions
3. **Bring-your-own** — Link/validate only

### git

Git configuration management.

```yaml
git:
  user:
    name: Your Name
    email: you@example.com
    signingkey: ABC123DEF456

  core:
    editor: nvim
    autocrlf: input
    pager: delta

  init:
    defaultBranch: main

  aliases:
    co: checkout
    br: branch
    ci: commit
    st: status
    lg: "log --oneline --graph --decorate"

  includes:
    - path: ~/.gitconfig.local
    - path: ~/.gitconfig.work
      condition: "gitdir:~/work/"
```

**Capabilities:**
- Generate ~/.gitconfig
- Identity separation via includes
- GPG signing configuration
- Conditional includes for multi-identity

### ssh

SSH configuration management.

```yaml
ssh:
  defaults:
    add_keys_to_agent: yes
    identity_file: ~/.ssh/id_ed25519

  hosts:
    - name: github.com
      user: git
      identity_file: ~/.ssh/github_ed25519

    - name: work-server
      hostname: server.work.example.com
      user: deploy
      port: 22
      identity_file: ~/.ssh/work_rsa

    - name: "*.internal"
      hostname: "%h.internal.company.com"
      proxy_jump: bastion
```

**Capabilities:**
- Generate ~/.ssh/config
- Host and Match block support
- Identity separation
- Never exports private keys
- Secret references only

### shell

Shell environment configuration.

```yaml
shell:
  default: zsh  # zsh | bash | fish

  # Framework
  framework: oh-my-zsh
  theme: robbyrussell

  plugins:
    - git
    - docker
    - kubectl
    - fzf
    - zsh-autosuggestions
    - zsh-syntax-highlighting

  # Starship prompt
  starship:
    enabled: true
    config: |
      [character]
      success_symbol = "[➜](bold green)"
      error_symbol = "[➜](bold red)"

  # Environment variables
  env:
    EDITOR: nvim
    VISUAL: nvim
    PAGER: less

  # Aliases
  aliases:
    ll: "ls -la"
    g: git
    k: kubectl

  # Path additions
  path:
    - ~/.local/bin
    - ~/go/bin
```

**Capabilities:**
- Shell selection and configuration
- Oh-My-Zsh, Fisher, Zinit support
- Starship prompt integration
- Environment variables
- Aliases and path management

### runtime

Tool version management (rtx/asdf compatible).

```yaml
runtime:
  manager: rtx  # rtx | asdf

  tools:
    node: "20.10.0"
    python: "3.12.0"
    go: "1.23.0"
    rust: "1.75.0"
    ruby: "3.3.0"
```

**Capabilities:**
- Install runtime manager
- Install tool versions
- Lock resolved versions
- Global and project-level versions

### nvim

Neovim configuration and plugin management.

```yaml
nvim:
  install: true

  # Preset configuration
  preset: balanced  # minimal | balanced | pro

  # Or custom config
  config:
    source: git
    repo: https://github.com/LazyVim/starter
    ref: main

  # External dependencies
  dependencies:
    - ripgrep
    - fd
    - lazygit
    - node  # For LSP servers

  # Plugin lock
  lazy_lock: true
```

**Capabilities:**
- Install Neovim via brew/apt
- Bootstrap from presets or git repos
- Headless plugin sync (`nvim --headless '+Lazy sync' +qa`)
- Lock via lazy-lock.json
- Doctor checks for external dependencies

**Presets:**
- **minimal** — Basic editing, no plugins
- **balanced** — LazyVim with essential plugins
- **pro** — Full IDE experience

### vscode

VS Code / Cursor configuration.

```yaml
vscode:
  extensions:
    - ms-python.python
    - golang.go
    - esbenp.prettier-vscode
    - dbaeumer.vscode-eslint
    - eamodio.gitlens
    - github.copilot

  settings:
    editor.fontSize: 14
    editor.tabSize: 2
    editor.formatOnSave: true
    editor.defaultFormatter: "esbenp.prettier-vscode"
    workbench.colorTheme: "One Dark Pro"

  keybindings:
    - key: "cmd+shift+f"
      command: "editor.action.formatDocument"
```

**Capabilities:**
- Install extensions by ID
- Apply settings.json
- Manage keybindings.json
- Lock installed versions (best-effort)
- Detect settings drift

## Provider Execution

### Step Interface

Each provider produces steps that implement:

```go
type Step interface {
    ID() string
    DependsOn() []string
    Check(ctx RunContext) (Status, error)   // satisfied | needs-apply
    Plan(ctx RunContext) (Diff, error)
    Apply(ctx RunContext) error
    Explain(ctx ExplainContext) Explanation
}
```

### Step Graph (DAG)

Steps are topologically sorted based on dependencies:

```
[brew: Install git]
         ↓
[git: Configure ~/.gitconfig]
         ↓
[ssh: Configure ~/.ssh/config]
```

### Idempotency

All steps are idempotent:
- `Check()` determines if action is needed
- `Apply()` only runs if `Check()` returns `needs-apply`
- Re-running is always safe

## Provider Development

Providers follow the ports & adapters pattern:

```
internal/
  provider/
    brew/
      provider.go   # Provider implementation
      steps.go      # Step definitions
      doctor.go     # Health checks
```

### Adding a New Provider

1. Implement the `Provider` interface
2. Define steps with idempotent `Apply()`
3. Add doctor checks
4. Register in the compiler

## What's Next?

- [Dotfile Management](/preflight/guides/dotfiles/) — Deep dive into the files provider
- [Architecture Overview](/preflight/architecture/overview/) — System design
- [CLI Commands](/preflight/cli/commands/) — Command reference
