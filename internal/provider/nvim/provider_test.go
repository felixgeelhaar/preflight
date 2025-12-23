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
