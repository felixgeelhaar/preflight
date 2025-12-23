package app

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestPreflight_New(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)
	if pf == nil {
		t.Fatal("New() returned nil")
	}
}

func TestPreflight_Plan_Integration(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create manifest
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0755); err != nil {
		t.Fatal(err)
	}

	// Create base layer with brew config
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - git
      - ripgrep
`
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Plan() returned nil plan")
	}
}

func TestPreflight_PrintPlan(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create manifest
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0755); err != nil {
		t.Fatal(err)
	}

	// Create base layer
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - git
`
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	plan, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	pf.PrintPlan(plan)

	output := buf.String()
	if output == "" {
		t.Error("PrintPlan() produced no output")
	}

	// Should contain the plan header
	if !contains(output, "Preflight Plan") {
		t.Errorf("output should contain 'Preflight Plan', got: %s", output)
	}
}

func TestPreflight_Plan_InvalidTarget(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir+"/layers", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	_, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "nonexistent")
	if err == nil {
		t.Error("Plan() should return error for nonexistent target")
	}
}

func TestPreflight_Plan_MissingConfig(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	_, err := pf.Plan(ctx, "/nonexistent/preflight.yaml", "default")
	if err == nil {
		t.Error("Plan() should return error for missing config")
	}
}

func TestPreflight_Plan_WithGitConfig(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create manifest
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0755); err != nil {
		t.Fatal(err)
	}

	// Create base layer with git config
	baseLayer := `
name: base
git:
  user:
    name: John Doe
    email: john@example.com
  core:
    editor: nvim
  alias:
    co: checkout
    st: status
`
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Plan() returned nil plan")
	}

	// The plan should include a git:config step
	pf.PrintPlan(plan)
	output := buf.String()

	if !contains(output, "git:config") {
		t.Errorf("output should contain 'git:config' step, got: %s", output)
	}
}

func TestPreflight_Plan_WithSSHConfig(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create manifest
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0755); err != nil {
		t.Fatal(err)
	}

	// Create base layer with SSH config
	baseLayer := `
name: base
ssh:
  defaults:
    addkeystoagent: true
    identitiesonly: true
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_ed25519
`
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Plan() returned nil plan")
	}

	// The plan should include an ssh:config step
	pf.PrintPlan(plan)
	output := buf.String()

	if !contains(output, "ssh:config") {
		t.Errorf("output should contain 'ssh:config' step, got: %s", output)
	}
}

func TestPreflight_Plan_WithRuntimeConfig(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create manifest
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0755); err != nil {
		t.Fatal(err)
	}

	// Create base layer with runtime config
	baseLayer := `
name: base
runtime:
  backend: rtx
  tools:
    - name: node
      version: "20.10.0"
    - name: golang
      version: "1.21.5"
`
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Plan() returned nil plan")
	}

	// The plan should include a runtime:tool-versions step
	pf.PrintPlan(plan)
	output := buf.String()

	if !contains(output, "runtime:tool-versions") {
		t.Errorf("output should contain 'runtime:tool-versions' step, got: %s", output)
	}
}

func TestPreflight_Plan_WithShellConfig(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create manifest
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0755); err != nil {
		t.Fatal(err)
	}

	// Create base layer with shell config
	baseLayer := `
name: base
shell:
  shells:
    - name: zsh
      framework: oh-my-zsh
      theme: robbyrussell
      plugins:
        - git
        - docker
  starship:
    enabled: true
    preset: nerd-font-symbols
`
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Plan() returned nil plan")
	}

	// The plan should include shell framework and starship steps
	pf.PrintPlan(plan)
	output := buf.String()

	if !contains(output, "shell:framework:zsh:oh-my-zsh") {
		t.Errorf("output should contain 'shell:framework:zsh:oh-my-zsh' step, got: %s", output)
	}

	if !contains(output, "shell:starship") {
		t.Errorf("output should contain 'shell:starship' step, got: %s", output)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
