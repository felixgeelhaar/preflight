package macos

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// DefaultsStep tests

func TestDefaultsStep_ID(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  true,
	}, runner)

	expected := "macos:defaults:com.apple.dock:autohide"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestDefaultsStep_DependsOn(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  true,
	}, runner)

	if len(step.DependsOn()) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(step.DependsOn()))
	}
}

func TestDefaultsStep_Check_Satisfied(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "com.apple.dock", "autohide"}, ports.CommandResult{
		Stdout:   "true\n",
		ExitCode: 0,
	})

	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  "true", // Value as string to match defaults output
	}, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestDefaultsStep_Check_NeedsApply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "com.apple.dock", "autohide"}, ports.CommandResult{
		Stdout:   "false\n",
		ExitCode: 0,
	})

	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  "true",
	}, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestDefaultsStep_Check_KeyDoesNotExist(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "com.apple.dock", "autohide"}, ports.CommandResult{
		Stderr:   "not found",
		ExitCode: 1,
	})

	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  true,
	}, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestDefaultsStep_Plan(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "com.apple.dock", "autohide"}, ports.CommandResult{
		Stdout:   "false\n",
		ExitCode: 0,
	})

	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  true,
	}, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeModify {
		t.Errorf("Diff type = %v, want %v", diff.Type(), compiler.DiffTypeModify)
	}
	if diff.Name() != "autohide" {
		t.Errorf("Diff name = %q, want %q", diff.Name(), "autohide")
	}
}

func TestDefaultsStep_Apply_Bool(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "com.apple.dock", "autohide", "-bool", "true"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  true,
	}, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestDefaultsStep_Apply_Int(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "NSGlobalDomain", "KeyRepeat", "-int", "2"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewDefaultsStep(Default{
		Domain: "NSGlobalDomain",
		Key:    "KeyRepeat",
		Type:   "int",
		Value:  2,
	}, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestDefaultsStep_Apply_String(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "NSGlobalDomain", "AppleShowScrollBars", "-string", "Always"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewDefaultsStep(Default{
		Domain: "NSGlobalDomain",
		Key:    "AppleShowScrollBars",
		Type:   "string",
		Value:  "Always",
	}, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestDefaultsStep_Apply_Failure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "com.apple.dock", "autohide", "-bool", "true"}, ports.CommandResult{
		Stderr:   "permission denied",
		ExitCode: 1,
	})

	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  true,
	}, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err == nil {
		t.Fatal("Apply() should fail")
	}
}

func TestDefaultsStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewDefaultsStep(Default{
		Domain: "com.apple.dock",
		Key:    "autohide",
		Type:   "bool",
		Value:  true,
	}, runner)

	explanation := step.Explain(compiler.NewExplainContext())
	if explanation.Summary() == "" {
		t.Error("Explain() summary should not be empty")
	}
	if len(explanation.DocLinks()) == 0 {
		t.Error("Explain() should have doc links")
	}
}

// DockStep tests

func TestDockStep_ID_Add(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewDockStep("Safari", true, runner)

	expected := "macos:dock:add:Safari"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestDockStep_ID_Remove(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewDockStep("FaceTime", false, runner)

	expected := "macos:dock:remove:FaceTime"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestDockStep_Check_Add_Satisfied(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("dockutil", []string{"--find", "Safari"}, ports.CommandResult{
		Stdout:   "Safari was found",
		ExitCode: 0,
	})

	step := NewDockStep("Safari", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestDockStep_Check_Add_NeedsApply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("dockutil", []string{"--find", "Safari"}, ports.CommandResult{
		Stderr:   "not found",
		ExitCode: 1,
	})

	step := NewDockStep("Safari", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestDockStep_Check_Remove_Satisfied(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("dockutil", []string{"--find", "FaceTime"}, ports.CommandResult{
		Stderr:   "not found",
		ExitCode: 1,
	})

	step := NewDockStep("FaceTime", false, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestDockStep_Plan_Add(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewDockStep("Safari", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Diff type = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
}

func TestDockStep_Plan_Remove(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewDockStep("FaceTime", false, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeRemove {
		t.Errorf("Diff type = %v, want %v", diff.Type(), compiler.DiffTypeRemove)
	}
}

func TestDockStep_Apply_Add(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("dockutil", []string{"--add", "Safari"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewDockStep("Safari", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestDockStep_Apply_Remove(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("dockutil", []string{"--remove", "FaceTime"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewDockStep("FaceTime", false, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestDockStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewDockStep("Safari", true, runner)

	explanation := step.Explain(compiler.NewExplainContext())
	if explanation.Summary() == "" {
		t.Error("Explain() summary should not be empty")
	}
}

// FinderStep tests

func TestFinderStep_ID(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewFinderStep("AppleShowAllFiles", true, runner)

	expected := "macos:finder:AppleShowAllFiles"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestFinderStep_Check_Satisfied(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "com.apple.finder", "AppleShowAllFiles"}, ports.CommandResult{
		Stdout:   "1\n",
		ExitCode: 0,
	})

	step := NewFinderStep("AppleShowAllFiles", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestFinderStep_Check_NeedsApply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "com.apple.finder", "AppleShowAllFiles"}, ports.CommandResult{
		Stdout:   "0\n",
		ExitCode: 0,
	})

	step := NewFinderStep("AppleShowAllFiles", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestFinderStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "com.apple.finder", "AppleShowAllFiles", "-bool", "true"}, ports.CommandResult{
		ExitCode: 0,
	})
	runner.AddResult("killall", []string{"Finder"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewFinderStep("AppleShowAllFiles", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestFinderStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewFinderStep("AppleShowAllFiles", true, runner)

	explanation := step.Explain(compiler.NewExplainContext())
	if explanation.Summary() == "" {
		t.Error("Explain() summary should not be empty")
	}
}

// KeyboardStep tests

func TestKeyboardStep_ID(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewKeyboardStep("KeyRepeat", 2, runner)

	expected := "macos:keyboard:KeyRepeat"
	if got := step.ID().String(); got != expected {
		t.Errorf("ID() = %q, want %q", got, expected)
	}
}

func TestKeyboardStep_Check_Satisfied(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "NSGlobalDomain", "KeyRepeat"}, ports.CommandResult{
		Stdout:   "2\n",
		ExitCode: 0,
	})

	step := NewKeyboardStep("KeyRepeat", 2, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestKeyboardStep_Check_NeedsApply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "NSGlobalDomain", "KeyRepeat"}, ports.CommandResult{
		Stdout:   "6\n",
		ExitCode: 0,
	})

	step := NewKeyboardStep("KeyRepeat", 2, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() status = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestKeyboardStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "NSGlobalDomain", "KeyRepeat", "-int", "2"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewKeyboardStep("KeyRepeat", 2, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestKeyboardStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewKeyboardStep("KeyRepeat", 2, runner)

	explanation := step.Explain(compiler.NewExplainContext())
	if explanation.Summary() == "" {
		t.Error("Explain() summary should not be empty")
	}
}
