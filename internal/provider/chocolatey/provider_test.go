package chocolatey

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	assert.Equal(t, "chocolatey", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_SkipsNonWindows(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSDarwin, "arm64", platform.EnvNative)
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps, "Should skip on non-Windows platforms")
}

func TestProvider_Compile_SkipsLinux(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSLinux, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps, "Should skip on Linux (non-WSL)")
}

func TestProvider_Compile_WorksOnWindows(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"packages": []interface{}{"git", "nodejs"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 2)
}

func TestProvider_Compile_WorksOnWSL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1, "Should work on WSL")
}

func TestProvider_Compile_Packages(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"packages": []interface{}{
				"git",
				map[string]interface{}{
					"name":    "nodejs",
					"version": "18.0.0",
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)

	// Verify step IDs
	assert.Equal(t, compiler.MustNewStepID("chocolatey:package:git"), steps[0].ID())
	assert.Equal(t, compiler.MustNewStepID("chocolatey:package:nodejs"), steps[1].ID())
}

func TestProvider_Compile_Sources(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"sources": []interface{}{
				map[string]interface{}{
					"name": "internal",
					"url":  "https://nuget.internal.com/v3/",
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)

	// Verify step ID
	assert.Equal(t, compiler.MustNewStepID("chocolatey:source:internal"), steps[0].ID())
}

func TestProvider_Compile_SourcesAndPackages(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"sources": []interface{}{
				map[string]interface{}{
					"name": "internal",
					"url":  "https://nuget.internal.com/v3/",
				},
			},
			"packages": []interface{}{
				"git",
				"nodejs",
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)

	// Sources should come before packages
	assert.Equal(t, compiler.MustNewStepID("chocolatey:source:internal"), steps[0].ID())
	assert.Equal(t, compiler.MustNewStepID("chocolatey:package:git"), steps[1].ID())
	assert.Equal(t, compiler.MustNewStepID("chocolatey:package:nodejs"), steps[2].ID())
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"packages": "not-a-list",
		},
	})
	_, err := p.Compile(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a list")
}

func TestProvider_Compile_NilPlatform(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := NewProvider(runner, nil) // nil platform

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"packages": []interface{}{"git"},
		},
	})
	steps, err := p.Compile(ctx)

	// With nil platform, should proceed (for testing purposes)
	require.NoError(t, err)
	assert.Len(t, steps, 1)
}

func TestProvider_ImplementsInterface(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := NewProvider(runner, plat)

	// Verify it implements the compiler.Provider interface
	var _ compiler.Provider = p
}
