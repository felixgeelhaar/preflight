package advisor

import (
	"path/filepath"
	"regexp"
	"strings"
)

// secretBasenamePatterns matches a path basename or substring that may carry
// a secret and therefore must not leave the machine. Intentionally
// conservative: a few false positives (e.g. a directory literally named
// "credentials") are far preferable to leaking a real key.
//
// Source guarantee: CLAUDE.md "Secrets never leave the machine."
var secretBasenamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^id_(rsa|ed25519|ecdsa|dsa|xmss)(\.pub)?$`),
	regexp.MustCompile(`(?i)^.*credentials.*$`),
	regexp.MustCompile(`(?i)^\.env(\..*)?$`),
	regexp.MustCompile(`(?i).*\.pem$`),
	regexp.MustCompile(`(?i).*\.key$`),
	regexp.MustCompile(`(?i).*token.*`),
	regexp.MustCompile(`(?i).*secret.*`),
	regexp.MustCompile(`(?i).*password.*`),
	regexp.MustCompile(`(?i)^aws_credentials$`),
	regexp.MustCompile(`(?i)^kubeconfig$|.*\.kubeconfig$`),
	regexp.MustCompile(`(?i)^netrc$|^\.netrc$`),
	// Extended after security audit (2026-05).
	regexp.MustCompile(`(?i).*\.gpg$|.*\.asc$|.*\.pgp$`),                                   // OpenPGP files
	regexp.MustCompile(`(?i)^secring\.|^pubring\.|^trustdb\.gpg$`),                         // GPG key rings
	regexp.MustCompile(`(?i).*\.kdbx$|.*\.kdb$`),                                           // KeePass DBs
	regexp.MustCompile(`(?i).*\.p12$|.*\.pfx$`),                                            // PKCS12 bundles
	regexp.MustCompile(`(?i).*\.ppk$`),                                                     // PuTTY keys
	regexp.MustCompile(`(?i).*\.keychain$|.*\.keychain-db$`),                               // macOS keychains
	regexp.MustCompile(`(?i)^cookies$|^cookies\.sqlite$|^login data$`),                     // Browser cookies + Chrome login DB
	regexp.MustCompile(`(?i)^auth\.json$|^\.dockercfg$`),                                   // Docker registry creds
	regexp.MustCompile(`(?i)^\.npmrc$|^\.pypirc$|^application_default_credentials\.json$`), // pkg publish + GCP ADC
}

// secretParentDirs identifies directories whose contents are presumed sensitive
// regardless of basename. Walking detects them anywhere in the path.
var secretParentDirs = map[string]struct{}{
	".ssh":   {},
	".gnupg": {},
	".aws":   {},
}

// RedactedPlaceholder is substituted in place of a sensitive path basename.
const RedactedPlaceholder = "[redacted]"

// RedactPath rewrites a filesystem path so that any sensitive basename, or any
// path that lives under a sensitive parent directory (.ssh, .gnupg, .aws), is
// replaced with RedactedPlaceholder. The non-sensitive directory prefix is
// preserved so the AI can still reason about where the file lives.
//
// "/home/alice/.ssh/id_rsa"             → "/home/alice/.ssh/[redacted]"
// "/home/alice/.ssh/work_deploy"        → "/home/alice/.ssh/[redacted]"  (parent dir)
// "/home/alice/.gnupg/secring.gpg"      → "/home/alice/.gnupg/[redacted]" (basename)
// "/home/alice/.aws/credentials"        → "/home/alice/.aws/[redacted]"
// "/home/alice/.config/nvim/init"       → unchanged
func RedactPath(path string) string {
	if path == "" {
		return path
	}
	dir, base := filepath.Split(path)
	if isSensitiveBasename(base) || hasSensitiveParent(dir) {
		return dir + RedactedPlaceholder
	}
	return path
}

// IsSecretPath reports whether RedactPath would alter the input.
func IsSecretPath(path string) bool {
	if path == "" {
		return false
	}
	dir, base := filepath.Split(path)
	return isSensitiveBasename(base) || hasSensitiveParent(dir)
}

// hasSensitiveParent walks the directory components of dir and returns true if
// any component matches a sensitive-parent name (e.g. .ssh, .gnupg, .aws).
// The check is component-exact, so a regular `sshconfig` directory does not
// trigger redaction.
func hasSensitiveParent(dir string) bool {
	if dir == "" {
		return false
	}
	cleaned := filepath.Clean(dir)
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		if _, ok := secretParentDirs[strings.ToLower(part)]; ok {
			return true
		}
	}
	return false
}

func isSensitiveBasename(base string) bool {
	for _, p := range secretBasenamePatterns {
		if p.MatchString(base) {
			return true
		}
	}
	return false
}

// commonPersonalDomains is the closed set of well-known consumer mail
// providers. Any other input is reported as "work" (i.e. presumably an
// employer or self-hosted domain). We deliberately do NOT echo the domain in
// any form: a 4-byte SHA prefix is reversible against a small enumeration and
// gave a false sense of anonymization.
var commonPersonalDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
	"icloud.com":     {},
	"me.com":         {},
	"mac.com":        {},
	"yahoo.com":      {},
	"outlook.com":    {},
	"hotmail.com":    {},
	"live.com":       {},
	"msn.com":        {},
	"protonmail.com": {},
	"proton.me":      {},
	"fastmail.com":   {},
	"tutanota.com":   {},
	"zoho.com":       {},
	"aol.com":        {},
}

// HashEmailDomain returns a coarse shape tag instead of the literal domain.
// The previous implementation returned a 4-byte SHA-256 prefix which is
// trivially reversible by enumerating the top ~10k email domains; that
// defeated the stated privacy guarantee. We now return only "personal" or
// "work" so AI prompts cannot be used to de-anonymize the user's employer.
//
// The function name and signature are preserved for callers; the contract has
// strengthened, not weakened.
func HashEmailDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return ""
	}
	if _, ok := commonPersonalDomains[domain]; ok {
		return "personal"
	}
	return "work"
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
