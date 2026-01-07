package apt

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/commandutil"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

const (
	aptReadyStepID  = "apt:ready"
	aptUpdateStepID = "apt:update"
)

// ReadyStep ensures apt is available.
type ReadyStep struct {
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewReadyStep creates a new ReadyStep.
func NewReadyStep(runner ports.CommandRunner) *ReadyStep {
	return &ReadyStep{
		id:     compiler.MustNewStepID(aptReadyStepID),
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *ReadyStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ReadyStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if apt is available.
func (s *ReadyStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if _, err := exec.LookPath("apt-get"); err == nil {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ReadyStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "apt", "ready", "", "available"), nil
}

// Apply reports that apt needs to be installed by the OS.
func (s *ReadyStep) Apply(_ compiler.RunContext) error {
	return fmt.Errorf("apt-get not found in PATH; install apt or use a supported package manager")
}

// Explain provides a human-readable explanation.
func (s *ReadyStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Ensure APT Available",
		"Validates that apt is available before managing packages.",
		nil,
	)
}

// UpdateStep refreshes apt package metadata.
type UpdateStep struct {
	id     compiler.StepID
	runner ports.CommandRunner
	deps   []compiler.StepID
}

// NewUpdateStep creates a new UpdateStep.
func NewUpdateStep(runner ports.CommandRunner, deps []compiler.StepID) *UpdateStep {
	return &UpdateStep{
		id:     compiler.MustNewStepID(aptUpdateStepID),
		runner: runner,
		deps:   deps,
	}
}

// ID returns the step identifier.
func (s *UpdateStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *UpdateStep) DependsOn() []compiler.StepID {
	return s.deps
}

// Check determines if apt metadata is up to date.
func (s *UpdateStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if _, err := exec.LookPath("apt-get"); err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // apt-get not found means needs install
	}

	info, err := os.Stat("/var/lib/apt/periodic/update-success-stamp")
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // File missing means needs update
	}
	if time.Since(info.ModTime()) < 24*time.Hour {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *UpdateStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "apt", "update", "", "refresh index"), nil
}

// Apply refreshes apt metadata.
func (s *UpdateStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "sudo", "apt-get", "update")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("apt-get not found in PATH; install apt or use a supported package manager")
		}
		return err
	}
	if !result.Success() {
		return fmt.Errorf("apt-get update failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *UpdateStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Refresh APT Index",
		"Updates the apt package list before installing packages.",
		nil,
	)
}

// PPAStep represents an apt PPA addition step.
type PPAStep struct {
	ppa    string
	id     compiler.StepID
	runner ports.CommandRunner
}

// ppaStepID returns the step ID string for a PPA.
func ppaStepID(ppa string) string {
	sanitizedPPA := strings.ReplaceAll(ppa, ":", "-")
	return "apt:ppa:" + sanitizedPPA
}

// NewPPAStep creates a new PPAStep.
func NewPPAStep(ppa string, runner ports.CommandRunner) *PPAStep {
	// Sanitize PPA name for step ID (replace colon with dash)
	id := compiler.MustNewStepID(ppaStepID(ppa))
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
	return []compiler.StepID{compiler.MustNewStepID(aptReadyStepID)}
}

// Check determines if the PPA is already added.
func (s *PPAStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	// Check if PPA source list file exists
	// PPAs are stored in /etc/apt/sources.list.d/
	// For now, we'll use apt-cache policy to check
	result, err := s.runner.Run(ctx.Context(), "apt-cache", "policy")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return compiler.StatusNeedsApply, nil
		}
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
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("add-apt-repository not found; install software-properties-common")
		}
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
	return []compiler.StepID{compiler.MustNewStepID(aptUpdateStepID)}
}

// Check determines if the package is already installed.
func (s *PackageStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "dpkg-query", "-W", "-f=${Package}\t${Version}\t${db:Status-Status}\n", s.pkg.Name)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return compiler.StatusNeedsApply, nil
		}
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
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("apt-get not found in PATH; install apt or use a supported package manager")
		}
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

// LockInfo returns lockfile information for this package.
func (s *PackageStep) LockInfo() (compiler.LockInfo, bool) {
	return compiler.LockInfo{
		Provider: "apt",
		Name:     s.pkg.Name,
		Version:  s.pkg.Version,
	}, true
}

// InstalledVersion returns the installed apt package version if available.
func (s *PackageStep) InstalledVersion(ctx compiler.RunContext) (string, bool, error) {
	result, err := s.runner.Run(ctx.Context(), "dpkg-query", "-W", "-f=${Version}\n", s.pkg.Name)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !result.Success() {
		return "", false, nil
	}
	version := strings.TrimSpace(result.Stdout)
	if version == "" {
		return "", false, nil
	}
	return version, true, nil
}
