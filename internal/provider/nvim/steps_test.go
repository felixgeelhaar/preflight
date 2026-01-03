package nvim_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/nvim"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresetStep_ID(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	assert.Equal(t, "nvim:preset:lazyvim", step.ID().String())
}

func TestPresetStep_DependsOn(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	assert.Empty(t, step.DependsOn())
}

func TestPresetStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPresetStep_Check_Installed(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Simulate nvim config directory exists
	configPath := ports.ExpandPath("~/.config/nvim")
	_ = fs.MkdirAll(configPath, 0o755)

	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPresetStep_Plan(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, diff.Summary())
	assert.Contains(t, diff.Summary(), "lazyvim")
}

func TestPresetStep_Explain(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "lazyvim")
}

func TestConfigRepoStep_ID(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	assert.Equal(t, "nvim:config-repo", step.ID().String())
}

func TestConfigRepoStep_Check_NotCloned(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigRepoStep_Check_Cloned(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Simulate nvim config directory exists
	configPath := ports.ExpandPath("~/.config/nvim")
	_ = fs.MkdirAll(configPath, 0o755)

	runner := mocks.NewCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestLazyLockStep_ID(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewLazyLockStep(fs, runner)

	assert.Equal(t, "nvim:lazy-lock", step.ID().String())
}

func TestLazyLockStep_DependsOn(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewLazyLockStep(fs, runner)
	deps := step.DependsOn()

	// lazy-lock depends on preset or config being installed first
	assert.Empty(t, deps) // For now, no explicit dependencies
}

func TestLazyLockStep_Check_NoLockFile(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewLazyLockStep(fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	// No lock file means nothing to sync
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestLazyLockStep_Check_WithLockFile(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	// Simulate lazy-lock.json exists in nvim config
	lockPath := ports.ExpandPath("~/.config/nvim/lazy-lock.json")
	fs.SetFileContent(lockPath, []byte(`{"plugin": {"commit": "abc123"}}`))

	step := nvim.NewLazyLockStep(fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

// Additional tests for Apply, Plan, and Explain methods

func TestPresetStep_Apply_LazyVim(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	configPath := ports.ExpandPath("~/.config/nvim")
	runner.AddResult("git", []string{"clone", "https://github.com/LazyVim/starter", configPath}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Cloning into...",
	})

	step := nvim.NewPresetStep("lazyvim", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestPresetStep_Apply_NvChad(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	configPath := ports.ExpandPath("~/.config/nvim")
	runner.AddResult("git", []string{"clone", "https://github.com/NvChad/starter", configPath}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Cloning into...",
	})

	step := nvim.NewPresetStep("nvchad", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestPresetStep_Apply_AstroNvim(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	configPath := ports.ExpandPath("~/.config/nvim")
	runner.AddResult("git", []string{"clone", "https://github.com/AstroNvim/template", configPath}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Cloning into...",
	})

	step := nvim.NewPresetStep("astronvim", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestPresetStep_Apply_Kickstart(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	configPath := ports.ExpandPath("~/.config/nvim")
	runner.AddResult("git", []string{"clone", "https://github.com/nvim-lua/kickstart.nvim", configPath}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Cloning into...",
	})

	step := nvim.NewPresetStep("kickstart", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestPresetStep_Apply_UnknownPreset(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()

	step := nvim.NewPresetStep("unknown-preset", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown preset")
}

func TestPresetStep_Apply_GitCloneFails(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	configPath := ports.ExpandPath("~/.config/nvim")
	runner.AddResult("git", []string{"clone", "https://github.com/LazyVim/starter", configPath}, ports.CommandResult{
		ExitCode: 128,
		Stderr:   "fatal: destination path already exists",
	})

	step := nvim.NewPresetStep("lazyvim", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestPresetStep_Explain_NvChad(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("nvchad", fs, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "nvchad")
}

func TestPresetStep_Explain_AstroNvim(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("astronvim", fs, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "astronvim")
}

func TestPresetStep_Explain_Kickstart(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewPresetStep("kickstart", fs, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "kickstart")
}

func TestConfigRepoStep_DependsOn(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestConfigRepoStep_Plan(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, diff.Summary())
	assert.Contains(t, diff.Summary(), "github.com/user/nvim-config")
}

func TestConfigRepoStep_Apply_Success(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	configPath := ports.ExpandPath("~/.config/nvim")
	runner.AddResult("git", []string{"clone", "https://github.com/user/nvim-config", configPath}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Cloning into...",
	})

	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestConfigRepoStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	configPath := ports.ExpandPath("~/.config/nvim")
	runner.AddResult("git", []string{"clone", "https://github.com/user/nvim-config", configPath}, ports.CommandResult{
		ExitCode: 128,
		Stderr:   "fatal: repository not found",
	})

	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestConfigRepoStep_Explain(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "github.com/user/nvim-config")
}

func TestLazyLockStep_Plan(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewLazyLockStep(fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, diff.Summary())
	assert.Contains(t, diff.Summary(), "lazy-lock")
}

func TestLazyLockStep_Apply_Success(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()

	// Mock successful nvim headless sync
	runner.AddResult("nvim", []string{"--headless", "+Lazy sync", "+qa"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "",
	})

	step := nvim.NewLazyLockStep(fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestLazyLockStep_Apply_NvimFails(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()

	// Mock failed nvim headless sync
	runner.AddResult("nvim", []string{"--headless", "+Lazy sync", "+qa"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Error: Lazy.nvim not found",
	})

	step := nvim.NewLazyLockStep(fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nvim sync failed")
}

func TestLazyLockStep_Explain(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	step := nvim.NewLazyLockStep(fs, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "lazy-lock")
}

// ConfigSourceStep tests

func TestConfigSourceStep_ID(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := nvim.NewConfigSourceStep("/home/user/dotfiles/.config/nvim", "~/.config/nvim", fs)

	assert.Equal(t, "nvim:config-source", step.ID().String())
}

func TestConfigSourceStep_DependsOn(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := nvim.NewConfigSourceStep("/home/user/dotfiles/.config/nvim", "~/.config/nvim", fs)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestConfigSourceStep_Check_CorrectSymlink(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"
	destPath := ports.ExpandPath("~/.config/nvim")

	// Simulate correct symlink exists
	fs.AddSymlink(destPath, sourcePath)

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestConfigSourceStep_Check_NoSymlink(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigSourceStep_Check_WrongTarget(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"
	destPath := ports.ExpandPath("~/.config/nvim")

	// Simulate symlink pointing to wrong target
	fs.AddSymlink(destPath, "/some/other/path")

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigSourceStep_Plan_NoExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, diff.Summary())
	// Name contains "symlink -> <sourcePath>"
	assert.Contains(t, diff.Name(), sourcePath)
}

func TestConfigSourceStep_Plan_ExistingSymlink(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"
	destPath := ports.ExpandPath("~/.config/nvim")

	// Simulate existing wrong symlink
	fs.AddSymlink(destPath, "/some/other/path")

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.OldValue(), "/some/other/path")
}

func TestConfigSourceStep_Plan_ExistingDirectory(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"
	destPath := ports.ExpandPath("~/.config/nvim")

	// Simulate existing directory (not a symlink)
	fs.AddDir(destPath)

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.OldValue(), "existing directory")
}

func TestConfigSourceStep_Apply_CreateNew(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
	// Verify symlink was created
	destPath := ports.ExpandPath("~/.config/nvim")
	isLink, target := fs.IsSymlink(destPath)
	assert.True(t, isLink)
	assert.Equal(t, sourcePath, target)
}

func TestConfigSourceStep_Apply_AlreadyCorrect(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"
	destPath := ports.ExpandPath("~/.config/nvim")

	// Already has correct symlink
	fs.AddSymlink(destPath, sourcePath)

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestConfigSourceStep_Apply_ReplaceWrongSymlink(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"
	destPath := ports.ExpandPath("~/.config/nvim")

	// Wrong symlink exists
	fs.AddSymlink(destPath, "/wrong/path")

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
	// Verify new symlink
	isLink, target := fs.IsSymlink(destPath)
	assert.True(t, isLink)
	assert.Equal(t, sourcePath, target)
}

func TestConfigSourceStep_Explain(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	sourcePath := "/home/user/dotfiles/.config/nvim"

	step := nvim.NewConfigSourceStep(sourcePath, "~/.config/nvim", fs)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), sourcePath)
	assert.Contains(t, exp.Detail(), "symlink")
}
