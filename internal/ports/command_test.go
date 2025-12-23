package ports

import (
	"testing"
)

func TestCommandResult_Success(t *testing.T) {
	result := CommandResult{
		ExitCode: 0,
		Stdout:   "output",
		Stderr:   "",
	}

	if !result.Success() {
		t.Error("Success() should be true for exit code 0")
	}
}

func TestCommandResult_Failure(t *testing.T) {
	result := CommandResult{
		ExitCode: 1,
		Stdout:   "",
		Stderr:   "error",
	}

	if result.Success() {
		t.Error("Success() should be false for non-zero exit code")
	}
}

func TestCommandCall(t *testing.T) {
	call := CommandCall{
		Command: "brew",
		Args:    []string{"install", "git"},
	}

	if call.Command != "brew" {
		t.Errorf("Command = %q, want %q", call.Command, "brew")
	}
	if len(call.Args) != 2 {
		t.Errorf("Args len = %d, want 2", len(call.Args))
	}
}
