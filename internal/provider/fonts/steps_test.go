package fonts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestNerdFontStep_ID(t *testing.T) {
	t.Parallel()

	step := NewNerdFontStep("JetBrainsMono", mocks.NewCommandRunner())
	assert.Equal(t, compiler.MustNewStepID("fonts:nerd:JetBrainsMono"), step.ID())
}

func TestNerdFontStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := NewNerdFontStep("JetBrainsMono", mocks.NewCommandRunner())
	deps := step.DependsOn()

	require.Len(t, deps, 1)
	assert.Equal(t, compiler.MustNewStepID("brew:tap:homebrew/cask-fonts"), deps[0])
}

func TestNerdFontStep_Check_Installed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask"}, ports.CommandResult{
		Stdout:   "font-jetbrains-mono-nerd-font\nfont-fira-code-nerd-font\n",
		ExitCode: 0,
	})

	step := NewNerdFontStep("JetBrainsMono", runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestNerdFontStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask"}, ports.CommandResult{
		Stdout:   "font-fira-code-nerd-font\n",
		ExitCode: 0,
	})

	step := NewNerdFontStep("JetBrainsMono", runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestNerdFontStep_Check_CommandFailed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"list", "--cask"}, ports.CommandResult{
		Stderr:   "brew not found",
		ExitCode: 1,
	})

	step := NewNerdFontStep("JetBrainsMono", runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestNerdFontStep_Plan(t *testing.T) {
	t.Parallel()

	step := NewNerdFontStep("JetBrainsMono", mocks.NewCommandRunner())
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "font-jetbrains-mono-nerd-font", diff.NewValue())
}

func TestNerdFontStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "--cask", "font-jetbrains-mono-nerd-font"}, ports.CommandResult{
		Stdout:   "==> Installing font-jetbrains-mono-nerd-font",
		ExitCode: 0,
	})

	step := NewNerdFontStep("JetBrainsMono", runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestNerdFontStep_Apply_Failed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "--cask", "font-jetbrains-mono-nerd-font"}, ports.CommandResult{
		Stderr:   "Cask 'font-jetbrains-mono-nerd-font' not found",
		ExitCode: 1,
	})

	step := NewNerdFontStep("JetBrainsMono", runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestNerdFontStep_Explain(t *testing.T) {
	t.Parallel()

	step := NewNerdFontStep("JetBrainsMono", mocks.NewCommandRunner())
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Summary(), "Nerd Font")
	assert.Contains(t, explanation.Detail(), "JetBrainsMono")
	assert.NotEmpty(t, explanation.DocLinks())
}

func TestNerdFontStep_CaskName(t *testing.T) {
	t.Parallel()

	step := NewNerdFontStep("FiraCode", mocks.NewCommandRunner())
	assert.Equal(t, "font-fira-code-nerd-font", step.CaskName())
}
