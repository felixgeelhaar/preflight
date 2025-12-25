package cursor

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

// getCursorConfigPath returns the Cursor configuration directory path.
func getCursorConfigPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User")
	case "linux":
		return filepath.Join(home, ".config", "Cursor", "User")
	default: // windows
		return filepath.Join(os.Getenv("APPDATA"), "Cursor", "User")
	}
}

// ExtensionStep represents a Cursor extension installation step.
type ExtensionStep struct {
	extension string
	id        compiler.StepID
	runner    ports.CommandRunner
}

// NewExtensionStep creates a new ExtensionStep.
func NewExtensionStep(extension string, runner ports.CommandRunner) *ExtensionStep {
	id := compiler.MustNewStepID("cursor:extension:" + extension)
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
	result, err := s.runner.Run(ctx.Context(), "cursor", "--list-extensions")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("cursor --list-extensions failed: %s", result.Stderr)
	}

	extensions := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, ext := range extensions {
		if strings.EqualFold(ext, s.extension) {
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
	result, err := s.runner.Run(ctx.Context(), "cursor", "--install-extension", s.extension)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("cursor --install-extension failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *ExtensionStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Cursor Extension",
		fmt.Sprintf("Installs the %s extension for Cursor editor", s.extension),
		[]string{
			"https://cursor.sh/",
			"https://marketplace.visualstudio.com",
		},
	).WithTradeoffs([]string{
		"+ Adds AI-powered features to Cursor",
		"+ Compatible with VS Code extensions",
		"- May impact editor performance",
	})
}

// SettingsStep represents a Cursor settings synchronization step.
type SettingsStep struct {
	settings map[string]interface{}
	id       compiler.StepID
	runner   ports.CommandRunner
}

// NewSettingsStep creates a new SettingsStep.
func NewSettingsStep(settings map[string]interface{}, runner ports.CommandRunner) *SettingsStep {
	return &SettingsStep{
		settings: settings,
		id:       compiler.MustNewStepID("cursor:settings"),
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
	settingsPath := filepath.Join(getCursorConfigPath(), "settings.json")

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
	settingsPath := filepath.Join(getCursorConfigPath(), "settings.json")

	// Read existing settings
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

	// Write back
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
		"Configure Cursor Settings",
		fmt.Sprintf("Updates %d settings in Cursor configuration", len(s.settings)),
		[]string{
			"https://cursor.sh/",
		},
	).WithTradeoffs([]string{
		"+ Customizes editor behavior",
		"+ Settings sync across machines",
	})
}

// KeybindingsStep represents a Cursor keybindings synchronization step.
type KeybindingsStep struct {
	keybindings []Keybinding
	id          compiler.StepID
	runner      ports.CommandRunner
}

// NewKeybindingsStep creates a new KeybindingsStep.
func NewKeybindingsStep(keybindings []Keybinding, runner ports.CommandRunner) *KeybindingsStep {
	return &KeybindingsStep{
		keybindings: keybindings,
		id:          compiler.MustNewStepID("cursor:keybindings"),
		runner:      runner,
	}
}

// ID returns the step identifier.
func (s *KeybindingsStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *KeybindingsStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the keybindings are already applied.
func (s *KeybindingsStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	keybindingsPath := filepath.Join(getCursorConfigPath(), "keybindings.json")

	_, err := os.ReadFile(keybindingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	// For simplicity, always mark as needs apply if we have keybindings
	// A more sophisticated check would compare the actual keybindings
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *KeybindingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "keybindings", "keybindings.json", "", fmt.Sprintf("%d keybindings", len(s.keybindings))), nil
}

// Apply writes the keybindings.
func (s *KeybindingsStep) Apply(_ compiler.RunContext) error {
	keybindingsPath := filepath.Join(getCursorConfigPath(), "keybindings.json")

	// Convert keybindings to JSON format
	kbList := make([]map[string]string, 0, len(s.keybindings))
	for _, kb := range s.keybindings {
		entry := map[string]string{
			"key":     kb.Key,
			"command": kb.Command,
		}
		if kb.When != "" {
			entry["when"] = kb.When
		}
		kbList = append(kbList, entry)
	}

	output, err := json.MarshalIndent(kbList, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(keybindingsPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(keybindingsPath, output, 0o644)
}

// Explain provides a human-readable explanation.
func (s *KeybindingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Cursor Keybindings",
		fmt.Sprintf("Sets %d custom keybindings", len(s.keybindings)),
		[]string{
			"https://cursor.sh/",
		},
	).WithTradeoffs([]string{
		"+ Customizes keyboard shortcuts",
		"- Overwrites existing keybindings file",
	})
}
