# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

1. **Do not** open a public issue
2. Email security concerns to the maintainers
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- Acknowledgment within 48 hours
- Status update within 7 days
- We'll work with you to understand and resolve the issue

### Security Principles

Preflight follows these security principles:

1. **Secrets never leave the machine** - Configuration uses references, not values
2. **No network calls without consent** - User must explicitly enable network features
3. **Minimal permissions** - Only request permissions that are necessary
4. **Audit trail** - All actions are logged and explainable

### Scope

Security issues in scope:

- Code execution vulnerabilities
- Path traversal attacks
- Privilege escalation
- Information disclosure
- Denial of service

Out of scope:

- Issues in dependencies (report to the upstream project)
- Social engineering attacks
- Physical access attacks

## Security Best Practices for Users

1. **Review plans before applying** - Always run `preflight plan` first
2. **Use version control** - Keep your configuration in git
3. **Protect your configuration** - Don't commit secrets
4. **Keep updated** - Use the latest version

Thank you for helping keep Preflight secure!
