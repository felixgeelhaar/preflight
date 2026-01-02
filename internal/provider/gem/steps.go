package gem

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// Step represents a Ruby gem installation step.
type Step struct {
	gem    Gem
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewStep creates a new gem Step.
func NewStep(gem Gem, runner ports.CommandRunner) *Step {
	id := compiler.MustNewStepID("gem:gem:" + gem.Name)
	return &Step{
		gem:    gem,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *Step) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *Step) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the gem is already installed.
func (s *Step) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	// Use gem list -i to check if gem is installed
	result, err := s.runner.Run(ctx.Context(), "gem", "list", "-i", s.gem.Name)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	if result.Success() && strings.TrimSpace(result.Stdout) == "true" {
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *Step) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	version := s.gem.Version
	if version == "" {
		version = "latest"
	}
	return compiler.NewDiff(compiler.DiffTypeAdd, "gem", s.gem.Name, "", version), nil
}

// Apply executes the gem installation.
func (s *Step) Apply(ctx compiler.RunContext) error {
	// Validate gem name before execution to prevent command injection
	if err := validation.ValidateGemName(s.gem.FullName()); err != nil {
		return fmt.Errorf("invalid gem name: %w", err)
	}

	args := []string{"install", s.gem.Name}
	if s.gem.Version != "" {
		args = append(args, "-v", s.gem.Version)
	}

	result, err := s.runner.Run(ctx.Context(), "gem", args...)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("gem install %s failed: %s", s.gem.Name, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *Step) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s gem via RubyGems.", s.gem.Name)
	if s.gem.Version != "" {
		desc += fmt.Sprintf(" Version: %s", s.gem.Version)
	}
	return compiler.NewExplanation(
		"Install Ruby Gem",
		desc,
		[]string{
			fmt.Sprintf("https://rubygems.org/gems/%s", s.gem.Name),
			"https://guides.rubygems.org/command-reference/",
		},
	).WithTradeoffs([]string{
		"+ Access to Ruby ecosystem tools",
		"+ Version pinning support",
		"+ Managed updates via 'gem update'",
		"- Requires Ruby to be installed",
		"- May conflict with bundled gems in Ruby projects",
	})
}
