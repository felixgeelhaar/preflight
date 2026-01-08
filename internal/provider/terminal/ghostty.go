package terminal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// GhosttyConfigStep manages Ghostty configuration.
type GhosttyConfigStep struct {
	cfg        *GhosttyConfig
	targetPath string
	configRoot string
	fs         ports.FileSystem
}

// NewGhosttyConfigStep creates a new Ghostty config step.
func NewGhosttyConfigStep(cfg *GhosttyConfig, targetPath, configRoot string, fs ports.FileSystem) *GhosttyConfigStep {
	return &GhosttyConfigStep{
		cfg:        cfg,
		targetPath: pathutil.ExpandPath(targetPath),
		configRoot: configRoot,
		fs:         fs,
	}
}

// ID returns the step identifier.
func (s *GhosttyConfigStep) ID() compiler.StepID {
	return compiler.MustNewStepID("terminal:ghostty:config")
}

// DependsOn returns dependencies.
func (s *GhosttyConfigStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if configuration needs to be applied.
func (s *GhosttyConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if s.cfg.Source != "" {
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

		// Copy mode
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

	// Settings merge mode
	if len(s.cfg.Settings) == 0 {
		return compiler.StatusSatisfied, nil
	}

	if !s.fs.Exists(s.targetPath) {
		return compiler.StatusNeedsApply, nil
	}

	existing, err := s.readGhosttyConfig()
	if err != nil {
		// Config file doesn't exist or can't be parsed - needs to be created
		return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: missing/invalid config means needs apply
	}

	for key, value := range s.cfg.Settings {
		if fmt.Sprintf("%v", existing[key]) != fmt.Sprintf("%v", value) {
			return compiler.StatusNeedsApply, nil
		}
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *GhosttyConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
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

	if len(s.cfg.Settings) == 0 {
		return compiler.NewDiff(
			compiler.DiffTypeNone,
			"config",
			s.targetPath,
			"",
			"no changes",
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
func (s *GhosttyConfigStep) Apply(_ compiler.RunContext) error {
	dir := filepath.Dir(s.targetPath)
	if err := s.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if s.cfg.Source != "" {
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

	// Settings merge mode
	if len(s.cfg.Settings) == 0 {
		return nil
	}

	existing, _ := s.readGhosttyConfig()
	if existing == nil {
		existing = make(map[string]interface{})
	}

	for key, value := range s.cfg.Settings {
		existing[key] = value
	}

	return s.writeGhosttyConfig(existing)
}

// readGhosttyConfig reads the existing config file.
func (s *GhosttyConfigStep) readGhosttyConfig() (map[string]interface{}, error) {
	content, err := s.fs.ReadFile(s.targetPath)
	if err != nil {
		return nil, err
	}

	settings := make(map[string]interface{})
	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			settings[key] = value
		}
	}

	return settings, scanner.Err()
}

// writeGhosttyConfig writes the config file.
func (s *GhosttyConfigStep) writeGhosttyConfig(settings map[string]interface{}) error {
	// Pre-allocate: header + empty + settings + trailing empty
	lines := make([]string, 0, len(settings)+3)
	lines = append(lines, "# Ghostty configuration managed by preflight")
	lines = append(lines, "")

	// Sort keys for deterministic output
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s = %v", key, settings[key]))
	}
	lines = append(lines, "")

	return s.fs.WriteFile(s.targetPath, []byte(strings.Join(lines, "\n")), 0o644)
}

// Explain provides context for this step.
func (s *GhosttyConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Ghostty",
		"Manages Ghostty terminal configuration",
		[]string{
			"https://ghostty.org/docs/config",
			"https://github.com/ghostty-org/ghostty",
		},
	).WithTradeoffs([]string{
		"+ Native performance (written in Zig)",
		"+ Modern GPU rendering",
		"+ Simple key=value configuration",
		"+ Cross-platform (macOS, Linux)",
		"- Relatively new project",
	})
}

// LockInfo returns lock information for this step.
func (s *GhosttyConfigStep) LockInfo() compiler.LockInfo {
	return compiler.LockInfo{}
}

// InstalledVersion returns the installed version (N/A for config).
func (s *GhosttyConfigStep) InstalledVersion(_ compiler.RunContext) (string, error) {
	return "", nil
}
