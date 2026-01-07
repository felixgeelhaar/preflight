package cargo

import (
	"context"
	"os/exec"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

type errRunner struct {
	err error
}

func (r errRunner) Run(_ context.Context, _ string, _ ...string) (ports.CommandResult, error) {
	return ports.CommandResult{}, r.err
}

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(nil)
	if got := provider.Name(); got != "cargo" {
		t.Errorf("Name() = %q, want %q", got, "cargo")
	}
}

func TestProvider_Compile_Empty(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestProvider_Compile_NoCargoSection(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"npm": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestProvider_Compile_Crates(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"cargo": map[string]interface{}{
			"crates": []interface{}{"ripgrep", "bat"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Compile() len = %d, want 2", len(steps))
	}

	// Verify step IDs
	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	if !ids["cargo:crate:ripgrep"] {
		t.Error("Missing cargo:crate:ripgrep step")
	}
	if !ids["cargo:crate:bat"] {
		t.Error("Missing cargo:crate:bat step")
	}
}

func TestProvider_Compile_CrateWithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"cargo": map[string]interface{}{
			"crates": []interface{}{"bat@0.22.1"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}
	if steps[0].ID().String() != "cargo:crate:bat" {
		t.Errorf("ID() = %q, want %q", steps[0].ID().String(), "cargo:crate:bat")
	}
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"cargo": map[string]interface{}{
			"crates": "not-a-list",
		},
	})
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Error("Compile() should return error for invalid config")
	}
}

func TestCrateStep_Check_Installed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("cargo", []string{"install", "--list"}, ports.CommandResult{
		Stdout:   "ripgrep v14.1.0:\n    rg",
		ExitCode: 0,
	})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestCrateStep_Check_NotInstalled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("cargo", []string{"install", "--list"}, ports.CommandResult{
		Stdout:   "bat v0.22.1:\n    bat",
		ExitCode: 0,
	})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestCrateStep_Check_CargoMissing(t *testing.T) {
	runner := errRunner{err: &exec.Error{Name: "cargo", Err: exec.ErrNotFound}}

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err == nil {
		t.Fatalf("Check() error = nil, want error")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusUnknown)
	}
}

func TestCrateStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("cargo", []string{"install", "ripgrep"}, ports.CommandResult{
		Stdout:   "Installing ripgrep v14.1.0",
		ExitCode: 0,
	})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestCrateStep_Apply_WithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("cargo", []string{"install", "bat", "--version", "0.22.1"}, ports.CommandResult{
		Stdout:   "Installing bat v0.22.1",
		ExitCode: 0,
	})

	step := NewCrateStep(Crate{Name: "bat", Version: "0.22.1"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestCrateStep_Plan(t *testing.T) {
	step := NewCrateStep(Crate{Name: "ripgrep", Version: "14.1.0"}, nil, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Resource() != "cargo-crate" {
		t.Errorf("Plan().Resource() = %q, want %q", diff.Resource(), "cargo-crate")
	}
	if diff.Name() != "ripgrep" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "ripgrep")
	}
	if diff.NewValue() != "14.1.0" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "14.1.0")
	}
}

func TestCrateStep_Explain(t *testing.T) {
	step := NewCrateStep(Crate{Name: "ripgrep", Version: "14.1.0"}, nil, nil)
	explainCtx := compiler.NewExplainContext()

	explanation := step.Explain(explainCtx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
	if explanation.Detail() == "" {
		t.Error("Explain().Detail() should not be empty")
	}
}

func TestParseCrate_Simple(t *testing.T) {
	crate := parseCrateString("ripgrep")
	if crate.Name != "ripgrep" {
		t.Errorf("Name = %q, want %q", crate.Name, "ripgrep")
	}
	if crate.Version != "" {
		t.Errorf("Version = %q, want %q", crate.Version, "")
	}
}

func TestParseCrate_WithVersion(t *testing.T) {
	crate := parseCrateString("bat@0.22.1")
	if crate.Name != "bat" {
		t.Errorf("Name = %q, want %q", crate.Name, "bat")
	}
	if crate.Version != "0.22.1" {
		t.Errorf("Version = %q, want %q", crate.Version, "0.22.1")
	}
}

func TestCrateStep_DependsOn(t *testing.T) {
	deps := []compiler.StepID{compiler.MustNewStepID("brew:formula:rust")}
	step := NewCrateStep(Crate{Name: "ripgrep"}, nil, deps)

	got := step.DependsOn()
	if len(got) != 1 {
		t.Fatalf("DependsOn() len = %d, want 1", len(got))
	}
	if got[0].String() != "brew:formula:rust" {
		t.Errorf("DependsOn()[0] = %q, want %q", got[0].String(), "brew:formula:rust")
	}
}

func TestCrateStep_DependsOn_Empty(t *testing.T) {
	step := NewCrateStep(Crate{Name: "ripgrep"}, nil, nil)

	got := step.DependsOn()
	if got != nil {
		t.Errorf("DependsOn() = %v, want nil", got)
	}
}

func TestCrateStep_LockInfo(t *testing.T) {
	step := NewCrateStep(Crate{Name: "ripgrep", Version: "14.1.0"}, nil, nil)

	info, ok := step.LockInfo()
	if !ok {
		t.Fatal("LockInfo() ok = false, want true")
	}
	if info.Provider != "cargo" {
		t.Errorf("LockInfo().Provider = %q, want %q", info.Provider, "cargo")
	}
	if info.Name != "ripgrep" {
		t.Errorf("LockInfo().Name = %q, want %q", info.Name, "ripgrep")
	}
	if info.Version != "14.1.0" {
		t.Errorf("LockInfo().Version = %q, want %q", info.Version, "14.1.0")
	}
}

func TestCrateStep_InstalledVersion_Found(t *testing.T) {
	runner := mocks.NewCommandRunner()
	// cargo install --list output format: "crate v1.2.3:" (colon is preserved in parsing)
	runner.AddResult("cargo", []string{"install", "--list"}, ports.CommandResult{
		Stdout:   "ripgrep v14.1.0:\n    rg\nbat v0.22.1:\n    bat",
		ExitCode: 0,
	})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if !found {
		t.Error("InstalledVersion() found = false, want true")
	}
	// Note: version includes trailing colon from cargo output format
	if version != "14.1.0:" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "14.1.0:")
	}
}

func TestCrateStep_InstalledVersion_NotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("cargo", []string{"install", "--list"}, ports.CommandResult{
		Stdout:   "bat v0.22.1:\n    bat",
		ExitCode: 0,
	})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false")
	}
	if version != "" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "")
	}
}

func TestCrateStep_InstalledVersion_CargoNotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("cargo", []string{"install", "--list"}, &commandNotFoundError{cmd: "cargo"})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false")
	}
	if version != "" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "")
	}
}

func TestCrateStep_InstalledVersion_CommandFailed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("cargo", []string{"install", "--list"}, ports.CommandResult{
		Stderr:   "error: some error",
		ExitCode: 1,
	})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false")
	}
	if version != "" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "")
	}
}

func TestCrateStep_Check_CargoMissing_WithDeps(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("cargo", []string{"install", "--list"}, &commandNotFoundError{cmd: "cargo"})

	deps := []compiler.StepID{compiler.MustNewStepID("brew:formula:rust")}
	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, deps)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestCrateStep_Check_CommandFailed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("cargo", []string{"install", "--list"}, ports.CommandResult{
		Stderr:   "error: some error",
		ExitCode: 1,
	})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err == nil {
		t.Fatal("Check() error = nil, want error")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusUnknown)
	}
}

func TestCrateStep_Apply_CargoNotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("cargo", []string{"install", "ripgrep"}, &commandNotFoundError{cmd: "cargo"})

	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err == nil {
		t.Error("Apply() error = nil, want error for cargo not found")
	}
}

func TestCrateStep_Apply_Failure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("cargo", []string{"install", "nonexistent"}, ports.CommandResult{
		Stderr:   "error: could not find 'nonexistent' in registry",
		ExitCode: 101,
	})

	step := NewCrateStep(Crate{Name: "nonexistent"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err == nil {
		t.Error("Apply() error = nil, want error for failed install")
	}
}

func TestCrateStep_Plan_NoVersion(t *testing.T) {
	step := NewCrateStep(Crate{Name: "ripgrep"}, nil, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.NewValue() != "latest" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "latest")
	}
}

func TestParseCrate_Map(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    Crate
		wantErr bool
	}{
		{
			name:  "map with name only",
			input: map[string]interface{}{"name": "ripgrep"},
			want:  Crate{Name: "ripgrep"},
		},
		{
			name:  "map with name and version",
			input: map[string]interface{}{"name": "ripgrep", "version": "14.1.0"},
			want:  Crate{Name: "ripgrep", Version: "14.1.0"},
		},
		{
			name:    "map missing name",
			input:   map[string]interface{}{"version": "1.0.0"},
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCrate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCrate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseCrate() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// commandNotFoundError implements exec.Error for testing command not found scenarios.
type commandNotFoundError struct {
	cmd string
}

func (e *commandNotFoundError) Error() string {
	return "exec: " + e.cmd + ": executable file not found in $PATH"
}

func (e *commandNotFoundError) Unwrap() error {
	return exec.ErrNotFound
}
