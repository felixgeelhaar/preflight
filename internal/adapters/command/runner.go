// Package command provides command execution adapters.
package command

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// RealRunner executes actual shell commands.
type RealRunner struct{}

// NewRealRunner creates a new RealRunner.
func NewRealRunner() *RealRunner {
	return &RealRunner{}
}

// Run executes a command and returns the result.
func (r *RealRunner) Run(ctx context.Context, command string, args ...string) (ports.CommandResult, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := ports.CommandResult{
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, err
	}

	return result, nil
}

// Ensure RealRunner implements ports.CommandRunner.
var _ ports.CommandRunner = (*RealRunner)(nil)
