package gotools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// ToolStep represents a Go tool installation step.
type ToolStep struct {
	tool   Tool
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewToolStep creates a new ToolStep.
func NewToolStep(tool Tool, runner ports.CommandRunner) *ToolStep {
	id := compiler.MustNewStepID("go:tool:" + tool.BinaryName())
	return &ToolStep{
		tool:   tool,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *ToolStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ToolStep) DependsOn() []compiler.StepID {
	return nil
}

// getGoBin returns the Go bin directory.
func getGoBin() string {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return gobin
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	return filepath.Join(gopath, "bin")
}

// Check determines if the tool is already installed.
func (s *ToolStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	// Check if binary exists in GOBIN
	binaryPath := filepath.Join(getGoBin(), s.tool.BinaryName())
	if _, err := os.Stat(binaryPath); err == nil {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ToolStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	version := s.tool.Version
	if version == "" {
		version = "latest"
	}
	return compiler.NewDiff(compiler.DiffTypeAdd, "go-tool", s.tool.BinaryName(), "", version), nil
}

// Apply executes the tool installation.
func (s *ToolStep) Apply(ctx compiler.RunContext) error {
	// Validate tool path before execution to prevent command injection
	if err := validation.ValidateGoTool(s.tool.FullName()); err != nil {
		return fmt.Errorf("invalid Go tool: %w", err)
	}

	result, err := s.runner.Run(ctx.Context(), "go", "install", s.tool.FullName())
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("go install %s failed: %s", s.tool.FullName(), result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *ToolStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s tool via 'go install'.", s.tool.BinaryName())
	if s.tool.Version != "" {
		desc += fmt.Sprintf(" Version: %s", s.tool.Version)
	}
	return compiler.NewExplanation(
		"Install Go Tool",
		desc,
		[]string{
			fmt.Sprintf("https://pkg.go.dev/%s", s.tool.Module),
			"https://go.dev/doc/go-get-install-deprecation",
		},
	).WithTradeoffs([]string{
		"+ Installs to $GOBIN for easy access",
		"+ Version pinning with @version syntax",
		"+ No dependency on external package managers",
		"- Requires Go to be installed",
		"- Each tool is compiled from source",
	})
}
