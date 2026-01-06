package deps

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/stretchr/testify/require"
)

func TestResolveToolDependency_ExplicitBrew(t *testing.T) {
	cfg := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"node"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	dep, ok := ResolveToolDependency(ctx, plat, ToolNode)
	require.True(t, ok)
	require.True(t, dep.Explicit)
	require.Equal(t, "brew", dep.Manager)
	require.Equal(t, "brew:formula:node", dep.StepID.String())
}

func TestResolveToolBootstrap_ImplicitBrew(t *testing.T) {
	cfg := map[string]interface{}{
		"npm": map[string]interface{}{
			"packages": []interface{}{"eslint"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	boot, ok := ResolveToolBootstrap(ctx, plat, ToolNode)
	require.True(t, ok)
	require.Equal(t, ToolNode, boot.Tool)
	require.Equal(t, "brew", boot.Manager)
	require.Equal(t, "node", boot.PackageName)
	require.Equal(t, "bootstrap:tool:node", boot.StepID.String())
}
