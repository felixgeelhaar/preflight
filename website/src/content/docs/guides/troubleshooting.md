---
title: Troubleshooting
description: Common issues, debugging techniques, and recovery procedures for Preflight.
---

This guide covers common issues, debugging techniques, and recovery procedures when using Preflight.

## Quick Diagnostics

When something goes wrong, start here:

```bash
# Check overall system state
preflight doctor

# See what would change
preflight plan --explain

# View verbose output
preflight apply --verbose
```

---

## Common Errors

### Configuration Errors

#### "manifest not found"

**Cause:** Preflight can't find `preflight.yaml`.

**Solutions:**
```bash
# Check current directory
ls preflight.yaml

# Specify config path explicitly
preflight plan --config ~/dotfiles/preflight.yaml

# Initialize new config
preflight init
```

#### "layer not found: layer_name"

**Cause:** A target references a layer that doesn't exist.

**Solutions:**
```bash
# List available layers
ls layers/

# Check manifest for typos
cat preflight.yaml | grep layers -A 20
```

Ensure the layer file exists: `layers/{layer_name}.yaml`

#### "invalid YAML syntax"

**Cause:** Malformed YAML in config files.

**Solutions:**
1. Check the line number in the error message
2. Validate YAML:
   ```bash
   # Using yq
   yq eval preflight.yaml

   # Using Python
   python -c "import yaml; yaml.safe_load(open('preflight.yaml'))"
   ```
3. Common issues:
   - Missing colons after keys
   - Incorrect indentation (use spaces, not tabs)
   - Unquoted special characters

#### "duplicate key: key_name"

**Cause:** Same key appears twice in YAML file.

**Solution:** Search for duplicates:
```bash
grep -n "key_name:" layers/*.yaml
```

---

### Apply Errors

#### "permission denied"

**Cause:** Preflight lacks permission to modify files.

**Solutions:**
```bash
# Check file ownership
ls -la ~/.zshrc

# Fix permissions (if you own the file)
chmod 644 ~/.zshrc

# Check if file is locked by another process
lsof ~/.zshrc
```

#### "file exists and is not a symlink"

**Cause:** A regular file exists where Preflight wants to create a symlink.

**Solutions:**
1. Back up the existing file:
   ```bash
   mv ~/.zshrc ~/.zshrc.backup
   ```
2. Let Preflight manage it:
   ```bash
   preflight apply
   ```
3. Or exclude from management in config:
   ```yaml
   files:
     exclude:
       - ~/.zshrc
   ```

#### "brew: command not found"

**Cause:** Homebrew not installed or not in PATH.

**Solutions:**
```bash
# Check if Homebrew is installed
which brew

# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Add to PATH (Apple Silicon)
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
eval "$(/opt/homebrew/bin/brew shellenv)"
```

#### "package not found"

**Cause:** Package name is incorrect or not available.

**Solutions:**
```bash
# Search for correct package name
brew search ripgrep

# Check if tap is needed
brew info ripgrep
```

Update your config with the correct name:
```yaml
packages:
  brew:
    formulae:
      - ripgrep  # Not 'rg' or 'ripgrep-cli'
```

---

### Doctor Errors

#### "drift detected"

**Cause:** Machine state differs from config.

**Solutions:**

1. View the differences:
   ```bash
   preflight doctor
   ```

2. Fix machine to match config:
   ```bash
   preflight doctor --fix
   ```

3. Or update config to match machine:
   ```bash
   preflight doctor --update-config
   ```

4. Preview changes first:
   ```bash
   preflight doctor --fix --dry-run
   preflight doctor --update-config --dry-run
   ```

#### "missing package"

**Cause:** A package in config is not installed.

**Solutions:**
```bash
# Install missing packages
preflight apply

# Or remove from config if not needed
# Edit layers/*.yaml to remove the package
```

---

## Debugging Techniques

### Verbose Output

Enable detailed logging:

```bash
# Show all steps
preflight apply --verbose

# Show even more detail
PREFLIGHT_DEBUG=1 preflight apply
```

### Dry Run Mode

Preview changes without applying:

```bash
# See what would happen
preflight plan --dry-run

# With full diffs
preflight plan --diff
```

### Explain Mode

Understand why actions are needed:

```bash
# See explanations for each step
preflight plan --explain

# With AI assistance (if configured)
preflight plan --explain --ai-provider openai
```

### Check Step by Step

Apply incrementally:

```bash
# Plan specific provider only
preflight plan --provider brew

# Apply specific target
preflight apply --target minimal
```

### Log Files

Check Preflight logs:

```bash
# Default log location
cat ~/.preflight/logs/preflight.log

# Recent errors only
grep ERROR ~/.preflight/logs/preflight.log | tail -20
```

---

## Recovery Procedures

### Restore from Snapshot

Preflight automatically snapshots files before modifying them:

```bash
# List available snapshots
preflight rollback

# Preview restoration
preflight rollback --to <snapshot-id> --dry-run

# Restore files
preflight rollback --to <snapshot-id>

# Restore most recent snapshot
preflight rollback --latest
```

### Manual Recovery

If Preflight won't run:

1. **Check for backups:**
   ```bash
   # Preflight backups
   ls ~/.preflight/snapshots/

   # Your git backups
   git -C ~/dotfiles log --oneline -5
   ```

2. **Restore from git:**
   ```bash
   cd ~/dotfiles
   git checkout HEAD -- .
   ```

3. **Restore individual files:**
   ```bash
   # Find in snapshots
   ls ~/.preflight/snapshots/*/files/

   # Copy back
   cp ~/.preflight/snapshots/<id>/files/.zshrc ~/.zshrc
   ```

### Reset Configuration

Start fresh while preserving data:

```bash
# Back up current config
cp -r ~/dotfiles ~/dotfiles.backup

# Re-capture current state
preflight capture --review
```

### Lock File Issues

If the lockfile is corrupted:

```bash
# Regenerate lockfile
rm preflight.lock
preflight lock update

# Or use intent mode (ignore lock)
preflight apply --mode intent
```

---

## Provider-Specific Issues

### Homebrew

**"Already installed but not linked"**
```bash
brew link --overwrite <formula>
```

**"Cask already installed"**
```bash
brew reinstall --cask <cask>
```

**"Permission denied in /usr/local"**
```bash
sudo chown -R $(whoami) /usr/local/*
```

### Git

**"gpg failed to sign the data"**
```bash
# Test GPG signing
echo "test" | gpg --clearsign

# Restart GPG agent
gpgconf --kill gpg-agent
gpg-agent --daemon
```

**"user.email is not set"**
```bash
# Verify config
git config --global user.email

# Set in layer
# layers/identity.work.yaml
git:
  user:
    email: your@email.com
```

### SSH

**"Permission denied (publickey)"**
```bash
# Check SSH key is loaded
ssh-add -l

# Add key
ssh-add ~/.ssh/id_ed25519

# Test connection
ssh -T git@github.com
```

### Neovim

**"Health check warnings"**
```bash
# Run Neovim health check
nvim +checkhealth

# Install missing dependencies
brew install ripgrep fd
```

**"Plugin errors after apply"**
```bash
# Sync plugins manually
nvim --headless "+Lazy! sync" +qa
```

---

## Performance Issues

### Slow Plan/Apply

**Cause:** Large configuration or many packages.

**Solutions:**
1. Use minimal target for testing:
   ```bash
   preflight apply --target minimal
   ```

2. Profile execution:
   ```bash
   time preflight plan
   ```

3. Reduce scope:
   ```yaml
   # Exclude heavy operations during testing
   packages:
     brew:
       # Comment out large lists temporarily
       # formulae: [...]
   ```

### High Memory Usage

**Cause:** Large layer files or many layers.

**Solutions:**
1. Split large layers:
   ```yaml
   # Instead of one huge layer, use:
   targets:
     default:
       - base
       - packages.core
       - packages.dev
       - editor.nvim
   ```

2. Use references instead of inline data:
   ```yaml
   files:
     links:
       - src: dotfiles/  # Directory reference
         dest: ~/.config/
   ```

---

## Environment-Specific Issues

### macOS

**"Operation not permitted" for system files**
```bash
# Check System Integrity Protection
csrutil status

# Some files cannot be modified
# Use alternatives in ~/Library/ instead
```

**Homebrew on Apple Silicon**
```bash
# Ensure correct PATH
echo $PATH | grep /opt/homebrew/bin

# Add to shell config
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
```

### Linux

**"apt requires sudo"**
```bash
# Preflight uses sudo automatically for apt
# Ensure user has sudo privileges
sudo -v
```

**"Package not found in apt"**
```bash
# Update package lists
sudo apt update

# Search for package
apt search <package>
```

---

## Getting More Help

### Gather Information

Before reporting an issue:

```bash
# Version info
preflight version

# System info
uname -a

# Config summary
cat preflight.yaml

# Full doctor report
preflight doctor --report markdown > doctor-report.md
```

### Report an Issue

1. Check [existing issues](https://github.com/felixgeelhaar/preflight/issues)
2. Create a new issue with:
   - Preflight version
   - Operating system
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant config snippets
   - Doctor report

### Community Support

- [GitHub Issues](https://github.com/felixgeelhaar/preflight/issues)
- [GitHub Discussions](https://github.com/felixgeelhaar/preflight/discussions)
