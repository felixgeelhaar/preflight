package winget

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/commandutil"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

const wingetReadyStepID = "winget:ready"

// ReadyStep ensures winget is available.
type ReadyStep struct {
	id       compiler.StepID
	platform *platform.Platform
}

// NewReadyStep creates a new ReadyStep.
func NewReadyStep(plat *platform.Platform) *ReadyStep {
	return &ReadyStep{
		id:       compiler.MustNewStepID(wingetReadyStepID),
		platform: plat,
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

// Check determines if winget is available.
func (s *ReadyStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if _, err := exec.LookPath(s.wingetCommand()); err == nil {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ReadyStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "winget", "ready", "", "available"), nil
}

// Apply reports that winget needs to be installed by the OS.
func (s *ReadyStep) Apply(_ compiler.RunContext) error {
	return fmt.Errorf("winget not found in PATH; install App Installer from Microsoft Store")
}

// Explain provides a human-readable explanation.
func (s *ReadyStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Ensure winget Available",
		"Validates that winget is available before installing packages.",
		[]string{"https://learn.microsoft.com/en-us/windows/package-manager/winget/"},
	)
}

func (s *ReadyStep) wingetCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "winget.exe"
	}
	return "winget"
}

// PackageStep represents a winget package installation step.
type PackageStep struct {
	pkg      Package
	id       compiler.StepID
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewPackageStep creates a new PackageStep.
func NewPackageStep(pkg Package, runner ports.CommandRunner, plat *platform.Platform) *PackageStep {
	id := compiler.MustNewStepID("winget:package:" + pkg.ID)
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
	return []compiler.StepID{compiler.MustNewStepID(wingetReadyStepID)}
}

// wingetCommand returns the appropriate winget command for the platform.
// In WSL, winget can be accessed via winget.exe.
func (s *PackageStep) wingetCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "winget.exe"
	}
	return "winget"
}

// Check determines if the package is already installed.
func (s *PackageStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	cmd := s.wingetCommand()
	result, err := s.runner.Run(ctx.Context(), cmd, "list", "--id", s.pkg.ID, "--exact", "--accept-source-agreements")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	// winget list returns exit code 0 if package is found
	if result.Success() && strings.Contains(result.Stdout, s.pkg.ID) {
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
	return compiler.NewDiff(compiler.DiffTypeAdd, "package", s.pkg.ID, "", version), nil
}

// Apply executes the package installation.
func (s *PackageStep) Apply(ctx compiler.RunContext) error {
	// Validate package ID before execution to prevent command injection
	if err := validation.ValidateWingetID(s.pkg.ID); err != nil {
		return fmt.Errorf("invalid package ID: %w", err)
	}

	// Validate source if specified
	if s.pkg.Source != "" {
		if err := validation.ValidateWingetSource(s.pkg.Source); err != nil {
			return fmt.Errorf("invalid source: %w", err)
		}
	}

	cmd := s.wingetCommand()
	args := []string{"install", "--id", s.pkg.ID, "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent"}

	if s.pkg.Version != "" {
		args = append(args, "--version", s.pkg.Version)
	}

	if s.pkg.Source != "" {
		args = append(args, "--source", s.pkg.Source)
	}

	result, err := s.runner.Run(ctx.Context(), cmd, args...)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("winget not found in PATH; install App Installer from Microsoft Store")
		}
		return err
	}
	if !result.Success() {
		return fmt.Errorf("winget install %s failed: %s", s.pkg.ID, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *PackageStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s package via Windows Package Manager (winget).", s.pkg.ID)
	if s.pkg.Version != "" {
		desc += fmt.Sprintf(" Version: %s.", s.pkg.Version)
	}
	if s.pkg.Source != "" {
		desc += fmt.Sprintf(" Source: %s.", s.pkg.Source)
	}

	tradeoffs := []string{
		"+ Native Windows package management",
		"+ Automatic updates via winget upgrade",
		"+ Silent installation without user interaction",
	}

	if s.platform != nil && s.platform.IsWSL() {
		tradeoffs = append(tradeoffs,
			"+ Installs Windows applications accessible from WSL",
			"- Runs as winget.exe (Windows interop required)",
		)
	}

	return compiler.NewExplanation(
		"Install Windows Package",
		desc,
		[]string{
			fmt.Sprintf("https://winget.run/pkg/%s", strings.ReplaceAll(s.pkg.ID, ".", "/")),
			"https://learn.microsoft.com/en-us/windows/package-manager/winget/",
		},
	).WithTradeoffs(tradeoffs)
}

// LockInfo returns lockfile information for this package.
func (s *PackageStep) LockInfo() (compiler.LockInfo, bool) {
	return compiler.LockInfo{
		Provider: "winget",
		Name:     s.pkg.ID,
		Version:  s.pkg.Version,
	}, true
}

// InstalledVersion returns the installed winget package version if available.
func (s *PackageStep) InstalledVersion(ctx compiler.RunContext) (string, bool, error) {
	cmd := s.wingetCommand()
	result, err := s.runner.Run(ctx.Context(), cmd, "list", "--id", s.pkg.ID, "--exact", "--accept-source-agreements")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !result.Success() {
		return "", false, nil
	}

	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		for i := 0; i < len(fields); i++ {
			if fields[i] == s.pkg.ID {
				if i+1 < len(fields) {
					version := strings.TrimSpace(fields[i+1])
					if version != "" {
						return version, true, nil
					}
				}
				return "", false, nil
			}
		}
	}

	return "", false, nil
}
