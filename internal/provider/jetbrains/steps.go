package jetbrains

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// =============================================================================
// PluginStep - Manages plugin installation for an IDE
// =============================================================================

// PluginStep represents a JetBrains plugin installation step.
type PluginStep struct {
	ide     IDE
	plugins []string
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewPluginStep creates a new PluginStep.
func NewPluginStep(ide IDE, plugins []string, runner ports.CommandRunner) *PluginStep {
	safeIDE := strings.ToLower(string(ide))
	return &PluginStep{
		ide:     ide,
		plugins: plugins,
		id:      compiler.MustNewStepID(fmt.Sprintf("jetbrains:%s:plugins", safeIDE)),
		runner:  runner,
	}
}

// ID returns the step identifier.
func (s *PluginStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *PluginStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the plugins are already installed.
func (s *PluginStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	discovery := NewDiscovery()
	pluginsDir := discovery.FindPluginsDir(s.ide)

	// Check if plugins directory exists
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return compiler.StatusNeedsApply, nil
		}
		return compiler.StatusUnknown, err
	}

	// Create set of installed plugins (folder names)
	installedSet := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			installedSet[strings.ToLower(entry.Name())] = true
		}
	}

	// Check if all desired plugins are installed
	for _, plugin := range s.plugins {
		// Plugin names might not match exactly, check for partial match
		pluginLower := strings.ToLower(plugin)
		found := false
		for installed := range installedSet {
			if strings.Contains(installed, pluginLower) || strings.Contains(pluginLower, installed) {
				found = true
				break
			}
		}
		if !found {
			return compiler.StatusNeedsApply, nil
		}
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *PluginStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"plugins",
		fmt.Sprintf("%s plugins", s.ide),
		"",
		fmt.Sprintf("%d plugins", len(s.plugins)),
	), nil
}

// Apply records the desired plugins (actual installation happens via IDE).
func (s *PluginStep) Apply(_ compiler.RunContext) error {
	discovery := NewDiscovery()
	configDir := discovery.FindConfigDir(s.ide)
	optionsDir := filepath.Join(configDir, "options")

	// Ensure options directory exists
	if err := os.MkdirAll(optionsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create options dir: %w", err)
	}

	// Create/update installed.txt to record desired plugins
	// This is a reference file - actual plugin installation requires IDE
	installedPath := filepath.Join(optionsDir, "preflight-plugins.txt")

	content := strings.Join(s.plugins, "\n")
	if err := os.WriteFile(installedPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write plugins file: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *PluginStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		fmt.Sprintf("Configure %s Plugins", s.ide),
		fmt.Sprintf("Records %d plugins to install for %s", len(s.plugins), s.ide),
		[]string{
			"https://plugins.jetbrains.com/",
			"https://www.jetbrains.com/help/idea/managing-plugins.html",
		},
	).WithTradeoffs([]string{
		"+ Declarative plugin management",
		"+ Works across all JetBrains IDEs",
		"- Requires IDE restart to apply",
		"- Some plugins need manual installation",
	})
}

// =============================================================================
// SettingsStep - Manages IDE-specific settings
// =============================================================================

// SettingsStep represents a JetBrains settings configuration step.
type SettingsStep struct {
	ide       IDE
	settings  map[string]interface{}
	keymap    string
	codeStyle string
	id        compiler.StepID
	runner    ports.CommandRunner
}

// NewSettingsStep creates a new SettingsStep.
func NewSettingsStep(ide IDE, settings map[string]interface{}, keymap, codeStyle string, runner ports.CommandRunner) *SettingsStep {
	safeIDE := strings.ToLower(string(ide))
	return &SettingsStep{
		ide:       ide,
		settings:  settings,
		keymap:    keymap,
		codeStyle: codeStyle,
		id:        compiler.MustNewStepID(fmt.Sprintf("jetbrains:%s:settings", safeIDE)),
		runner:    runner,
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
	optionsDir := discovery.FindOptionsDir(s.ide)

	// Check if options directory exists
	if _, err := os.Stat(optionsDir); os.IsNotExist(err) {
		return compiler.StatusNeedsApply, nil
	}

	// For simplicity, always mark as needs apply if we have settings
	if len(s.settings) > 0 || s.keymap != "" || s.codeStyle != "" {
		return compiler.StatusNeedsApply, nil
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *SettingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	count := len(s.settings)
	if s.keymap != "" {
		count++
	}
	if s.codeStyle != "" {
		count++
	}
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"settings",
		fmt.Sprintf("%s settings", s.ide),
		"",
		fmt.Sprintf("%d settings", count),
	), nil
}

// Apply writes the settings.
func (s *SettingsStep) Apply(_ compiler.RunContext) error {
	discovery := NewDiscovery()
	optionsDir := discovery.FindOptionsDir(s.ide)

	// Ensure options directory exists
	if err := os.MkdirAll(optionsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create options dir: %w", err)
	}

	// Write keymap selection if specified
	if s.keymap != "" {
		keymapConfig := fmt.Sprintf(`<application>
  <component name="KeymapManager">
    <active_keymap name="%s" />
  </component>
</application>`, s.keymap)

		keymapPath := filepath.Join(optionsDir, "keymap.xml")
		if err := os.WriteFile(keymapPath, []byte(keymapConfig), 0o644); err != nil {
			return fmt.Errorf("failed to write keymap config: %w", err)
		}
	}

	// Write code style selection if specified
	if s.codeStyle != "" {
		codeStyleConfig := fmt.Sprintf(`<application>
  <component name="CodeStyleSettingsManager">
    <option name="USE_PER_PROJECT_SETTINGS" value="false" />
    <option name="PREFERRED_PROJECT_CODE_STYLE" value="%s" />
  </component>
</application>`, s.codeStyle)

		codeStylePath := filepath.Join(optionsDir, "code.style.schemes.xml")
		if err := os.WriteFile(codeStylePath, []byte(codeStyleConfig), 0o644); err != nil {
			return fmt.Errorf("failed to write code style config: %w", err)
		}
	}

	// Write custom settings if specified
	if len(s.settings) > 0 {
		// Create a generic options file for custom settings
		var settingsEntries []string
		for key, value := range s.settings {
			settingsEntries = append(settingsEntries, fmt.Sprintf(`    <option name="%s" value="%v" />`, key, value))
		}

		settingsConfig := fmt.Sprintf(`<application>
  <component name="PreflightSettings">
%s
  </component>
</application>`, strings.Join(settingsEntries, "\n"))

		settingsPath := filepath.Join(optionsDir, "preflight.xml")
		if err := os.WriteFile(settingsPath, []byte(settingsConfig), 0o644); err != nil {
			return fmt.Errorf("failed to write settings config: %w", err)
		}
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *SettingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		fmt.Sprintf("Configure %s Settings", s.ide),
		fmt.Sprintf("Configures settings for %s IDE", s.ide),
		[]string{
			"https://www.jetbrains.com/help/idea/configuring-project-and-ide-settings.html",
		},
	).WithTradeoffs([]string{
		"+ Declarative configuration",
		"+ Settings preserved across updates",
		"- Some settings require IDE restart",
	})
}

// =============================================================================
// SettingsSyncStep - Manages JetBrains Settings Sync
// =============================================================================

// SettingsSyncStep represents a JetBrains Settings Sync configuration step.
type SettingsSyncStep struct {
	ide       IDE
	config    *SettingsSyncConfig
	id        compiler.StepID
	runner    ports.CommandRunner
}

// NewSettingsSyncStep creates a new SettingsSyncStep.
func NewSettingsSyncStep(ide IDE, config *SettingsSyncConfig, runner ports.CommandRunner) *SettingsSyncStep {
	safeIDE := strings.ToLower(string(ide))
	return &SettingsSyncStep{
		ide:    ide,
		config: config,
		id:     compiler.MustNewStepID(fmt.Sprintf("jetbrains:%s:settingssync", safeIDE)),
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *SettingsSyncStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *SettingsSyncStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if settings sync is already configured.
func (s *SettingsSyncStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	discovery := NewDiscovery()
	optionsDir := discovery.FindOptionsDir(s.ide)

	settingsSyncPath := filepath.Join(optionsDir, "settingsSync.xml")
	if _, err := os.Stat(settingsSyncPath); os.IsNotExist(err) {
		return compiler.StatusNeedsApply, nil
	}

	// For simplicity, always check config matches
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *SettingsSyncStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	status := "disabled"
	if s.config.Enabled {
		status = "enabled"
	}
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"settingssync",
		fmt.Sprintf("%s Settings Sync", s.ide),
		"",
		status,
	), nil
}

// settingsSyncXML represents the XML structure for Settings Sync configuration.
type settingsSyncXML struct {
	XMLName   xml.Name             `xml:"application"`
	Component settingsSyncComponent `xml:"component"`
}

type settingsSyncComponent struct {
	Name    string                `xml:"name,attr"`
	Options []settingsSyncOption  `xml:"option"`
}

type settingsSyncOption struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Apply configures settings sync.
func (s *SettingsSyncStep) Apply(_ compiler.RunContext) error {
	discovery := NewDiscovery()
	optionsDir := discovery.FindOptionsDir(s.ide)

	// Ensure options directory exists
	if err := os.MkdirAll(optionsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create options dir: %w", err)
	}

	// Create settings sync configuration
	syncConfig := settingsSyncXML{
		Component: settingsSyncComponent{
			Name: "SettingsSyncSettings",
			Options: []settingsSyncOption{
				{Name: "syncEnabled", Value: fmt.Sprintf("%t", s.config.Enabled)},
				{Name: "syncPlugins", Value: fmt.Sprintf("%t", s.config.SyncPlugins)},
				{Name: "syncUI", Value: fmt.Sprintf("%t", s.config.SyncUI)},
				{Name: "syncCodeStyles", Value: fmt.Sprintf("%t", s.config.SyncCodeStyles)},
				{Name: "syncKeymaps", Value: fmt.Sprintf("%t", s.config.SyncKeymaps)},
			},
		},
	}

	output, err := xml.MarshalIndent(syncConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings sync config: %w", err)
	}

	settingsSyncPath := filepath.Join(optionsDir, "settingsSync.xml")
	xmlContent := xml.Header + string(output)
	if err := os.WriteFile(settingsSyncPath, []byte(xmlContent), 0o644); err != nil {
		return fmt.Errorf("failed to write settings sync config: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *SettingsSyncStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		fmt.Sprintf("Configure %s Settings Sync", s.ide),
		"Configures JetBrains Settings Sync for cloud-based settings synchronization",
		[]string{
			"https://www.jetbrains.com/help/idea/sharing-your-ide-settings.html#IDE_settings_sync",
		},
	).WithTradeoffs([]string{
		"+ Settings sync across machines",
		"+ Automatic backup of settings",
		"- Requires JetBrains account",
	})
}
