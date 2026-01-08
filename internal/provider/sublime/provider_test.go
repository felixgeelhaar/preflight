package sublime_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/sublime"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := sublime.NewProvider(runner)

	assert.Equal(t, "sublime", provider.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := sublime.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := sublime.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"sublime": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithPackages(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := sublime.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"sublime": map[string]interface{}{
			"packages": []interface{}{
				"Package Control",
				"SublimeLinter",
				"Catppuccin",
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "sublime:packages", steps[0].ID().String())
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := sublime.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"sublime": map[string]interface{}{
			"settings": map[string]interface{}{
				"font_size":         14,
				"tab_size":          4,
				"translate_tabs_to_spaces": true,
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "sublime:settings", steps[0].ID().String())
}

func TestProvider_Compile_WithTheme(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := sublime.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"sublime": map[string]interface{}{
			"theme":        "Adaptive.sublime-theme",
			"color_scheme": "Packages/Catppuccin/Mocha.tmTheme",
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "sublime:settings", steps[0].ID().String())
}

func TestProvider_Compile_WithKeybindings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := sublime.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"sublime": map[string]interface{}{
			"keybindings": []interface{}{
				map[string]interface{}{
					"keys":    []interface{}{"ctrl+shift+p"},
					"command": "show_overlay",
					"args":    map[string]interface{}{"overlay": "command_palette"},
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "sublime:keybindings", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := sublime.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"sublime": map[string]interface{}{
			"packages": []interface{}{
				"Package Control",
				"SublimeLinter",
			},
			"settings": map[string]interface{}{
				"font_size": 14,
			},
			"theme":        "Adaptive.sublime-theme",
			"color_scheme": "Packages/Catppuccin/Mocha.tmTheme",
			"keybindings": []interface{}{
				map[string]interface{}{
					"keys":    []interface{}{"ctrl+shift+p"},
					"command": "show_overlay",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 3) // packages + settings + keybindings
}
