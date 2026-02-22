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

func TestFrameworkStep_Apply_OhMyZsh(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	runner.AddResult("/bin/bash", []string{"-c", `RUNZSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"`}, ports.CommandResult{ExitCode: 0})

	step := shell.NewFrameworkStepWith(sc, fs, runner)
	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "/bin/bash", calls[0].Command)
}

func TestFrameworkStep_Apply_Fisher(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "fish",
		Framework: "fisher",
	}
	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	runner.AddResult("fish", []string{"-c", `curl -sL https://raw.githubusercontent.com/jorgebucaran/fisher/main/functions/fisher.fish | source && fisher install jorgebucaran/fisher`}, ports.CommandResult{ExitCode: 0})

	step := shell.NewFrameworkStepWith(sc, fs, runner)
	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestFrameworkStep_Apply_OhMyFish(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "fish",
		Framework: "oh-my-fish",
	}
	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	runner.AddResult("/bin/bash", []string{"-c", "curl -L https://get.oh-my.fish | fish"}, ports.CommandResult{ExitCode: 0})

	step := shell.NewFrameworkStepWith(sc, fs, runner)
	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestFrameworkStep_Apply_UnsupportedFramework(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "unknown",
	}
	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()

	step := shell.NewFrameworkStepWith(sc, fs, runner)
	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported framework")
}

func TestFrameworkStep_Apply_CommandFails(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	runner.AddResult("/bin/bash", []string{"-c", `RUNZSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"`}, ports.CommandResult{ExitCode: 1, Stderr: "curl failed"})

	step := shell.NewFrameworkStepWith(sc, fs, runner)
	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "oh-my-zsh install failed")
}

func TestFrameworkStep_Apply_NoRunner(t *testing.T) {
	t.Parallel()

	sc := shell.Entry{
		Name:      "zsh",
		Framework: "oh-my-zsh",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewFrameworkStepWithFS(sc, fs)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "command runner not configured")
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

func TestPluginStep_Check_NotEnabled(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	zshrcPath := ports.ExpandPath("~/.zshrc")
	fs.AddFile(zshrcPath, "plugins=(docker)\n")

	step := shell.NewPluginStepWithFS("zsh", "oh-my-zsh", "git", fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPluginStep_Check_Enabled(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	zshrcPath := ports.ExpandPath("~/.zshrc")
	fs.AddFile(zshrcPath, "plugins=(git docker)\n")

	step := shell.NewPluginStepWithFS("zsh", "oh-my-zsh", "git", fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPluginStep_Check_NoConfigFile(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := shell.NewPluginStepWithFS("zsh", "oh-my-zsh", "git", fs)
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

func TestPluginStep_Apply_AddsToExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	zshrcPath := ports.ExpandPath("~/.zshrc")
	fs.AddFile(zshrcPath, "plugins=(docker)\n")

	step := shell.NewPluginStepWithFS("zsh", "oh-my-zsh", "git", fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
	content, _ := fs.ReadFile(zshrcPath)
	assert.Contains(t, string(content), "git")
	assert.Contains(t, string(content), "docker")
}

func TestPluginStep_Apply_CreatesConfig(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := shell.NewPluginStepWithFS("zsh", "oh-my-zsh", "git", fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
	zshrcPath := ports.ExpandPath("~/.zshrc")
	content, _ := fs.ReadFile(zshrcPath)
	assert.Contains(t, string(content), "plugins=(git)")
}

func TestPluginStep_Explain(t *testing.T) {
	t.Parallel()

	step := shell.NewPluginStep("zsh", "oh-my-zsh", "git")
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "git")
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
	runner := mocks.NewCommandRunner()
	pluginPath := ports.ExpandPath("~/.oh-my-zsh/custom/plugins/zsh-autosuggestions")
	runner.AddResult("git", []string{"clone", "zsh-users/zsh-autosuggestions", pluginPath}, ports.CommandResult{ExitCode: 0})

	step := shell.NewCustomPluginStepWith("zsh", "oh-my-zsh", plugin, fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "git", calls[0].Command)
	assert.Contains(t, calls[0].Args, "clone")
}

func TestCustomPluginStep_Apply_CloneFails(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	pluginPath := ports.ExpandPath("~/.oh-my-zsh/custom/plugins/zsh-autosuggestions")
	runner.AddResult("git", []string{"clone", "zsh-users/zsh-autosuggestions", pluginPath}, ports.CommandResult{ExitCode: 1, Stderr: "not found"})

	step := shell.NewCustomPluginStepWith("zsh", "oh-my-zsh", plugin, fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestCustomPluginStep_Apply_NoRunner(t *testing.T) {
	t.Parallel()

	plugin := shell.CustomPlugin{
		Name: "zsh-autosuggestions",
		Repo: "zsh-users/zsh-autosuggestions",
	}
	fs := mocks.NewFileSystem()
	step := shell.NewCustomPluginStepWithFS("zsh", "oh-my-zsh", plugin, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "command runner not configured")
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

func TestEnvStep_ID(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	step := shell.NewEnvStep("zsh", env)

	assert.Equal(t, "shell:env:zsh", step.ID().String())
}

func TestEnvStep_DependsOn(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	step := shell.NewEnvStep("zsh", env)

	assert.Empty(t, step.DependsOn())
}

func TestEnvStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	fs := mocks.NewFileSystem()
	step := shell.NewEnvStepWithFS("zsh", env, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestEnvStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	fs := mocks.NewFileSystem()
	zshrcPath := ports.ExpandPath("~/.zshrc")
	// Write the exact expected managed block
	content := "# >>> preflight env >>>\nexport EDITOR=\"nvim\"\n# <<< preflight env <<<\n"
	fs.AddFile(zshrcPath, content)

	step := shell.NewEnvStepWithFS("zsh", env, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
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
	fs := mocks.NewFileSystem()
	step := shell.NewEnvStepWithFS("zsh", env, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)

	zshrcPath := ports.ExpandPath("~/.zshrc")
	content, _ := fs.ReadFile(zshrcPath)
	assert.Contains(t, string(content), "# >>> preflight env >>>")
	assert.Contains(t, string(content), `export EDITOR="nvim"`)
	assert.Contains(t, string(content), "# <<< preflight env <<<")
}

func TestEnvStep_Apply_PreservesExisting(t *testing.T) {
	t.Parallel()

	env := map[string]string{"EDITOR": "nvim"}
	fs := mocks.NewFileSystem()
	zshrcPath := ports.ExpandPath("~/.zshrc")
	fs.AddFile(zshrcPath, "# my custom config\nsource something\n")

	step := shell.NewEnvStepWithFS("zsh", env, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
	content, _ := fs.ReadFile(zshrcPath)
	assert.Contains(t, string(content), "# my custom config")
	assert.Contains(t, string(content), `export EDITOR="nvim"`)
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

func TestAliasStep_ID(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	step := shell.NewAliasStep("zsh", aliases)

	assert.Equal(t, "shell:aliases:zsh", step.ID().String())
}

func TestAliasStep_DependsOn(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	step := shell.NewAliasStep("zsh", aliases)

	assert.Empty(t, step.DependsOn())
}

func TestAliasStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	fs := mocks.NewFileSystem()
	step := shell.NewAliasStepWithFS("zsh", aliases, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAliasStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{"ll": "ls -la"}
	fs := mocks.NewFileSystem()
	zshrcPath := ports.ExpandPath("~/.zshrc")
	content := "# >>> preflight aliases >>>\nalias ll=\"ls -la\"\n# <<< preflight aliases <<<\n"
	fs.AddFile(zshrcPath, content)

	step := shell.NewAliasStepWithFS("zsh", aliases, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
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
	fs := mocks.NewFileSystem()
	step := shell.NewAliasStepWithFS("zsh", aliases, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)

	zshrcPath := ports.ExpandPath("~/.zshrc")
	content, _ := fs.ReadFile(zshrcPath)
	assert.Contains(t, string(content), "# >>> preflight aliases >>>")
	assert.Contains(t, string(content), `alias ll="ls -la"`)
	assert.Contains(t, string(content), "# <<< preflight aliases <<<")
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

func TestStarshipStep_ID(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
		Preset:  "nerd-font-symbols",
	}
	step := shell.NewStarshipStep(cfg)

	assert.Equal(t, "shell:starship", step.ID().String())
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

	starshipPath := ports.ExpandPath("~/.config/starship.toml")
	content, readErr := fs.ReadFile(starshipPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "Managed by preflight")
	assert.Contains(t, string(content), "nerd-font-symbols")
}

func TestStarshipStep_Apply_NoPreset(t *testing.T) {
	t.Parallel()

	cfg := shell.StarshipConfig{
		Enabled: true,
	}
	fs := mocks.NewFileSystem()
	step := shell.NewStarshipStepWithFS(cfg, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)

	starshipPath := ports.ExpandPath("~/.config/starship.toml")
	content, readErr := fs.ReadFile(starshipPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "Managed by preflight")
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

func TestFisherPluginStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := shell.NewFisherPluginStepWithFS("jorgebucaran/autopair.fish", fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestFisherPluginStep_Check_Installed(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	pluginsPath := ports.ExpandPath("~/.config/fish/fish_plugins")
	fs.AddFile(pluginsPath, "jorgebucaran/autopair.fish\njorgebucaran/hydro\n")

	step := shell.NewFisherPluginStepWithFS("jorgebucaran/autopair.fish", fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
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
	runner := mocks.NewCommandRunner()
	runner.AddResult("fish", []string{"-c", "fisher install jorgebucaran/autopair.fish"}, ports.CommandResult{ExitCode: 0})

	step := shell.NewFisherPluginStepWith("jorgebucaran/autopair.fish", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "fish", calls[0].Command)
}

func TestFisherPluginStep_Apply_Fails(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	runner.AddResult("fish", []string{"-c", "fisher install jorgebucaran/autopair.fish"}, ports.CommandResult{ExitCode: 1, Stderr: "not found"})

	step := shell.NewFisherPluginStepWith("jorgebucaran/autopair.fish", fs, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "fisher install")
}

func TestFisherPluginStep_Apply_NoRunner(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := shell.NewFisherPluginStepWithFS("jorgebucaran/autopair.fish", fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "command runner not configured")
}

func TestFisherPluginStep_Explain(t *testing.T) {
	t.Parallel()

	step := shell.NewFisherPluginStep("jorgebucaran/autopair.fish")
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "autopair")
}
