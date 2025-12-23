package command

import (
	"context"
	"testing"
)

func TestNewRealRunner(t *testing.T) {
	runner := NewRealRunner()
	if runner == nil {
		t.Error("NewRealRunner() should not return nil")
	}
}

func TestRealRunner_Run_Success(t *testing.T) {
	runner := NewRealRunner()

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

func TestRealRunner_Run_Failure(t *testing.T) {
	runner := NewRealRunner()

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

func TestRealRunner_Run_NotFound(t *testing.T) {
	runner := NewRealRunner()

	_, err := runner.Run(context.Background(), "nonexistent-command-12345")
	if err == nil {
		t.Error("Run() should return error for non-existent command")
	}
}

func TestRealRunner_Run_WithStderr(t *testing.T) {
	runner := NewRealRunner()

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

func TestRealRunner_Run_ContextCancellation(t *testing.T) {
	runner := NewRealRunner()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := runner.Run(ctx, "sleep", "10")
	if err == nil {
		t.Error("Run() should return error for cancelled context")
	}
}
