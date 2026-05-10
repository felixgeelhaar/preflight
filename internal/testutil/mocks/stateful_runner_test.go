package mocks

import (
	"context"
	"strings"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

func TestStatefulCommandRunner_BrewListReflectsInstall(t *testing.T) {
	t.Parallel()
	r := NewStatefulCommandRunner()
	ctx := context.Background()

	// Before install: list is empty.
	res, err := r.Run(ctx, "brew", "list", "--formula")
	if err != nil || res.Stdout != "" {
		t.Fatalf("expected empty list before install, got stdout=%q err=%v", res.Stdout, err)
	}

	// Install ripgrep.
	if _, err := r.Run(ctx, "brew", "install", "ripgrep"); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// After install: list contains ripgrep.
	res, err = r.Run(ctx, "brew", "list", "--formula")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(res.Stdout, "ripgrep") {
		t.Errorf("expected ripgrep in list output, got %q", res.Stdout)
	}
}

func TestStatefulCommandRunner_AptDpkgQueryReflectsInstall(t *testing.T) {
	t.Parallel()
	r := NewStatefulCommandRunner()
	ctx := context.Background()

	// Before install: dpkg-query exits 1.
	res, _ := r.Run(ctx, "dpkg-query", "-W", "-f=${db:Status-Status}", "curl")
	if res.Success() {
		t.Fatal("dpkg-query must fail when package is not installed")
	}

	// Install via sudo apt-get install -y.
	if _, err := r.Run(ctx, "sudo", "apt-get", "install", "-y", "curl"); err != nil {
		t.Fatalf("install: %v", err)
	}

	// After install: dpkg-query reports "installed".
	res, err := r.Run(ctx, "dpkg-query", "-W", "-f=${db:Status-Status}", "curl")
	if err != nil {
		t.Fatalf("dpkg-query: %v", err)
	}
	if !res.Success() {
		t.Error("dpkg-query must succeed after install")
	}
	if !strings.Contains(res.Stdout, "installed") {
		t.Errorf("expected 'installed' in dpkg-query stdout, got %q", res.Stdout)
	}
}

func TestStatefulCommandRunner_SeedInstalled(t *testing.T) {
	t.Parallel()
	r := NewStatefulCommandRunner()
	r.SeedInstalled("brew", "fzf")

	res, _ := r.Run(context.Background(), "brew", "list", "--formula")
	if !strings.Contains(res.Stdout, "fzf") {
		t.Errorf("seeded package should appear in list, got %q", res.Stdout)
	}
}

func TestStatefulCommandRunner_HandlerOverride(t *testing.T) {
	t.Parallel()
	r := NewStatefulCommandRunner()
	r.AddHandler("custom-tool", nil, func(_ []string) (ports.CommandResult, error) {
		return ports.CommandResult{Stdout: "ok", ExitCode: 0}, nil
	})

	res, err := r.Run(context.Background(), "custom-tool", "any", "args")
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if res.Stdout != "ok" {
		t.Errorf("handler output = %q, want %q", res.Stdout, "ok")
	}
}

func TestStatefulCommandRunner_GemListReflectsInstall(t *testing.T) {
	t.Parallel()
	r := NewStatefulCommandRunner()
	ctx := context.Background()

	// Before install: gem list -i returns 1.
	res, _ := r.Run(ctx, "gem", "list", "-i", "rails")
	if res.Success() {
		t.Fatal("gem list -i must fail before install")
	}

	if _, err := r.Run(ctx, "gem", "install", "rails"); err != nil {
		t.Fatalf("install: %v", err)
	}

	res, err := r.Run(ctx, "gem", "list", "-i", "rails")
	if err != nil || !res.Success() {
		t.Errorf("gem list -i must succeed after install, got %+v err=%v", res, err)
	}
}

func TestStatefulCommandRunner_PipShowReflectsInstall(t *testing.T) {
	t.Parallel()
	r := NewStatefulCommandRunner()
	ctx := context.Background()

	res, _ := r.Run(ctx, "pip", "show", "black")
	if res.Success() {
		t.Fatal("pip show must fail before install")
	}

	if _, err := r.Run(ctx, "pip", "install", "--user", "black"); err != nil {
		t.Fatalf("install: %v", err)
	}

	res, err := r.Run(ctx, "pip", "show", "black")
	if err != nil || !res.Success() {
		t.Errorf("pip show must succeed after install, got %+v err=%v", res, err)
	}
	if !strings.Contains(res.Stdout, "black") {
		t.Errorf("pip show stdout missing package name: %q", res.Stdout)
	}
}

func TestStatefulCommandRunner_RecordsCalls(t *testing.T) {
	t.Parallel()
	r := NewStatefulCommandRunner()
	_, _ = r.Run(context.Background(), "brew", "install", "fzf")
	calls := r.Calls()
	if len(calls) != 1 || calls[0].Command != "brew" || calls[0].Args[0] != "install" {
		t.Errorf("expected 1 call to brew install, got %+v", calls)
	}
}
