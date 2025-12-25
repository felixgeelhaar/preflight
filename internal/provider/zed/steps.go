package zed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// getZedConfigPath returns the Zed configuration directory path.
func getZedConfigPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, ".config", "zed")
	case "linux":
		return filepath.Join(home, ".config", "zed")
	default:
		return filepath.Join(home, ".config", "zed")
	}
}

// ExtensionStep represents a Zed extension installation step.
type ExtensionStep struct {
	extension string
	id        compiler.StepID
	runner    ports.CommandRunner
}

// NewExtensionStep creates a new ExtensionStep.
func NewExtensionStep(extension string, runner ports.CommandRunner) *ExtensionStep {
	id := compiler.MustNewStepID("zed:extension:" + extension)
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
func (s *ExtensionStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	// Check extensions directory
	extPath := filepath.Join(getZedConfigPath(), "extensions", "installed", s.extension)
	if _, err := os.Stat(extPath); err == nil {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ExtensionStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "extension", s.extension, "", s.extension), nil
}

// Apply installs the extension.
func (s *ExtensionStep) Apply(_ compiler.RunContext) error {
	// Zed extensions are installed via the editor or by adding to extensions.json
	// For now, we'll add to the settings
	settingsPath := filepath.Join(getZedConfigPath(), "settings.json")

	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// Add extension to auto_install_extensions
	var extensions []interface{}
	if existing, ok := settings["auto_install_extensions"].([]interface{}); ok {
		extensions = existing
	}

	// Check if already in list
	found := false
	for _, ext := range extensions {
		if extStr, ok := ext.(string); ok && extStr == s.extension {
			found = true
			break
		}
	}
	if !found {
		extensions = append(extensions, s.extension)
		settings["auto_install_extensions"] = extensions
	}

	output, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(settingsPath, output, 0o644)
}

// Explain provides a human-readable explanation.
func (s *ExtensionStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Zed Extension",
		fmt.Sprintf("Configures %s extension for Zed editor", s.extension),
		[]string{
			"https://zed.dev/extensions",
		},
	).WithTradeoffs([]string{
		"+ Adds language support or features to Zed",
		"+ Native extensions are fast and lightweight",
	})
}

// SettingsStep represents a Zed settings synchronization step.
type SettingsStep struct {
	settings map[string]interface{}
	id       compiler.StepID
	runner   ports.CommandRunner
}

// NewSettingsStep creates a new SettingsStep.
func NewSettingsStep(settings map[string]interface{}, runner ports.CommandRunner) *SettingsStep {
	return &SettingsStep{
		settings: settings,
		id:       compiler.MustNewStepID("zed:settings"),
		runner:   runner,
	}
}

// ID returns the step identifier.
func (s *SettingsStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *SettingsStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the settings are already applied.
func (s *SettingsStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	settingsPath := filepath.Join(getZedConfigPath(), "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	var current map[string]interface{}
	if err := json.Unmarshal(data, &current); err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // Invalid JSON means we need to apply fresh config
	}

	// Check if all desired settings are present
	for key, value := range s.settings {
		if currentVal, ok := current[key]; !ok || fmt.Sprintf("%v", currentVal) != fmt.Sprintf("%v", value) {
			return compiler.StatusNeedsApply, nil
		}
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *SettingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "settings", "settings.json", "", fmt.Sprintf("%d settings", len(s.settings))), nil
}

// Apply writes the settings.
func (s *SettingsStep) Apply(_ compiler.RunContext) error {
	settingsPath := filepath.Join(getZedConfigPath(), "settings.json")

	var current map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		_ = json.Unmarshal(data, &current)
	}
	if current == nil {
		current = make(map[string]interface{})
	}

	// Merge new settings
	for key, value := range s.settings {
		current[key] = value
	}

	output, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(settingsPath, output, 0o644)
}

// Explain provides a human-readable explanation.
func (s *SettingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Zed Settings",
		fmt.Sprintf("Updates %d settings in Zed configuration", len(s.settings)),
		[]string{
			"https://zed.dev/docs/configuring-zed",
		},
	).WithTradeoffs([]string{
		"+ Customizes editor behavior",
		"+ JSONC format supports comments",
	})
}

// KeymapStep represents a Zed keymap synchronization step.
type KeymapStep struct {
	keymap []KeyBinding
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewKeymapStep creates a new KeymapStep.
func NewKeymapStep(keymap []KeyBinding, runner ports.CommandRunner) *KeymapStep {
	return &KeymapStep{
		keymap: keymap,
		id:     compiler.MustNewStepID("zed:keymap"),
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *KeymapStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *KeymapStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the keymap is already applied.
func (s *KeymapStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	keymapPath := filepath.Join(getZedConfigPath(), "keymap.json")

	_, err := os.ReadFile(keymapPath)
	if err != nil {
		if os.IsNotExist(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *KeymapStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "keymap", "keymap.json", "", fmt.Sprintf("%d contexts", len(s.keymap))), nil
}

// Apply writes the keymap.
func (s *KeymapStep) Apply(_ compiler.RunContext) error {
	keymapPath := filepath.Join(getZedConfigPath(), "keymap.json")

	// Convert keymap to Zed format
	keymapList := make([]map[string]interface{}, 0, len(s.keymap))
	for _, kb := range s.keymap {
		entry := map[string]interface{}{
			"bindings": kb.Bindings,
		}
		if kb.Context != "" {
			entry["context"] = kb.Context
		}
		keymapList = append(keymapList, entry)
	}

	output, err := json.MarshalIndent(keymapList, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(keymapPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(keymapPath, output, 0o644)
}

// Explain provides a human-readable explanation.
func (s *KeymapStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Zed Keymap",
		fmt.Sprintf("Sets custom keybindings in %d contexts", len(s.keymap)),
		[]string{
			"https://zed.dev/docs/key-bindings",
		},
	).WithTradeoffs([]string{
		"+ Customizes keyboard shortcuts",
		"+ Context-aware bindings",
	})
}

// ThemeStep represents a Zed theme configuration step.
type ThemeStep struct {
	theme  string
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewThemeStep creates a new ThemeStep.
func NewThemeStep(theme string, runner ports.CommandRunner) *ThemeStep {
	return &ThemeStep{
		theme:  theme,
		id:     compiler.MustNewStepID("zed:theme:" + theme),
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *ThemeStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ThemeStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the theme is already set.
func (s *ThemeStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	settingsPath := filepath.Join(getZedConfigPath(), "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // File not existing means we need to apply
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // Invalid JSON means we need to apply fresh config
	}

	if theme, ok := settings["theme"].(string); ok && strings.EqualFold(theme, s.theme) {
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ThemeStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "theme", "theme", "", s.theme), nil
}

// Apply sets the theme.
func (s *ThemeStep) Apply(_ compiler.RunContext) error {
	settingsPath := filepath.Join(getZedConfigPath(), "settings.json")

	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	settings["theme"] = s.theme

	output, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(settingsPath, output, 0o644)
}

// Explain provides a human-readable explanation.
func (s *ThemeStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Set Zed Theme",
		fmt.Sprintf("Sets Zed theme to %s", s.theme),
		[]string{
			"https://zed.dev/docs/themes",
		},
	).WithTradeoffs([]string{
		"+ Customizes visual appearance",
	})
}
