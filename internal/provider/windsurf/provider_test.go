package windsurf_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/windsurf"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := windsurf.NewProvider(runner)

	assert.Equal(t, "windsurf", provider.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := windsurf.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := windsurf.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"windsurf": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithExtensions(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := windsurf.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"windsurf": map[string]interface{}{
			"extensions": []interface{}{
				"golang.go",
				"rust-lang.rust-analyzer",
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 2)
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := windsurf.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"windsurf": map[string]interface{}{
			"settings": map[string]interface{}{
				"editor.fontSize": 14,
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
}

func TestProvider_Compile_WithKeybindings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := windsurf.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"windsurf": map[string]interface{}{
			"keybindings": []interface{}{
				map[string]interface{}{
					"key":     "ctrl+shift+p",
					"command": "workbench.action.showCommands",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := windsurf.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"windsurf": map[string]interface{}{
			"extensions": []interface{}{
				"golang.go",
			},
			"settings": map[string]interface{}{
				"editor.fontSize": 14,
			},
			"keybindings": []interface{}{
				map[string]interface{}{
					"key":     "ctrl+shift+p",
					"command": "workbench.action.showCommands",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 3) // 1 extension + 1 settings + 1 keybindings
}
