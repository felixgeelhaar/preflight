package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// ITerm2SettingsStep manages iTerm2 preferences via defaults command.
type ITerm2SettingsStep struct {
	cfg    *ITerm2Config
	runner ports.CommandRunner
}

// NewITerm2SettingsStep creates a new iTerm2 settings step.
func NewITerm2SettingsStep(cfg *ITerm2Config, runner ports.CommandRunner) *ITerm2SettingsStep {
	return &ITerm2SettingsStep{
		cfg:    cfg,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *ITerm2SettingsStep) ID() compiler.StepID {
	return compiler.MustNewStepID("terminal:iterm2:settings")
}

// DependsOn returns dependencies.
func (s *ITerm2SettingsStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if settings need to be applied.
func (s *ITerm2SettingsStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if len(s.cfg.Settings) == 0 {
		return compiler.StatusSatisfied, nil
	}

	for key, value := range s.cfg.Settings {
		current, err := s.readDefault(key)
		if err != nil {
			// Setting doesn't exist or can't be read - needs to be applied
			return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: missing setting means needs apply
		}
		if current != fmt.Sprintf("%v", value) {
			return compiler.StatusNeedsApply, nil
		}
	}

	return compiler.StatusSatisfied, nil
}

// readDefault reads a defaults value for iTerm2.
func (s *ITerm2SettingsStep) readDefault(key string) (string, error) {
	result, err := s.runner.Run(context.Background(), "defaults", "read", "com.googlecode.iterm2", key)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// Plan returns the diff for this step.
func (s *ITerm2SettingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	if len(s.cfg.Settings) == 0 {
		return compiler.NewDiff(
			compiler.DiffTypeNone,
			"settings",
			"com.googlecode.iterm2",
			"",
			"no changes",
		), nil
	}

	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"settings",
		"com.googlecode.iterm2",
		"",
		fmt.Sprintf("update %d preferences", len(s.cfg.Settings)),
	), nil
}

// Apply writes the settings via defaults command.
func (s *ITerm2SettingsStep) Apply(_ compiler.RunContext) error {
	for key, value := range s.cfg.Settings {
		var typeFlag string
		var valueStr string

		switch v := value.(type) {
		case bool:
			typeFlag = "-bool"
			if v {
				valueStr = "YES"
			} else {
				valueStr = "NO"
			}
		case int, int64, float64:
			typeFlag = "-float"
			valueStr = fmt.Sprintf("%v", v)
		case string:
			typeFlag = "-string"
			valueStr = v
		default:
			typeFlag = "-string"
			valueStr = fmt.Sprintf("%v", v)
		}

		_, err := s.runner.Run(context.Background(), "defaults", "write", "com.googlecode.iterm2", key, typeFlag, valueStr)
		if err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}
	return nil
}

// Explain provides context for this step.
func (s *ITerm2SettingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure iTerm2 Settings",
		"Manages iTerm2 preferences via macOS defaults system",
		[]string{
			"https://iterm2.com/documentation-preferences.html",
			"https://github.com/gnachman/iTerm2",
		},
	).WithTradeoffs([]string{
		"+ Native macOS integration",
		"+ Extensive customization options",
		"+ Triggers, profiles, and automation",
		"- macOS only",
		"- Some settings require app restart",
	})
}

// LockInfo returns lock information for this step.
func (s *ITerm2SettingsStep) LockInfo() compiler.LockInfo {
	return compiler.LockInfo{}
}

// InstalledVersion returns the installed version (N/A for config).
func (s *ITerm2SettingsStep) InstalledVersion(_ compiler.RunContext) (string, error) {
	return "", nil
}

// ITerm2ProfilesStep manages iTerm2 dynamic profiles.
type ITerm2ProfilesStep struct {
	cfg         *ITerm2Config
	profilesDir string
	fs          ports.FileSystem
}

// NewITerm2ProfilesStep creates a new iTerm2 profiles step.
func NewITerm2ProfilesStep(cfg *ITerm2Config, profilesDir string, fs ports.FileSystem) *ITerm2ProfilesStep {
	return &ITerm2ProfilesStep{
		cfg:         cfg,
		profilesDir: profilesDir,
		fs:          fs,
	}
}

// ID returns the step identifier.
func (s *ITerm2ProfilesStep) ID() compiler.StepID {
	return compiler.MustNewStepID("terminal:iterm2:profiles")
}

// DependsOn returns dependencies.
func (s *ITerm2ProfilesStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if profiles need to be applied.
func (s *ITerm2ProfilesStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if len(s.cfg.DynamicProfiles) == 0 {
		return compiler.StatusSatisfied, nil
	}

	profilePath := filepath.Join(s.profilesDir, "preflight-profiles.json")
	if !s.fs.Exists(profilePath) {
		return compiler.StatusNeedsApply, nil
	}

	// Compare existing profiles with desired
	content, err := s.fs.ReadFile(profilePath)
	if err != nil {
		// Profile file can't be read - needs to be created/updated
		return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: unreadable profile means needs apply
	}

	desired, err := s.generateProfilesJSON()
	if err != nil {
		return compiler.StatusUnknown, err
	}

	if string(content) == desired {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ITerm2ProfilesStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	if len(s.cfg.DynamicProfiles) == 0 {
		return compiler.NewDiff(
			compiler.DiffTypeNone,
			"profiles",
			s.profilesDir,
			"",
			"no changes",
		), nil
	}

	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"profiles",
		filepath.Join(s.profilesDir, "preflight-profiles.json"),
		"",
		fmt.Sprintf("create/update %d dynamic profiles", len(s.cfg.DynamicProfiles)),
	), nil
}

// Apply creates the dynamic profiles file.
func (s *ITerm2ProfilesStep) Apply(_ compiler.RunContext) error {
	if len(s.cfg.DynamicProfiles) == 0 {
		return nil
	}

	if err := s.fs.MkdirAll(s.profilesDir, 0o755); err != nil {
		return fmt.Errorf("failed to create profiles directory: %w", err)
	}

	content, err := s.generateProfilesJSON()
	if err != nil {
		return err
	}

	profilePath := filepath.Join(s.profilesDir, "preflight-profiles.json")
	return s.fs.WriteFile(profilePath, []byte(content), 0o644)
}

// generateProfilesJSON generates the dynamic profiles JSON.
func (s *ITerm2ProfilesStep) generateProfilesJSON() (string, error) {
	profiles := make([]map[string]interface{}, len(s.cfg.DynamicProfiles))

	for i, p := range s.cfg.DynamicProfiles {
		profile := map[string]interface{}{
			"Name": p.Name,
		}

		if p.GUID != "" {
			profile["Guid"] = p.GUID
		}
		if p.Font != "" {
			profile["Normal Font"] = p.Font
		}
		if p.FontSize > 0 {
			profile["Normal Font Size"] = p.FontSize
		}
		if p.ColorScheme != "" {
			profile["Color Preset"] = p.ColorScheme
		}
		for k, v := range p.Custom {
			profile[k] = v
		}

		profiles[i] = profile
	}

	wrapper := map[string]interface{}{
		"Profiles": profiles,
	}

	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal profiles: %w", err)
	}

	return string(data), nil
}

// Explain provides context for this step.
func (s *ITerm2ProfilesStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure iTerm2 Dynamic Profiles",
		"Creates iTerm2 dynamic profiles for easy sharing and versioning",
		[]string{
			"https://iterm2.com/documentation-dynamic-profiles.html",
		},
	).WithTradeoffs([]string{
		"+ Profiles are version-controlled",
		"+ Easy to share across machines",
		"+ Auto-loaded by iTerm2",
		"- Dynamic profiles have some limitations vs regular profiles",
	})
}

// LockInfo returns lock information for this step.
func (s *ITerm2ProfilesStep) LockInfo() compiler.LockInfo {
	return compiler.LockInfo{}
}

// InstalledVersion returns the installed version (N/A for config).
func (s *ITerm2ProfilesStep) InstalledVersion(_ compiler.RunContext) (string, error) {
	return "", nil
}
