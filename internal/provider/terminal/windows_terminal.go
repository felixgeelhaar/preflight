package terminal

import (
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// WindowsTerminalConfigStep manages Windows Terminal configuration.
type WindowsTerminalConfigStep struct {
	cfg        *WindowsTerminalConfig
	targetPath string
	fs         ports.FileSystem
}

// NewWindowsTerminalConfigStep creates a new Windows Terminal config step.
func NewWindowsTerminalConfigStep(cfg *WindowsTerminalConfig, targetPath string, fs ports.FileSystem) *WindowsTerminalConfigStep {
	return &WindowsTerminalConfigStep{
		cfg:        cfg,
		targetPath: pathutil.ExpandPath(targetPath),
		fs:         fs,
	}
}

// ID returns the step identifier.
func (s *WindowsTerminalConfigStep) ID() compiler.StepID {
	return compiler.MustNewStepID("terminal:windows-terminal:config")
}

// DependsOn returns dependencies.
func (s *WindowsTerminalConfigStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if configuration needs to be applied.
func (s *WindowsTerminalConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	hasChanges := len(s.cfg.Settings) > 0 ||
		len(s.cfg.Profiles) > 0 ||
		len(s.cfg.Schemes) > 0

	if !hasChanges {
		return compiler.StatusSatisfied, nil
	}

	if !s.fs.Exists(s.targetPath) {
		return compiler.StatusNeedsApply, nil
	}

	// Read existing config and compare
	existing, err := s.readConfig()
	if err != nil {
		return compiler.StatusNeedsApply, nil
	}

	// Check if settings need update
	for key, value := range s.cfg.Settings {
		if fmt.Sprintf("%v", existing[key]) != fmt.Sprintf("%v", value) {
			return compiler.StatusNeedsApply, nil
		}
	}

	// Check profiles and schemes would require more complex comparison
	// For simplicity, if profiles or schemes are configured, mark as needs-apply
	if len(s.cfg.Profiles) > 0 || len(s.cfg.Schemes) > 0 {
		return compiler.StatusNeedsApply, nil
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *WindowsTerminalConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	hasChanges := len(s.cfg.Settings) > 0 ||
		len(s.cfg.Profiles) > 0 ||
		len(s.cfg.Schemes) > 0

	if !hasChanges {
		return compiler.NewDiff(
			compiler.DiffTypeNone,
			"config",
			s.targetPath,
			"",
			"no changes",
		), nil
	}

	var desc string
	if len(s.cfg.Settings) > 0 {
		desc = fmt.Sprintf("update %d settings", len(s.cfg.Settings))
	}
	if len(s.cfg.Profiles) > 0 {
		if desc != "" {
			desc += ", "
		}
		desc += fmt.Sprintf("%d profiles", len(s.cfg.Profiles))
	}
	if len(s.cfg.Schemes) > 0 {
		if desc != "" {
			desc += ", "
		}
		desc += fmt.Sprintf("%d color schemes", len(s.cfg.Schemes))
	}

	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"config",
		s.targetPath,
		"",
		desc,
	), nil
}

// Apply writes the configuration.
func (s *WindowsTerminalConfigStep) Apply(_ compiler.RunContext) error {
	hasChanges := len(s.cfg.Settings) > 0 ||
		len(s.cfg.Profiles) > 0 ||
		len(s.cfg.Schemes) > 0

	if !hasChanges {
		return nil
	}

	// Read existing config or create new one
	existing, _ := s.readConfig()
	if existing == nil {
		existing = make(map[string]interface{})
	}

	// Merge settings
	for key, value := range s.cfg.Settings {
		existing[key] = value
	}

	// Add/update profiles
	if len(s.cfg.Profiles) > 0 {
		if err := s.mergeProfiles(existing); err != nil {
			return err
		}
	}

	// Add color schemes
	if len(s.cfg.Schemes) > 0 {
		if err := s.mergeSchemes(existing); err != nil {
			return err
		}
	}

	return s.writeConfig(existing)
}

// readConfig reads the existing settings.json.
func (s *WindowsTerminalConfigStep) readConfig() (map[string]interface{}, error) {
	content, err := s.fs.ReadFile(s.targetPath)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// writeConfig writes the settings.json.
func (s *WindowsTerminalConfigStep) writeConfig(config map[string]interface{}) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return s.fs.WriteFile(s.targetPath, data, 0o644)
}

// mergeProfiles merges profiles into the config.
func (s *WindowsTerminalConfigStep) mergeProfiles(config map[string]interface{}) error {
	profiles, ok := config["profiles"].(map[string]interface{})
	if !ok {
		profiles = map[string]interface{}{
			"defaults": map[string]interface{}{},
			"list":     []interface{}{},
		}
		config["profiles"] = profiles
	}

	list, ok := profiles["list"].([]interface{})
	if !ok {
		list = []interface{}{}
	}

	for _, p := range s.cfg.Profiles {
		profile := map[string]interface{}{
			"name": p.Name,
		}
		if p.GUID != "" {
			profile["guid"] = p.GUID
		}
		if p.CommandLine != "" {
			profile["commandline"] = p.CommandLine
		}
		if p.ColorScheme != "" {
			profile["colorScheme"] = p.ColorScheme
		}
		if p.FontFace != "" {
			profile["font"] = map[string]interface{}{
				"face": p.FontFace,
			}
			if p.FontSize > 0 {
				profile["font"].(map[string]interface{})["size"] = p.FontSize
			}
		}
		if p.UseAcrylic != nil {
			profile["useAcrylic"] = *p.UseAcrylic
		}
		if p.AcrylicOpacity != nil {
			profile["acrylicOpacity"] = *p.AcrylicOpacity
		}

		// Find and update existing profile or append
		found := false
		for i, existing := range list {
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if existingMap["name"] == p.Name || (p.GUID != "" && existingMap["guid"] == p.GUID) {
					// Merge into existing
					for k, v := range profile {
						existingMap[k] = v
					}
					list[i] = existingMap
					found = true
					break
				}
			}
		}
		if !found {
			list = append(list, profile)
		}
	}

	profiles["list"] = list
	return nil
}

// mergeSchemes merges color schemes into the config.
func (s *WindowsTerminalConfigStep) mergeSchemes(config map[string]interface{}) error {
	schemes, ok := config["schemes"].([]interface{})
	if !ok {
		schemes = []interface{}{}
	}

	for _, cs := range s.cfg.Schemes {
		scheme := map[string]interface{}{
			"name":       cs.Name,
			"background": cs.Background,
			"foreground": cs.Foreground,
		}

		// Add optional colors
		if cs.Black != "" {
			scheme["black"] = cs.Black
		}
		if cs.Red != "" {
			scheme["red"] = cs.Red
		}
		if cs.Green != "" {
			scheme["green"] = cs.Green
		}
		if cs.Yellow != "" {
			scheme["yellow"] = cs.Yellow
		}
		if cs.Blue != "" {
			scheme["blue"] = cs.Blue
		}
		if cs.Purple != "" {
			scheme["purple"] = cs.Purple
		}
		if cs.Cyan != "" {
			scheme["cyan"] = cs.Cyan
		}
		if cs.White != "" {
			scheme["white"] = cs.White
		}
		if cs.BrightBlack != "" {
			scheme["brightBlack"] = cs.BrightBlack
		}
		if cs.BrightRed != "" {
			scheme["brightRed"] = cs.BrightRed
		}
		if cs.BrightGreen != "" {
			scheme["brightGreen"] = cs.BrightGreen
		}
		if cs.BrightYellow != "" {
			scheme["brightYellow"] = cs.BrightYellow
		}
		if cs.BrightBlue != "" {
			scheme["brightBlue"] = cs.BrightBlue
		}
		if cs.BrightPurple != "" {
			scheme["brightPurple"] = cs.BrightPurple
		}
		if cs.BrightCyan != "" {
			scheme["brightCyan"] = cs.BrightCyan
		}
		if cs.BrightWhite != "" {
			scheme["brightWhite"] = cs.BrightWhite
		}
		if cs.CursorColor != "" {
			scheme["cursorColor"] = cs.CursorColor
		}
		if cs.SelectionColor != "" {
			scheme["selectionBackground"] = cs.SelectionColor
		}

		// Find and update existing scheme or append
		found := false
		for i, existing := range schemes {
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if existingMap["name"] == cs.Name {
					schemes[i] = scheme
					found = true
					break
				}
			}
		}
		if !found {
			schemes = append(schemes, scheme)
		}
	}

	config["schemes"] = schemes
	return nil
}

// Explain provides context for this step.
func (s *WindowsTerminalConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Windows Terminal",
		"Manages Windows Terminal settings.json configuration",
		[]string{
			"https://docs.microsoft.com/en-us/windows/terminal/",
			"https://docs.microsoft.com/en-us/windows/terminal/customize-settings/startup",
		},
	).WithTradeoffs([]string{
		"+ Modern GPU-accelerated terminal for Windows",
		"+ JSON configuration (easy to version control)",
		"+ Multiple profiles and tabs",
		"+ Rich color scheme support",
		"- Windows only",
	})
}

// LockInfo returns lock information for this step.
func (s *WindowsTerminalConfigStep) LockInfo() compiler.LockInfo {
	return compiler.LockInfo{}
}

// InstalledVersion returns the installed version (N/A for config).
func (s *WindowsTerminalConfigStep) InstalledVersion(_ compiler.RunContext) (string, error) {
	return "", nil
}
