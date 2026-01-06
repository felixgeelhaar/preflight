package bootstrap

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	tooldeps "github.com/felixgeelhaar/preflight/internal/domain/deps"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/require"
)

type fakeStep struct {
	id compiler.StepID
}

func (f *fakeStep) ID() compiler.StepID { return f.id }
func (f *fakeStep) DependsOn() []compiler.StepID {
	return nil
}
func (f *fakeStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return compiler.StatusSatisfied, nil
}
func (f *fakeStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.Diff{}, nil
}
func (f *fakeStep) Apply(_ compiler.RunContext) error { return nil }
func (f *fakeStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.Explanation{}
}

func TestProviderCompile_ImplicitToolBootstrap(t *testing.T) {
	cfg := map[string]interface{}{
		"npm": map[string]interface{}{
			"packages": []interface{}{"eslint"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	provider := NewProvider(runner, plat)
	steps, err := provider.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 2)

	ids := []string{steps[0].ID().String(), steps[1].ID().String()}
	require.ElementsMatch(t, []string{"brew:install", "bootstrap:tool:node"}, ids)
}

func TestProviderCompile_ExplicitToolConfigured(t *testing.T) {
	cfg := map[string]interface{}{
		"npm": map[string]interface{}{
			"packages": []interface{}{"eslint"},
		},
		"brew": map[string]interface{}{
			"formulae": []interface{}{"node"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	provider := NewProvider(runner, plat)
	steps, err := provider.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 0)
}

func TestRequiredTools(t *testing.T) {
	cfg := map[string]interface{}{
		"cargo": map[string]interface{}{
			"crates": []interface{}{"rg"},
		},
		"npm": map[string]interface{}{
			"packages": []interface{}{"eslint"},
		},
		"pip": map[string]interface{}{
			"packages": []interface{}{"requests"},
		},
		"gem": map[string]interface{}{
			"gems": []interface{}{"rails"},
		},
		"gotools": map[string]interface{}{
			"tools": []interface{}{"golang.org/x/tools/gopls"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	tools := requiredTools(ctx)
	require.Equal(t, []tooldeps.Tool{
		tooldeps.ToolRust,
		tooldeps.ToolNode,
		tooldeps.ToolPython,
		tooldeps.ToolRuby,
		tooldeps.ToolGo,
	}, tools)
}

func TestManagerConfigured(t *testing.T) {
	cfg := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"node"},
		},
		"apt": map[string]interface{}{
			"packages": []interface{}{"curl"},
		},
		"winget": map[string]interface{}{
			"packages": []interface{}{"Git.Git"},
		},
		"chocolatey": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
		"scoop": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	require.True(t, managerConfigured(ctx, "brew"))
	require.True(t, managerConfigured(ctx, "apt"))
	require.True(t, managerConfigured(ctx, "winget"))
	require.True(t, managerConfigured(ctx, "chocolatey"))
	require.True(t, managerConfigured(ctx, "scoop"))
	require.False(t, managerConfigured(ctx, "unknown"))
}

func TestEnsureManagerSteps(t *testing.T) {
	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	cfg := map[string]interface{}{}
	ctx := compiler.NewCompileContext(cfg)
	var steps []compiler.Step
	added := make(map[string]struct{})

	dep := provider.ensureManagerSteps(ctx, "apt", &steps, added)
	require.Equal(t, "apt:update", dep.String())
	require.Len(t, steps, 2)

	steps = nil
	added = make(map[string]struct{})
	cfg = map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"node"},
		},
	}
	ctx = compiler.NewCompileContext(cfg)
	dep = provider.ensureManagerSteps(ctx, "brew", &steps, added)
	require.Equal(t, "brew:install", dep.String())
	require.Len(t, steps, 0)

	steps = nil
	added = make(map[string]struct{})
	dep = provider.ensureManagerSteps(ctx, "unknown", &steps, added)
	require.Equal(t, "", dep.String())

	steps = nil
	added = make(map[string]struct{})
	dep = provider.ensureManagerSteps(ctx, "winget", &steps, added)
	require.Equal(t, "winget:ready", dep.String())
	require.Len(t, steps, 1)

	steps = nil
	added = make(map[string]struct{})
	dep = provider.ensureManagerSteps(ctx, "chocolatey", &steps, added)
	require.Equal(t, "chocolatey:install", dep.String())
	require.Len(t, steps, 1)

	steps = nil
	added = make(map[string]struct{})
	dep = provider.ensureManagerSteps(ctx, "scoop", &steps, added)
	require.Equal(t, "scoop:install", dep.String())
	require.Len(t, steps, 1)
}

func TestAppendStep(t *testing.T) {
	var steps []compiler.Step
	added := make(map[string]struct{})

	appendStep(&steps, added, nil)
	require.Len(t, steps, 0)

	step := &fakeStep{id: compiler.MustNewStepID("test:step")}
	appendStep(&steps, added, step)
	require.Len(t, steps, 1)

	appendStep(&steps, added, step)
	require.Len(t, steps, 1)
}
