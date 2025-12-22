package ports

import (
	"context"
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

func TestMockCommandRunner(t *testing.T) {
	runner := NewMockCommandRunner()
	runner.AddResult("brew", []string{"--version"}, CommandResult{
		ExitCode: 0,
		Stdout:   "Homebrew 4.2.0",
	})

	result, err := runner.Run(context.Background(), "brew", "--version")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Stdout != "Homebrew 4.2.0" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "Homebrew 4.2.0")
	}
}

func TestMockCommandRunner_NotFound(t *testing.T) {
	runner := NewMockCommandRunner()

	_, err := runner.Run(context.Background(), "unknown", "command")
	if err == nil {
		t.Error("Run() should return error for unregistered command")
	}
}

func TestMockCommandRunner_RecordsCalls(t *testing.T) {
	runner := NewMockCommandRunner()
	runner.AddResult("brew", []string{"install", "git"}, CommandResult{ExitCode: 0})
	runner.AddResult("brew", []string{"install", "curl"}, CommandResult{ExitCode: 0})

	_, _ = runner.Run(context.Background(), "brew", "install", "git")
	_, _ = runner.Run(context.Background(), "brew", "install", "curl")

	calls := runner.Calls()
	if len(calls) != 2 {
		t.Fatalf("Calls() len = %d, want 2", len(calls))
	}
	if calls[0].Command != "brew" {
		t.Errorf("calls[0].Command = %q, want %q", calls[0].Command, "brew")
	}
	if calls[0].Args[0] != "install" || calls[0].Args[1] != "git" {
		t.Errorf("calls[0].Args = %v, want [install git]", calls[0].Args)
	}
}

func TestNewRealCommandRunner(t *testing.T) {
	runner := NewRealCommandRunner()
	if runner == nil {
		t.Error("NewRealCommandRunner() should not return nil")
	}
}

func TestRealCommandRunner_Run_Success(t *testing.T) {
	runner := NewRealCommandRunner()

	result, err := runner.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Success() {
		t.Error("Run() should succeed for 'echo hello'")
	}
	if result.Stdout != "hello\n" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "hello\n")
	}
}

func TestRealCommandRunner_Run_Failure(t *testing.T) {
	runner := NewRealCommandRunner()

	result, err := runner.Run(context.Background(), "false")
	if err != nil {
		t.Fatalf("Run() error = %v (should return result with exit code, not error)", err)
	}
	if result.Success() {
		t.Error("Run() should fail for 'false' command")
	}
	if result.ExitCode == 0 {
		t.Error("ExitCode should be non-zero for 'false' command")
	}
}

func TestRealCommandRunner_Run_NotFound(t *testing.T) {
	runner := NewRealCommandRunner()

	_, err := runner.Run(context.Background(), "nonexistent-command-12345")
	if err == nil {
		t.Error("Run() should return error for non-existent command")
	}
}

func TestRealCommandRunner_Run_WithStderr(t *testing.T) {
	runner := NewRealCommandRunner()

	result, err := runner.Run(context.Background(), "sh", "-c", "echo error >&2; exit 1")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Success() {
		t.Error("Run() should fail")
	}
	if result.Stderr != "error\n" {
		t.Errorf("Stderr = %q, want %q", result.Stderr, "error\n")
	}
}
