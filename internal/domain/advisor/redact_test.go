package advisor

import (
	"strings"
	"testing"
)

func TestRedactPath_SensitiveBasenames(t *testing.T) {
	t.Parallel()
	cases := []string{
		"/home/alice/.ssh/id_rsa",
		"/home/alice/.ssh/id_ed25519.pub",
		"/home/alice/.ssh/id_xmss",
		"/home/alice/.aws/credentials",
		"/Users/bob/.env",
		"/Users/bob/.env.local",
		"/etc/ssl/server.pem",
		"/var/keys/private.key",
		"/home/work/api_token",
		"/home/work/MY_SECRET_KEY",
		"/home/work/.netrc",
		// Extended coverage added 2026-05.
		"/home/alice/.gnupg/secring.gpg",
		"/home/alice/keys/passwords.kdbx",
		"/home/alice/certs/identity.p12",
		"/home/alice/certs/server.pfx",
		"/home/alice/.putty/work.ppk",
		"/Users/alice/Library/Keychains/login.keychain-db",
		"/Users/alice/Library/Application Support/Google/Chrome/Default/Cookies",
		"/Users/alice/Library/Application Support/Google/Chrome/Default/Login Data",
		"/home/alice/.docker/auth.json",
		"/home/alice/.gcloud/application_default_credentials.json",
		"/home/alice/dev/.npmrc",
		"/home/alice/dev/.pypirc",
		// Sensitive parent dir, non-matching basename.
		"/home/alice/.ssh/work_deploy",
		"/home/alice/.ssh/github_deploy",
		"/home/alice/.gnupg/openpgp-revocs.d/A1B2C3D4.rev",
		"/home/alice/.aws/cli/cache/some_cache",
		// Extended patterns (2026-05 second pass).
		"/home/alice/certs/server.crt",
		"/home/alice/certs/server.csr",
		"/home/alice/certs/server.cert",
		"/home/alice/keystores/jvm.jks",
		"/home/alice/keystores/jvm.bks",
		"/home/alice/secrets/group_vars.vault",
		"/home/alice/Library/MobileDevice/Provisioning Profiles/abcd.mobileprovision",
		"/etc/wpa_supplicant.conf",
		"/etc/hostapd.conf",
		"/Users/alice/Library/Application Support/Firefox/Profiles/x/signons.sqlite",
		"/Users/alice/Library/Application Support/Firefox/Profiles/x/logins.json",
		"/Users/alice/Library/Application Support/Google/Chrome/Default/Web Data",
		"/Users/alice/Library/Application Support/Google/Chrome/Default/Web Data-journal",
		"/Users/alice/Library/Application Support/Firefox/Profiles/x/cookies.sqlite-wal",
		"/home/alice/.git-credentials",
		// Non-id_-prefixed SSH keys outside ~/.ssh.
		"/home/alice/keys/work_rsa",
		"/home/alice/keys/github_ed25519.pub",
		"/home/alice/keys/deploy_ecdsa",
	}
	for _, in := range cases {
		out := RedactPath(in)
		if out == in {
			t.Errorf("RedactPath(%q) did not redact", in)
		}
		if !strings.HasSuffix(out, RedactedPlaceholder) {
			t.Errorf("RedactPath(%q) = %q, want suffix %q", in, out, RedactedPlaceholder)
		}
	}
}

func TestRedactPath_BenignPaths(t *testing.T) {
	t.Parallel()
	cases := []string{
		"/home/alice/.config/nvim/init.lua",
		"/Users/bob/.zshrc",
		"/etc/hosts",
		"",
	}
	for _, in := range cases {
		if got := RedactPath(in); got != in {
			t.Errorf("RedactPath(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestIsSecretPath(t *testing.T) {
	t.Parallel()
	if !IsSecretPath("/home/x/.ssh/id_rsa") {
		t.Error("expected id_rsa to be recognised as secret")
	}
	if IsSecretPath("/home/x/.zshrc") {
		t.Error("expected .zshrc to be considered safe")
	}
}

func TestHashEmailDomain_ShapeOnly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"gmail.com", "personal"},
		{"GMAIL.com", "personal"},
		{"icloud.com", "personal"},
		{"proton.me", "personal"},
		{"anthropic.com", "work"},
		{"acme-corp.io", "work"},
		{"my-self-hosted.example", "work"},
		{"", ""},
	}
	for _, c := range cases {
		if got := HashEmailDomain(c.in); got != c.want {
			t.Errorf("HashEmailDomain(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestHashEmailDomain_DoesNotEchoDomain is the security-critical guarantee:
// no part of the literal domain — even hashed/truncated — appears in the
// output. The previous SHA-prefix implementation was reversible against an
// enumeration of common domains; this test fails fast if a future refactor
// reintroduces it.
func TestHashEmailDomain_DoesNotEchoDomain(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"anthropic.com", "very-secret-internal.example"} {
		got := HashEmailDomain(in)
		if strings.Contains(got, in) || strings.Contains(got, in[:5]) {
			t.Errorf("HashEmailDomain(%q) = %q must not echo any prefix of input", in, got)
		}
		if len(got) > 16 {
			t.Errorf("HashEmailDomain(%q) = %q is unexpectedly long; suggests hash leakage", in, got)
		}
	}
}

func TestRedactCapturedItems(t *testing.T) {
	t.Parallel()
	in := []CapturedItem{
		{Path: "/home/x/.ssh/id_rsa", Type: "file"},
		{Path: "/home/x/.zshrc", Type: "file"},
	}
	out := RedactCapturedItems(in)
	if out[0].Path == in[0].Path {
		t.Errorf("expected first item to be redacted, got %q", out[0].Path)
	}
	if out[1].Path != in[1].Path {
		t.Errorf("expected second item unchanged, got %q", out[1].Path)
	}
	if in[0].Path != "/home/x/.ssh/id_rsa" {
		t.Error("RedactCapturedItems must not mutate the input slice")
	}
}

// TestBuildCaptureAnalysisPrompt_RedactsSecrets is the integration guarantee:
// secret-bearing path basenames must NOT appear in the prompt sent to the AI.
func TestBuildCaptureAnalysisPrompt_RedactsSecrets(t *testing.T) {
	t.Parallel()
	req := CaptureAnalysisRequest{
		Items: []CapturedItem{
			{Path: "/home/x/.ssh/id_rsa", Type: "file", Description: "ssh key"},
			{Path: "/home/x/.aws/credentials", Type: "file"},
			{Path: "/home/x/.zshrc", Type: "file"},
			{Path: "/home/x/.env.production", Type: "file"},
		},
	}
	prompt := BuildCaptureAnalysisPrompt(req)
	body := prompt.UserPrompt()

	for _, leak := range []string{"id_rsa", "credentials", ".env.production"} {
		if strings.Contains(body, leak) {
			t.Errorf("prompt body must not contain %q (secret leak)\nbody:\n%s", leak, body)
		}
	}
	if !strings.Contains(body, ".zshrc") {
		t.Errorf("benign path .zshrc should still appear in prompt body")
	}
	if !strings.Contains(body, RedactedPlaceholder) {
		t.Errorf("expected redaction placeholder %q in prompt body", RedactedPlaceholder)
	}
}

// TestNoRawSecretLeak ensures that no known sensitive path basename appears
// in the output of RedactCapturedItems serialised back as strings.
func TestNoRawSecretLeak(t *testing.T) {
	t.Parallel()
	in := []CapturedItem{
		{Path: "/home/x/.ssh/id_rsa"},
		{Path: "/home/x/.aws/credentials"},
		{Path: "/home/x/.env.production"},
	}
	out := RedactCapturedItems(in)
	for _, item := range out {
		if strings.Contains(item.Path, "id_rsa") ||
			strings.Contains(item.Path, "credentials") ||
			strings.Contains(item.Path, ".env.") {
			t.Errorf("redacted output still contains secret-looking text: %q", item.Path)
		}
	}
}
