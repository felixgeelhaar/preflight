package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildFeedbackURL_IncludesPlatform(t *testing.T) {
	t.Parallel()

	u := buildFeedbackURL("linux", "amd64", "v9.9.9")

	if !strings.HasPrefix(u, "https://github.com/felixgeelhaar/preflight/discussions/new?") {
		t.Errorf("URL missing GitHub Discussions prefix: %s", u)
	}

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("URL parse: %v", err)
	}
	body := parsed.Query().Get("body")
	for _, want := range []string{"linux", "amd64", "v9.9.9", "What were you trying to do"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q:\n%s", want, body)
		}
	}
}

func TestBuildFeedbackURL_DoesNotIncludeSecrets(t *testing.T) {
	t.Parallel()

	// Defense in depth: the URL must never carry env-derived secret-looking
	// strings. The function only takes goos/goarch/version arguments — this
	// test guards against a future refactor that adds richer auto-context.
	u := buildFeedbackURL("darwin", "arm64", "v0.0.0")
	for _, banned := range []string{
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY",
		"AWS_SECRET", "GITHUB_TOKEN", "id_rsa", ".aws/credentials",
	} {
		if strings.Contains(u, banned) {
			t.Errorf("feedback URL must not contain %q:\n%s", banned, u)
		}
	}
}
