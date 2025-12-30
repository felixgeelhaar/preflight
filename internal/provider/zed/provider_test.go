package zed_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/zed"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := zed.NewProvider(runner)

	assert.Equal(t, "zed", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := zed.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := zed.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"zed": map[string]interface{}{},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithExtensions(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := zed.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"zed": map[string]interface{}{
			"extensions": []interface{}{"python", "go"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, "zed:extension:python", steps[0].ID().String())
	assert.Equal(t, "zed:extension:go", steps[1].ID().String())
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := zed.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"zed": map[string]interface{}{
			"settings": map[string]interface{}{
				"tab_size": 4,
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "zed:settings", steps[0].ID().String())
}

func TestProvider_Compile_WithKeymap(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := zed.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"zed": map[string]interface{}{
			"keymap": []interface{}{
				map[string]interface{}{
					"context": "Editor",
					"bindings": map[string]interface{}{
						"ctrl-p": "file_finder::Toggle",
					},
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "zed:keymap", steps[0].ID().String())
}

func TestProvider_Compile_WithTheme(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := zed.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"zed": map[string]interface{}{
			"theme": "one-dark",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "zed:theme:one-dark", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := zed.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"zed": map[string]interface{}{
			"extensions": []interface{}{"python"},
			"settings": map[string]interface{}{
				"tab_size": 4,
			},
			"keymap": []interface{}{
				map[string]interface{}{
					"context": "Editor",
					"bindings": map[string]interface{}{
						"ctrl-p": "file_finder::Toggle",
					},
				},
			},
			"theme": "one-dark",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 4)
	assert.Equal(t, "zed:extension:python", steps[0].ID().String())
	assert.Equal(t, "zed:settings", steps[1].ID().String())
	assert.Equal(t, "zed:keymap", steps[2].ID().String())
	assert.Equal(t, "zed:theme:one-dark", steps[3].ID().String())
}
