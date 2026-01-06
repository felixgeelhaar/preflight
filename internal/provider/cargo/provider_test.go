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
