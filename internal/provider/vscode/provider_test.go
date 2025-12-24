package vscode_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/provider/vscode"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner, nil)

	assert.Equal(t, "vscode", p.Name())
}

func TestProvider_Compile_Empty(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner, nil)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithExtensions(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner, nil)

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
	p := vscode.NewProvider(fs, runner, nil)

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
	p := vscode.NewProvider(fs, runner, nil)

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
	p := vscode.NewProvider(fs, runner, nil)

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

func TestProvider_Compile_WithWSL_OnWindows(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := vscode.NewProvider(fs, runner, plat)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{
				"ms-python.python",
			},
			"wsl": map[string]interface{}{
				"auto_install": true,
				"extensions": []interface{}{
					"golang.go",
					"rust-lang.rust-analyzer",
				},
				"settings": map[string]interface{}{
					"terminal.integrated.shell.linux": "/bin/bash",
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// 1 regular extension + 1 WSL setup + 2 WSL extensions + 1 WSL settings = 5 steps
	assert.Len(t, steps, 5)
	assert.Equal(t, "vscode:extension:ms-python_python", steps[0].ID().String())
	assert.Equal(t, "vscode:wsl:setup", steps[1].ID().String())
	assert.Equal(t, "vscode:wsl:extension:golang_go", steps[2].ID().String())
	assert.Equal(t, "vscode:wsl:extension:rust-lang_rust-analyzer", steps[3].ID().String())
	assert.Equal(t, "vscode:wsl:settings", steps[4].ID().String())
}

func TestProvider_Compile_WithWSL_OnWSL(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	p := vscode.NewProvider(fs, runner, plat)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"wsl": map[string]interface{}{
				"extensions": []interface{}{
					"golang.go",
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// 1 WSL setup + 1 WSL extension = 2 steps
	assert.Len(t, steps, 2)
	assert.Equal(t, "vscode:wsl:setup", steps[0].ID().String())
	assert.Equal(t, "vscode:wsl:extension:golang_go", steps[1].ID().String())
}

func TestProvider_Compile_WithWSL_OnMacOS_Skipped(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSDarwin, "arm64", platform.EnvNative)
	p := vscode.NewProvider(fs, runner, plat)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{
				"ms-python.python",
			},
			"wsl": map[string]interface{}{
				"auto_install": true,
				"extensions": []interface{}{
					"golang.go",
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// Only 1 regular extension - WSL config is skipped on macOS
	assert.Len(t, steps, 1)
	assert.Equal(t, "vscode:extension:ms-python_python", steps[0].ID().String())
}

func TestProvider_Compile_WithWSL_NilPlatform_Skipped(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	p := vscode.NewProvider(fs, runner, nil)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"wsl": map[string]interface{}{
				"auto_install": true,
			},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// WSL config is skipped when platform is nil
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithWSL_Distro(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	p := vscode.NewProvider(fs, runner, plat)

	raw := map[string]interface{}{
		"vscode": map[string]interface{}{
			"wsl": map[string]interface{}{
				"distro": "Ubuntu-22.04",
				"extensions": []interface{}{
					"golang.go",
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// 1 WSL setup + 1 WSL extension = 2 steps
	assert.Len(t, steps, 2)
}
