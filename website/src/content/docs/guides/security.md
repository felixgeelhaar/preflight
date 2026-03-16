---
title: Plugin Security
description: Defense-in-depth security model for Preflight plugins.
---

Preflight uses a **defense-in-depth security model** for plugins. Each layer provides independent protection, so compromising one layer doesn't compromise the system.

## Security Stack

```
┌─────────────────────────────────────┐
│  Layer 4: Sandbox (WASM isolation)  │  ← Can't escape even if malicious
├─────────────────────────────────────┤
│  Layer 3: Capabilities (permissions)│  ← Can only do what's declared
├─────────────────────────────────────┤
│  Layer 2: Signatures (identity)     │  ← Know who published it
├─────────────────────────────────────┤
│  Layer 1: Integrity (hashes)        │  ← Know it wasn't tampered with
└─────────────────────────────────────┘
```

Each layer catches different attack vectors:

| Layer | Protects Against |
|-------|------------------|
| Integrity | Tampering, MITM, corrupted downloads |
| Signatures | Unknown publishers, impersonation |
| Capabilities | Privilege escalation, unauthorized access |
| Sandbox | Code execution, system compromise |

---

## Layer 1: Integrity Verification

All catalogs and plugins include SHA256 checksums for every file.

### Catalog Manifest

```yaml
# catalog-manifest.yaml
version: "1.0"
name: company-devtools
files:
  catalog.yaml: "sha256:abc123..."
  presets/dev.yaml: "sha256:def456..."
  presets/prod.yaml: "sha256:789ghi..."
```

### Verification Process

1. Download catalog from URL
2. Compute SHA256 of each file
3. Compare against manifest hashes
4. **Fail immediately** if mismatch

```bash
# Verify catalog integrity
preflight catalog verify company-devtools
```

---

## Layer 2: Signature Verification

Publishers sign catalogs with cryptographic keys. Preflight verifies signatures before loading.

### Supported Key Types

| Type | Format | Use Case |
|------|--------|----------|
| SSH | ED25519 | Developer signing |
| GPG | RSA/ED25519 | Traditional PKI |
| Sigstore | Keyless | CI/CD pipelines |

### Trust Management

```bash
# List trusted keys
preflight trust list

# Add a trusted key
preflight trust add ~/.ssh/id_ed25519.pub

# Remove a key
preflight trust remove key-fingerprint

# Show key details
preflight trust show key-fingerprint
```

### Trust Levels

| Level | Description | Verification |
|-------|-------------|--------------|
| `builtin` | Embedded in Preflight binary | Automatic |
| `verified` | Signed by trusted key | Signature check |
| `community` | Hash verified only | User approval |
| `untrusted` | No verification | Explicit flag |

### Signed Catalog Example

```yaml
# catalog-manifest.yaml
version: "1.0"
name: company-devtools
publisher:
  name: "DevOps Team"
  email: "devops@company.com"
signature:
  type: ssh
  key_id: "SHA256:abc123..."
  data: "base64-encoded-signature"
```

---

## Layer 3: Capability-Based Permissions

Plugins must declare what they need. Preflight enforces these declarations.

### Capability Categories

| Category | Actions | Example |
|----------|---------|---------|
| `files` | read, write | `files:read` |
| `packages` | brew, apt | `packages:brew` |
| `shell` | execute | `shell:execute` |
| `network` | fetch | `network:fetch` |
| `secrets` | read, write | `secrets:read` |
| `system` | modify | `system:modify` |

### Declaring Capabilities

```yaml
# plugin.yaml
capabilities:
  - name: files:read
    justification: Read configuration files

  - name: shell:execute
    justification: Run validation commands
    optional: true

  - name: network:fetch
    justification: Download tool releases
```

### Dangerous Capabilities

Some capabilities require explicit approval:

| Capability | Why Dangerous |
|------------|---------------|
| `shell:execute` | Can run arbitrary commands |
| `files:write` | Can modify system files |
| `secrets:write` | Can leak credentials |
| `system:modify` | Can change system settings |

### Policy Enforcement

```yaml
# preflight.yaml
security:
  blocked_capabilities:
    - secrets:write
    - system:modify

  require_approval: true
```

### Content Security Policy (CSP)

Block dangerous patterns in shell commands:

```yaml
security:
  csp_deny:
    - pattern: "curl.*|.*sh"
      reason: "Piped curl to shell is dangerous"
    - pattern: "chmod.*777"
      reason: "World-writable permissions"
    - pattern: "sudo.*"
      reason: "Sudo not allowed"
    - pattern: "rm.*-rf.*/"
      reason: "Recursive delete of root paths"

  csp_warn:
    - pattern: ".*eval.*"
      reason: "Eval can execute arbitrary code"
```

---

## Layer 4: WASM Sandbox

Plugins can compile to WebAssembly for complete isolation. The sandbox:

- Runs in isolated VM with no direct system access
- Cannot access files, network, or shell without explicit bindings
- Has resource limits to prevent abuse
- Uses deterministic execution

### Sandbox Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `full` | Complete isolation, no side effects | Audit unknown plugins |
| `restricted` | Limited to declared capabilities | Normal operation |
| `trusted` | Full access like builtin | Verified publishers |

### Resource Limits

| Resource | Default | Purpose |
|----------|---------|---------|
| Memory | 64 MB | Prevent memory exhaustion |
| CPU time | 30 seconds | Prevent infinite loops |
| File descriptors | 32 | Limit open files |
| Output | 1 MB | Limit stdout/stderr |

### Host Function Bindings

Plugins access the host through controlled bindings:

```go
// Available host functions
preflight.log_info(message)    // Always available
preflight.log_warn(message)    // Always available
preflight.log_error(message)   // Always available
preflight.read_file(path)      // Requires files:read
preflight.write_file(path, data) // Requires files:write
preflight.exec(cmd, args)      // Requires shell:execute
preflight.fetch(url)           // Requires network:fetch
```

### WASM Plugin Manifest

```yaml
# plugin.yaml
id: my-secure-plugin
name: My Secure Plugin
version: 1.0.0
module: plugin.wasm
checksum: sha256:abc123def456...

capabilities:
  - name: files:read
    justification: Read user config files
  - name: network:fetch
    justification: Check for updates
    optional: true
```

---

## SLSA Attestation Verification

Preflight can verify SLSA provenance for locked packages, ensuring that packages were built by trusted builders using reproducible processes.

### Verification Process

```bash
# Verify all locked packages have valid attestations
preflight lock verify-attestations
```

This checks:

1. Each `PackageLock` entry has an `AttestationRef`
2. The in-toto statement is valid and well-formed
3. The SLSA provenance predicate meets the configured policy level (L0-L4)
4. Sigstore bundle signatures are verified against the transparency log
5. Builder identity matches the list of trusted builders

### Attestation Policy

Configure attestation requirements in `preflight.yaml`:

```yaml
security:
  attestation:
    min_slsa_level: 2          # Require at least SLSA L2
    trusted_builders:
      - https://github.com/actions/runner
      - https://github.com/slsa-framework/slsa-github-generator
    max_age: 720h              # Reject attestations older than 30 days
    require_sigstore: true     # Require Sigstore keyless signatures
```

### SLSA Levels

| Level | Requirements |
|-------|-------------|
| L0 | No guarantees |
| L1 | Build process documented |
| L2 | Signed provenance, hosted build |
| L3 | Hardened build platform |
| L4 | Two-party review, hermetic builds |

---

## Identity-Based Trust Elevation

Enterprise OIDC identity extends the existing Sigstore trust model. When a user authenticates via `preflight identity login`, their identity claims can be used to:

- **Elevate plugin trust levels** — Plugins signed by a verified corporate identity are automatically trusted
- **Gate fleet operations** — Require authenticated identity before fleet apply
- **Audit attribution** — All operations are attributed to the authenticated identity

### Configuration

```yaml
identity:
  providers:
    corporate:
      issuer: https://login.company.com
      client_id: preflight-cli
      scopes: [openid, profile, email, groups]
```

### Trust Chain

```
OIDC Provider → Device Auth Flow → ID Token → Sigstore Keyless → Attestation
```

When Sigstore keyless signing is used, the OIDC identity token becomes the basis for signing. This creates a verifiable chain from enterprise identity to package attestation.

---

## Marketplace Security Scanning

Preflight can automatically scan marketplace packages for vulnerabilities during installation using nox (with Grype or Trivy as fallbacks).

### Automatic Scanning

```bash
# Install with automatic security scan
preflight marketplace install nvim-pro

# Skip scanning (not recommended)
preflight marketplace install nvim-pro --skip-scan
```

### Manual Scanning

```bash
# Scan an installed package
preflight marketplace scan nvim-pro

# Only report high and critical vulnerabilities
preflight marketplace scan nvim-pro --min-severity high
```

### Scan Policy

Configure scan behavior in `preflight.yaml`:

```yaml
security:
  marketplace_scan:
    enabled: true
    scanner: nox                # nox (primary), grype, or trivy
    block_severity: critical    # Block install if severity >= threshold
    skip_patterns:
      - "test-*"               # Skip scanning test packages
```

If no scanner binary is available on the system, Preflight logs a warning and continues without scanning (graceful fallback).

---

## Security Auditing

### Audit External Catalogs

```bash
# Run security audit
preflight catalog audit company-devtools
```

Audit checks for:

| Pattern | Severity | Description |
|---------|----------|-------------|
| `curl.*\|.*sh` | Critical | Remote code execution |
| `sudo`, `doas` | High | Privilege escalation |
| `rm -rf /` | High | Destructive operation |
| `chmod 777` | Medium | Insecure permissions |
| `API_KEY=`, `TOKEN=` | Medium | Hardcoded secrets |
| `eval` | Low | Dynamic code execution |

### Audit Output

```
Audit Results for company-devtools
══════════════════════════════════

Critical Issues: 0
High Issues: 1
Medium Issues: 2
Low Issues: 3

HIGH: presets/deploy.yaml:15
  Pattern: sudo docker build
  Reason: Privilege escalation via sudo

MEDIUM: presets/dev.yaml:23
  Pattern: chmod 755 /usr/local/bin
  Reason: System path modification

Recommendation: Review flagged patterns before installing
```

---

## Security Best Practices

### For Users

1. **Only install from trusted sources**
   ```bash
   # Good: Verified publisher
   preflight catalog add https://company.com/catalog.yaml

   # Risky: Unknown source
   preflight catalog add https://random-site.com/catalog.yaml --untrusted
   ```

2. **Review capabilities before approval**
   ```bash
   # See what a plugin requires
   preflight plugin info suspicious-plugin
   ```

3. **Run audits before installing**
   ```bash
   preflight catalog audit new-catalog
   ```

4. **Use restrictive policies**
   ```yaml
   security:
     blocked_capabilities:
       - secrets:write
       - system:modify
     require_approval: true
   ```

### For Plugin Authors

1. **Request minimal capabilities**
   - Only declare what you need
   - Use `optional: true` for non-essential features

2. **Provide justifications**
   ```yaml
   capabilities:
     - name: files:read
       justification: Read ~/.config/myapp/settings.yaml
   ```

3. **Sign your releases**
   ```bash
   # Sign catalog with SSH key
   ssh-keygen -Y sign -f ~/.ssh/id_ed25519 -n preflight catalog.yaml
   ```

4. **Use WASM for sensitive operations**
   - Compile to WebAssembly for maximum isolation
   - Let users run in `full` sandbox mode for auditing

---

## Threat Model

### What We Protect Against

| Threat | Mitigation |
|--------|------------|
| Malicious catalog | Signature verification + audit |
| Tampered download | SHA256 integrity checks |
| Privilege escalation | Capability-based permissions |
| Code execution | WASM sandbox isolation |
| Data exfiltration | Network capability control |
| Resource exhaustion | Sandbox resource limits |

### What We Don't Protect Against

| Threat | Reason |
|--------|--------|
| Compromised trusted key | User must manage key security |
| Social engineering | User must verify trust decisions |
| Kernel exploits | Outside Preflight's scope |
| Hardware attacks | Outside Preflight's scope |

---

## What's Next?

- [Plugins](/preflight/guides/plugins/) — Plugin development guide
- [Providers](/preflight/guides/providers/) — Built-in providers
- [CLI Commands](/preflight/cli/commands/) — Full CLI reference
