package starship

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/pelletier/go-toml/v2"
)

// getStarshipConfigPath returns the Starship configuration file path.
func getStarshipConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "starship.toml")
}

// InstallStep represents a Starship installation step.
type InstallStep struct {
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewInstallStep creates a new InstallStep.
func NewInstallStep(runner ports.CommandRunner) *InstallStep {
	return &InstallStep{
		id:     compiler.MustNewStepID("starship:install"),
		runner: runner,
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

// Check determines if Starship is already installed.
func (s *InstallStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "starship", "--version")
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // Starship not installed means we need to apply
	}
	if result.Success() {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *InstallStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "starship", "starship", "", "latest"), nil
}

// Apply installs Starship.
func (s *InstallStep) Apply(ctx compiler.RunContext) error {
	// Try brew first on macOS
	result, err := s.runner.Run(ctx.Context(), "brew", "install", "starship")
	if err == nil && result.Success() {
		return nil
	}

	// Fallback to curl installer
	result, err = s.runner.Run(ctx.Context(), "sh", "-c", "curl -sS https://starship.rs/install.sh | sh -s -- -y")
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("starship install failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *InstallStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Starship",
		"Installs the Starship cross-shell prompt",
		[]string{
			"https://starship.rs/",
			"https://github.com/starship/starship",
		},
	).WithTradeoffs([]string{
		"+ Minimal, fast, customizable prompt",
		"+ Works across all shells",
		"+ Great out-of-the-box defaults",
	})
}

// ConfigStep represents a Starship configuration step.
type ConfigStep struct {
	settings   map[string]interface{}
	preset     string
	installDep compiler.StepID
	id         compiler.StepID
	runner     ports.CommandRunner
}

// NewConfigStep creates a new ConfigStep.
func NewConfigStep(settings map[string]interface{}, preset string, installDep compiler.StepID, runner ports.CommandRunner) *ConfigStep {
	return &ConfigStep{
		settings:   settings,
		preset:     preset,
		installDep: installDep,
		id:         compiler.MustNewStepID("starship:config"),
		runner:     runner,
	}
}

// ID returns the step identifier.
func (s *ConfigStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ConfigStep) DependsOn() []compiler.StepID {
	return []compiler.StepID{s.installDep}
}

// Check determines if the configuration is applied.
func (s *ConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := getStarshipConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return compiler.StatusNeedsApply, nil
	}

	// If we have a preset, always apply
	if s.preset != "" {
		return compiler.StatusNeedsApply, nil
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *ConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	details := fmt.Sprintf("%d settings", len(s.settings))
	if s.preset != "" {
		details = fmt.Sprintf("preset: %s", s.preset)
	}
	return compiler.NewDiff(compiler.DiffTypeModify, "config", "starship.toml", "", details), nil
}

// Apply writes the configuration.
func (s *ConfigStep) Apply(ctx compiler.RunContext) error {
	configPath := getStarshipConfigPath()

	// Create config directory if needed
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	// If preset is specified, use starship preset command
	if s.preset != "" {
		result, err := s.runner.Run(ctx.Context(), "starship", "preset", s.preset, "-o", configPath)
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("starship preset failed: %s", result.Stderr)
		}
		return nil
	}

	// Write settings as TOML
	output, err := toml.Marshal(s.settings)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, output, 0o644)
}

// Explain provides a human-readable explanation.
func (s *ConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Configures Starship with %d settings", len(s.settings))
	if s.preset != "" {
		desc = fmt.Sprintf("Applies the %s preset to Starship", s.preset)
	}
	return compiler.NewExplanation(
		"Configure Starship",
		desc,
		[]string{
			"https://starship.rs/config/",
			"https://starship.rs/presets/",
		},
	).WithTradeoffs([]string{
		"+ Customizes prompt appearance and behavior",
		"+ Presets provide themed configurations",
	})
}

// ShellIntegrationStep represents a shell integration step.
type ShellIntegrationStep struct {
	shell      string
	installDep compiler.StepID
	id         compiler.StepID
	runner     ports.CommandRunner
}

// NewShellIntegrationStep creates a new ShellIntegrationStep.
func NewShellIntegrationStep(shell string, installDep compiler.StepID, runner ports.CommandRunner) *ShellIntegrationStep {
	return &ShellIntegrationStep{
		shell:      shell,
		installDep: installDep,
		id:         compiler.MustNewStepID("starship:shell:" + shell),
		runner:     runner,
	}
}

// ID returns the step identifier.
func (s *ShellIntegrationStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ShellIntegrationStep) DependsOn() []compiler.StepID {
	return []compiler.StepID{s.installDep}
}

// Check determines if shell integration is configured.
func (s *ShellIntegrationStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	home, _ := os.UserHomeDir()
	var configPath string

	switch s.shell {
	case "zsh":
		configPath = filepath.Join(home, ".zshrc")
	case "bash":
		configPath = filepath.Join(home, ".bashrc")
	case "fish":
		configPath = filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return compiler.StatusUnknown, fmt.Errorf("unsupported shell: %s", s.shell)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // File not existing means we need to apply
	}

	if strings.Contains(string(data), "starship init") {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ShellIntegrationStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "integration", s.shell, "", "starship init"), nil
}

// Apply adds shell integration.
func (s *ShellIntegrationStep) Apply(_ compiler.RunContext) error {
	home, _ := os.UserHomeDir()
	var configPath string
	var initLine string

	switch s.shell {
	case "zsh":
		configPath = filepath.Join(home, ".zshrc")
		initLine = `eval "$(starship init zsh)"`
	case "bash":
		configPath = filepath.Join(home, ".bashrc")
		initLine = `eval "$(starship init bash)"`
	case "fish":
		configPath = filepath.Join(home, ".config", "fish", "config.fish")
		initLine = "starship init fish | source"
	default:
		return fmt.Errorf("unsupported shell: %s", s.shell)
	}

	// Read existing config
	content, _ := os.ReadFile(configPath)

	// Check if already present
	if strings.Contains(string(content), "starship init") {
		return nil
	}

	// Append init line
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		content = append(content, '\n')
	}
	content = append(content, []byte("\n# Starship prompt\n"+initLine+"\n")...)

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, content, 0o644)
}

// Explain provides a human-readable explanation.
func (s *ShellIntegrationStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Shell Integration",
		fmt.Sprintf("Adds Starship initialization to %s configuration", s.shell),
		[]string{
			"https://starship.rs/guide/#step-2-set-up-your-shell-to-use-starship",
		},
	).WithTradeoffs([]string{
		"+ Enables Starship prompt in your shell",
		"- Requires shell restart to take effect",
	})
}
