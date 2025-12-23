package mocks

import (
	"context"
	"sync"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

func TestCommandRunner_AddResult(t *testing.T) {
	runner := NewCommandRunner()
	runner.AddResult("brew", []string{"--version"}, ports.CommandResult{
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

func TestCommandRunner_NotFound(t *testing.T) {
	runner := NewCommandRunner()

	_, err := runner.Run(context.Background(), "unknown", "command")
	if err == nil {
		t.Error("Run() should return error for unregistered command")
	}
}

func TestCommandRunner_RecordsCalls(t *testing.T) {
	runner := NewCommandRunner()
	runner.AddResult("brew", []string{"install", "git"}, ports.CommandResult{ExitCode: 0})
	runner.AddResult("brew", []string{"install", "curl"}, ports.CommandResult{ExitCode: 0})

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

func TestCommandRunner_Reset(t *testing.T) {
	runner := NewCommandRunner()
	runner.AddResult("brew", []string{"--version"}, ports.CommandResult{ExitCode: 0})
	_, _ = runner.Run(context.Background(), "brew", "--version")

	runner.Reset()

	calls := runner.Calls()
	if len(calls) != 0 {
		t.Error("Reset() should clear all calls")
	}

	_, err := runner.Run(context.Background(), "brew", "--version")
	if err == nil {
		t.Error("Reset() should clear all results")
	}
}

func TestCommandRunner_ThreadSafety(t *testing.T) {
	runner := NewCommandRunner()

	// Add some results
	for i := 0; i < 100; i++ {
		runner.AddResult("cmd", []string{string(rune('a' + i%26))}, ports.CommandResult{ExitCode: 0})
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = runner.Run(context.Background(), "cmd", string(rune('a'+idx%26)))
			_ = runner.Calls()
		}(i)
	}

	wg.Wait()

	// Should not panic or have data races
	calls := runner.Calls()
	if len(calls) != 100 {
		t.Errorf("Expected 100 calls, got %d", len(calls))
	}
}
