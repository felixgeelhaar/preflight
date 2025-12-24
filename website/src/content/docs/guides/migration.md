---
title: Migration Guide
description: Migrate to Preflight from manual dotfiles, chezmoi, yadm, stow, or shell scripts.
---

This guide helps you migrate your existing dotfiles setup to Preflight, whether you're coming from manual dotfiles, another dotfiles manager, or custom scripts.

## Before You Start

Preflight captures your current machine state automatically. You don't need to migrate manually:

```bash
# Capture your current setup
preflight capture

# Review findings in TUI
# Accept/reject each item
# Assign to appropriate layers
```

This works regardless of how your dotfiles are currently managed.

---

## From Manual Dotfiles

If you're managing dotfiles with a simple git repository and symlinks:

### Step 1: Capture Current State

```bash
preflight capture --include git,shell,nvim
```

This detects:
- Git configuration (user, aliases, core settings)
- Shell config (zsh/bash, aliases, environment variables)
- Neovim configuration

### Step 2: Review and Accept

The TUI shows each captured item. Use:
- `y` — Accept item
- `n` — Reject item
- `l` — Assign to a different layer
- `e` — Edit before accepting

### Step 3: Import Existing Dotfiles

If you have custom dotfiles in a repository:

```yaml
# layers/base.yaml
files:
  links:
    - src: dotfiles/.zshrc
      dest: ~/.zshrc
    - src: dotfiles/.gitconfig.local
      dest: ~/.gitconfig.local
    - src: dotfiles/nvim
      dest: ~/.config/nvim
```

### What Changes

| Before | After |
|--------|-------|
| Manual symlinks | `preflight apply` creates links |
| Edit files directly | Edit layers, apply changes |
| Mental tracking | `preflight doctor` detects drift |
| Manual updates | `preflight capture` finds new settings |

---

## From chezmoi

If you're using [chezmoi](https://chezmoi.io/):

### Key Differences

| chezmoi | Preflight |
|---------|-----------|
| Templates with Go syntax | YAML-native config |
| `~/.local/share/chezmoi` source | `preflight.yaml` + `layers/` |
| `chezmoi apply` | `preflight apply` |
| `.chezmoiignore` | `files.exclude` in config |
| Scripting with `.chezmoiscripts` | Provider steps with dependencies |

### Migration Steps

1. **Export chezmoi state:**
   ```bash
   # List all managed files
   chezmoi managed

   # Export current state
   chezmoi dump > chezmoi-state.yaml
   ```

2. **Capture with Preflight:**
   ```bash
   preflight capture
   ```

3. **Map templates:**

   chezmoi template:
   ```
   {{ .email }}
   ```

   Preflight equivalent (in layer):
   ```yaml
   git:
     user:
       email: "{{ .Values.email }}"
   ```

   Or use environment variables:
   ```yaml
   git:
     user:
       email: ${EMAIL}
   ```

4. **Migrate scripts:**

   chezmoi script (`run_once_install-packages.sh`):
   ```bash
   #!/bin/bash
   brew install ripgrep fzf
   ```

   Preflight equivalent:
   ```yaml
   packages:
     brew:
       formulae:
         - ripgrep
         - fzf
   ```

### Gradual Migration

Run both tools during migration:

```bash
# Use chezmoi for files not yet migrated
chezmoi apply

# Use Preflight for migrated sections
preflight apply --target minimal
```

---

## From yadm

If you're using [yadm](https://yadm.io/):

### Key Differences

| yadm | Preflight |
|------|-----------|
| Bare git repo in `$HOME` | Explicit config in project directory |
| Class-based alternates | Layer-based composition |
| Encryption with GPG | Secrets by reference only |
| Bootstrap scripts | Provider-based steps |

### Migration Steps

1. **List yadm-managed files:**
   ```bash
   yadm list
   ```

2. **Capture with Preflight:**
   ```bash
   preflight capture
   ```

3. **Map alternates to layers:**

   yadm alternate (`~/.gitconfig##class.work`):
   ```ini
   [user]
       email = work@company.com
   ```

   Preflight layer (`layers/identity.work.yaml`):
   ```yaml
   name: identity.work
   git:
     user:
       email: work@company.com
   ```

4. **Define targets:**
   ```yaml
   # preflight.yaml
   targets:
     work:
       - base
       - identity.work
       - role.dev
     personal:
       - base
       - identity.personal
   ```

### Encrypted Files

yadm encrypts files with GPG. Preflight doesn't store secrets:

```yaml
# Instead of encrypted files, use references
ssh:
  hosts:
    - name: production
      identity_file: ~/.ssh/prod_key  # Key stays on disk
```

For sensitive config, use environment variables:
```yaml
git:
  user:
    signingkey: ${GPG_KEY_ID}
```

---

## From GNU Stow

If you're using [GNU Stow](https://www.gnu.org/software/stow/):

### Key Differences

| stow | Preflight |
|------|-----------|
| Directory-based packages | YAML layers |
| `stow -t ~ package` | `preflight apply` |
| Symlinks only | Symlinks + templating + packages + more |
| Manual conflict resolution | Drift detection with doctor |

### Migration Steps

1. **Identify stow packages:**
   ```bash
   ls ~/dotfiles/
   # zsh/  git/  nvim/  tmux/
   ```

2. **Capture with Preflight:**
   ```bash
   preflight capture
   ```

3. **Map packages to layers:**

   Stow package (`dotfiles/zsh/.zshrc`):
   ```bash
   stow -t ~ zsh
   ```

   Preflight config:
   ```yaml
   # layers/base.yaml
   files:
     links:
       - src: dotfiles/zsh/.zshrc
         dest: ~/.zshrc
   ```

4. **Consolidate configuration:**

   Instead of managing raw dotfiles, declare intent:
   ```yaml
   shell:
     default: zsh
     framework: oh-my-zsh
     plugins:
       - git
       - docker
     aliases:
       ll: "ls -la"
   ```

### Hybrid Approach

Keep stow for some packages while migrating:

```yaml
# Use files.links for stow-like behavior
files:
  links:
    # Keep existing stow structure
    - src: dotfiles/tmux/.tmux.conf
      dest: ~/.tmux.conf

    # Or link entire directories
    - src: dotfiles/nvim
      dest: ~/.config/nvim
```

---

## From Ansible/Shell Scripts

If you're using Ansible playbooks or shell scripts:

### Key Differences

| Ansible/Scripts | Preflight |
|-----------------|-----------|
| Imperative tasks | Declarative config |
| YAML playbooks / bash | YAML layers |
| `ansible-playbook` | `preflight apply` |
| Manual idempotency | Built-in idempotency |
| Task ordering | Automatic dependency resolution |

### Migration Steps

1. **Map Ansible tasks to Preflight config:**

   Ansible task:
   ```yaml
   - name: Install packages
     homebrew:
       name:
         - ripgrep
         - fzf
         - bat
       state: present
   ```

   Preflight config:
   ```yaml
   packages:
     brew:
       formulae:
         - ripgrep
         - fzf
         - bat
   ```

2. **Map shell scripts to providers:**

   Shell script:
   ```bash
   # Install oh-my-zsh
   sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

   # Set zsh as default
   chsh -s $(which zsh)
   ```

   Preflight config:
   ```yaml
   shell:
     default: zsh
     framework: oh-my-zsh
     theme: robbyrussell
   ```

3. **Convert conditionals to targets:**

   Ansible conditionals:
   ```yaml
   - name: Work git config
     when: profile == "work"
   ```

   Preflight targets:
   ```yaml
   targets:
     work:
       - base
       - identity.work
   ```

### Common Patterns

| Ansible Module | Preflight Provider |
|----------------|-------------------|
| `homebrew` | `packages.brew` |
| `apt` | `packages.apt` |
| `git_config` | `git` |
| `file` (link) | `files.links` |
| `template` | `files.templates` |
| `shell` | Not needed (declarative) |

---

## Migration Checklist

Before switching fully to Preflight:

- [ ] Run `preflight capture` to capture current state
- [ ] Review all captured items in TUI
- [ ] Assign items to appropriate layers (base, identity, role)
- [ ] Run `preflight plan` to verify expected changes
- [ ] Run `preflight apply --dry-run` to simulate
- [ ] Back up existing dotfiles repository
- [ ] Run `preflight apply` to apply configuration
- [ ] Run `preflight doctor` to verify everything is correct
- [ ] Test on a fresh machine or VM

## Getting Help

If you encounter issues during migration:

1. Check the [Troubleshooting Guide](/preflight/guides/troubleshooting/)
2. Run `preflight doctor` to diagnose problems
3. Use `preflight plan --explain` to understand changes
4. Open an issue on [GitHub](https://github.com/felixgeelhaar/preflight/issues)
