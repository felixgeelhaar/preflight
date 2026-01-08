package helix_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/helix"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	assert.Equal(t, "helix", provider.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"helix": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithSource(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"helix": map[string]interface{}{
			"source": "dotfiles/.config/helix/config.toml",
			"link":   true,
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "helix:config", steps[0].ID().String())
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"helix": map[string]interface{}{
			"editor": map[string]interface{}{
				"line-number": "relative",
				"mouse":       false,
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "helix:config", steps[0].ID().String())
}

func TestProvider_Compile_WithLanguages(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"helix": map[string]interface{}{
			"languages": "dotfiles/.config/helix/languages.toml",
			"link":      true,
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "helix:languages", steps[0].ID().String())
}

func TestProvider_Compile_WithTheme(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"helix": map[string]interface{}{
			"theme": "catppuccin_mocha",
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "helix:theme", steps[0].ID().String())
}

func TestProvider_Compile_WithCustomTheme(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"helix": map[string]interface{}{
			"theme":        "my_custom_theme",
			"theme_source": "dotfiles/.config/helix/themes/my_custom_theme.toml",
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "helix:theme", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"helix": map[string]interface{}{
			"editor": map[string]interface{}{
				"line-number": "relative",
			},
			"languages": "dotfiles/.config/helix/languages.toml",
			"theme":     "catppuccin_mocha",
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 3) // config + languages + theme
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := helix.NewProvider(runner)

	// Cannot have both source and settings
	ctx := compiler.NewCompileContext(map[string]interface{}{
		"helix": map[string]interface{}{
			"source": "config.toml",
			"editor": map[string]interface{}{
				"line-number": "relative",
			},
		},
	})
	_, err := provider.Compile(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify both")
}
