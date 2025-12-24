---
title: Dotfile Management
description: Managing configuration files with Preflight's files provider.
---

Preflight provides comprehensive dotfile management through the files provider, supporting symlinks, templates, and automatic drift detection.

## Dotfile Modes

Preflight supports three modes for managing dotfiles:

### 1. Generated

Preflight completely owns the file. Best for beginners and non-engineers.

```yaml
files:
  generated:
    - dest: ~/.gitconfig
      provider: git  # Generated from git provider config
```

### 2. Template + Override

Preflight manages a base; users can extend safely.

```yaml
files:
  templates:
    - src: templates/zshrc.tmpl
      dest: ~/.zshrc
      vars:
        plugins: ["git", "docker", "fzf"]
```

Template example (`templates/zshrc.tmpl`):

```bash
# Managed by Preflight - DO NOT EDIT ABOVE THIS LINE
export ZSH="$HOME/.oh-my-zsh"
plugins=({{ range .plugins }}{{ . }} {{ end }})
source $ZSH/oh-my-zsh.sh

# === USER CUSTOMIZATION BELOW ===
# Add your custom configuration here
```

### 3. Bring-Your-Own

Preflight links/validates only; never rewrites.

```yaml
files:
  links:
    - src: dotfiles/nvim
      dest: ~/.config/nvim
```

## Configuration

### Symbolic Links

Create symlinks from your dotfiles directory:

```yaml
files:
  links:
    # Single file
    - src: dotfiles/.zshrc
      dest: ~/.zshrc

    # Directory
    - src: dotfiles/nvim
      dest: ~/.config/nvim

    # With backup
    - src: dotfiles/.vimrc
      dest: ~/.vimrc
      backup: true  # Creates ~/.vimrc.bak if exists
```

### Templates

Render Go templates with configuration values:

```yaml
files:
  templates:
    - src: templates/gitconfig.tmpl
      dest: ~/.gitconfig
      vars:
        name: "{{ .Git.User.Name }}"
        email: "{{ .Git.User.Email }}"

    - src: templates/starship.toml.tmpl
      dest: ~/.config/starship.toml
```

Template syntax:

```
# templates/gitconfig.tmpl
[user]
    name = {{ .name }}
    email = {{ .email }}
{{ if .signingkey }}
    signingkey = {{ .signingkey }}
{{ end }}

[core]
    editor = {{ .editor | default "vim" }}
```

### Copies

One-time file copies (doesn't overwrite existing):

```yaml
files:
  copies:
    - src: defaults/.vimrc
      dest: ~/.vimrc
      overwrite: false  # Skip if exists

    - src: defaults/settings.json
      dest: ~/.config/Code/User/settings.json
      overwrite: true   # Always overwrite
```

## Drift Detection

Preflight tracks file state to detect external changes.

### How It Works

1. After applying, Preflight records file hashes in `~/.preflight/state.json`
2. `preflight doctor` compares current hashes to recorded state
3. Differences are reported as drift

### Checking for Drift

```bash
preflight doctor
```

Output:

```
Drift Detection

! ~/.zshrc
  └─ Modified externally
  └─ Expected hash: abc123
  └─ Current hash:  def456

! ~/.config/nvim/init.lua
  └─ Modified externally

2 files have drifted from applied state
```

### Resolving Drift

**Option 1: Converge machine to config**

```bash
preflight doctor --fix
```

**Option 2: Update config to match machine**

```bash
preflight doctor --update-config
```

**Option 3: Preview changes first**

```bash
preflight doctor --update-config --dry-run
```

## Automatic Snapshots

Before any file modification, Preflight creates a backup.

### Snapshot Location

```
~/.preflight/snapshots/
  2024-12-24T10:30:00/
    .zshrc
    .gitconfig
  2024-12-24T14:15:00/
    .zshrc
```

### Restoring from Snapshot

```bash
# List snapshots
ls ~/.preflight/snapshots/

# Manual restore
cp ~/.preflight/snapshots/2024-12-24T10:30:00/.zshrc ~/.zshrc
```

## Three-Way Merge

When both config and file have changed, Preflight uses three-way merge.

### Change Detection

| Scenario | Resolution |
|----------|------------|
| Only config changed | Apply config |
| Only file changed | Offer `--update-config` |
| Both changed identically | Clean merge |
| Both changed differently | Conflict markers |

### Conflict Markers

When conflicts occur, Preflight generates Git-style markers:

```
<<<<<<< ours (config)
export EDITOR=nvim
=======
export EDITOR=vim
>>>>>>> theirs (file)
```

Or diff3-style with base content:

```
<<<<<<< ours (config)
export EDITOR=nvim
||||||| base
export EDITOR=nano
=======
export EDITOR=vim
>>>>>>> theirs (file)
```

### Resolving Conflicts

Edit the file manually and remove conflict markers, then run:

```bash
preflight apply
```

## File Organization

Recommended dotfiles structure:

```
~/dotfiles/
  preflight.yaml
  layers/
    base.yaml
    identity.work.yaml
  dotfiles/
    .zshrc
    .gitignore_global
    nvim/
      init.lua
      lua/
  templates/
    gitconfig.tmpl
    starship.toml.tmpl
  defaults/
    .vimrc
```

## Security

### Never Store Secrets

```yaml
# Bad - secrets in config
git:
  user:
    signingkey: "-----BEGIN PGP PRIVATE KEY-----..."

# Good - reference to file
git:
  user:
    signingkey: ABC123DEF  # Key ID only
```

### Redaction During Capture

When running `preflight capture`, sensitive data is automatically redacted:

- API tokens
- Private keys
- Passwords
- Credential files

## Best Practices

### 1. Use Version Control

```bash
cd ~/dotfiles
git init
git add .
git commit -m "Initial dotfiles"
```

### 2. Separate Concerns

```yaml
files:
  links:
    # Shell config
    - src: dotfiles/shell/.zshrc
      dest: ~/.zshrc

    # Editor config
    - src: dotfiles/nvim
      dest: ~/.config/nvim

    # Git config (via provider, not direct link)
```

### 3. Test Changes

```bash
# Preview what will change
preflight plan --diff

# Apply with confirmation
preflight apply
```

### 4. Regular Doctor Checks

```bash
# Add to your workflow
preflight doctor
```

## What's Next?

- [Providers](/preflight/guides/providers/) — All available providers
- [CLI Commands](/preflight/cli/commands/) — Full command reference
- [TDD Workflow](/preflight/development/tdd/) — Development practices
