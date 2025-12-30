package ghcli_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/ghcli"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ExtensionStep Tests
// =============================================================================

func TestExtensionStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	assert.Equal(t, "ghcli:extension:dlvhdr/gh-dash", step.ID().String())
}

func TestExtensionStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestExtensionStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"extension", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "dlvhdr/gh-dash\tgithub/gh-copilot\n",
	})

	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestExtensionStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"extension", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "github/gh-copilot\n",
	})

	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestExtensionStep_Check_CommandError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"extension", "list"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "not authenticated",
	})

	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestExtensionStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "extension", diff.Resource())
	assert.Equal(t, "dlvhdr/gh-dash", diff.Name())
}

func TestExtensionStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"extension", "install", "dlvhdr/gh-dash"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestExtensionStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"extension", "install", "dlvhdr/gh-dash"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "extension not found",
	})

	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extension not found")
}

func TestExtensionStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewExtensionStep("dlvhdr/gh-dash", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Install GitHub CLI Extension", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "dlvhdr/gh-dash")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// AliasStep Tests
// =============================================================================

func TestAliasStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	assert.Equal(t, "ghcli:alias:co", step.ID().String())
}

func TestAliasStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestAliasStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"alias", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "co: pr checkout\nprc: pr create\n",
	})

	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestAliasStep_Check_NeedsApply_NotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"alias", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "prc: pr create\n",
	})

	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAliasStep_Check_NeedsApply_DifferentCommand(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"alias", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "co: pr view\n",
	})

	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAliasStep_Check_CommandError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"alias", "list"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "not authenticated",
	})

	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestAliasStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "alias", diff.Resource())
	assert.Equal(t, "co", diff.Name())
	assert.Equal(t, "pr checkout", diff.NewValue())
}

func TestAliasStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"alias", "set", "co", "pr checkout"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestAliasStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"alias", "set", "co", "pr checkout"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "invalid alias",
	})

	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid alias")
}

func TestAliasStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewAliasStep("co", "pr checkout", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Set GitHub CLI Alias", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "co")
	assert.Contains(t, explanation.Detail(), "pr checkout")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// ConfigStep Tests
// =============================================================================

func TestConfigStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewConfigStep("editor", "nvim", runner)

	assert.Equal(t, "ghcli:config:editor", step.ID().String())
}

func TestConfigStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewConfigStep("editor", "nvim", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestConfigStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"config", "get", "editor"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "nvim\n",
	})

	step := ghcli.NewConfigStep("editor", "nvim", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestConfigStep_Check_NeedsApply_Different(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"config", "get", "editor"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "vim\n",
	})

	step := ghcli.NewConfigStep("editor", "nvim", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Check_NeedsApply_NotSet(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"config", "get", "editor"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "",
	})

	step := ghcli.NewConfigStep("editor", "nvim", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"config", "get", "editor"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "vim\n",
	})

	step := ghcli.NewConfigStep("editor", "nvim", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "config", diff.Resource())
	assert.Equal(t, "editor", diff.Name())
	assert.Equal(t, "vim", diff.OldValue())
	assert.Equal(t, "nvim", diff.NewValue())
}

func TestConfigStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"config", "set", "editor", "nvim"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := ghcli.NewConfigStep("editor", "nvim", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestConfigStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{"config", "set", "editor", "nvim"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "invalid key",
	})

	step := ghcli.NewConfigStep("editor", "nvim", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key")
}

func TestConfigStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := ghcli.NewConfigStep("editor", "nvim", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Set GitHub CLI Config", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "editor")
	assert.Contains(t, explanation.Detail(), "nvim")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}
