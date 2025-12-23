package shell_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/shell"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameworkStep_ID(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	step := shell.NewFrameworkStep(sc)

	assert.Equal(t, "shell:framework:zsh:oh-my-zsh", step.ID().String())
}

func TestFrameworkStep_DependsOn(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	step := shell.NewFrameworkStep(sc)

	assert.Empty(t, step.DependsOn())
}

func TestFrameworkStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewFrameworkStepWithFS(sc, fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestFrameworkStep_Check_Installed(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	fs := mocks.NewFileSystem()
	// Simulate oh-my-zsh is installed (directory exists)
	ohmyzshPath := ports.ExpandPath("~/.oh-my-zsh")
	_ = fs.MkdirAll(ohmyzshPath, 0o755)

	step := shell.NewFrameworkStepWithFS(sc, fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestFrameworkStep_Plan(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	step := shell.NewFrameworkStep(sc)

	ctx := compiler.NewRunContext(context.TODO())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, diff.Summary())
	assert.Contains(t, diff.Summary(), "oh-my-zsh")
}

func TestFrameworkStep_Explain(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	step := shell.NewFrameworkStep(sc)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "oh-my-zsh")
}

func TestPluginStep_ID(t *testing.T) {
	t.Parallel()

	step := shell.NewPluginStep("zsh", "oh-my-zsh", "git")

	assert.Equal(t, "shell:plugin:zsh:git", step.ID().String())
}

func TestPluginStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := shell.NewPluginStep("zsh", "oh-my-zsh", "git")
	deps := step.DependsOn()

	// Plugin should depend on framework being installed
	require.Len(t, deps, 1)
	assert.Equal(t, "shell:framework:zsh:oh-my-zsh", deps[0].String())
}

func TestCustomPluginStep_ID(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	step := shell.NewCustomPluginStep("zsh", "oh-my-zsh", plugin)

	assert.Equal(t, "shell:custom-plugin:zsh:zsh-autosuggestions", step.ID().String())
}

func TestCustomPluginStep_DependsOn(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	step := shell.NewCustomPluginStep("zsh", "oh-my-zsh", plugin)
	deps := step.DependsOn()

	require.Len(t, deps, 1)
	assert.Equal(t, "shell:framework:zsh:oh-my-zsh", deps[0].String())
}

func TestCustomPluginStep_Check_NotCloned(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewCustomPluginStepWithFS("zsh", "oh-my-zsh", plugin, fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestCustomPluginStep_Check_Cloned(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	fs := mocks.NewFileSystem()
	// Simulate plugin is already cloned (directory exists)
	pluginPath := ports.ExpandPath("~/.oh-my-zsh/custom/plugins/zsh-autosuggestions")
	_ = fs.MkdirAll(pluginPath, 0o755)

	step := shell.NewCustomPluginStepWithFS("zsh", "oh-my-zsh", plugin, fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestEnvStep_ID(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	step := shell.NewEnvStep("zsh", env)

	assert.Equal(t, "shell:env:zsh", step.ID().String())
}

func TestAliasStep_ID(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	step := shell.NewAliasStep("zsh", aliases)

	assert.Equal(t, "shell:aliases:zsh", step.ID().String())
}

func TestStarshipStep_ID(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
		Preset:  "nerd-font-symbols",
	}
	step := shell.NewStarshipStep(cfg)

	assert.Equal(t, "shell:starship", step.ID().String())
}

func TestStarshipStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
		Preset:  "nerd-font-symbols",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewStarshipStepWithFS(cfg, fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestStarshipStep_Check_Installed(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
		Preset:  "nerd-font-symbols",
	}
	fs := mocks.NewFileSystem()
	// Simulate starship config exists
	starshipPath := ports.ExpandPath("~/.config/starship.toml")
	fs.SetFileContent(starshipPath, []byte("# starship config"))

	step := shell.NewStarshipStepWithFS(cfg, fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestFisherPluginStep_ID(t *testing.T) {
	t.Parallel()

	step := shell.NewFisherPluginStep("jorgebucaran/autopair.fish")

	// Plugin names with dots are sanitized to dashes in step IDs
	assert.Equal(t, "shell:fisher:jorgebucaran/autopair-fish", step.ID().String())
}

func TestFisherPluginStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := shell.NewFisherPluginStep("jorgebucaran/autopair.fish")
	deps := step.DependsOn()

	// Fisher plugin depends on fisher framework being installed
	require.Len(t, deps, 1)
	assert.Equal(t, "shell:framework:fish:fisher", deps[0].String())
}

func TestFrameworkStep_Apply(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewFrameworkStepWithFS(sc, fs)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestFrameworkStep_FrameworkPath_Fisher(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "fish",
		Framework: "fisher",
	}
	fs := mocks.NewFileSystem()
	fisherPath := ports.ExpandPath("~/.config/fish/functions/fisher.fish")
	fs.AddFile(fisherPath, "# fisher")

	step := shell.NewFrameworkStepWithFS(sc, fs)
	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestFrameworkStep_FrameworkPath_OhMyFish(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "fish",
		Framework: "oh-my-fish",
	}
	fs := mocks.NewFileSystem()
	omfPath := ports.ExpandPath("~/.local/share/omf")
	_ = fs.MkdirAll(omfPath, 0o755)

	step := shell.NewFrameworkStepWithFS(sc, fs)
	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestFrameworkStep_FrameworkPath_Unknown(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "unknown-framework",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewFrameworkStepWithFS(sc, fs)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPluginStep_Check(t *testing.T) {
	t.Parallel()

	step := shell.NewPluginStep("zsh", "oh-my-zsh", "git")
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPluginStep_Plan(t *testing.T) {
	t.Parallel()

	step := shell.NewPluginStep("zsh", "oh-my-zsh", "git")
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.Summary(), "git")
}

func TestPluginStep_Apply(t *testing.T) {
	t.Parallel()

	step := shell.NewPluginStep("zsh", "oh-my-zsh", "git")
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPluginStep_Explain(t *testing.T) {
	t.Parallel()

	step := shell.NewPluginStep("zsh", "oh-my-zsh", "git")
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "git")
}

func TestCustomPluginStep_Plan(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	step := shell.NewCustomPluginStep("zsh", "oh-my-zsh", plugin)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.Summary(), "zsh-autosuggestions")
}

func TestCustomPluginStep_Apply(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewCustomPluginStepWithFS("zsh", "oh-my-zsh", plugin, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestCustomPluginStep_Explain(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	step := shell.NewCustomPluginStep("zsh", "oh-my-zsh", plugin)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "zsh-autosuggestions")
}

func TestCustomPluginStep_PluginPath_Unknown(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "some-plugin",
		Repo: "owner/some-plugin",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewCustomPluginStepWithFS("fish", "unknown", plugin, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestEnvStep_DependsOn(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	step := shell.NewEnvStep("zsh", env)

	assert.Empty(t, step.DependsOn())
}

func TestEnvStep_Check(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	step := shell.NewEnvStep("zsh", env)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestEnvStep_Plan(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim", "SHELL": "zsh"}
	step := shell.NewEnvStep("zsh", env)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.Summary(), "2 variables")
}

func TestEnvStep_Apply(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	step := shell.NewEnvStep("zsh", env)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestEnvStep_Explain(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	step := shell.NewEnvStep("zsh", env)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "environment")
}

func TestAliasStep_DependsOn(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	step := shell.NewAliasStep("zsh", aliases)

	assert.Empty(t, step.DependsOn())
}

func TestAliasStep_Check(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	step := shell.NewAliasStep("zsh", aliases)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAliasStep_Plan(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la", "la": "ls -a"}
	step := shell.NewAliasStep("zsh", aliases)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.Summary(), "2 aliases")
}

func TestAliasStep_Apply(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	step := shell.NewAliasStep("zsh", aliases)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestAliasStep_Explain(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	step := shell.NewAliasStep("zsh", aliases)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "aliases")
}

func TestStarshipStep_DependsOn(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
		Preset:  "nerd-font-symbols",
	}
	step := shell.NewStarshipStep(cfg)

	assert.Empty(t, step.DependsOn())
}

func TestStarshipStep_Plan(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
		Preset:  "nerd-font-symbols",
	}
	step := shell.NewStarshipStep(cfg)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.Summary(), "nerd-font-symbols")
}

func TestStarshipStep_Plan_NoPreset(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
	}
	step := shell.NewStarshipStep(cfg)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.Summary(), "starship")
}

func TestStarshipStep_Apply(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
		Preset:  "nerd-font-symbols",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewStarshipStepWithFS(cfg, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestStarshipStep_Explain(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
		Preset:  "nerd-font-symbols",
	}
	step := shell.NewStarshipStep(cfg)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "nerd-font-symbols")
}

func TestStarshipStep_Explain_NoPreset(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
	}
	step := shell.NewStarshipStep(cfg)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "starship")
}

func TestFisherPluginStep_Check(t *testing.T) {
	t.Parallel()

	step := shell.NewFisherPluginStep("jorgebucaran/autopair.fish")
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestFisherPluginStep_Plan(t *testing.T) {
	t.Parallel()

	step := shell.NewFisherPluginStep("jorgebucaran/autopair.fish")
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.Summary(), "autopair")
}

func TestFisherPluginStep_Apply(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := shell.NewFisherPluginStepWithFS("jorgebucaran/autopair.fish", fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestFisherPluginStep_Explain(t *testing.T) {
	t.Parallel()

	step := shell.NewFisherPluginStep("jorgebucaran/autopair.fish")
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "autopair")
}
