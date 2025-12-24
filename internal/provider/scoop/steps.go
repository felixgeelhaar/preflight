package scoop

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

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
	return nil
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
	if s.pkg.Bucket != "" {
		bucketID := compiler.MustNewStepID("scoop:bucket:" + s.pkg.Bucket)
		return []compiler.StepID{bucketID}
	}
	return nil
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
