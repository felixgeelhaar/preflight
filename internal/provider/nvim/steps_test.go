package nvim_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/nvim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresetStep_ID(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	runner := ports.NewMockCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	assert.Equal(t, "nvim:preset:lazyvim", step.ID().String())
}

func TestPresetStep_DependsOn(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	runner := ports.NewMockCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	assert.Empty(t, step.DependsOn())
}

func TestPresetStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	runner := ports.NewMockCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPresetStep_Check_Installed(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	// Simulate nvim config directory exists
	configPath := ports.ExpandPath("~/.config/nvim")
	_ = fs.MkdirAll(configPath, 0755)

	runner := ports.NewMockCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPresetStep_Plan(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	runner := ports.NewMockCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, diff.Summary())
	assert.Contains(t, diff.Summary(), "lazyvim")
}

func TestPresetStep_Explain(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	runner := ports.NewMockCommandRunner()
	step := nvim.NewPresetStep("lazyvim", fs, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "lazyvim")
}

func TestConfigRepoStep_ID(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	runner := ports.NewMockCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	assert.Equal(t, "nvim:config-repo", step.ID().String())
}

func TestConfigRepoStep_Check_NotCloned(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	runner := ports.NewMockCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigRepoStep_Check_Cloned(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	// Simulate nvim config directory exists
	configPath := ports.ExpandPath("~/.config/nvim")
	_ = fs.MkdirAll(configPath, 0755)

	runner := ports.NewMockCommandRunner()
	step := nvim.NewConfigRepoStep("https://github.com/user/nvim-config", fs, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestLazyLockStep_ID(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	step := nvim.NewLazyLockStep(fs)

	assert.Equal(t, "nvim:lazy-lock", step.ID().String())
}

func TestLazyLockStep_DependsOn(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	step := nvim.NewLazyLockStep(fs)
	deps := step.DependsOn()

	// lazy-lock depends on preset or config being installed first
	assert.Empty(t, deps) // For now, no explicit dependencies
}

func TestLazyLockStep_Check_NoLockFile(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	step := nvim.NewLazyLockStep(fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	// No lock file means nothing to sync
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestLazyLockStep_Check_WithLockFile(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	// Simulate lazy-lock.json exists in dotfiles (source)
	lockPath := ports.ExpandPath("~/.config/nvim/lazy-lock.json")
	fs.SetFileContent(lockPath, []byte(`{"plugin": {"commit": "abc123"}}`))

	step := nvim.NewLazyLockStep(fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}
