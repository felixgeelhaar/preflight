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
		"/home/alice/.aws/credentials",
		"/Users/bob/.env",
		"/Users/bob/.env.local",
		"/etc/ssl/server.pem",
		"/var/keys/private.key",
		"/home/work/api_token",
		"/home/work/MY_SECRET_KEY",
		"/home/work/.netrc",
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

func TestHashEmailDomain_Stable(t *testing.T) {
	t.Parallel()
	a := HashEmailDomain("example.com")
	b := HashEmailDomain("EXAMPLE.com")
	if a != b {
		t.Errorf("HashEmailDomain not case-insensitive: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, "domain-") {
		t.Errorf("hash %q missing prefix", a)
	}
	if HashEmailDomain("example.com") == HashEmailDomain("other.com") {
		t.Error("different domains hashed to the same value")
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
