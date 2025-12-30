---
title: Fleet Management
description: Manage configurations across multiple machines with SSH-based fleet operations.
---

Fleet management allows you to apply preflight configurations across multiple machines simultaneously. This is useful for managing development teams, server clusters, or personal multi-device setups.

## Quick Start

```bash
# Create fleet inventory
preflight fleet init

# List all hosts
preflight fleet list

# Test connectivity
preflight fleet ping

# Apply configuration to all hosts
preflight fleet apply
```

## Fleet Inventory

The fleet inventory defines your managed hosts. Create `fleet.yaml` in your repository:

```yaml
version: 1

hosts:
  workstation-01:
    hostname: dev-ws-01.internal
    user: admin
    port: 22
    ssh_key: ~/.ssh/fleet_ed25519
    tags: [workstation, darwin, engineering]
    groups: [dev-team]

  workstation-02:
    hostname: dev-ws-02.internal
    user: admin
    tags: [workstation, darwin, engineering]
    groups: [dev-team]

  server-prod-01:
    hostname: prod-01.example.com
    user: deploy
    tags: [server, linux, production]
    groups: [production]

groups:
  production:
    hosts: [server-prod-*]
    policies: [require-approval, maintenance-window]

  dev-team:
    hosts: [workstation-*]
    policies: []

defaults:
  ssh_timeout: 30s
  max_parallel: 10
```

## Targeting Hosts

Fleet commands support flexible targeting:

```bash
# By group
preflight fleet apply --target @production

# By tag
preflight fleet apply --target tag:darwin

# By glob pattern
preflight fleet apply --target "server-*"

# All hosts except a tag
preflight fleet apply --target @all --exclude tag:prod

# Multiple targets
preflight fleet apply --target @dev-team --target tag:linux
```

## Commands Reference

### fleet list

List hosts matching the target filter:

```bash
preflight fleet list                    # All hosts
preflight fleet list --target @production
preflight fleet list --tags             # Show tags for each host
preflight fleet list --groups           # Show group membership
preflight fleet list --json             # JSON output for scripting
```

### fleet ping

Test SSH connectivity to hosts:

```bash
preflight fleet ping                    # All hosts
preflight fleet ping --target @production
preflight fleet ping --timeout 30s      # Custom timeout
```

### fleet plan

Generate execution plan for all hosts:

```bash
preflight fleet plan                    # Show what would be done
preflight fleet plan --target @dev-team
preflight fleet plan --output json      # JSON for CI/CD
```

### fleet apply

Apply configuration to hosts:

```bash
preflight fleet apply                   # All hosts
preflight fleet apply --target @production
preflight fleet apply --strategy rolling   # One host at a time
preflight fleet apply --strategy parallel  # All at once
preflight fleet apply --strategy canary    # Canary first, then rolling
```

### fleet status

Show current state across the fleet:

```bash
preflight fleet status                  # All hosts
preflight fleet status --target @production
preflight fleet status --json
```

### fleet diff

Compare local configuration with fleet state:

```bash
preflight fleet diff                    # Show differences
preflight fleet diff --target tag:darwin
```

## Execution Strategies

### Parallel

Execute on all hosts simultaneously (up to `max_parallel`):

```yaml
defaults:
  max_parallel: 10  # Run on 10 hosts at once
```

```bash
preflight fleet apply --strategy parallel
```

### Rolling

Execute one batch at a time, waiting for completion:

```bash
preflight fleet apply --strategy rolling
```

### Canary

Execute on a canary host first, then rolling deployment:

```yaml
hosts:
  canary-host:
    hostname: canary.example.com
    tags: [canary]  # Mark as canary
```

```bash
preflight fleet apply --strategy canary
```

## Fleet Policies

Define policies for host groups:

```yaml
groups:
  production:
    hosts: [server-prod-*]
    policies:
      - require-approval     # Require explicit approval
      - maintenance-window   # Only apply during maintenance windows
```

### Maintenance Windows

Define when changes can be applied:

```yaml
policies:
  maintenance-window:
    windows:
      - day: saturday
        start: "02:00"
        end: "06:00"
        timezone: America/Los_Angeles
```

## SSH Configuration

Fleet uses SSH for remote execution. Configure authentication:

### SSH Key

```yaml
hosts:
  workstation-01:
    ssh_key: ~/.ssh/fleet_ed25519
```

### SSH Agent

If no `ssh_key` is specified, fleet uses ssh-agent:

```bash
# Add your key to the agent
ssh-add ~/.ssh/fleet_ed25519

# Verify it's loaded
ssh-add -l
```

### SSH Config

Fleet respects your `~/.ssh/config`:

```
Host dev-ws-*
    User admin
    IdentityFile ~/.ssh/fleet_ed25519
    StrictHostKeyChecking accept-new

Host prod-*
    User deploy
    IdentityFile ~/.ssh/production_key
    ProxyJump bastion.example.com
```

## Git Sync Integration

Combine fleet with sync for distributed configuration:

```bash
# Push local config to remote
preflight sync --push

# Apply to fleet (each host pulls from remote)
preflight fleet apply

# Or sync directly from fleet master
preflight fleet sync --push
```

## Troubleshooting

### Connection Issues

```bash
# Test SSH connectivity
preflight fleet ping --target problematic-host

# Verbose output
preflight fleet ping --target host-01 -v

# Check SSH config
ssh -v user@hostname
```

### Partial Failures

When some hosts fail:

```bash
# Retry failed hosts only
preflight fleet apply --retry-failed

# Skip failed hosts
preflight fleet apply --continue-on-error
```

### Audit Trail

All fleet operations are logged:

```bash
# View fleet events
preflight audit show --type fleet

# Filter by host
preflight audit show --filter "host=workstation-01"
```

## Best Practices

1. **Use SSH keys**, not passwords
2. **Test with `--dry-run`** before applying
3. **Start with canary** deployments for production
4. **Set up maintenance windows** for critical systems
5. **Review the plan** before applying to multiple hosts
6. **Use tags** for logical grouping beyond hostnames

## Example Workflows

### Development Team Setup

```bash
# Set up new developer machine
preflight fleet apply --target new-hire-laptop

# Update all developer workstations
preflight fleet apply --target @dev-team
```

### Production Deployment

```bash
# Plan the deployment
preflight fleet plan --target @production

# Canary first
preflight fleet apply --target tag:canary

# Rolling deployment
preflight fleet apply --target @production --strategy rolling
```

### Multi-Environment Sync

```bash
# Sync to staging
preflight fleet apply --target @staging

# Verify staging
preflight fleet status --target @staging

# Promote to production
preflight fleet apply --target @production
```
