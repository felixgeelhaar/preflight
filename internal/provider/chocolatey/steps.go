package chocolatey

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// SourceStep represents a Chocolatey source configuration step.
type SourceStep struct {
	source   Source
	id       compiler.StepID
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewSourceStep creates a new SourceStep.
func NewSourceStep(source Source, runner ports.CommandRunner, plat *platform.Platform) *SourceStep {
	id := compiler.MustNewStepID("chocolatey:source:" + source.Name)
	return &SourceStep{
		source:   source,
		id:       id,
		runner:   runner,
		platform: plat,
	}
}

// ID returns the step identifier.
func (s *SourceStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *SourceStep) DependsOn() []compiler.StepID {
	return nil
}

// chocoCommand returns the appropriate choco command for the platform.
func (s *SourceStep) chocoCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "choco.exe"
	}
	return "choco"
}

// Check determines if the source is already configured.
func (s *SourceStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	cmd := s.chocoCommand()
	result, err := s.runner.Run(ctx.Context(), cmd, "source", "list")
	if err != nil {
		return compiler.StatusUnknown, err
	}

	if result.Success() {
		// Parse source list output to find our source
		lines := strings.Split(result.Stdout, "\n")
		for _, line := range lines {
			// Format: "name - url | Priority: N | Disabled"
			if strings.HasPrefix(strings.TrimSpace(line), s.source.Name+" ") {
				// Check if URL matches
				if strings.Contains(line, s.source.URL) {
					return compiler.StatusSatisfied, nil
				}
			}
		}
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *SourceStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "source", s.source.Name, "", s.source.URL), nil
}

// Apply executes the source configuration.
func (s *SourceStep) Apply(ctx compiler.RunContext) error {
	// Validate source name
	if err := validation.ValidateChocoSource(s.source.Name); err != nil {
		return fmt.Errorf("invalid source name: %w", err)
	}

	// Validate URL
	if err := validation.ValidateURL(s.source.URL); err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	}

	cmd := s.chocoCommand()
	args := []string{"source", "add", "-n=" + s.source.Name, "-s=" + s.source.URL}

	if s.source.Priority > 0 {
		args = append(args, fmt.Sprintf("--priority=%d", s.source.Priority))
	}

	result, err := s.runner.Run(ctx.Context(), cmd, args...)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("choco source add %s failed: %s", s.source.Name, result.Stderr)
	}

	// Disable if requested
	if s.source.Disabled {
		result, err = s.runner.Run(ctx.Context(), cmd, "source", "disable", "-n="+s.source.Name)
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("choco source disable %s failed: %s", s.source.Name, result.Stderr)
		}
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *SourceStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Configures the '%s' Chocolatey source at %s.", s.source.Name, s.source.URL)
	if s.source.Priority > 0 {
		desc += fmt.Sprintf(" Priority: %d.", s.source.Priority)
	}
	if s.source.Disabled {
		desc += " (Disabled)"
	}

	return compiler.NewExplanation(
		"Configure Chocolatey Source",
		desc,
		[]string{
			"https://docs.chocolatey.org/en-us/choco/commands/source",
		},
	).WithTradeoffs([]string{
		"+ Custom package sources for internal/private packages",
		"+ Control over package origins",
		"- Requires source URL to be accessible",
	})
}

// PackageStep represents a Chocolatey package installation step.
type PackageStep struct {
	pkg      Package
	id       compiler.StepID
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewPackageStep creates a new PackageStep.
func NewPackageStep(pkg Package, runner ports.CommandRunner, plat *platform.Platform) *PackageStep {
	id := compiler.MustNewStepID("chocolatey:package:" + pkg.Name)
	return &PackageStep{
		pkg:      pkg,
		id:       id,
		runner:   runner,
		platform: plat,
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

// chocoCommand returns the appropriate choco command for the platform.
func (s *PackageStep) chocoCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "choco.exe"
	}
	return "choco"
}

// Check determines if the package is already installed.
func (s *PackageStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	cmd := s.chocoCommand()
	result, err := s.runner.Run(ctx.Context(), cmd, "list", "--local-only", "--exact", s.pkg.Name)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	// choco list --local-only returns the package if installed
	if result.Success() && strings.Contains(strings.ToLower(result.Stdout), strings.ToLower(s.pkg.Name)) {
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
	return compiler.NewDiff(compiler.DiffTypeAdd, "package", s.pkg.Name, "", version), nil
}

// Apply executes the package installation.
func (s *PackageStep) Apply(ctx compiler.RunContext) error {
	// Validate package name
	if err := validation.ValidateChocoPackage(s.pkg.Name); err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	// Validate source if specified
	if s.pkg.Source != "" {
		if err := validation.ValidateChocoSource(s.pkg.Source); err != nil {
			return fmt.Errorf("invalid source: %w", err)
		}
	}

	cmd := s.chocoCommand()
	args := []string{"install", s.pkg.Name, "-y", "--no-progress"}

	if s.pkg.Version != "" {
		args = append(args, "--version="+s.pkg.Version)
	}

	if s.pkg.Source != "" {
		args = append(args, "--source="+s.pkg.Source)
	}

	if s.pkg.Args != "" {
		args = append(args, "--package-parameters="+s.pkg.Args)
	}

	result, err := s.runner.Run(ctx.Context(), cmd, args...)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("choco install %s failed: %s", s.pkg.Name, result.Stderr)
	}

	// Pin package if requested
	if s.pkg.Pin {
		result, err = s.runner.Run(ctx.Context(), cmd, "pin", "add", "-n="+s.pkg.Name)
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("choco pin %s failed: %s", s.pkg.Name, result.Stderr)
		}
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *PackageStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s package via Chocolatey.", s.pkg.Name)
	if s.pkg.Version != "" {
		desc += fmt.Sprintf(" Version: %s.", s.pkg.Version)
	}
	if s.pkg.Source != "" {
		desc += fmt.Sprintf(" Source: %s.", s.pkg.Source)
	}
	if s.pkg.Pin {
		desc += " (Pinned to prevent upgrades)"
	}

	tradeoffs := []string{
		"+ Popular Windows package manager",
		"+ Large package repository",
		"+ Silent installation",
		"+ Version pinning available",
	}

	if s.platform != nil && s.platform.IsWSL() {
		tradeoffs = append(tradeoffs,
			"+ Installs Windows applications accessible from WSL",
			"- Runs as choco.exe (Windows interop required)",
		)
	}

	return compiler.NewExplanation(
		"Install Chocolatey Package",
		desc,
		[]string{
			fmt.Sprintf("https://community.chocolatey.org/packages/%s", s.pkg.Name),
			"https://docs.chocolatey.org/en-us/choco/commands/install",
		},
	).WithTradeoffs(tradeoffs)
}

// Ensure steps implement compiler.Step.
var (
	_ compiler.Step = (*SourceStep)(nil)
	_ compiler.Step = (*PackageStep)(nil)
)
