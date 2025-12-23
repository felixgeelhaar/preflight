package brew

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestTapStep_ID(t *testing.T) {
	step := NewTapStep("homebrew/cask", nil)
	expected := "brew:tap:homebrew/cask"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestTapStep_DependsOn(t *testing.T) {
	step := NewTapStep("homebrew/cask", nil)
	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestTapStep_Check_AlreadyTapped(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"tap"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "homebrew/cask\nhomebrew/core\n",
	})

	step := NewTapStep("homebrew/cask", runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestTapStep_Check_NotTapped(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"tap"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "homebrew/core\n",
	})

	step := NewTapStep("homebrew/cask", runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestTapStep_Plan(t *testing.T) {
	step := NewTapStep("homebrew/cask", nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Name() != "homebrew/cask" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "homebrew/cask")
	}
}

func TestTapStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"tap", "homebrew/cask"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewTapStep("homebrew/cask", runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	calls := runner.Calls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Command != "brew" {
		t.Errorf("Command = %q, want %q", calls[0].Command, "brew")
	}
}

func TestTapStep_Apply_Failure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"tap", "invalid/tap"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Error: invalid tap",
	})

	step := NewTapStep("invalid/tap", runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error on failure")
	}
}

func TestFormulaStep_ID(t *testing.T) {
	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, nil)
	expected := "brew:formula:git"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestFormulaStep_ID_WithTap(t *testing.T) {
	formula := Formula{Name: "neovim", Tap: "homebrew/core"}
	step := NewFormulaStep(formula, nil)
	expected := "brew:formula:homebrew/core/neovim"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestFormulaStep_DependsOn_WithTap(t *testing.T) {
	formula := Formula{Name: "neovim", Tap: "homebrew/core"}
	step := NewFormulaStep(formula, nil)
	deps := step.DependsOn()
	if len(deps) != 1 {
		t.Fatalf("DependsOn() len = %d, want 1", len(deps))
	}
	if deps[0].String() != "brew:tap:homebrew/core" {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), "brew:tap:homebrew/core")
	}
}

func TestFormulaStep_Check_Installed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--formula"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "git\ncurl\nwget\n",
	})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestFormulaStep_Check_NotInstalled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--formula"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "curl\nwget\n",
	})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestFormulaStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "git"}, ports.CommandResult{
		ExitCode: 0,
	})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestFormulaStep_Apply_WithArgs(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "neovim", "--HEAD"}, ports.CommandResult{
		ExitCode: 0,
	})

	formula := Formula{Name: "neovim", Args: []string{"--HEAD"}}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	calls := runner.Calls()
	if len(calls[0].Args) != 3 {
		t.Errorf("Args len = %d, want 3", len(calls[0].Args))
	}
}

func TestCaskStep_ID(t *testing.T) {
	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, nil)
	expected := "brew:cask:docker"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestCaskStep_Check_Installed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "docker\nslack\n",
	})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestCaskStep_Check_NotInstalled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "slack\n",
	})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestCaskStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "--cask", "docker"}, ports.CommandResult{
		ExitCode: 0,
	})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestCaskStep_DependsOn_WithTap(t *testing.T) {
	cask := Cask{Name: "font-fira-code", Tap: "homebrew/cask-fonts"}
	step := NewCaskStep(cask, nil)
	deps := step.DependsOn()
	if len(deps) != 1 {
		t.Fatalf("DependsOn() len = %d, want 1", len(deps))
	}
	if deps[0].String() != "brew:tap:homebrew/cask-fonts" {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), "brew:tap:homebrew/cask-fonts")
	}
}

func TestCaskStep_DependsOn_NoTap(t *testing.T) {
	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, nil)
	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestFormulaStep_DependsOn_NoTap(t *testing.T) {
	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, nil)
	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestTapStep_Explain(t *testing.T) {
	step := NewTapStep("homebrew/cask", nil)
	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
	if explanation.Detail() == "" {
		t.Error("Explain().Detail() should not be empty")
	}
}

func TestFormulaStep_Plan(t *testing.T) {
	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Name() != "git" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "git")
	}
}

func TestFormulaStep_Explain(t *testing.T) {
	formula := Formula{Name: "neovim", Args: []string{"--HEAD"}}
	step := NewFormulaStep(formula, nil)
	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
	if explanation.Detail() == "" {
		t.Error("Explain().Detail() should not be empty")
	}
}

func TestCaskStep_Plan(t *testing.T) {
	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Name() != "docker" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "docker")
	}
}

func TestCaskStep_Explain(t *testing.T) {
	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, nil)
	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
	if explanation.Detail() == "" {
		t.Error("Explain().Detail() should not be empty")
	}
}

func TestFormulaStep_Apply_Failure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "invalid-formula"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Error: No formula found",
	})

	formula := Formula{Name: "invalid-formula"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error on failure")
	}
}

func TestCaskStep_Apply_Failure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "--cask", "invalid-cask"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Error: No cask found",
	})

	cask := Cask{Name: "invalid-cask"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error on failure")
	}
}

func TestTapStep_Check_CommandError(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"tap"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Homebrew not installed",
	})

	step := NewTapStep("homebrew/cask", runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err == nil {
		t.Error("Check() should return error on command failure")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusUnknown)
	}
}

func TestFormulaStep_Check_CommandError(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--formula"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Homebrew not installed",
	})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err == nil {
		t.Error("Check() should return error on command failure")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusUnknown)
	}
}

func TestCaskStep_Check_CommandError(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Homebrew not installed",
	})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err == nil {
		t.Error("Check() should return error on command failure")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusUnknown)
	}
}
