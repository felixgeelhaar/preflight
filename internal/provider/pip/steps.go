package pip

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// PackageStep represents a pip package installation step.
type PackageStep struct {
	pkg    Package
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewPackageStep creates a new PackageStep.
func NewPackageStep(pkg Package, runner ports.CommandRunner) *PackageStep {
	id := compiler.MustNewStepID("pip:package:" + pkg.Name)
	return &PackageStep{
		pkg:    pkg,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *PackageStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *PackageStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the package is already installed.
func (s *PackageStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	// Try pip show to check if package is installed
	result, err := s.runner.Run(ctx.Context(), "pip", "show", s.pkg.Name)
	if err != nil {
		// Try pip3 as fallback
		result, err = s.runner.Run(ctx.Context(), "pip3", "show", s.pkg.Name)
		if err != nil {
			return compiler.StatusUnknown, err
		}
	}

	if result.Success() && strings.Contains(result.Stdout, "Name:") {
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PackageStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	version := s.pkg.Version
	if version == "" {
		version = "latest"
	}
	return compiler.NewDiff(compiler.DiffTypeAdd, "pip-package", s.pkg.Name, "", version), nil
}

// Apply executes the package installation.
func (s *PackageStep) Apply(ctx compiler.RunContext) error {
	// Validate package name before execution to prevent command injection
	if err := validation.ValidatePipPackage(s.pkg.FullName()); err != nil {
		return fmt.Errorf("invalid pip package: %w", err)
	}

	// Install to user directory with --user flag
	result, err := s.runner.Run(ctx.Context(), "pip", "install", "--user", s.pkg.FullName())
	if err != nil {
		// Try pip3 as fallback
		result, err = s.runner.Run(ctx.Context(), "pip3", "install", "--user", s.pkg.FullName())
		if err != nil {
			return err
		}
	}
	if !result.Success() {
		return fmt.Errorf("pip install --user %s failed: %s", s.pkg.FullName(), result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *PackageStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s package via pip to user directory.", s.pkg.Name)
	if s.pkg.Version != "" {
		desc += fmt.Sprintf(" Version: %s", s.pkg.Version)
	}
	return compiler.NewExplanation(
		"Install pip Package",
		desc,
		[]string{
			fmt.Sprintf("https://pypi.org/project/%s/", s.pkg.Name),
			"https://pip.pypa.io/en/stable/cli/pip_install/",
		},
	).WithTradeoffs([]string{
		"+ Installs to user directory (no sudo required)",
		"+ Version pinning with specifiers (==, >=, etc.)",
		"+ Access to Python ecosystem tools",
		"- Requires Python to be installed",
		"- May conflict with system Python packages",
	})
}
