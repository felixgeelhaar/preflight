package terminal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// HyperConfigStep manages Hyper terminal configuration.
// Note: Hyper uses JavaScript, so we only support source/link mode, not settings merge.
type HyperConfigStep struct {
	cfg        *HyperConfig
	targetPath string
	configRoot string
	fs         ports.FileSystem
}

// NewHyperConfigStep creates a new Hyper config step.
func NewHyperConfigStep(cfg *HyperConfig, targetPath, configRoot string, fs ports.FileSystem) *HyperConfigStep {
	return &HyperConfigStep{
		cfg:        cfg,
		targetPath: pathutil.ExpandPath(targetPath),
		configRoot: configRoot,
		fs:         fs,
	}
}

// ID returns the step identifier.
func (s *HyperConfigStep) ID() compiler.StepID {
	return compiler.MustNewStepID("terminal:hyper:config")
}

// DependsOn returns dependencies.
func (s *HyperConfigStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if configuration needs to be applied.
func (s *HyperConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if s.cfg.Source == "" {
		return compiler.StatusSatisfied, nil
	}

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

// Plan returns the diff for this step.
func (s *HyperConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	if s.cfg.Source == "" {
		return compiler.NewDiff(
			compiler.DiffTypeNone,
			"config",
			s.targetPath,
			"",
			"no source specified",
		), nil
	}

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

// Apply writes the configuration.
func (s *HyperConfigStep) Apply(_ compiler.RunContext) error {
	if s.cfg.Source == "" {
		return nil
	}

	dir := filepath.Dir(s.targetPath)
	if err := s.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

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

// Explain provides context for this step.
func (s *HyperConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Hyper",
		"Manages Hyper terminal configuration (JavaScript-based)",
		[]string{
			"https://hyper.is/",
			"https://github.com/vercel/hyper",
		},
	).WithTradeoffs([]string{
		"+ Web technologies (Electron-based)",
		"+ Rich plugin ecosystem",
		"+ Cross-platform support",
		"+ Easy theming with CSS",
		"- Higher resource usage (Electron)",
		"- JavaScript config has learning curve",
	})
}

// LockInfo returns lock information for this step.
func (s *HyperConfigStep) LockInfo() compiler.LockInfo {
	return compiler.LockInfo{}
}

// InstalledVersion returns the installed version (N/A for config).
func (s *HyperConfigStep) InstalledVersion(_ compiler.RunContext) (string, error) {
	return "", nil
}
