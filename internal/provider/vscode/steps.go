package vscode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// ExtensionStep manages VSCode extension installation.
type ExtensionStep struct {
	extension string
	id        compiler.StepID
	runner    ports.CommandRunner
}

// NewExtensionStep creates a new ExtensionStep.
func NewExtensionStep(extension string, runner ports.CommandRunner) *ExtensionStep {
	// Replace dots with underscores for valid step ID (dots not allowed in StepID pattern)
	safeExt := strings.ReplaceAll(extension, ".", "_")
	id := compiler.MustNewStepID(fmt.Sprintf("vscode:extension:%s", safeExt))
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

// DependsOn returns dependencies for this step.
func (s *ExtensionStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if the extension is installed.
func (s *ExtensionStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "code", "--list-extensions")
	if err != nil {
		return compiler.StatusUnknown, err
	}

	// Check if extension is in the list
	extensions := strings.Split(result.Stdout, "\n")
	for _, ext := range extensions {
		if strings.TrimSpace(ext) == s.extension {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ExtensionStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"extension",
		s.extension,
		"",
		fmt.Sprintf("Install VSCode extension %s", s.extension),
	), nil
}

// Apply installs the extension.
func (s *ExtensionStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "code", "--install-extension", s.extension, "--force")
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("extension install failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides context for this step.
func (s *ExtensionStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install VSCode Extension",
		fmt.Sprintf("Install the VSCode extension %s using the 'code' CLI", s.extension),
		[]string{fmt.Sprintf("https://marketplace.visualstudio.com/items?itemName=%s", s.extension)},
	)
}

// SettingsStep manages VSCode settings.json.
type SettingsStep struct {
	settings map[string]interface{}
	id       compiler.StepID
	fs       ports.FileSystem
}

// NewSettingsStep creates a new SettingsStep.
func NewSettingsStep(settings map[string]interface{}, fs ports.FileSystem) *SettingsStep {
	id := compiler.MustNewStepID("vscode:settings")
	return &SettingsStep{
		settings: settings,
		id:       id,
		fs:       fs,
	}
}

// ID returns the step identifier.
func (s *SettingsStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *SettingsStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if settings need to be applied.
func (s *SettingsStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	settingsPath := s.getSettingsPath()
	if !s.fs.Exists(settingsPath) {
		return compiler.StatusNeedsApply, nil
	}

	// For now, if settings exist, consider comparing content
	// Future: deep compare settings
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *SettingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"settings",
		"settings.json",
		"",
		"Merge VSCode settings.json",
	), nil
}

// Apply writes the settings file.
func (s *SettingsStep) Apply(_ compiler.RunContext) error {
	settingsPath := s.getSettingsPath()

	// Merge with existing settings
	existingSettings := make(map[string]interface{})
	if s.fs.Exists(settingsPath) {
		content, err := s.fs.ReadFile(settingsPath)
		if err == nil {
			_ = json.Unmarshal(content, &existingSettings)
		}
	}

	// Merge new settings into existing
	for k, v := range s.settings {
		existingSettings[k] = v
	}

	// Write merged settings
	data, err := json.MarshalIndent(existingSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := s.fs.WriteFile(settingsPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}

// Explain provides context for this step.
func (s *SettingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure VSCode Settings",
		"Merge settings into VSCode settings.json, preserving existing settings while adding/updating managed ones",
		[]string{"https://code.visualstudio.com/docs/getstarted/settings"},
	)
}

func (s *SettingsStep) getSettingsPath() string {
	// Cross-platform path resolution would be more complex in production
	return ports.ExpandPath("~/.config/Code/User/settings.json")
}

// KeybindingsStep manages VSCode keybindings.json.
type KeybindingsStep struct {
	keybindings []Keybinding
	id          compiler.StepID
	fs          ports.FileSystem
}

// NewKeybindingsStep creates a new KeybindingsStep.
func NewKeybindingsStep(keybindings []Keybinding, fs ports.FileSystem) *KeybindingsStep {
	id := compiler.MustNewStepID("vscode:keybindings")
	return &KeybindingsStep{
		keybindings: keybindings,
		id:          id,
		fs:          fs,
	}
}

// ID returns the step identifier.
func (s *KeybindingsStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *KeybindingsStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if keybindings need to be applied.
func (s *KeybindingsStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	keybindingsPath := s.getKeybindingsPath()
	if !s.fs.Exists(keybindingsPath) {
		return compiler.StatusNeedsApply, nil
	}

	// Future: compare existing keybindings
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *KeybindingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"keybindings",
		"keybindings.json",
		"",
		"Update VSCode keybindings.json",
	), nil
}

// Apply writes the keybindings file.
func (s *KeybindingsStep) Apply(_ compiler.RunContext) error {
	keybindingsPath := s.getKeybindingsPath()

	data, err := json.MarshalIndent(s.keybindings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keybindings: %w", err)
	}

	if err := s.fs.WriteFile(keybindingsPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write keybindings: %w", err)
	}

	return nil
}

// Explain provides context for this step.
func (s *KeybindingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure VSCode Keybindings",
		"Write custom keybindings to VSCode keybindings.json",
		[]string{"https://code.visualstudio.com/docs/getstarted/keybindings"},
	)
}

func (s *KeybindingsStep) getKeybindingsPath() string {
	return ports.ExpandPath("~/.config/Code/User/keybindings.json")
}
