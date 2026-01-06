package winget

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWingetProvider_Name(t *testing.T) {
	t.Parallel()

	provider := NewProvider(nil, nil)

	assert.Equal(t, "winget", provider.Name())
}

func TestWingetProvider_Compile_Empty(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestWingetProvider_Compile_NoWingetSection(t *testing.T) {
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

func TestWingetProvider_Compile_Packages(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"winget": map[string]interface{}{
			"packages": []interface{}{
				"Microsoft.VisualStudioCode",
				"Git.Git",
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)

	// Verify step IDs
	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	assert.True(t, ids[wingetReadyStepID])
	assert.True(t, ids["winget:package:Microsoft.VisualStudioCode"])
	assert.True(t, ids["winget:package:Git.Git"])
}

func TestWingetProvider_Compile_PackagesWithVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"winget": map[string]interface{}{
			"packages": []interface{}{
				map[string]interface{}{
					"id":      "Microsoft.VisualStudioCode",
					"version": "1.85.0",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
}

func TestWingetProvider_Compile_InvalidConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"winget": map[string]interface{}{
			"packages": "not-a-list",
		},
	})
	_, err := provider.Compile(ctx)

	assert.Error(t, err)
}

func TestWingetProvider_Compile_SkipsOnNonWindows(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSDarwin, "arm64", platform.EnvNative)
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"winget": map[string]interface{}{
			"packages": []interface{}{"Microsoft.VisualStudioCode"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestWingetProvider_Compile_WorksOnWSL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	provider := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"winget": map[string]interface{}{
			"packages": []interface{}{"Microsoft.VisualStudioCode"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
}

func TestWingetProvider_Compile_WorksWithNilPlatform(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner, nil)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"winget": map[string]interface{}{
			"packages": []interface{}{"Microsoft.VisualStudioCode"},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
}
