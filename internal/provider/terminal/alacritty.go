package terminal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// AlacrittyConfigStep manages Alacritty configuration.
type AlacrittyConfigStep struct {
	cfg        *AlacrittyConfig
	globalCfg  *Config
	targetPath string
	configRoot string
	fs         ports.FileSystem
}

// NewAlacrittyConfigStep creates a new Alacritty config step.
func NewAlacrittyConfigStep(cfg *AlacrittyConfig, globalCfg *Config, targetPath, configRoot string, fs ports.FileSystem) *AlacrittyConfigStep {
	return &AlacrittyConfigStep{
		cfg:        cfg,
		globalCfg:  globalCfg,
		targetPath: pathutil.ExpandPath(targetPath),
		configRoot: configRoot,
		fs:         fs,
	}
}

// ID returns the step identifier.
func (s *AlacrittyConfigStep) ID() compiler.StepID {
	return compiler.MustNewStepID("terminal:alacritty:config")
}

// DependsOn returns dependencies.
func (s *AlacrittyConfigStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if configuration needs to be applied.
func (s *AlacrittyConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if s.cfg.Source != "" {
		return s.checkSourceMode()
	}
	return s.checkSettingsMode()
}

func (s *AlacrittyConfigStep) checkSourceMode() (compiler.StepStatus, error) {
	sourcePath := filepath.Join(s.configRoot, s.cfg.Source)

	if s.cfg.Link {
		// Check if symlink exists and points to correct target
		linkTarget, err := os.Readlink(s.targetPath)
		if err != nil {
			return compiler.StatusNeedsApply, nil
		}

		// Resolve to absolute path for comparison
		if !filepath.IsAbs(linkTarget) {
			linkTarget = filepath.Join(filepath.Dir(s.targetPath), linkTarget)
		}

		if linkTarget == sourcePath {
			return compiler.StatusSatisfied, nil
		}
		return compiler.StatusNeedsApply, nil
	}

	// Copy mode - compare file hashes
	if !s.fs.Exists(s.targetPath) {
		return compiler.StatusNeedsApply, nil
	}

	srcHash, err := s.fs.FileHash(sourcePath)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	dstHash, err := s.fs.FileHash(s.targetPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil
	}

	if srcHash == dstHash {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

func (s *AlacrittyConfigStep) checkSettingsMode() (compiler.StepStatus, error) {
	// If no settings and no global font/theme, nothing to do
	hasGlobalFont := s.globalCfg != nil && s.globalCfg.Font != nil
	hasGlobalTheme := s.globalCfg != nil && s.globalCfg.Theme != nil
	if len(s.cfg.Settings) == 0 && !hasGlobalFont && !hasGlobalTheme {
		return compiler.StatusSatisfied, nil
	}

	if !s.fs.Exists(s.targetPath) {
		return compiler.StatusNeedsApply, nil
	}

	// For now, always apply settings mode (could add deep comparison later)
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *AlacrittyConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
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
func (s *AlacrittyConfigStep) Apply(_ compiler.RunContext) error {
	// Ensure directory exists
	dir := filepath.Dir(s.targetPath)
	if err := s.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if s.cfg.Source != "" {
		return s.applySourceMode()
	}
	return s.applySettingsMode()
}

func (s *AlacrittyConfigStep) applySourceMode() error {
	sourcePath := filepath.Join(s.configRoot, s.cfg.Source)

	// Remove existing file/symlink
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

func (s *AlacrittyConfigStep) applySettingsMode() error {
	// Read existing config or start fresh
	existing := make(map[string]interface{})
	if s.fs.Exists(s.targetPath) {
		content, err := s.fs.ReadFile(s.targetPath)
		if err == nil {
			_ = toml.Unmarshal(content, &existing)
		}
	}

	// Merge settings
	for k, v := range s.cfg.Settings {
		existing[k] = v
	}

	// Apply global font if set
	if s.globalCfg != nil && s.globalCfg.Font != nil {
		fontConfig := map[string]interface{}{
			"normal": map[string]interface{}{
				"family": s.globalCfg.Font.Family,
			},
			"size": s.globalCfg.Font.Size,
		}
		existing["font"] = fontConfig
	}

	// Write TOML
	output, err := toml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return s.fs.WriteFile(s.targetPath, output, 0o644)
}

// Explain provides context for this step.
func (s *AlacrittyConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Alacritty",
		"Manages Alacritty terminal configuration",
		[]string{
			"https://alacritty.org/config-alacritty.html",
			"https://github.com/alacritty/alacritty",
		},
	).WithTradeoffs([]string{
		"+ GPU-accelerated, fast terminal",
		"+ Cross-platform (macOS, Linux, Windows)",
		"+ TOML configuration is human-readable",
		"- No tabs or splits (use tmux)",
	})
}

// LockInfo returns lock information for this step.
func (s *AlacrittyConfigStep) LockInfo() compiler.LockInfo {
	return compiler.LockInfo{}
}

// InstalledVersion returns the installed version (N/A for config).
func (s *AlacrittyConfigStep) InstalledVersion(_ compiler.RunContext) (string, error) {
	return "", nil
}
