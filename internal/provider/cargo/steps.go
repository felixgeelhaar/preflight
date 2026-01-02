package cargo

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// CrateStep represents a Cargo crate installation step.
type CrateStep struct {
	crate  Crate
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewCrateStep creates a new CrateStep.
func NewCrateStep(crate Crate, runner ports.CommandRunner) *CrateStep {
	id := compiler.MustNewStepID("cargo:crate:" + crate.Name)
	return &CrateStep{
		crate:  crate,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *CrateStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *CrateStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the crate is already installed.
func (s *CrateStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	// Use cargo install --list to check installed crates
	result, err := s.runner.Run(ctx.Context(), "cargo", "install", "--list")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("cargo install --list failed: %s", result.Stderr)
	}

	// Parse output to find installed crates
	// Format: "crate-name v1.2.3:" followed by binaries
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, s.crate.Name+" ") {
			return compiler.StatusSatisfied, nil
		}
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *CrateStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	version := s.crate.Version
	if version == "" {
		version = "latest"
	}
	return compiler.NewDiff(compiler.DiffTypeAdd, "cargo-crate", s.crate.Name, "", version), nil
}

// Apply executes the crate installation.
func (s *CrateStep) Apply(ctx compiler.RunContext) error {
	// Validate crate name before execution to prevent command injection
	if err := validation.ValidateCargoCrate(s.crate.FullName()); err != nil {
		return fmt.Errorf("invalid cargo crate: %w", err)
	}

	args := []string{"install", s.crate.Name}
	if s.crate.Version != "" {
		args = append(args, "--version", s.crate.Version)
	}

	result, err := s.runner.Run(ctx.Context(), "cargo", args...)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("cargo install %s failed: %s", s.crate.Name, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *CrateStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s crate via Cargo.", s.crate.Name)
	if s.crate.Version != "" {
		desc += fmt.Sprintf(" Version: %s", s.crate.Version)
	}
	return compiler.NewExplanation(
		"Install Cargo Crate",
		desc,
		[]string{
			fmt.Sprintf("https://crates.io/crates/%s", s.crate.Name),
			"https://doc.rust-lang.org/cargo/commands/cargo-install.html",
		},
	).WithTradeoffs([]string{
		"+ Access to Rust ecosystem tools",
		"+ Version pinning with --version flag",
		"+ Compiles optimized binaries",
		"- Requires Rust toolchain to be installed",
		"- Compilation can be slow for large crates",
	})
}
