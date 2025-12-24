# Plugin Security Design

## Overview

This document outlines the security model for Preflight plugins/external catalogs.

## Threat Model

### Risks
1. **Malicious presets** - Presets that install malware or backdoors
2. **Supply chain attacks** - Compromised upstream catalog sources
3. **Privilege escalation** - Plugins requesting excessive permissions
4. **Data exfiltration** - Plugins capturing secrets or system info
5. **Typosquatting** - Fake presets with similar names to popular ones

### Trust Levels

| Level | Description | Verification |
|-------|-------------|--------------|
| `builtin` | Embedded in binary | Compiled-in |
| `verified` | Signed by known publisher | GPG/Sigstore signature |
| `community` | Hash-verified, user-reviewed | SHA256 + user approval |
| `untrusted` | No verification | Explicit `--allow-untrusted` |

## Security Mechanisms

### 1. Integrity Verification

Every external catalog must include integrity hashes:

```yaml
# catalog-manifest.yaml
version: "1.0"
integrity:
  algorithm: sha256
  files:
    catalog.yaml: "sha256:abc123..."
    presets/base.yaml: "sha256:def456..."
```

**Verification flow:**
1. Download catalog manifest
2. Verify manifest signature (if signed)
3. Download individual files
4. Verify each file against manifest hash
5. Reject if any hash mismatch

### 2. Signature Verification

Support for cryptographic signatures:

```yaml
signature:
  type: gpg  # or: sigstore, ssh
  keyid: "ABCD1234EFGH5678"
  fingerprint: "1234 5678 ABCD EFGH..."
  sig: |
    -----BEGIN PGP SIGNATURE-----
    ...
    -----END PGP SIGNATURE-----
```

**Trusted keys registry:**
- `~/.preflight/trusted-keys/` - User's trusted GPG keys
- `preflight trust add <keyid>` - Add key to trusted set
- `preflight trust list` - List trusted keys
- `preflight trust remove <keyid>` - Remove trusted key

### 3. Capability-Based Permissions

Plugins declare required capabilities:

```yaml
capabilities:
  - files:read     # Read dotfiles
  - files:write    # Write dotfiles
  - packages:brew  # Install Homebrew packages
  - packages:apt   # Install APT packages
  - shell:execute  # Run shell commands
  - network:fetch  # Fetch from network
  - secrets:read   # Access secrets (SSH keys, etc.)
```

**Enforcement:**
- Plugins only get capabilities they declare
- User must approve capabilities on first install
- Dangerous capabilities require explicit confirmation

### 4. Sandbox Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `full` | Complete isolation, no side effects | Preview/audit |
| `restricted` | Limited to declared capabilities | Normal operation |
| `trusted` | Full access (like builtin) | Verified publishers |

### 5. Content Security Policy

Presets are validated against security rules:

```yaml
# security-policy.yaml
deny:
  # Dangerous patterns
  - pattern: "curl.*|.*sh"
    reason: "Piped curl to shell is dangerous"

  - pattern: "chmod.*777"
    reason: "World-writable permissions"

  - pattern: "sudo.*"
    reason: "Sudo commands not allowed in presets"

  - pattern: "rm.*-rf.*/"
    reason: "Recursive delete of root paths"

warn:
  - pattern: ".*eval.*"
    reason: "Eval can execute arbitrary code"
```

## Implementation Phases

> **Note:** These phases align with v3.x in the PRD (see section 15).

### Phase 1: Foundation (v3.0)
- External catalog YAML loading
- SHA256 integrity verification
- User approval for new catalogs
- `preflight catalog add/remove/list`

### Phase 2: Signatures (v3.1)
- GPG signature verification
- Trusted keys management
- Publisher verification
- Sigstore integration (keyless signing)

### Phase 3: Capabilities (v3.2)
- Capability declaration format
- Permission enforcement
- Interactive approval flow
- Audit logging

### Phase 4: Sandbox (v3.3)
- WASM plugin runtime
- Process isolation
- Resource limits
- Network policy enforcement

## CLI Commands

```bash
# Catalog management
preflight catalog list                    # List installed catalogs
preflight catalog add <url>               # Add external catalog
preflight catalog remove <name>           # Remove catalog
preflight catalog verify                  # Verify all catalog integrity
preflight catalog audit <name>            # Security audit of catalog

# Trust management
preflight trust list                      # List trusted publishers
preflight trust add <keyid>               # Trust a GPG key
preflight trust remove <keyid>            # Untrust a key
preflight trust show <keyid>              # Show key details

# Security
preflight security scan                   # Scan config for issues
preflight security policy                 # Show security policy
preflight security audit                  # Full security audit
```

## Configuration

```yaml
# preflight.yaml
security:
  # Minimum trust level for catalogs
  min_trust_level: community  # builtin, verified, community, untrusted

  # Require signature verification
  require_signatures: false

  # Auto-approve known publishers
  auto_approve_publishers:
    - "Anthropic <security@anthropic.com>"
    - "Preflight Official <team@preflight.dev>"

  # Blocked capabilities
  blocked_capabilities:
    - secrets:read
    - shell:execute

  # Custom security policy
  policy: ~/.preflight/security-policy.yaml
```

## Audit Logging

All plugin operations are logged:

```json
{
  "timestamp": "2024-12-24T12:00:00Z",
  "event": "catalog_installed",
  "catalog": "company-devtools",
  "source": "https://company.com/catalog.yaml",
  "integrity": "sha256:abc123...",
  "signature_verified": true,
  "signer": "devops@company.com",
  "capabilities_granted": ["files:write", "packages:brew"],
  "user": "jane"
}
```

## Security Checklist for Plugin Authors

- [ ] Sign your catalog with GPG or Sigstore
- [ ] Include integrity hashes for all files
- [ ] Declare minimum required capabilities
- [ ] Document what each preset does
- [ ] Avoid shell command execution where possible
- [ ] Never hardcode secrets or credentials
- [ ] Use version pinning for dependencies
- [ ] Provide a security contact

## Future Considerations

1. **Plugin Marketplace** - Curated registry with security reviews
2. **Automated Scanning** - CI/CD integration for preset scanning
3. **Reproducible Builds** - Verify catalog source matches published
4. **Dependency Scanning** - Check for vulnerable package versions
5. **Runtime Monitoring** - Detect anomalous plugin behavior
