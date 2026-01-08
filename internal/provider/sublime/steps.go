package sublime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// getSublimeConfigDir returns the Sublime Text configuration directory path.
func getSublimeConfigDir() string {
	discovery := NewDiscovery()
	return discovery.FindConfigDir()
}

// =============================================================================
// PackagesStep - Manages Package Control installed packages
// =============================================================================

// PackagesStep represents a Sublime Text package installation step.
type PackagesStep struct {
	packages []string
	id       compiler.StepID
	runner   ports.CommandRunner
}

// NewPackagesStep creates a new PackagesStep.
func NewPackagesStep(packages []string, runner ports.CommandRunner) *PackagesStep {
	return &PackagesStep{
		packages: packages,
		id:       compiler.MustNewStepID("sublime:packages"),
		runner:   runner,
	}
}

// ID returns the step identifier.
func (s *PackagesStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *PackagesStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the packages are already configured.
func (s *PackagesStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	discovery := NewDiscovery()
	pkgControlPath := discovery.FindPackageControlPath()

	data, err := os.ReadFile(pkgControlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return compiler.StatusNeedsApply, nil
	}

	installedPkgs, ok := config["installed_packages"].([]interface{})
	if !ok {
		return compiler.StatusNeedsApply, nil
	}

	// Check if all desired packages are listed
	installedSet := make(map[string]bool)
	for _, pkg := range installedPkgs {
		if pkgName, ok := pkg.(string); ok {
			installedSet[pkgName] = true
		}
	}

	for _, pkg := range s.packages {
		if !installedSet[pkg] {
			return compiler.StatusNeedsApply, nil
		}
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *PackagesStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "packages", "Package Control.sublime-settings", "", fmt.Sprintf("%d packages", len(s.packages))), nil
}

// Apply writes the package configuration.
func (s *PackagesStep) Apply(_ compiler.RunContext) error {
	configDir := getSublimeConfigDir()
	pkgControlPath := filepath.Join(configDir, "Package Control.sublime-settings")

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Read existing config
	var config map[string]interface{}
	if data, err := os.ReadFile(pkgControlPath); err == nil {
		_ = json.Unmarshal(data, &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	// Get existing installed packages
	installedPkgs := make([]string, 0)
	if existing, ok := config["installed_packages"].([]interface{}); ok {
		for _, pkg := range existing {
			if pkgName, ok := pkg.(string); ok {
				installedPkgs = append(installedPkgs, pkgName)
			}
		}
	}

	// Add new packages
	existingSet := make(map[string]bool)
	for _, pkg := range installedPkgs {
		existingSet[pkg] = true
	}
	for _, pkg := range s.packages {
		if !existingSet[pkg] {
			installedPkgs = append(installedPkgs, pkg)
		}
	}

	config["installed_packages"] = installedPkgs

	// Write back with pretty formatting
	output, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(pkgControlPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *PackagesStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Sublime Text Packages",
		fmt.Sprintf("Configures %d packages for Package Control to install", len(s.packages)),
		[]string{
			"https://packagecontrol.io/",
			"https://www.sublimetext.com/docs/packages.html",
		},
	).WithTradeoffs([]string{
		"+ Declarative package management",
		"+ Package Control handles installation",
		"- Requires Package Control to be installed",
	})
}

// =============================================================================
// SettingsStep - Manages Preferences.sublime-settings
// =============================================================================

// SettingsStep represents a Sublime Text settings step.
type SettingsStep struct {
	settings    map[string]interface{}
	theme       string
	colorScheme string
	id          compiler.StepID
	runner      ports.CommandRunner
}

// NewSettingsStep creates a new SettingsStep.
func NewSettingsStep(settings map[string]interface{}, theme, colorScheme string, runner ports.CommandRunner) *SettingsStep {
	return &SettingsStep{
		settings:    settings,
		theme:       theme,
		colorScheme: colorScheme,
		id:          compiler.MustNewStepID("sublime:settings"),
		runner:      runner,
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
	discovery := NewDiscovery()
	prefsPath := discovery.FindPreferencesPath()

	data, err := os.ReadFile(prefsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	var current map[string]interface{}
	if err := json.Unmarshal(data, &current); err != nil {
		return compiler.StatusNeedsApply, nil
	}

	// Check if all desired settings are present
	for key, value := range s.settings {
		if currentVal, ok := current[key]; !ok || fmt.Sprintf("%v", currentVal) != fmt.Sprintf("%v", value) {
			return compiler.StatusNeedsApply, nil
		}
	}

	// Check theme
	if s.theme != "" {
		if currentTheme, _ := current["theme"].(string); currentTheme != s.theme {
			return compiler.StatusNeedsApply, nil
		}
	}

	// Check color scheme
	if s.colorScheme != "" {
		if currentCS, _ := current["color_scheme"].(string); currentCS != s.colorScheme {
			return compiler.StatusNeedsApply, nil
		}
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *SettingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	count := len(s.settings)
	if s.theme != "" {
		count++
	}
	if s.colorScheme != "" {
		count++
	}
	return compiler.NewDiff(compiler.DiffTypeModify, "settings", "Preferences.sublime-settings", "", fmt.Sprintf("%d settings", count)), nil
}

// Apply writes the settings.
func (s *SettingsStep) Apply(_ compiler.RunContext) error {
	configDir := getSublimeConfigDir()
	prefsPath := filepath.Join(configDir, "Preferences.sublime-settings")

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Read existing settings
	var current map[string]interface{}
	if data, err := os.ReadFile(prefsPath); err == nil {
		_ = json.Unmarshal(data, &current)
	}
	if current == nil {
		current = make(map[string]interface{})
	}

	// Merge new settings
	for key, value := range s.settings {
		current[key] = value
	}

	// Set theme if specified
	if s.theme != "" {
		current["theme"] = s.theme
	}

	// Set color scheme if specified
	if s.colorScheme != "" {
		current["color_scheme"] = s.colorScheme
	}

	// Write back with pretty formatting
	output, err := json.MarshalIndent(current, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(prefsPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *SettingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Sublime Text Settings",
		fmt.Sprintf("Updates %d settings in Preferences.sublime-settings", len(s.settings)),
		[]string{
			"https://www.sublimetext.com/docs/settings.html",
		},
	).WithTradeoffs([]string{
		"+ Customizes editor behavior",
		"+ Settings sync across machines",
	})
}

// =============================================================================
// KeybindingsStep - Manages keybindings
// =============================================================================

// KeybindingsStep represents a Sublime Text keybindings step.
type KeybindingsStep struct {
	keybindings []Keybinding
	id          compiler.StepID
	runner      ports.CommandRunner
}

// NewKeybindingsStep creates a new KeybindingsStep.
func NewKeybindingsStep(keybindings []Keybinding, runner ports.CommandRunner) *KeybindingsStep {
	return &KeybindingsStep{
		keybindings: keybindings,
		id:          compiler.MustNewStepID("sublime:keybindings"),
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
	discovery := NewDiscovery()
	keybindingsPath := discovery.FindKeybindingsPath()

	_, err := os.ReadFile(keybindingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	// For simplicity, always mark as needs apply if we have keybindings
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *KeybindingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "keybindings", "sublime-keymap", "", fmt.Sprintf("%d keybindings", len(s.keybindings))), nil
}

// Apply writes the keybindings.
func (s *KeybindingsStep) Apply(_ compiler.RunContext) error {
	discovery := NewDiscovery()
	configDir := getSublimeConfigDir()
	keybindingsPath := discovery.FindKeybindingsPath()

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Convert keybindings to JSON format
	kbList := make([]map[string]interface{}, 0, len(s.keybindings))
	for _, kb := range s.keybindings {
		entry := map[string]interface{}{
			"keys":    kb.Keys,
			"command": kb.Command,
		}
		if len(kb.Args) > 0 {
			entry["args"] = kb.Args
		}
		if len(kb.Context) > 0 {
			ctxList := make([]map[string]interface{}, 0, len(kb.Context))
			for _, ctx := range kb.Context {
				ctxEntry := map[string]interface{}{
					"key": ctx.Key,
				}
				if ctx.Operator != "" {
					ctxEntry["operator"] = ctx.Operator
				}
				if ctx.Operand != nil {
					ctxEntry["operand"] = ctx.Operand
				}
				ctxList = append(ctxList, ctxEntry)
			}
			entry["context"] = ctxList
		}
		kbList = append(kbList, entry)
	}

	output, err := json.MarshalIndent(kbList, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal keybindings: %w", err)
	}

	if err := os.WriteFile(keybindingsPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write keybindings: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *KeybindingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Sublime Text Keybindings",
		fmt.Sprintf("Sets %d custom keybindings", len(s.keybindings)),
		[]string{
			"https://www.sublimetext.com/docs/key_bindings.html",
		},
	).WithTradeoffs([]string{
		"+ Customizes keyboard shortcuts",
		"- Overwrites existing user keybindings file",
	})
}
