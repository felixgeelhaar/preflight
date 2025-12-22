// Package ports defines interfaces for external dependencies.
package ports

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// CommandResult represents the result of executing a shell command.
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Success returns true if the command exited with code 0.
func (r CommandResult) Success() bool {
	return r.ExitCode == 0
}

// CommandRunner executes shell commands.
type CommandRunner interface {
	Run(ctx context.Context, command string, args ...string) (CommandResult, error)
}

// RealCommandRunner executes actual shell commands.
type RealCommandRunner struct{}

// NewRealCommandRunner creates a new RealCommandRunner.
func NewRealCommandRunner() *RealCommandRunner {
	return &RealCommandRunner{}
}

// Run executes a command and returns the result.
func (r *RealCommandRunner) Run(ctx context.Context, command string, args ...string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, err
	}

	result.ExitCode = 0
	return result, nil
}

// MockCommandRunner is a test double for CommandRunner.
type MockCommandRunner struct {
	results map[string]CommandResult
	calls   []CommandCall
}

// CommandCall records a command invocation.
type CommandCall struct {
	Command string
	Args    []string
}

// NewMockCommandRunner creates a new MockCommandRunner.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		results: make(map[string]CommandResult),
		calls:   make([]CommandCall, 0),
	}
}

// AddResult registers an expected command and its result.
func (m *MockCommandRunner) AddResult(command string, args []string, result CommandResult) {
	key := buildKey(command, args)
	m.results[key] = result
}

// Run executes a mock command.
func (m *MockCommandRunner) Run(_ context.Context, command string, args ...string) (CommandResult, error) {
	m.calls = append(m.calls, CommandCall{
		Command: command,
		Args:    args,
	})

	key := buildKey(command, args)
	if result, ok := m.results[key]; ok {
		return result, nil
	}

	return CommandResult{}, fmt.Errorf("no mock result for command: %s %v", command, args)
}

// Calls returns all recorded command invocations.
func (m *MockCommandRunner) Calls() []CommandCall {
	return m.calls
}

// buildKey creates a unique key for a command and its arguments.
func buildKey(command string, args []string) string {
	return command + ":" + strings.Join(args, ":")
}
