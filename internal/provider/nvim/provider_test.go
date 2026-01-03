package nvim_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/nvim"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	assert.Equal(t, "nvim", p.Name())
}

func TestProvider_Compile_Empty(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithPreset(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"preset": "lazyvim",
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "nvim:preset:lazyvim", steps[0].ID().String())
}

func TestProvider_Compile_WithConfigRepo(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"config_repo": "https://github.com/user/nvim-config",
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "nvim:config-repo", steps[0].ID().String())
}

func TestProvider_Compile_PresetWithLazyLock(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"preset":         "lazyvim",
			"plugin_manager": "lazy",
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// preset step + lazy-lock step
	assert.Len(t, steps, 2)
}

func TestProvider_Compile_CustomPreset_NoSteps(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	// "custom" preset means user has their own config - no step should be created
	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"preset": "custom",
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps, "custom preset should not create any steps")
}

func TestProvider_Compile_WithConfigSource_HomeMirrored(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	// Simulate home-mirrored nvim config exists
	configRoot := "/home/user/dotfiles"
	nvimPath := configRoot + "/.config/nvim"
	fs.AddDir(nvimPath)

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"config_source": ".config/nvim",
		},
	}
	ctx := compiler.NewCompileContext(raw).WithConfigRoot(configRoot)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "nvim:config-source", steps[0].ID().String())
}

func TestProvider_Compile_WithConfigSource_TargetOverride(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	// Simulate target-specific nvim config exists
	configRoot := "/home/user/dotfiles"
	workNvimPath := configRoot + "/.config.work/nvim"
	fs.AddDir(workNvimPath)

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"config_source": ".config/nvim",
		},
	}
	ctx := compiler.NewCompileContext(raw).WithConfigRoot(configRoot).WithTarget("work")
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "nvim:config-source", steps[0].ID().String())
}

func TestProvider_Compile_WithConfigSource_FallbackToShared(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	// Simulate only shared config exists (no target-specific)
	configRoot := "/home/user/dotfiles"
	sharedNvimPath := configRoot + "/.config/nvim"
	fs.AddDir(sharedNvimPath)

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"config_source": ".config/nvim",
		},
	}
	ctx := compiler.NewCompileContext(raw).WithConfigRoot(configRoot).WithTarget("work")
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
}

func TestProvider_Compile_WithConfigSource_LegacyPath(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	// Simulate legacy dotfiles/ structure
	configRoot := "/home/user/dotfiles"
	legacyPath := configRoot + "/dotfiles/.config/nvim"
	fs.AddDir(legacyPath)

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"config_source": ".config/nvim",
		},
	}
	ctx := compiler.NewCompileContext(raw).WithConfigRoot(configRoot)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
}

func TestProvider_Compile_WithConfigSource_NotFound(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	configRoot := "/home/user/dotfiles"
	// No config exists

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"config_source": ".config/nvim",
		},
	}
	ctx := compiler.NewCompileContext(raw).WithConfigRoot(configRoot)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// Should have no steps when config_source path doesn't exist
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithConfigSource_PathTraversal(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := nvim.NewProvider(fs, runner)

	configRoot := "/home/user/dotfiles"

	raw := map[string]interface{}{
		"nvim": map[string]interface{}{
			"config_source": "../../../etc/passwd",
		},
	}
	ctx := compiler.NewCompileContext(raw).WithConfigRoot(configRoot)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// Path traversal should be rejected, no steps
	assert.Empty(t, steps)
}
