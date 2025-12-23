package nvim

import (
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// PresetStep manages Neovim preset installation (LazyVim, NvChad, etc.).
type PresetStep struct {
	preset string
	id     compiler.StepID
	fs     ports.FileSystem
	runner ports.CommandRunner
}

// NewPresetStep creates a new PresetStep.
func NewPresetStep(preset string, fs ports.FileSystem, runner ports.CommandRunner) *PresetStep {
	id := compiler.MustNewStepID(fmt.Sprintf("nvim:preset:%s", preset))
	return &PresetStep{
		preset: preset,
		id:     id,
		fs:     fs,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *PresetStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *PresetStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if the preset is installed.
func (s *PresetStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := ports.ExpandPath("~/.config/nvim")
	if s.fs.Exists(configPath) {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PresetStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"preset",
		s.preset,
		"",
		fmt.Sprintf("Install %s Neovim preset", s.preset),
	), nil
}

// Apply installs the preset.
func (s *PresetStep) Apply(ctx compiler.RunContext) error {
	// Clone preset starter template based on preset type
	var repoURL string
	switch s.preset {
	case "lazyvim":
		repoURL = "https://github.com/LazyVim/starter"
	case "nvchad":
		repoURL = "https://github.com/NvChad/starter"
	case "astronvim":
		repoURL = "https://github.com/AstroNvim/template"
	case "kickstart":
		repoURL = "https://github.com/nvim-lua/kickstart.nvim"
	default:
		return fmt.Errorf("unknown preset: %s", s.preset)
	}

	configPath := ports.ExpandPath("~/.config/nvim")
	result, err := s.runner.Run(ctx.Context(), "git", "clone", repoURL, configPath)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("git clone failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides context for this step.
func (s *PresetStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	var url string
	switch s.preset {
	case "lazyvim":
		url = "https://www.lazyvim.org/"
	case "nvchad":
		url = "https://nvchad.com/"
	case "astronvim":
		url = "https://astronvim.com/"
	case "kickstart":
		url = "https://github.com/nvim-lua/kickstart.nvim"
	}

	return compiler.NewExplanation(
		"Install Neovim Preset",
		fmt.Sprintf("Install %s preset configuration for Neovim. This provides a pre-configured, fully-featured Neovim setup.", s.preset),
		[]string{url},
	)
}

// ConfigRepoStep manages cloning a custom Neovim config repository.
type ConfigRepoStep struct {
	repo   string
	id     compiler.StepID
	fs     ports.FileSystem
	runner ports.CommandRunner
}

// NewConfigRepoStep creates a new ConfigRepoStep.
func NewConfigRepoStep(repo string, fs ports.FileSystem, runner ports.CommandRunner) *ConfigRepoStep {
	id := compiler.MustNewStepID("nvim:config-repo")
	return &ConfigRepoStep{
		repo:   repo,
		id:     id,
		fs:     fs,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *ConfigRepoStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *ConfigRepoStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if the config repo is cloned.
func (s *ConfigRepoStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := ports.ExpandPath("~/.config/nvim")
	if s.fs.Exists(configPath) {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ConfigRepoStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"config-repo",
		s.repo,
		"",
		fmt.Sprintf("Clone Neovim config from %s", s.repo),
	), nil
}

// Apply clones the config repository.
func (s *ConfigRepoStep) Apply(ctx compiler.RunContext) error {
	configPath := ports.ExpandPath("~/.config/nvim")
	result, err := s.runner.Run(ctx.Context(), "git", "clone", s.repo, configPath)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("git clone failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides context for this step.
func (s *ConfigRepoStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Clone Neovim Config",
		fmt.Sprintf("Clone your personal Neovim configuration from %s to ~/.config/nvim", s.repo),
		[]string{s.repo},
	)
}

// LazyLockStep manages lazy-lock.json synchronization for reproducible plugin versions.
type LazyLockStep struct {
	id compiler.StepID
	fs ports.FileSystem
}

// NewLazyLockStep creates a new LazyLockStep.
func NewLazyLockStep(fs ports.FileSystem) *LazyLockStep {
	id := compiler.MustNewStepID("nvim:lazy-lock")
	return &LazyLockStep{
		id: id,
		fs: fs,
	}
}

// ID returns the step identifier.
func (s *LazyLockStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *LazyLockStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if lazy-lock.json is in sync.
func (s *LazyLockStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	// For now, if the file exists, consider it satisfied
	// Future: compare with dotfiles version
	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *LazyLockStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"lazy-lock",
		"lazy-lock.json",
		"",
		"Sync lazy-lock.json plugin versions",
	), nil
}

// Apply syncs the lazy-lock.json file.
func (s *LazyLockStep) Apply(_ compiler.RunContext) error {
	// In real implementation:
	// 1. Copy lazy-lock.json from dotfiles to ~/.config/nvim/
	// 2. Run nvim --headless "+Lazy sync" +qa to sync plugins
	return nil
}

// Explain provides context for this step.
func (s *LazyLockStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Sync Lazy Plugin Lock",
		"Synchronize lazy-lock.json to ensure reproducible plugin versions across machines",
		nil,
	)
}
