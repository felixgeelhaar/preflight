package ghcli

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// ExtensionStep represents a GitHub CLI extension installation step.
type ExtensionStep struct {
	extension string
	id        compiler.StepID
	runner    ports.CommandRunner
}

// NewExtensionStep creates a new ExtensionStep.
func NewExtensionStep(extension string, runner ports.CommandRunner) *ExtensionStep {
	id := compiler.MustNewStepID("ghcli:extension:" + extension)
	return &ExtensionStep{
		extension: extension,
		id:        id,
		runner:    runner,
	}
}

// ID returns the step identifier.
func (s *ExtensionStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ExtensionStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the extension is already installed.
func (s *ExtensionStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "gh", "extension", "list")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("gh extension list failed: %s", result.Stderr)
	}

	// Check if extension is in the list
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, s.extension) {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ExtensionStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "extension", s.extension, "", s.extension), nil
}

// Apply installs the extension.
func (s *ExtensionStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "gh", "extension", "install", s.extension)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("gh extension install failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *ExtensionStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install GitHub CLI Extension",
		fmt.Sprintf("Installs the %s extension for GitHub CLI", s.extension),
		[]string{
			fmt.Sprintf("https://github.com/%s", s.extension),
			"https://cli.github.com/manual/gh_extension",
		},
	).WithTradeoffs([]string{
		"+ Extends gh capabilities",
		"+ Easy updates via 'gh extension upgrade'",
		"- Requires gh authentication",
	})
}

// AliasStep represents a GitHub CLI alias step.
type AliasStep struct {
	name    string
	command string
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewAliasStep creates a new AliasStep.
func NewAliasStep(name, command string, runner ports.CommandRunner) *AliasStep {
	id := compiler.MustNewStepID("ghcli:alias:" + name)
	return &AliasStep{
		name:    name,
		command: command,
		id:      id,
		runner:  runner,
	}
}

// ID returns the step identifier.
func (s *AliasStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *AliasStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the alias is already set.
func (s *AliasStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "gh", "alias", "list")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("gh alias list failed: %s", result.Stderr)
	}

	// Check if alias exists with correct command
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == s.name {
			if strings.TrimSpace(parts[1]) == s.command {
				return compiler.StatusSatisfied, nil
			}
			return compiler.StatusNeedsApply, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *AliasStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "alias", s.name, "", s.command), nil
}

// Apply sets the alias.
func (s *AliasStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "gh", "alias", "set", s.name, s.command)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("gh alias set failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *AliasStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Set GitHub CLI Alias",
		fmt.Sprintf("Creates alias '%s' for command '%s'", s.name, s.command),
		[]string{
			"https://cli.github.com/manual/gh_alias",
		},
	).WithTradeoffs([]string{
		"+ Shortens common commands",
		"+ Syncs across machines via gh config",
	})
}

// ConfigStep represents a GitHub CLI config step.
type ConfigStep struct {
	key    string
	value  string
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewConfigStep creates a new ConfigStep.
func NewConfigStep(key, value string, runner ports.CommandRunner) *ConfigStep {
	id := compiler.MustNewStepID("ghcli:config:" + key)
	return &ConfigStep{
		key:    key,
		value:  value,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *ConfigStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ConfigStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the config is already set.
func (s *ConfigStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "gh", "config", "get", s.key)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	currentValue := strings.TrimSpace(result.Stdout)
	if currentValue == s.value {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ConfigStep) Plan(ctx compiler.RunContext) (compiler.Diff, error) {
	result, _ := s.runner.Run(ctx.Context(), "gh", "config", "get", s.key)
	currentValue := strings.TrimSpace(result.Stdout)
	return compiler.NewDiff(compiler.DiffTypeModify, "config", s.key, currentValue, s.value), nil
}

// Apply sets the config.
func (s *ConfigStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "gh", "config", "set", s.key, s.value)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("gh config set failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *ConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Set GitHub CLI Config",
		fmt.Sprintf("Sets %s to %s", s.key, s.value),
		[]string{
			"https://cli.github.com/manual/gh_config",
		},
	).WithTradeoffs([]string{
		"+ Configures gh behavior globally",
	})
}
