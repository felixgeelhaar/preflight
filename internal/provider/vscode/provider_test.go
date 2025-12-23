package vscode_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/vscode"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner)

	assert.Equal(t, "vscode", p.Name())
}

func TestProvider_Compile_Empty(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithExtensions(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{
				"ms-python.python",
				"golang.go",
			},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 2)
	assert.Equal(t, "vscode:extension:ms-python_python", steps[0].ID().String())
	assert.Equal(t, "vscode:extension:golang_go", steps[1].ID().String())
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"settings": map[string]interface{}{
				"editor.fontSize": 14,
			},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "vscode:settings", steps[0].ID().String())
}

func TestProvider_Compile_WithKeybindings(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"keybindings": []interface{}{
				map[string]interface{}{
					"key":     "ctrl+shift+p",
					"command": "workbench.action.showCommands",
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "vscode:keybindings", steps[0].ID().String())
}

func TestProvider_Compile_Full(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{
				"ms-python.python",
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
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// 2 extensions + 1 settings + 1 keybindings = 4 steps
	assert.Len(t, steps, 4)
}
