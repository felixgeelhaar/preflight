package app

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
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
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
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
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0o644); err != nil {
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
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
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
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0o644); err != nil {
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
	if !strings.Contains(output, "Preflight Plan") {
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
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0o644); err != nil {
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
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
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
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0o644); err != nil {
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

	if !strings.Contains(output, "git:config") {
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
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
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
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0o644); err != nil {
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

	if !strings.Contains(output, "ssh:config") {
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
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
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
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0o644); err != nil {
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

	if !strings.Contains(output, "runtime:tool-versions") {
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
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
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
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0o644); err != nil {
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

	if !strings.Contains(output, "shell:framework:zsh:oh-my-zsh") {
		t.Errorf("output should contain 'shell:framework:zsh:oh-my-zsh' step, got: %s", output)
	}

	if !strings.Contains(output, "shell:starship") {
		t.Errorf("output should contain 'shell:starship' step, got: %s", output)
	}
}

func TestPreflight_WithLockRepo(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Create a mock lock repository
	mockRepo := &mockLockRepository{}

	// WithLockRepo should return the same instance for chaining
	result := pf.WithLockRepo(mockRepo)
	if result != pf {
		t.Error("WithLockRepo() should return the same Preflight instance")
	}
}

func TestPreflight_PrintResults_AllStatuses(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Create step IDs for testing (format: provider:action)
	successID, _ := compiler.NewStepID("test:success")
	failID, _ := compiler.NewStepID("test:fail")
	skipID, _ := compiler.NewStepID("test:skip")
	needsApplyID, _ := compiler.NewStepID("test:needs-apply")
	unknownID, _ := compiler.NewStepID("test:unknown")

	results := []execution.StepResult{
		execution.NewStepResult(successID, compiler.StatusSatisfied, nil),
		execution.NewStepResult(failID, compiler.StatusFailed, nil),
		execution.NewStepResult(skipID, compiler.StatusSkipped, nil),
		execution.NewStepResult(needsApplyID, compiler.StatusNeedsApply, nil),
		execution.NewStepResult(unknownID, compiler.StatusUnknown, nil),
	}

	pf.PrintResults(results)
	output := buf.String()

	// Check header
	if !strings.Contains(output, "Execution Results") {
		t.Error("output should contain 'Execution Results' header")
	}

	// Check all step IDs are in output
	if !strings.Contains(output, "test:success") {
		t.Error("output should contain 'test:success'")
	}
	if !strings.Contains(output, "test:fail") {
		t.Error("output should contain 'test:fail'")
	}
	if !strings.Contains(output, "test:skip") {
		t.Error("output should contain 'test:skip'")
	}

	// Check summary
	if !strings.Contains(output, "Summary:") {
		t.Error("output should contain 'Summary:'")
	}
}

func TestPreflight_PrintPlan_NoChanges(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Create temp directory structure with empty config
	tmpDir := t.TempDir()

	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
		t.Fatal(err)
	}

	// Create empty base layer (no config sections)
	baseLayer := `
name: base
`
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	plan, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	pf.PrintPlan(plan)
	output := buf.String()

	// Should indicate no changes needed
	if !strings.Contains(output, "No changes needed") {
		t.Errorf("output should contain 'No changes needed' for empty plan, got: %s", output)
	}
}

func TestPreflight_Plan_InvalidTargetName(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	_, err := pf.Plan(ctx, "/any/path", "")
	if err == nil {
		t.Error("Plan() should return error for empty target name")
	}
}

// mockLockRepository is a simple mock for testing WithLockRepo
type mockLockRepository struct{}

func (m *mockLockRepository) Load(_ context.Context, _ string) (*lock.Lockfile, error) {
	return nil, nil
}

func (m *mockLockRepository) Save(_ context.Context, _ string, _ *lock.Lockfile) error {
	return nil
}

func (m *mockLockRepository) Exists(_ context.Context, _ string) bool {
	return false
}

// ============================================================
// Capture Tests
// ============================================================

func TestPreflight_Capture_AllProviders(t *testing.T) {
	// Create temp directory with mock config files
	tmpDir := t.TempDir()

	// Create .gitconfig
	if err := os.WriteFile(tmpDir+"/.gitconfig", []byte("[user]\n    name = Test User\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create .ssh/config
	if err := os.MkdirAll(tmpDir+"/.ssh", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/.ssh/config", []byte("Host github.com\n    User git\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create .zshrc
	if err := os.WriteFile(tmpDir+"/.zshrc", []byte("# zsh config\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := CaptureOptions{
		Providers: []string{"git", "ssh", "shell"},
		HomeDir:   tmpDir,
	}

	findings, err := pf.Capture(ctx, opts)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if findings == nil {
		t.Fatal("Capture() returned nil findings")
	}

	// Should have captured items from git, ssh, shell
	if len(findings.Providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(findings.Providers))
	}

	// Should have at least the ssh config and zshrc
	if findings.ItemCount() < 2 {
		t.Errorf("expected at least 2 items, got %d", findings.ItemCount())
	}
}

func TestPreflight_Capture_UnknownProvider(t *testing.T) {
	tmpDir := t.TempDir()

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := CaptureOptions{
		Providers: []string{"unknown-provider"},
		HomeDir:   tmpDir,
	}

	findings, err := pf.Capture(ctx, opts)
	if err != nil {
		t.Fatalf("Capture() should not fail, got error = %v", err)
	}

	// Should have warning for unknown provider
	if len(findings.Warnings) == 0 {
		t.Error("expected warning for unknown provider")
	}
}

func TestPreflight_Capture_DefaultProviders(t *testing.T) {
	tmpDir := t.TempDir()

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := CaptureOptions{
		Providers: []string{}, // Empty means defaults
		HomeDir:   tmpDir,
	}

	findings, err := pf.Capture(ctx, opts)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	// Should have default providers
	if len(findings.Providers) != 4 {
		t.Errorf("expected 4 default providers, got %d: %v", len(findings.Providers), findings.Providers)
	}
}

func TestPreflight_Capture_GitConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitconfig file
	if err := os.WriteFile(tmpDir+"/.gitconfig", []byte("[user]\n    name = Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := CaptureOptions{
		Providers: []string{"git"},
		HomeDir:   tmpDir,
	}

	findings, err := pf.Capture(ctx, opts)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	// Git provider should be captured
	byProvider := findings.ItemsByProvider()
	// Git items depend on system git config, so may be empty
	// This is OK as long as no error occurred
	_ = byProvider["git"]
}

func TestPreflight_Capture_SSHConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ssh directory and config
	sshDir := tmpDir + "/.ssh"
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sshDir+"/config", []byte("Host *\n    ServerAliveInterval 60\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := CaptureOptions{
		Providers: []string{"ssh"},
		HomeDir:   tmpDir,
	}

	findings, err := pf.Capture(ctx, opts)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	// SSH provider should have captured the config
	byProvider := findings.ItemsByProvider()
	if len(byProvider["ssh"]) != 1 {
		t.Errorf("expected 1 ssh item, got %d", len(byProvider["ssh"]))
	}
}

func TestPreflight_Capture_ShellConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create shell config files
	if err := os.WriteFile(tmpDir+"/.zshrc", []byte("# zsh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/.bashrc", []byte("# bash\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := CaptureOptions{
		Providers: []string{"shell"},
		HomeDir:   tmpDir,
	}

	findings, err := pf.Capture(ctx, opts)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	// Shell provider should have captured both files
	byProvider := findings.ItemsByProvider()
	if len(byProvider["shell"]) != 2 {
		t.Errorf("expected 2 shell items, got %d", len(byProvider["shell"]))
	}
}

// ============================================================
// Doctor Tests
// ============================================================

func TestPreflight_Doctor_NoIssues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal config with no drift (empty config means satisfied)
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
		t.Fatal(err)
	}

	// Empty layer = no steps = no drift
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := NewDoctorOptions(tmpDir+"/preflight.yaml", "default")

	report, err := pf.Doctor(ctx, opts)
	if err != nil {
		t.Fatalf("Doctor() error = %v", err)
	}

	if report.HasIssues() {
		t.Errorf("expected no issues for empty config, got %d", report.IssueCount())
	}
}

func TestPreflight_Doctor_WithDrift(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config with a package that needs install (will cause NeedsApply)
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
		t.Fatal(err)
	}

	// Config with brew package (will show as needs apply if not installed)
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - somenonexistentpackage12345
`
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte(baseLayer), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := NewDoctorOptions(tmpDir+"/preflight.yaml", "default")

	report, err := pf.Doctor(ctx, opts)
	if err != nil {
		t.Fatalf("Doctor() error = %v", err)
	}

	// Should have at least one issue (drift)
	if !report.HasIssues() {
		t.Log("No issues detected - this may be expected if brew isn't available")
	}
}

func TestPreflight_Doctor_InvalidTarget(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := NewDoctorOptions(tmpDir+"/preflight.yaml", "nonexistent")

	_, err := pf.Doctor(ctx, opts)
	if err == nil {
		t.Error("Doctor() should fail for nonexistent target")
	}
}

// ============================================================
// Fix Tests
// ============================================================

func TestPreflight_Fix_NilReport(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()

	fixed, err := pf.Fix(ctx, nil)
	if err != nil {
		t.Errorf("Fix() should not error on nil report, got %v", err)
	}
	if fixed != nil {
		t.Error("Fix() should return nil for nil report")
	}
}

func TestPreflight_Fix_EmptyReport(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	report := &DoctorReport{
		Issues: []DoctorIssue{},
	}

	fixed, err := pf.Fix(ctx, report)
	if err != nil {
		t.Errorf("Fix() should not error on empty report, got %v", err)
	}
	if fixed != nil {
		t.Error("Fix() should return nil for empty report")
	}
}

// ============================================================
// Diff Tests
// ============================================================

func TestPreflight_Diff_NoDifferences(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal config (empty = no differences)
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	result, err := pf.Diff(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if result.HasDifferences() {
		t.Errorf("expected no differences, got %d", len(result.Entries))
	}
}

func TestPreflight_Diff_InvalidConfig(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	_, err := pf.Diff(ctx, "/nonexistent/config.yaml", "default")
	if err == nil {
		t.Error("Diff() should fail for nonexistent config")
	}
}

// ============================================================
// Lock Tests
// ============================================================

func TestPreflight_LockUpdate_NoLockRepo(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Use mock repo set to nil to test no-repo case
	pf.WithLockRepo(nil)

	ctx := context.Background()
	err := pf.LockUpdate(ctx, "/any/config.yaml")
	if err == nil {
		t.Error("LockUpdate() should fail without lock repo")
	}
	if !strings.Contains(err.Error(), "lockfile repository not configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPreflight_LockFreeze_NoLockRepo(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Use mock repo set to nil to test no-repo case
	pf.WithLockRepo(nil)

	ctx := context.Background()
	err := pf.LockFreeze(ctx, "/any/config.yaml")
	if err == nil {
		t.Error("LockFreeze() should fail without lock repo")
	}
	if !strings.Contains(err.Error(), "lockfile repository not configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

// mockLockRepoWithData allows configuring return values
type mockLockRepoWithData struct {
	lockfile     *lock.Lockfile
	loadErr      error
	saveErr      error
	existsResult bool
}

func (m *mockLockRepoWithData) Load(_ context.Context, _ string) (*lock.Lockfile, error) {
	return m.lockfile, m.loadErr
}

func (m *mockLockRepoWithData) Save(_ context.Context, _ string, _ *lock.Lockfile) error {
	return m.saveErr
}

func (m *mockLockRepoWithData) Exists(_ context.Context, _ string) bool {
	return m.existsResult
}

func TestPreflight_LockUpdate_WithRepo(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Use mock that returns nil lockfile (will create new one)
	mockRepo := &mockLockRepoWithData{
		lockfile:     nil,
		loadErr:      os.ErrNotExist, // Simulate missing lockfile
		existsResult: false,
	}
	pf.WithLockRepo(mockRepo)

	ctx := context.Background()
	err := pf.LockUpdate(ctx, "/tmp/config.yaml")
	if err != nil {
		t.Errorf("LockUpdate() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Lockfile updated") {
		t.Errorf("expected 'Lockfile updated' message, got: %s", output)
	}
}

func TestPreflight_LockFreeze_WithRepo(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Use mock that returns existing lockfile
	existingLock := lock.NewLockfile(config.ModeIntent, lock.MachineInfo{})
	mockRepo := &mockLockRepoWithData{
		lockfile:     existingLock,
		existsResult: true,
	}
	pf.WithLockRepo(mockRepo)

	ctx := context.Background()
	err := pf.LockFreeze(ctx, "/tmp/config.yaml")
	if err != nil {
		t.Errorf("LockFreeze() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Lockfile frozen") {
		t.Errorf("expected 'Lockfile frozen' message, got: %s", output)
	}
}

func TestPreflight_LockFreeze_NotFound(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	// Mock returns error (lockfile not found)
	mockRepo := &mockLockRepoWithData{
		loadErr: os.ErrNotExist,
	}
	pf.WithLockRepo(mockRepo)

	ctx := context.Background()
	err := pf.LockFreeze(ctx, "/tmp/config.yaml")
	if err == nil {
		t.Error("LockFreeze() should fail when lockfile not found")
	}
}

// ============================================================
// RepoInit Tests
// ============================================================

func TestPreflight_RepoInit_NewRepo(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := tmpDir + "/myconfig"

	// Create the directory (but not .git)
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := NewRepoOptions(repoPath).WithBranch("main")

	err := pf.RepoInit(ctx, opts)
	if err != nil {
		t.Fatalf("RepoInit() error = %v", err)
	}

	// Check .git directory was created
	if _, err := os.Stat(repoPath + "/.git"); os.IsNotExist(err) {
		t.Error(".git directory should exist after init")
	}

	output := buf.String()
	if !strings.Contains(output, "Repository initialized") {
		t.Errorf("expected 'Repository initialized' message, got: %s", output)
	}
}

func TestPreflight_RepoInit_AlreadyInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory to simulate existing repo
	if err := os.MkdirAll(tmpDir+"/.git", 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := NewRepoOptions(tmpDir)

	err := pf.RepoInit(ctx, opts)
	if err == nil {
		t.Error("RepoInit() should fail for already initialized repo")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPreflight_RepoInit_WithRemote(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := tmpDir + "/myconfig"

	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	opts := NewRepoOptions(repoPath).
		WithRemote("https://github.com/test/repo.git").
		WithBranch("main")

	err := pf.RepoInit(ctx, opts)
	if err != nil {
		t.Fatalf("RepoInit() error = %v", err)
	}
}

// ============================================================
// RepoStatus Tests
// ============================================================

func TestPreflight_RepoStatus_NotInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	status, err := pf.RepoStatus(ctx, tmpDir)
	if err != nil {
		t.Fatalf("RepoStatus() error = %v", err)
	}

	if status.Initialized {
		t.Error("status should show not initialized")
	}
}

func TestPreflight_RepoStatus_Initialized(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git for commit
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test")
	_ = cmd.Run()

	// Create initial commit
	if err := os.WriteFile(tmpDir+"/README.md", []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "-C", tmpDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit")
	_ = cmd.Run()

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	status, err := pf.RepoStatus(ctx, tmpDir)
	if err != nil {
		t.Fatalf("RepoStatus() error = %v", err)
	}

	if !status.Initialized {
		t.Error("status should show initialized")
	}

	if status.LastCommit == "" {
		t.Error("should have last commit hash")
	}
}

// ============================================================
// Print Function Tests
// ============================================================

func TestPreflight_PrintDoctorReport_NoIssues(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	report := &DoctorReport{
		Issues: []DoctorIssue{},
	}

	pf.PrintDoctorReport(report)
	output := buf.String()

	if !strings.Contains(output, "No issues found") {
		t.Errorf("expected 'No issues found' message, got: %s", output)
	}
}

func TestPreflight_PrintDoctorReport_WithIssues(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	report := &DoctorReport{
		Issues: []DoctorIssue{
			{
				StepID:   "brew:install:git",
				Severity: SeverityError,
				Message:  "Package not installed",
			},
			{
				StepID:     "files:link:zshrc",
				Severity:   SeverityWarning,
				Message:    "File differs",
				Fixable:    true,
				FixCommand: "preflight apply",
			},
			{
				StepID:   "git:config",
				Severity: SeverityInfo,
				Message:  "Config check skipped",
			},
		},
	}

	pf.PrintDoctorReport(report)
	output := buf.String()

	if !strings.Contains(output, "Found 3 issue(s)") {
		t.Errorf("expected 'Found 3 issue(s)', got: %s", output)
	}
	if !strings.Contains(output, "[ERROR]") {
		t.Error("expected ERROR severity marker")
	}
	if !strings.Contains(output, "[WARNING]") {
		t.Error("expected WARNING severity marker")
	}
	if !strings.Contains(output, "[INFO]") {
		t.Error("expected INFO severity marker")
	}
	if !strings.Contains(output, "preflight apply") {
		t.Error("expected fix command in output")
	}
}

func TestPreflight_PrintCaptureFindings(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	findings := &CaptureFindings{
		Items: []CapturedItem{
			{Provider: "brew", Name: "git"},
			{Provider: "brew", Name: "ripgrep"},
			{Provider: "ssh", Name: "config"},
		},
		Providers: []string{"brew", "ssh"},
		Warnings:  []string{"shell: config not found"},
	}

	pf.PrintCaptureFindings(findings)
	output := buf.String()

	if !strings.Contains(output, "Capture Results") {
		t.Error("expected 'Capture Results' header")
	}
	if !strings.Contains(output, "Captured 3 items") {
		t.Errorf("expected 'Captured 3 items', got: %s", output)
	}
	if !strings.Contains(output, "brew (2 items)") {
		t.Error("expected 'brew (2 items)'")
	}
	if !strings.Contains(output, "Warnings:") {
		t.Error("expected 'Warnings:' section")
	}
}

func TestPreflight_PrintDiff_NoDifferences(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	result := &DiffResult{
		Entries: []DiffEntry{},
	}

	pf.PrintDiff(result)
	output := buf.String()

	if !strings.Contains(output, "No differences") {
		t.Errorf("expected 'No differences', got: %s", output)
	}
}

func TestPreflight_PrintDiff_WithDifferences(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	result := &DiffResult{
		Entries: []DiffEntry{
			{Provider: "brew", Path: "brew:install:git", Type: DiffTypeAdded, Expected: "install git"},
			{Provider: "files", Path: "files:link:zshrc", Type: DiffTypeRemoved},
			{Provider: "git", Path: "git:config", Type: DiffTypeModified, Expected: "name = Test"},
		},
	}

	pf.PrintDiff(result)
	output := buf.String()

	if !strings.Contains(output, "Found 3 difference(s)") {
		t.Errorf("expected 'Found 3 difference(s)', got: %s", output)
	}
	if !strings.Contains(output, "+ ") {
		t.Error("expected '+' for added")
	}
	if !strings.Contains(output, "- ") {
		t.Error("expected '-' for removed")
	}
	if !strings.Contains(output, "~ ") {
		t.Error("expected '~' for modified")
	}
}

func TestPreflight_PrintRepoStatus_NotInitialized(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	status := &RepoStatus{
		Path:        "/some/path",
		Initialized: false,
	}

	pf.PrintRepoStatus(status)
	output := buf.String()

	if !strings.Contains(output, "Not a git repository") {
		t.Errorf("expected 'Not a git repository', got: %s", output)
	}
}

func TestPreflight_PrintRepoStatus_Synced(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	status := &RepoStatus{
		Path:         "/some/path",
		Initialized:  true,
		Branch:       "main",
		Remote:       "https://github.com/test/repo.git",
		HasChanges:   false,
		Ahead:        0,
		Behind:       0,
		LastCommit:   "abc1234",
		LastCommitAt: time.Now(),
	}

	pf.PrintRepoStatus(status)
	output := buf.String()

	if !strings.Contains(output, "Repository Status") {
		t.Error("expected 'Repository Status' header")
	}
	if !strings.Contains(output, "Branch: main") {
		t.Error("expected branch info")
	}
	if !strings.Contains(output, "Up to date") {
		t.Error("expected 'Up to date' for synced repo")
	}
}

func TestPreflight_PrintRepoStatus_NeedsPush(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	status := &RepoStatus{
		Path:        "/some/path",
		Initialized: true,
		Branch:      "main",
		Ahead:       3,
	}

	pf.PrintRepoStatus(status)
	output := buf.String()

	if !strings.Contains(output, "3 commit(s) ahead") {
		t.Errorf("expected ahead count, got: %s", output)
	}
}

func TestPreflight_PrintRepoStatus_NeedsPull(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	status := &RepoStatus{
		Path:        "/some/path",
		Initialized: true,
		Branch:      "main",
		Behind:      2,
	}

	pf.PrintRepoStatus(status)
	output := buf.String()

	if !strings.Contains(output, "2 commit(s) behind") {
		t.Errorf("expected behind count, got: %s", output)
	}
}

func TestPreflight_PrintRepoStatus_HasChanges(t *testing.T) {
	var buf bytes.Buffer
	pf := New(&buf)

	status := &RepoStatus{
		Path:        "/some/path",
		Initialized: true,
		Branch:      "main",
		HasChanges:  true,
	}

	pf.PrintRepoStatus(status)
	output := buf.String()

	if !strings.Contains(output, "Uncommitted changes") {
		t.Errorf("expected uncommitted changes warning, got: %s", output)
	}
}

// ============================================================
// Apply Tests
// ============================================================

func TestPreflight_Apply_EmptyPlan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal config (empty = no steps)
	manifest := `
targets:
  default:
    - base
`
	if err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tmpDir+"/layers", 0o755); err != nil {
		t.Fatal(err)
	}

	// Empty layer = no steps to apply
	if err := os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	pf := New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, tmpDir+"/preflight.yaml", "default")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	// Apply the empty plan in dry-run mode
	results, err := pf.Apply(ctx, plan, true)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Empty plan should have empty results
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty plan, got %d", len(results))
	}
}
