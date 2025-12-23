// Package ports defines interfaces for external dependencies.
package ports

import (
	"context"
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

// CommandCall records a command invocation.
type CommandCall struct {
	Command string
	Args    []string
}

// CommandRunner executes shell commands.
type CommandRunner interface {
	Run(ctx context.Context, command string, args ...string) (CommandResult, error)
}
