package advisor

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"strings"
)

// SecretPattern matches a path basename or substring that may carry a secret
// and therefore must not leave the machine. Intentionally conservative: a few
// false positives (e.g. a directory literally named "credentials") are far
// preferable to leaking a real key.
//
// Source guarantee: CLAUDE.md "Secrets never leave the machine."
var secretBasenamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^id_(rsa|ed25519|ecdsa|dsa)(\.pub)?$`),
	regexp.MustCompile(`(?i)^.*credentials.*$`),
	regexp.MustCompile(`(?i)^\.env(\..*)?$`),
	regexp.MustCompile(`(?i).*\.pem$`),
	regexp.MustCompile(`(?i).*\.key$`),
	regexp.MustCompile(`(?i).*token.*`),
	regexp.MustCompile(`(?i).*secret.*`),
	regexp.MustCompile(`(?i).*password.*`),
	regexp.MustCompile(`(?i)^aws_credentials$`),
	regexp.MustCompile(`(?i)^kubeconfig$`),
	regexp.MustCompile(`(?i)^netrc$|^\.netrc$`),
}

// RedactedPlaceholder is substituted in place of a sensitive path basename.
const RedactedPlaceholder = "[redacted]"

// RedactPath rewrites a filesystem path so that any sensitive basename is
// replaced with RedactedPlaceholder while preserving the directory shape so
// the AI can still reason about where the file lives.
//
// "/home/alice/.ssh/id_rsa"        → "/home/alice/.ssh/[redacted]"
// "/home/alice/.aws/credentials"   → "/home/alice/.aws/[redacted]"
// "/home/alice/.config/nvim/init"  → unchanged
func RedactPath(path string) string {
	if path == "" {
		return path
	}
	dir, base := filepath.Split(path)
	if isSensitiveBasename(base) {
		return dir + RedactedPlaceholder
	}
	return path
}

// IsSecretPath reports whether RedactPath would alter the input.
func IsSecretPath(path string) bool {
	if path == "" {
		return false
	}
	_, base := filepath.Split(path)
	return isSensitiveBasename(base)
}

func isSensitiveBasename(base string) bool {
	for _, p := range secretBasenamePatterns {
		if p.MatchString(base) {
			return true
		}
	}
	return false
}

// HashEmailDomain replaces the domain part of an email address with the first
// 8 hex characters of its SHA-256, retaining only the high-level shape ("work"
// vs "personal") without sending the literal domain to the AI provider.
func HashEmailDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(domain))
	return "domain-" + hex.EncodeToString(sum[:4])
}

// RedactCapturedItems returns a copy of items with sensitive paths redacted.
// Use this on every CapturedItem slice before it leaves the machine.
func RedactCapturedItems(items []CapturedItem) []CapturedItem {
	out := make([]CapturedItem, len(items))
	for i, item := range items {
		item.Path = RedactPath(item.Path)
		out[i] = item
	}
	return out
}

// RedactEmailDomains returns hashed forms of the supplied email-domain list.
func RedactEmailDomains(domains []string) []string {
	out := make([]string, len(domains))
	for i, d := range domains {
		out[i] = HashEmailDomain(d)
	}
	return out
}
