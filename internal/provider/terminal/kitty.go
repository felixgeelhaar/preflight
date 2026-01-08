package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// KittyConfigStep manages Kitty configuration.
type KittyConfigStep struct {
	cfg        *KittyConfig
	globalCfg  *Config
	targetPath string
	configRoot string
	fs         ports.FileSystem
}

// NewKittyConfigStep creates a new Kitty config step.
func NewKittyConfigStep(cfg *KittyConfig, globalCfg *Config, targetPath, configRoot string, fs ports.FileSystem) *KittyConfigStep {
	return &KittyConfigStep{
		cfg:        cfg,
		globalCfg:  globalCfg,
		targetPath: pathutil.ExpandPath(targetPath),
		configRoot: configRoot,
		fs:         fs,
	}
}

// ID returns the step identifier.
func (s *KittyConfigStep) ID() compiler.StepID {
	return compiler.MustNewStepID("terminal:kitty:config")
}

// DependsOn returns dependencies.
func (s *KittyConfigStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if configuration needs to be applied.
func (s *KittyConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if s.cfg.Source != "" {
		return s.checkSourceMode()
	}
	return s.checkSettingsMode()
}

func (s *KittyConfigStep) checkSourceMode() (compiler.StepStatus, error) {
	sourcePath := filepath.Join(s.configRoot, s.cfg.Source)

	if s.cfg.Link {
		linkTarget, err := os.Readlink(s.targetPath)
		if err != nil {
			// Symlink doesn't exist or can't be read - needs to be created
			return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: missing symlink means needs apply
		}

		if !filepath.IsAbs(linkTarget) {
			linkTarget = filepath.Join(filepath.Dir(s.targetPath), linkTarget)
		}

		if linkTarget == sourcePath {
			return compiler.StatusSatisfied, nil
		}
		return compiler.StatusNeedsApply, nil
	}

	if !s.fs.Exists(s.targetPath) {
		return compiler.StatusNeedsApply, nil
	}

	srcHash, err := s.fs.FileHash(sourcePath)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	dstHash, err := s.fs.FileHash(s.targetPath)
	if err != nil {
		// Target file can't be read - needs to be created/updated
		return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: unreadable target means needs apply
	}

	if srcHash == dstHash {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

func (s *KittyConfigStep) checkSettingsMode() (compiler.StepStatus, error) {
	hasGlobalFont := s.globalCfg != nil && s.globalCfg.Font != nil
	if len(s.cfg.Settings) == 0 && !hasGlobalFont && s.cfg.Theme == "" {
		return compiler.StatusSatisfied, nil
	}

	if !s.fs.Exists(s.targetPath) {
		return compiler.StatusNeedsApply, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *KittyConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	if s.cfg.Source != "" {
		action := "copy"
		if s.cfg.Link {
			action = "symlink"
		}
		return compiler.NewDiff(
			compiler.DiffTypeModify,
			"config",
			s.targetPath,
			"",
			fmt.Sprintf("%s from %s", action, s.cfg.Source),
		), nil
	}

	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"config",
		s.targetPath,
		"",
		fmt.Sprintf("merge %d settings", len(s.cfg.Settings)),
	), nil
}

// Apply writes the configuration.
func (s *KittyConfigStep) Apply(_ compiler.RunContext) error {
	dir := filepath.Dir(s.targetPath)
	if err := s.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if s.cfg.Source != "" {
		return s.applySourceMode()
	}
	return s.applySettingsMode()
}

func (s *KittyConfigStep) applySourceMode() error {
	sourcePath := filepath.Join(s.configRoot, s.cfg.Source)

	if s.fs.Exists(s.targetPath) {
		if err := s.fs.Remove(s.targetPath); err != nil {
			return fmt.Errorf("failed to remove existing config: %w", err)
		}
	}

	if s.cfg.Link {
		return s.fs.CreateSymlink(sourcePath, s.targetPath)
	}
	return s.fs.CopyFile(sourcePath, s.targetPath)
}

func (s *KittyConfigStep) applySettingsMode() error {
	// Read existing config
	var lines []string
	if s.fs.Exists(s.targetPath) {
		content, err := s.fs.ReadFile(s.targetPath)
		if err == nil {
			lines = strings.Split(string(content), "\n")
		}
	}

	// Track which settings we've applied
	applied := make(map[string]bool)

	// Update existing lines
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.SplitN(trimmed, " ", 2)
		if len(parts) < 1 {
			continue
		}

		key := parts[0]
		if val, ok := s.cfg.Settings[key]; ok {
			lines[i] = fmt.Sprintf("%s %v", key, val)
			applied[key] = true
		}
	}

	// Add new settings
	for key, val := range s.cfg.Settings {
		if !applied[key] {
			lines = append(lines, fmt.Sprintf("%s %v", key, val))
		}
	}

	// Apply global font
	if s.globalCfg != nil && s.globalCfg.Font != nil {
		if !applied["font_family"] {
			lines = append(lines, fmt.Sprintf("font_family %s", s.globalCfg.Font.Family))
		}
		if !applied["font_size"] {
			lines = append(lines, fmt.Sprintf("font_size %.1f", s.globalCfg.Font.Size))
		}
	}

	// Apply theme (include theme_name)
	if s.cfg.Theme != "" && !applied["include"] {
		lines = append(lines, fmt.Sprintf("include themes/%s.conf", s.cfg.Theme))
	}

	content := strings.Join(lines, "\n")
	return s.fs.WriteFile(s.targetPath, []byte(content), 0o644)
}

// Explain provides context for this step.
func (s *KittyConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Kitty",
		"Manages Kitty terminal configuration",
		[]string{
			"https://sw.kovidgoyal.net/kitty/conf/",
			"https://github.com/kovidgoyal/kitty",
		},
	).WithTradeoffs([]string{
		"+ GPU-accelerated with OpenGL",
		"+ Built-in tabs and splits",
		"+ Extensive customization",
		"+ Image support in terminal",
	})
}

// LockInfo returns lock information for this step.
func (s *KittyConfigStep) LockInfo() compiler.LockInfo {
	return compiler.LockInfo{}
}

// InstalledVersion returns the installed version (N/A for config).
func (s *KittyConfigStep) InstalledVersion(_ compiler.RunContext) (string, error) {
	return "", nil
}
