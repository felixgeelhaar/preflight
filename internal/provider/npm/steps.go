package npm

import (
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// PackageStep represents an npm global package installation step.
type PackageStep struct {
	pkg    Package
	id     compiler.StepID
	runner ports.CommandRunner
}

// sanitizeStepID converts a package name to a valid step ID component.
// Scoped packages like @org/pkg become org/pkg for the step ID.
func sanitizeStepID(name string) string {
	if len(name) > 0 && name[0] == '@' {
		return name[1:] // Strip leading @
	}
	return name
}

// NewPackageStep creates a new PackageStep.
func NewPackageStep(pkg Package, runner ports.CommandRunner) *PackageStep {
	id := compiler.MustNewStepID("npm:package:" + sanitizeStepID(pkg.Name))
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

// Check determines if the package is already installed globally.
func (s *PackageStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "npm", "list", "-g", "--depth=0", "--json")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	// npm list returns exit code 1 when no packages match, but still outputs JSON
	// so we check the output regardless of exit code

	var npmList struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &npmList); err != nil {
		return compiler.StatusUnknown, fmt.Errorf("failed to parse npm list output: %w", err)
	}

	if _, found := npmList.Dependencies[s.pkg.Name]; found {
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
	return compiler.NewDiff(compiler.DiffTypeAdd, "npm-package", s.pkg.Name, "", version), nil
}

// Apply executes the package installation.
func (s *PackageStep) Apply(ctx compiler.RunContext) error {
	// Validate package name before execution to prevent command injection
	if err := validation.ValidateNpmPackage(s.pkg.FullName()); err != nil {
		return fmt.Errorf("invalid npm package: %w", err)
	}

	result, err := s.runner.Run(ctx.Context(), "npm", "install", "-g", s.pkg.FullName())
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("npm install -g %s failed: %s", s.pkg.FullName(), result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *PackageStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s package globally via npm.", s.pkg.Name)
	if s.pkg.Version != "" {
		desc += fmt.Sprintf(" Version: %s", s.pkg.Version)
	}
	return compiler.NewExplanation(
		"Install npm Global Package",
		desc,
		[]string{
			fmt.Sprintf("https://www.npmjs.com/package/%s", s.pkg.Name),
			"https://docs.npmjs.com/cli/install",
		},
	).WithTradeoffs([]string{
		"+ Globally accessible CLI tool",
		"+ Managed updates via 'npm update -g'",
		"- Global packages can have version conflicts",
		"- Requires Node.js to be installed",
	})
}
