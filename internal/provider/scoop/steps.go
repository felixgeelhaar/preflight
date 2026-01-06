package scoop

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

const scoopInstallStepID = "scoop:install"

// InstallStep ensures Scoop is installed.
type InstallStep struct {
	id       compiler.StepID
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewInstallStep creates a new InstallStep.
func NewInstallStep(runner ports.CommandRunner, plat *platform.Platform) *InstallStep {
	return &InstallStep{
		id:       compiler.MustNewStepID(scoopInstallStepID),
		runner:   runner,
		platform: plat,
	}
}

// ID returns the step identifier.
func (s *InstallStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *InstallStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if Scoop is installed.
func (s *InstallStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if _, err := exec.LookPath(s.scoopCommand()); err == nil {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *InstallStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "scoop", "install", "", "latest"), nil
}

// Apply installs Scoop using the official script.
func (s *InstallStep) Apply(ctx compiler.RunContext) error {
	cmd := s.powerShellCommand()
	script := "iwr -useb https://get.scoop.sh | iex"
	result, err := s.runner.Run(ctx.Context(), cmd, "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("scoop install failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *InstallStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Scoop",
		"Installs Scoop to enable package management on Windows.",
		[]string{"https://scoop.sh/"},
	)
}

func (s *InstallStep) scoopCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "scoop.cmd"
	}
	return "scoop"
}

func (s *InstallStep) powerShellCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "powershell.exe"
	}
	return "powershell"
}

// BucketStep represents a Scoop bucket addition step.
type BucketStep struct {
	bucket   Bucket
	id       compiler.StepID
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewBucketStep creates a new BucketStep.
func NewBucketStep(bucket Bucket, runner ports.CommandRunner, plat *platform.Platform) *BucketStep {
	id := compiler.MustNewStepID("scoop:bucket:" + bucket.Name)
	return &BucketStep{
		bucket:   bucket,
		id:       id,
		runner:   runner,
		platform: plat,
	}
}

// ID returns the step identifier.
func (s *BucketStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *BucketStep) DependsOn() []compiler.StepID {
	return []compiler.StepID{compiler.MustNewStepID(scoopInstallStepID)}
}

// scoopCommand returns the appropriate scoop command for the platform.
func (s *BucketStep) scoopCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "scoop.cmd"
	}
	return "scoop"
}

// Check determines if the bucket is already added.
func (s *BucketStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	cmd := s.scoopCommand()
	result, err := s.runner.Run(ctx.Context(), cmd, "bucket", "list")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("scoop bucket list failed: %s", result.Stderr)
	}

	// Parse bucket list output
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, line := range lines {
		// Bucket names appear in the first column
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == s.bucket.Name {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *BucketStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "bucket", s.bucket.Name, "", s.bucket.Name), nil
}

// Apply executes the bucket addition.
func (s *BucketStep) Apply(ctx compiler.RunContext) error {
	// Validate bucket name before execution
	if err := validation.ValidateScoopBucket(s.bucket.Name); err != nil {
		return fmt.Errorf("invalid bucket name: %w", err)
	}

	cmd := s.scoopCommand()
	args := []string{"bucket", "add", s.bucket.Name}

	// Add URL if specified
	if s.bucket.URL != "" {
		args = append(args, s.bucket.URL)
	}

	result, err := s.runner.Run(ctx.Context(), cmd, args...)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("scoop not found in PATH; install Scoop first")
		}
		return err
	}
	if !result.Success() {
		return fmt.Errorf("scoop bucket add %s failed: %s", s.bucket.Name, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *BucketStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Adds the %s bucket to Scoop, enabling installation of packages from this repository.", s.bucket.Name)
	if s.bucket.URL != "" {
		desc += fmt.Sprintf(" Using custom URL: %s", s.bucket.URL)
	}

	tradeoffs := []string{
		"+ Access to additional packages not in main bucket",
		"- Third-party buckets may have less stability",
		"- Requires trust in the bucket maintainer",
	}

	if s.platform != nil && s.platform.IsWSL() {
		tradeoffs = append(tradeoffs,
			"+ Adds Windows Scoop bucket from WSL",
			"- Runs as scoop.cmd (Windows interop required)",
		)
	}

	return compiler.NewExplanation(
		"Add Scoop Bucket",
		desc,
		[]string{
			"https://scoop.sh/",
			"https://github.com/ScoopInstaller/Scoop/wiki/Buckets",
		},
	).WithTradeoffs(tradeoffs)
}

// PackageStep represents a Scoop package installation step.
type PackageStep struct {
	pkg      Package
	id       compiler.StepID
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewPackageStep creates a new PackageStep.
func NewPackageStep(pkg Package, runner ports.CommandRunner, plat *platform.Platform) *PackageStep {
	id := compiler.MustNewStepID("scoop:package:" + pkg.FullName())
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
	deps := []compiler.StepID{compiler.MustNewStepID(scoopInstallStepID)}
	if s.pkg.Bucket != "" {
		bucketID := compiler.MustNewStepID("scoop:bucket:" + s.pkg.Bucket)
		deps = append(deps, bucketID)
	}
	return deps
}

// scoopCommand returns the appropriate scoop command for the platform.
func (s *PackageStep) scoopCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "scoop.cmd"
	}
	return "scoop"
}

// Check determines if the package is already installed.
func (s *PackageStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	cmd := s.scoopCommand()
	result, err := s.runner.Run(ctx.Context(), cmd, "list")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("scoop list failed: %s", result.Stderr)
	}

	// Parse installed package list
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == s.pkg.Name {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PackageStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	version := s.pkg.Version
	if version == "" {
		version = "latest"
	}
	return compiler.NewDiff(compiler.DiffTypeAdd, "package", s.pkg.FullName(), "", version), nil
}

// Apply executes the package installation.
func (s *PackageStep) Apply(ctx compiler.RunContext) error {
	// Validate package name before execution
	if err := validation.ValidatePackageName(s.pkg.Name); err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	cmd := s.scoopCommand()
	args := []string{"install", s.pkg.FullName()}

	result, err := s.runner.Run(ctx.Context(), cmd, args...)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("scoop not found in PATH; install Scoop first")
		}
		return err
	}
	if !result.Success() {
		return fmt.Errorf("scoop install %s failed: %s", s.pkg.FullName(), result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *PackageStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s package via Scoop.", s.pkg.Name)
	if s.pkg.Bucket != "" {
		desc += fmt.Sprintf(" From bucket: %s.", s.pkg.Bucket)
	}
	if s.pkg.Version != "" {
		desc += fmt.Sprintf(" Version: %s.", s.pkg.Version)
	}

	tradeoffs := []string{
		"+ Portable installation, no admin required",
		"+ Managed updates via 'scoop update'",
		"+ Clean uninstall without registry remnants",
	}

	if s.platform != nil && s.platform.IsWSL() {
		tradeoffs = append(tradeoffs,
			"+ Installs Windows applications accessible from WSL",
			"- Runs as scoop.cmd (Windows interop required)",
		)
	}

	return compiler.NewExplanation(
		"Install Scoop Package",
		desc,
		[]string{
			fmt.Sprintf("https://scoop.sh/#/%s", s.pkg.Name),
			"https://github.com/ScoopInstaller/Scoop",
		},
	).WithTradeoffs(tradeoffs)
}

// LockInfo returns lockfile information for this package.
func (s *PackageStep) LockInfo() (compiler.LockInfo, bool) {
	return compiler.LockInfo{
		Provider: "scoop",
		Name:     s.pkg.FullName(),
		Version:  s.pkg.Version,
	}, true
}

// InstalledVersion returns the installed Scoop package version if available.
func (s *PackageStep) InstalledVersion(ctx compiler.RunContext) (string, bool, error) {
	cmd := s.scoopCommand()
	result, err := s.runner.Run(ctx.Context(), cmd, "list", s.pkg.Name)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !result.Success() {
		return "", false, nil
	}
	for _, line := range strings.Split(result.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == s.pkg.Name {
			version := strings.TrimSpace(fields[1])
			if version != "" {
				return version, true, nil
			}
		}
	}
	return "", false, nil
}
