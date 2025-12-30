package cursor_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/cursor"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := cursor.NewProvider(runner)

	assert.Equal(t, "cursor", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := cursor.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := cursor.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"cursor": map[string]interface{}{},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithExtensions(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := cursor.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"cursor": map[string]interface{}{
			"extensions": []interface{}{"ms-python.python", "golang.go"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, "cursor:extension:ms-python.python", steps[0].ID().String())
	assert.Equal(t, "cursor:extension:golang.go", steps[1].ID().String())
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := cursor.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"cursor": map[string]interface{}{
			"settings": map[string]interface{}{
				"editor.fontSize": 14,
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "cursor:settings", steps[0].ID().String())
}

func TestProvider_Compile_WithKeybindings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := cursor.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"cursor": map[string]interface{}{
			"keybindings": []interface{}{
				map[string]interface{}{
					"key":     "ctrl+shift+p",
					"command": "workbench.action.showCommands",
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "cursor:keybindings", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := cursor.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"cursor": map[string]interface{}{
			"extensions": []interface{}{"ms-python.python"},
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
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)
	// 1 extension + 1 settings + 1 keybindings
	assert.Equal(t, "cursor:extension:ms-python.python", steps[0].ID().String())
	assert.Equal(t, "cursor:settings", steps[1].ID().String())
	assert.Equal(t, "cursor:keybindings", steps[2].ID().String())
}
