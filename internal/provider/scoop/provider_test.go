package scoop

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoopProvider_Name(t *testing.T) {
	t.Parallel()

	provider := NewProvider(nil, nil)

	assert.Equal(t, "scoop", provider.Name())
}

func TestScoopProvider_Compile_Empty(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestScoopProvider_Compile_NoScoopSection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestScoopProvider_Compile_Buckets(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"scoop": map[string]interface{}{
			"buckets": []interface{}{"extras", "versions"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)

	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	assert.True(t, ids[scoopInstallStepID])
	assert.True(t, ids["scoop:bucket:extras"])
	assert.True(t, ids["scoop:bucket:versions"])
}

func TestScoopProvider_Compile_Packages(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"scoop": map[string]interface{}{
			"packages": []interface{}{"git", "curl"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)
}

func TestScoopProvider_Compile_Full(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"scoop": map[string]interface{}{
			"buckets":  []interface{}{"extras"},
			"packages": []interface{}{"git", "neovim"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 4)

	// First step should be install
	assert.Equal(t, scoopInstallStepID, steps[0].ID().String())
}

func TestScoopProvider_Compile_InvalidConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"scoop": map[string]interface{}{
			"buckets": "not-a-list",
		},
	})
	_, err := provider.Compile(ctx)

	assert.Error(t, err)
}

func TestScoopProvider_Compile_SkipsOnNonWindows(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSDarwin, "arm64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"scoop": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestScoopProvider_Compile_WorksOnWSL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"scoop": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
}

func TestScoopProvider_Compile_WorksWithNilPlatform(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner, nil)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"scoop": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
}

func TestScoopProvider_Compile_PackageWithBucket(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"scoop": map[string]interface{}{
			"packages": []interface{}{
				map[string]interface{}{
					"name":   "neovim",
					"bucket": "extras",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)

	// Package should depend on bucket
	deps := steps[1].DependsOn()
	require.Len(t, deps, 2)
	assert.Equal(t, scoopInstallStepID, deps[0].String())
	assert.Equal(t, "scoop:bucket:extras", deps[1].String())
}
