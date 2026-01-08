package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// getTPMPath returns the TPM installation path.
// Uses dynamic discovery to find existing TPM installation.
func getTPMPath() string {
	discovery := NewDiscovery()

	// Check if TPM exists in known location
	if found := discovery.FindTPMPath(); found != "" {
		return found
	}

	// Return best-practice path for new installation
	return discovery.TPMBestPracticePath()
}

// getTmuxConfigPath returns the tmux configuration file path.
// Uses dynamic discovery: checks TMUX_CONF env var first, then XDG, then legacy paths.
func getTmuxConfigPath() string {
	discovery := NewDiscovery()

	// Check if config exists in any known location
	if found := discovery.FindConfig(); found != "" {
		return found
	}

	// Return best-practice path for new configs
	return discovery.BestPracticePath()
}

// TPMStep represents a TPM (Tmux Plugin Manager) installation step.
type TPMStep struct {
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewTPMStep creates a new TPMStep.
func NewTPMStep(runner ports.CommandRunner) *TPMStep {
	return &TPMStep{
		id:     compiler.MustNewStepID("tmux:tpm"),
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *TPMStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *TPMStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if TPM is already installed.
func (s *TPMStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	tpmPath := getTPMPath()
	if _, err := os.Stat(tpmPath); err == nil {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *TPMStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "tpm", "tmux-plugins/tpm", "", getTPMPath()), nil
}

// Apply installs TPM.
func (s *TPMStep) Apply(ctx compiler.RunContext) error {
	tpmPath := getTPMPath()

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(tpmPath), 0o755); err != nil {
		return err
	}

	// Clone TPM repository
	result, err := s.runner.Run(ctx.Context(), "git", "clone", "https://github.com/tmux-plugins/tpm", tpmPath)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("git clone tpm failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *TPMStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install TPM",
		"Installs Tmux Plugin Manager for managing tmux plugins",
		[]string{
			"https://github.com/tmux-plugins/tpm",
		},
	).WithTradeoffs([]string{
		"+ Enables easy plugin management",
		"+ Plugins auto-install on first run",
		"- Requires git",
	})
}

// PluginStep represents a tmux plugin installation step.
type PluginStep struct {
	plugin string
	tpmDep compiler.StepID
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewPluginStep creates a new PluginStep.
func NewPluginStep(plugin string, tpmDep compiler.StepID, runner ports.CommandRunner) *PluginStep {
	id := compiler.MustNewStepID("tmux:plugin:" + plugin)
	return &PluginStep{
		plugin: plugin,
		tpmDep: tpmDep,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *PluginStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *PluginStep) DependsOn() []compiler.StepID {
	return []compiler.StepID{s.tpmDep}
}

// Check determines if the plugin is configured.
func (s *PluginStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := getTmuxConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // File not existing means we need to apply
	}

	// Check if plugin is in config
	if strings.Contains(string(data), s.plugin) {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PluginStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "plugin", s.plugin, "", s.plugin), nil
}

// Apply adds the plugin to tmux config.
func (s *PluginStep) Apply(_ compiler.RunContext) error {
	configPath := getTmuxConfigPath()

	// Read existing config
	var content []byte
	content, _ = os.ReadFile(configPath)

	// Add plugin line if not present
	pluginLine := fmt.Sprintf("set -g @plugin '%s'\n", s.plugin)
	if !strings.Contains(string(content), s.plugin) {
		content = append(content, []byte(pluginLine)...)
	}

	// Ensure parent directory exists (important for XDG paths)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(configPath, content, 0o644)
}

// Explain provides a human-readable explanation.
func (s *PluginStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure tmux Plugin",
		fmt.Sprintf("Adds %s to tmux configuration", s.plugin),
		[]string{
			fmt.Sprintf("https://github.com/%s", s.plugin),
		},
	).WithTradeoffs([]string{
		"+ Extends tmux functionality",
		"+ Install with prefix + I in tmux",
	})
}

// ConfigStep represents a tmux configuration step.
type ConfigStep struct {
	settings   map[string]string
	configFile string
	id         compiler.StepID
	runner     ports.CommandRunner
}

// NewConfigStep creates a new ConfigStep.
func NewConfigStep(settings map[string]string, configFile string, runner ports.CommandRunner) *ConfigStep {
	return &ConfigStep{
		settings:   settings,
		configFile: configFile,
		id:         compiler.MustNewStepID("tmux:config"),
		runner:     runner,
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

// Check determines if the config is applied.
func (s *ConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := getTmuxConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // File not existing means we need to apply
	}

	// Check if all settings are present
	for key, value := range s.settings {
		expectedLine := fmt.Sprintf("set -g %s %s", key, value)
		if !strings.Contains(string(data), expectedLine) {
			return compiler.StatusNeedsApply, nil
		}
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *ConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "config", "tmux.conf", "", fmt.Sprintf("%d settings", len(s.settings))), nil
}

// Apply writes the configuration.
func (s *ConfigStep) Apply(_ compiler.RunContext) error {
	configPath := getTmuxConfigPath()

	// Read existing config or start fresh
	var content []byte
	content, _ = os.ReadFile(configPath)
	lines := strings.Split(string(content), "\n")

	// Build new content preserving existing lines but updating settings
	var newLines []string
	settingsApplied := make(map[string]bool)

	for _, line := range lines {
		updated := false
		for key, value := range s.settings {
			if strings.HasPrefix(strings.TrimSpace(line), fmt.Sprintf("set -g %s", key)) {
				newLines = append(newLines, fmt.Sprintf("set -g %s %s", key, value))
				settingsApplied[key] = true
				updated = true
				break
			}
		}
		if !updated {
			newLines = append(newLines, line)
		}
	}

	// Add any settings that weren't updated
	for key, value := range s.settings {
		if !settingsApplied[key] {
			newLines = append(newLines, fmt.Sprintf("set -g %s %s", key, value))
		}
	}

	// Ensure parent directory exists (important for XDG paths)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0o644)
}

// Explain provides a human-readable explanation.
func (s *ConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure tmux",
		fmt.Sprintf("Updates %d tmux settings", len(s.settings)),
		[]string{
			"https://github.com/tmux/tmux/wiki/Getting-Started",
		},
	).WithTradeoffs([]string{
		"+ Customizes tmux behavior",
		"+ Reload with 'tmux source ~/.tmux.conf'",
	})
}
