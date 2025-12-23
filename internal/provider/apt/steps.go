package apt

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// PPAStep represents an apt PPA addition step.
type PPAStep struct {
	ppa    string
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewPPAStep creates a new PPAStep.
func NewPPAStep(ppa string, runner ports.CommandRunner) *PPAStep {
	// Sanitize PPA name for step ID (replace colon with dash)
	sanitizedPPA := strings.ReplaceAll(ppa, ":", "-")
	id := compiler.MustNewStepID("apt:ppa:" + sanitizedPPA)
	return &PPAStep{
		ppa:    ppa,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *PPAStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *PPAStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the PPA is already added.
func (s *PPAStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	// Check if PPA source list file exists
	// PPAs are stored in /etc/apt/sources.list.d/
	// For now, we'll use apt-cache policy to check
	result, err := s.runner.Run(ctx.Context(), "apt-cache", "policy")
	if err != nil {
		return compiler.StatusUnknown, err
	}

	// Check if the PPA URL is in the policy output
	ppaURL := strings.TrimPrefix(s.ppa, "ppa:")
	if strings.Contains(result.Stdout, ppaURL) {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PPAStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "ppa", s.ppa, "", s.ppa), nil
}

// Apply executes the PPA addition.
func (s *PPAStep) Apply(ctx compiler.RunContext) error {
	// Validate PPA name before execution to prevent command injection
	if err := validation.ValidatePPA(s.ppa); err != nil {
		return fmt.Errorf("invalid PPA: %w", err)
	}

	result, err := s.runner.Run(ctx.Context(), "sudo", "add-apt-repository", "-y", s.ppa)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("add-apt-repository %s failed: %s", s.ppa, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *PPAStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Add APT PPA",
		fmt.Sprintf("Adds the %s Personal Package Archive to apt sources, enabling installation of packages from this repository.", s.ppa),
		nil,
	)
}

// PackageStep represents an apt package installation step.
type PackageStep struct {
	pkg    Package
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewPackageStep creates a new PackageStep.
func NewPackageStep(pkg Package, runner ports.CommandRunner) *PackageStep {
	id := compiler.MustNewStepID("apt:package:" + pkg.Name)
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
	result, err := s.runner.Run(ctx.Context(), "dpkg-query", "-W", "-f=${Package}\t${Version}\t${db:Status-Status}\n", s.pkg.Name)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	// dpkg-query returns exit code 1 if package not found
	if !result.Success() {
		return compiler.StatusNeedsApply, nil
	}

	// Check if package is installed
	if strings.Contains(result.Stdout, "installed") {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PackageStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	version := "latest"
	if s.pkg.Version != "" {
		version = s.pkg.Version
	}
	return compiler.NewDiff(compiler.DiffTypeAdd, "package", s.pkg.Name, "", version), nil
}

// Apply executes the package installation.
func (s *PackageStep) Apply(ctx compiler.RunContext) error {
	// Validate package name before execution to prevent command injection
	if err := validation.ValidatePackageName(s.pkg.Name); err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	pkgSpec := s.pkg.Name
	if s.pkg.Version != "" && s.pkg.Version != "latest" {
		// Also validate version string
		if err := validation.ValidatePackageName(s.pkg.Version); err != nil {
			return fmt.Errorf("invalid package version: %w", err)
		}
		pkgSpec = s.pkg.FullName()
	}

	result, err := s.runner.Run(ctx.Context(), "sudo", "apt-get", "install", "-y", pkgSpec)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("apt-get install %s failed: %s", pkgSpec, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *PackageStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s package via apt.", s.pkg.Name)
	if s.pkg.Version != "" && s.pkg.Version != "latest" {
		desc += fmt.Sprintf(" Version: %s", s.pkg.Version)
	}
	return compiler.NewExplanation(
		"Install APT Package",
		desc,
		nil,
	)
}
