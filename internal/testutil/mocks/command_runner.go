// Package mocks provides test doubles for testing.
package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// CommandRunner is a thread-safe test double for ports.CommandRunner.
type CommandRunner struct {
	mu      sync.RWMutex
	results map[string]ports.CommandResult
	errors  map[string]error
	calls   []ports.CommandCall
}

// NewCommandRunner creates a new CommandRunner mock.
func NewCommandRunner() *CommandRunner {
	return &CommandRunner{
		results: make(map[string]ports.CommandResult),
		errors:  make(map[string]error),
		calls:   make([]ports.CommandCall, 0),
	}
}

// AddResult registers an expected command and its result.
func (m *CommandRunner) AddResult(command string, args []string, result ports.CommandResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := buildKey(command, args)
	m.results[key] = result
}

// AddError registers an expected command that should return an error.
func (m *CommandRunner) AddError(command string, args []string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := buildKey(command, args)
	m.errors[key] = err
}

// Run executes a mock command.
func (m *CommandRunner) Run(_ context.Context, command string, args ...string) (ports.CommandResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, ports.CommandCall{
		Command: command,
		Args:    args,
	})
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := buildKey(command, args)

	// Check for registered error first
	if err, ok := m.errors[key]; ok {
		return ports.CommandResult{}, err
	}

	if result, ok := m.results[key]; ok {
		return result, nil
	}

	return ports.CommandResult{}, fmt.Errorf("no mock result for command: %s %v", command, args)
}

// Calls returns all recorded command invocations.
func (m *CommandRunner) Calls() []ports.CommandCall {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent data races
	calls := make([]ports.CommandCall, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// Reset clears all registered results, errors, and recorded calls.
func (m *CommandRunner) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results = make(map[string]ports.CommandResult)
	m.errors = make(map[string]error)
	m.calls = make([]ports.CommandCall, 0)
}

// buildKey creates a unique key for a command and its arguments.
func buildKey(command string, args []string) string {
	return command + ":" + strings.Join(args, ":")
}

// Ensure CommandRunner implements ports.CommandRunner.
var _ ports.CommandRunner = (*CommandRunner)(nil)
