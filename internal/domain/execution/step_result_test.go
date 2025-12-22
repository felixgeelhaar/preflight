package execution

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

func TestStepResult_Success(t *testing.T) {
	stepID, _ := compiler.NewStepID("brew:install:git")
	result := NewStepResult(stepID, compiler.StatusSatisfied, nil)

	if !result.StepID().Equals(stepID) {
		t.Errorf("StepID() = %v, want %v", result.StepID(), stepID)
	}
	if result.Status() != compiler.StatusSatisfied {
		t.Errorf("Status() = %v, want %v", result.Status(), compiler.StatusSatisfied)
	}
	if result.Error() != nil {
		t.Errorf("Error() = %v, want nil", result.Error())
	}
	if !result.Success() {
		t.Error("Success() should be true for satisfied status")
	}
}

func TestStepResult_Failure(t *testing.T) {
	stepID, _ := compiler.NewStepID("brew:install:git")
	result := NewStepResult(stepID, compiler.StatusFailed, errInstallFailed)

	if result.Status() != compiler.StatusFailed {
		t.Errorf("Status() = %v, want %v", result.Status(), compiler.StatusFailed)
	}
	if result.Error() == nil {
		t.Error("Error() should not be nil for failed status")
	}
	if result.Success() {
		t.Error("Success() should be false for failed status")
	}
}

// errInstallFailed is a test error for failure cases.
var errInstallFailed = compiler.ErrEmptyStepID // reuse an existing error for simplicity

func TestStepResult_WithDuration(t *testing.T) {
	stepID, _ := compiler.NewStepID("brew:install:git")
	result := NewStepResult(stepID, compiler.StatusSatisfied, nil).
		WithDuration(500 * time.Millisecond)

	if result.Duration() != 500*time.Millisecond {
		t.Errorf("Duration() = %v, want %v", result.Duration(), 500*time.Millisecond)
	}
}

func TestStepResult_WithDiff(t *testing.T) {
	stepID, _ := compiler.NewStepID("brew:install:git")
	diff := compiler.NewDiff(compiler.DiffTypeAdd, "package", "git", "", "2.43.0")
	result := NewStepResult(stepID, compiler.StatusSatisfied, nil).
		WithDiff(diff)

	if result.Diff().Type() != compiler.DiffTypeAdd {
		t.Errorf("Diff().Type() = %v, want %v", result.Diff().Type(), compiler.DiffTypeAdd)
	}
}

func TestStepResult_Skipped(t *testing.T) {
	stepID, _ := compiler.NewStepID("nvim:install:plugin")
	result := NewStepResult(stepID, compiler.StatusSkipped, nil)

	if result.Status() != compiler.StatusSkipped {
		t.Errorf("Status() = %v, want %v", result.Status(), compiler.StatusSkipped)
	}
	if !result.Skipped() {
		t.Error("Skipped() should be true")
	}
}
