//go:build e2e

package framework

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// Result represents the result of running a command.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// Success returns true if the command exited with code 0.
func (r *Result) Success() bool {
	return r.ExitCode == 0 && r.Err == nil
}

// Contains checks if stdout contains the given substring.
func (r *Result) Contains(s string) bool {
	return strings.Contains(r.Stdout, s)
}

// StderrContains checks if stderr contains the given substring.
func (r *Result) StderrContains(s string) bool {
	return strings.Contains(r.Stderr, s)
}

// Runner executes preflight commands in a test environment.
type Runner struct {
	t   *testing.T
	env *Environment
}

// NewRunner creates a new command runner.
func NewRunner(t *testing.T, env *Environment) *Runner {
	return &Runner{
		t:   t,
		env: env,
	}
}

// Run executes the preflight command with the given arguments.
func (r *Runner) Run(args ...string) *Result {
	r.t.Helper()

	cmd := exec.Command(r.env.BinaryPath(), args...)
	cmd.Dir = r.env.ConfigDir()

	// Set environment variables
	cmd.Env = append(cmd.Env,
		"HOME="+r.env.HomeDir(),
		"PATH="+r.env.BinaryPath()+":"+r.env.RootDir(),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		result.Err = nil // Exit code is not an error
	} else if err != nil {
		result.ExitCode = -1
	}

	return result
}

// RunWithConfig executes a command with an explicit config path.
func (r *Runner) RunWithConfig(configPath string, args ...string) *Result {
	r.t.Helper()

	fullArgs := append([]string{"--config", configPath}, args...)
	return r.Run(fullArgs...)
}

// Version runs the version command.
func (r *Runner) Version() *Result {
	return r.Run("version")
}

// Plan runs the plan command.
func (r *Runner) Plan() *Result {
	configPath := r.env.ConfigDir() + "/preflight.yaml"
	return r.Run("plan", "--config", configPath)
}

// Apply runs the apply command with --yes flag.
func (r *Runner) Apply() *Result {
	configPath := r.env.ConfigDir() + "/preflight.yaml"
	return r.Run("apply", "--config", configPath, "--yes")
}

// ApplyDryRun runs the apply command with --dry-run flag.
func (r *Runner) ApplyDryRun() *Result {
	configPath := r.env.ConfigDir() + "/preflight.yaml"
	return r.Run("apply", "--config", configPath, "--dry-run")
}

// Diff runs the diff command.
func (r *Runner) Diff() *Result {
	configPath := r.env.ConfigDir() + "/preflight.yaml"
	return r.Run("diff", "--config", configPath)
}

// Scenario provides a fluent interface for writing BDD-style tests.
type Scenario struct {
	t      *testing.T
	env    *Environment
	runner *Runner
	result *Result
}

// NewScenario creates a new test scenario.
func NewScenario(t *testing.T) *Scenario {
	env := NewEnvironment(t)
	return &Scenario{
		t:      t,
		env:    env,
		runner: NewRunner(t, env),
	}
}

// Given sets up the test preconditions.
func (s *Scenario) Given(description string, setup func(*Environment)) *Scenario {
	s.t.Helper()
	s.t.Logf("Given %s", description)
	setup(s.env)
	return s
}

// When executes the action under test.
func (s *Scenario) When(description string, action func(*Runner) *Result) *Scenario {
	s.t.Helper()
	s.t.Logf("When %s", description)
	s.result = action(s.runner)
	return s
}

// Then asserts the expected outcome.
func (s *Scenario) Then(description string, assertion func(*testing.T, *Result)) *Scenario {
	s.t.Helper()
	s.t.Logf("Then %s", description)
	assertion(s.t, s.result)
	return s
}

// And is an alias for Then for chaining assertions.
func (s *Scenario) And(description string, assertion func(*testing.T, *Result)) *Scenario {
	return s.Then(description, assertion)
}

// Environment returns the test environment for direct access.
func (s *Scenario) Environment() *Environment {
	return s.env
}

// Result returns the last command result.
func (s *Scenario) Result() *Result {
	return s.result
}
