package brew

import (
	"context"
	"os/exec"
	"strings"
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
	if len(deps) != 1 {
		t.Errorf("DependsOn() len = %d, want 1", len(deps))
	}
	if deps[0].String() != brewInstallStepID {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), brewInstallStepID)
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
	if len(deps) != 2 {
		t.Fatalf("DependsOn() len = %d, want 2", len(deps))
	}
	if deps[0].String() != brewInstallStepID {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), brewInstallStepID)
	}
	if deps[1].String() != "brew:tap:homebrew/core" {
		t.Errorf("DependsOn()[1] = %q, want %q", deps[1].String(), "brew:tap:homebrew/core")
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
	if len(deps) != 2 {
		t.Fatalf("DependsOn() len = %d, want 2", len(deps))
	}
	if deps[0].String() != brewInstallStepID {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), brewInstallStepID)
	}
	if deps[1].String() != "brew:tap:homebrew/cask-fonts" {
		t.Errorf("DependsOn()[1] = %q, want %q", deps[1].String(), "brew:tap:homebrew/cask-fonts")
	}
}

func TestCaskStep_DependsOn_NoTap(t *testing.T) {
	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, nil)
	deps := step.DependsOn()
	if len(deps) != 1 {
		t.Errorf("DependsOn() len = %d, want 1", len(deps))
	}
	if deps[0].String() != brewInstallStepID {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), brewInstallStepID)
	}
}

func TestFormulaStep_DependsOn_NoTap(t *testing.T) {
	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, nil)
	deps := step.DependsOn()
	if len(deps) != 1 {
		t.Errorf("DependsOn() len = %d, want 1", len(deps))
	}
	if deps[0].String() != brewInstallStepID {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), brewInstallStepID)
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

// --- InstallStep tests ---

func TestInstallStep_ID(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil)
	if got := step.ID().String(); got != brewInstallStepID {
		t.Errorf("ID() = %q, want %q", got, brewInstallStepID)
	}
}

func TestInstallStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil)
	deps := step.DependsOn()
	if deps != nil {
		t.Errorf("DependsOn() = %v, want nil", deps)
	}
}

func TestInstallStep_Check(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	// On macOS with brew installed, this should return Satisfied.
	// On systems without brew, it returns NeedsApply.
	// Either is valid; we just ensure no error occurs.
	if status != compiler.StatusSatisfied && status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want StatusSatisfied or StatusNeedsApply", status)
	}
}

func TestInstallStep_Plan(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Name() != "install" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "install")
	}
}

func TestInstallStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("/bin/bash", []string{"-c", "curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | /bin/bash"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Homebrew installed successfully",
	})

	step := NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestInstallStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// No result registered, runner returns error

	step := NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error when runner fails")
	}
}

func TestInstallStep_Apply_CommandFailure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("/bin/bash", []string{"-c", "curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | /bin/bash"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "installation failed",
	})

	step := NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error on non-zero exit code")
	}
}

func TestInstallStep_Explain(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil)
	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
	if explanation.Detail() == "" {
		t.Error("Explain().Detail() should not be empty")
	}
}

// --- TapStep Check with command-not-found ---

func TestTapStep_Check_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("brew", []string{"tap"}, &exec.Error{Name: "brew", Err: exec.ErrNotFound})

	step := NewTapStep("homebrew/cask", runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v, want nil for command-not-found", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

// --- TapStep Apply with command-not-found ---

func TestTapStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("brew", []string{"tap", "homebrew/cask"}, &exec.Error{Name: "brew", Err: exec.ErrNotFound})

	step := NewTapStep("homebrew/cask", runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error when brew not found")
	}
	if !strings.Contains(err.Error(), "brew not found") {
		t.Errorf("Apply() error = %q, want message containing 'brew not found'", err.Error())
	}
}

// --- FormulaStep Check with command-not-found ---

func TestFormulaStep_Check_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("brew", []string{"list", "--formula"}, &exec.Error{Name: "brew", Err: exec.ErrNotFound})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v, want nil for command-not-found", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

// --- FormulaStep Apply with command-not-found ---

func TestFormulaStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("brew", []string{"install", "git"}, &exec.Error{Name: "brew", Err: exec.ErrNotFound})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error when brew not found")
	}
	if !strings.Contains(err.Error(), "brew not found") {
		t.Errorf("Apply() error = %q, want message containing 'brew not found'", err.Error())
	}
}

// --- CaskStep Check with command-not-found ---

func TestCaskStep_Check_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("brew", []string{"list", "--cask"}, &exec.Error{Name: "brew", Err: exec.ErrNotFound})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v, want nil for command-not-found", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

// --- CaskStep Apply with command-not-found ---

func TestCaskStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("brew", []string{"install", "--cask", "docker"}, &exec.Error{Name: "brew", Err: exec.ErrNotFound})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error when brew not found")
	}
	if !strings.Contains(err.Error(), "brew not found") {
		t.Errorf("Apply() error = %q, want message containing 'brew not found'", err.Error())
	}
}

// --- FormulaStep InstalledVersion tests ---

func TestFormulaStep_InstalledVersion_Found(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--versions", "git"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "git 2.43.0\n",
	})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(ctx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if !found {
		t.Error("InstalledVersion() found = false, want true")
	}
	if version != "2.43.0" {
		t.Errorf("InstalledVersion() version = %q, want %q", version, "2.43.0")
	}
}

func TestFormulaStep_InstalledVersion_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--versions", "nonexistent"}, ports.CommandResult{
		ExitCode: 1,
		Stdout:   "",
	})

	formula := Formula{Name: "nonexistent"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(ctx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false")
	}
	if version != "" {
		t.Errorf("InstalledVersion() version = %q, want empty", version)
	}
}

func TestFormulaStep_InstalledVersion_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("brew", []string{"list", "--versions", "git"}, &exec.Error{Name: "brew", Err: exec.ErrNotFound})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(ctx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v, want nil for command-not-found", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false")
	}
	if version != "" {
		t.Errorf("InstalledVersion() version = %q, want empty", version)
	}
}

func TestFormulaStep_InstalledVersion_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// No result registered, returns generic error

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	_, _, err := step.InstalledVersion(ctx)
	if err == nil {
		t.Error("InstalledVersion() should return error on runner failure")
	}
}

func TestFormulaStep_InstalledVersion_ShortOutput(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--versions", "git"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "git",
	})

	formula := Formula{Name: "git"}
	step := NewFormulaStep(formula, runner)
	ctx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(ctx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false (only one field)")
	}
	if version != "" {
		t.Errorf("InstalledVersion() version = %q, want empty", version)
	}
}

// --- CaskStep InstalledVersion tests ---

func TestCaskStep_InstalledVersion_Found(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask", "--versions", "docker"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "docker 4.25.0\n",
	})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(ctx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if !found {
		t.Error("InstalledVersion() found = false, want true")
	}
	if version != "4.25.0" {
		t.Errorf("InstalledVersion() version = %q, want %q", version, "4.25.0")
	}
}

func TestCaskStep_InstalledVersion_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask", "--versions", "nonexistent"}, ports.CommandResult{
		ExitCode: 1,
		Stdout:   "",
	})

	cask := Cask{Name: "nonexistent"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(ctx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false")
	}
	if version != "" {
		t.Errorf("InstalledVersion() version = %q, want empty", version)
	}
}

func TestCaskStep_InstalledVersion_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("brew", []string{"list", "--cask", "--versions", "docker"}, &exec.Error{Name: "brew", Err: exec.ErrNotFound})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(ctx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v, want nil for command-not-found", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false")
	}
	if version != "" {
		t.Errorf("InstalledVersion() version = %q, want empty", version)
	}
}

func TestCaskStep_InstalledVersion_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// No result registered, returns generic error

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	_, _, err := step.InstalledVersion(ctx)
	if err == nil {
		t.Error("InstalledVersion() should return error on runner failure")
	}
}

func TestCaskStep_InstalledVersion_ShortOutput(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask", "--versions", "docker"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "docker",
	})

	cask := Cask{Name: "docker"}
	step := NewCaskStep(cask, runner)
	ctx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(ctx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false (only one field)")
	}
	if version != "" {
		t.Errorf("InstalledVersion() version = %q, want empty", version)
	}
}
