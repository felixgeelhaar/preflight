package helix

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/pelletier/go-toml/v2"
)

// getHelixConfigDir returns the Helix configuration directory path.
func getHelixConfigDir() string {
	discovery := NewDiscovery()
	return discovery.FindConfigDir()
}

// =============================================================================
// ConfigStep - Manages config.toml (copy/link or generate from settings)
// =============================================================================

// ConfigStep represents a Helix config.toml management step.
type ConfigStep struct {
	source   string                 // Source file to copy/link (optional)
	link     bool                   // Whether to symlink
	settings map[string]interface{} // Settings to merge
	editor   map[string]interface{} // Editor settings
	keys     map[string]interface{} // Keys settings
	id       compiler.StepID
	runner   ports.CommandRunner
}

// NewConfigStep creates a new ConfigStep.
func NewConfigStep(source string, link bool, settings, editor, keys map[string]interface{}, runner ports.CommandRunner) *ConfigStep {
	return &ConfigStep{
		source:   source,
		link:     link,
		settings: settings,
		editor:   editor,
		keys:     keys,
		id:       compiler.MustNewStepID("helix:config"),
		runner:   runner,
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

// Check determines if the config is already applied.
func (s *ConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := filepath.Join(getHelixConfigDir(), "config.toml")

	if s.source != "" {
		// Check if symlink or copy matches
		info, err := os.Lstat(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				return compiler.StatusNeedsApply, nil
			}
			return compiler.StatusUnknown, err
		}

		if s.link {
			// Should be a symlink
			if info.Mode()&os.ModeSymlink == 0 {
				return compiler.StatusNeedsApply, nil
			}
			// Check if symlink points to correct target
			target, err := os.Readlink(configPath)
			if err != nil {
				return compiler.StatusUnknown, err
			}
			absSource, _ := filepath.Abs(s.source)
			if target != absSource && target != s.source {
				return compiler.StatusNeedsApply, nil
			}
			return compiler.StatusSatisfied, nil
		}

		// For copy, just check if file exists (could compare content)
		if info.IsDir() {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusSatisfied, nil
	}

	// Settings-based config
	if !fileExists(configPath) {
		return compiler.StatusNeedsApply, nil
	}

	// Could compare settings here, but for simplicity always apply
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	if s.source != "" {
		action := "copy"
		if s.link {
			action = "link"
		}
		return compiler.NewDiff(compiler.DiffTypeModify, "config", "config.toml", "", fmt.Sprintf("%s from %s", action, s.source)), nil
	}
	return compiler.NewDiff(compiler.DiffTypeModify, "config", "config.toml", "", "merge settings"), nil
}

// Apply writes the config.
func (s *ConfigStep) Apply(_ compiler.RunContext) error {
	configDir := getHelixConfigDir()
	configPath := filepath.Join(configDir, "config.toml")

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	if s.source != "" {
		// Remove existing file/symlink
		_ = os.Remove(configPath)

		if s.link {
			absSource, err := filepath.Abs(s.source)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}
			if err := os.Symlink(absSource, configPath); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}
			return nil
		}

		// Copy file
		content, err := os.ReadFile(s.source)
		if err != nil {
			return fmt.Errorf("failed to read source: %w", err)
		}
		if err := os.WriteFile(configPath, content, 0o644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
		return nil
	}

	// Generate config from settings
	config := make(map[string]interface{})

	// Read existing config if present
	if data, err := os.ReadFile(configPath); err == nil {
		_ = toml.Unmarshal(data, &config)
	}

	// Merge top-level settings
	for k, v := range s.settings {
		config[k] = v
	}

	// Merge editor settings
	if len(s.editor) > 0 {
		editorSection, ok := config["editor"].(map[string]interface{})
		if !ok {
			editorSection = make(map[string]interface{})
		}
		for k, v := range s.editor {
			editorSection[k] = v
		}
		config["editor"] = editorSection
	}

	// Merge keys settings
	if len(s.keys) > 0 {
		keysSection, ok := config["keys"].(map[string]interface{})
		if !ok {
			keysSection = make(map[string]interface{})
		}
		for k, v := range s.keys {
			keysSection[k] = v
		}
		config["keys"] = keysSection
	}

	// Write config
	output, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *ConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	detail := "Configures Helix editor settings"
	if s.source != "" {
		if s.link {
			detail = fmt.Sprintf("Symlinks config.toml from %s", s.source)
		} else {
			detail = fmt.Sprintf("Copies config.toml from %s", s.source)
		}
	}

	return compiler.NewExplanation(
		"Configure Helix",
		detail,
		[]string{
			"https://docs.helix-editor.com/configuration.html",
		},
	).WithTradeoffs([]string{
		"+ Modal editing with modern defaults",
		"+ Built-in LSP support",
		"+ Minimal configuration required",
	})
}

// =============================================================================
// LanguagesStep - Manages languages.toml
// =============================================================================

// LanguagesStep represents a Helix languages.toml management step.
type LanguagesStep struct {
	source string
	link   bool
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewLanguagesStep creates a new LanguagesStep.
func NewLanguagesStep(source string, link bool, runner ports.CommandRunner) *LanguagesStep {
	return &LanguagesStep{
		source: source,
		link:   link,
		id:     compiler.MustNewStepID("helix:languages"),
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *LanguagesStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *LanguagesStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the languages config is already applied.
func (s *LanguagesStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	langPath := filepath.Join(getHelixConfigDir(), "languages.toml")

	info, err := os.Lstat(langPath)
	if err != nil {
		if os.IsNotExist(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	if s.link {
		if info.Mode()&os.ModeSymlink == 0 {
			return compiler.StatusNeedsApply, nil
		}
		target, err := os.Readlink(langPath)
		if err != nil {
			return compiler.StatusUnknown, err
		}
		absSource, _ := filepath.Abs(s.source)
		if target != absSource && target != s.source {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *LanguagesStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	action := "copy"
	if s.link {
		action = "link"
	}
	return compiler.NewDiff(compiler.DiffTypeModify, "config", "languages.toml", "", fmt.Sprintf("%s from %s", action, s.source)), nil
}

// Apply writes the languages config.
func (s *LanguagesStep) Apply(_ compiler.RunContext) error {
	configDir := getHelixConfigDir()
	langPath := filepath.Join(configDir, "languages.toml")

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Remove existing file/symlink
	_ = os.Remove(langPath)

	if s.link {
		absSource, err := filepath.Abs(s.source)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		if err := os.Symlink(absSource, langPath); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
		return nil
	}

	// Copy file
	content, err := os.ReadFile(s.source)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}
	if err := os.WriteFile(langPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write languages.toml: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *LanguagesStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	detail := fmt.Sprintf("Configures language-specific settings from %s", s.source)

	return compiler.NewExplanation(
		"Configure Helix Languages",
		detail,
		[]string{
			"https://docs.helix-editor.com/languages.html",
		},
	).WithTradeoffs([]string{
		"+ Customizes LSP and formatter per language",
		"+ Supports language injections",
	})
}

// =============================================================================
// ThemeStep - Manages theme installation
// =============================================================================

// ThemeStep represents a Helix theme installation step.
type ThemeStep struct {
	theme       string // Theme name to set in config
	themeSource string // Custom theme file to install
	id          compiler.StepID
	runner      ports.CommandRunner
}

// NewThemeStep creates a new ThemeStep.
func NewThemeStep(theme, themeSource string, runner ports.CommandRunner) *ThemeStep {
	return &ThemeStep{
		theme:       theme,
		themeSource: themeSource,
		id:          compiler.MustNewStepID("helix:theme"),
		runner:      runner,
	}
}

// ID returns the step identifier.
func (s *ThemeStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ThemeStep) DependsOn() []compiler.StepID {
	// Theme step should run after config step
	return []compiler.StepID{compiler.MustNewStepID("helix:config")}
}

// Check determines if the theme is already applied.
func (s *ThemeStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	// If we have a custom theme source, check if it's installed
	if s.themeSource != "" {
		themeName := filepath.Base(s.themeSource)
		themeName = themeName[:len(themeName)-len(filepath.Ext(themeName))]
		themePath := filepath.Join(getHelixConfigDir(), "themes", themeName+".toml")

		if !fileExists(themePath) {
			return compiler.StatusNeedsApply, nil
		}
	}

	// Check if theme is set in config
	configPath := filepath.Join(getHelixConfigDir(), "config.toml")
	if !fileExists(configPath) {
		return compiler.StatusNeedsApply, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	var config map[string]interface{}
	if err := toml.Unmarshal(data, &config); err != nil {
		// Config file is invalid or corrupted - needs to be rewritten
		return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: invalid config means needs apply
	}

	currentTheme, _ := config["theme"].(string)
	if currentTheme != s.theme && s.theme != "" {
		return compiler.StatusNeedsApply, nil
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *ThemeStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	themeName := s.theme
	if themeName == "" && s.themeSource != "" {
		themeName = filepath.Base(s.themeSource)
		themeName = themeName[:len(themeName)-len(filepath.Ext(themeName))]
	}
	return compiler.NewDiff(compiler.DiffTypeModify, "theme", themeName, "", fmt.Sprintf("set theme to %s", themeName)), nil
}

// Apply installs and sets the theme.
func (s *ThemeStep) Apply(_ compiler.RunContext) error {
	configDir := getHelixConfigDir()

	// Install custom theme if specified
	if s.themeSource != "" {
		themesDir := filepath.Join(configDir, "themes")
		if err := os.MkdirAll(themesDir, 0o755); err != nil {
			return fmt.Errorf("failed to create themes dir: %w", err)
		}

		themeName := filepath.Base(s.themeSource)
		themeName = themeName[:len(themeName)-len(filepath.Ext(themeName))]
		themePath := filepath.Join(themesDir, themeName+".toml")

		content, err := os.ReadFile(s.themeSource)
		if err != nil {
			return fmt.Errorf("failed to read theme source: %w", err)
		}
		if err := os.WriteFile(themePath, content, 0o644); err != nil {
			return fmt.Errorf("failed to write theme: %w", err)
		}
	}

	// Set theme in config.toml
	if s.theme != "" {
		configPath := filepath.Join(configDir, "config.toml")

		var config map[string]interface{}
		if data, err := os.ReadFile(configPath); err == nil {
			_ = toml.Unmarshal(data, &config)
		}
		if config == nil {
			config = make(map[string]interface{})
		}

		config["theme"] = s.theme

		output, err := toml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return fmt.Errorf("failed to create config dir: %w", err)
		}

		if err := os.WriteFile(configPath, output, 0o644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *ThemeStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	themeName := s.theme
	if themeName == "" && s.themeSource != "" {
		themeName = filepath.Base(s.themeSource)
	}

	return compiler.NewExplanation(
		"Set Helix Theme",
		fmt.Sprintf("Sets the editor theme to %s", themeName),
		[]string{
			"https://docs.helix-editor.com/themes.html",
		},
	).WithTradeoffs([]string{
		"+ Customizes editor appearance",
		"+ Many built-in themes available",
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
