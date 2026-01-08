package terminal

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())
	assert.Equal(t, "terminal", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptyTerminalConfig(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_AlacrittyConfig(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"alacritty": map[string]interface{}{
				"source": "alacritty/alacritty.toml",
				"link":   true,
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg).WithConfigRoot("/tmp/dotfiles")

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:alacritty:config", steps[0].ID().String())
}

func TestProvider_Compile_KittyConfig(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"kitty": map[string]interface{}{
				"settings": map[string]interface{}{
					"font_family": "JetBrains Mono",
					"font_size":   12,
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:kitty:config", steps[0].ID().String())
}

func TestProvider_Compile_WezTermConfig(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"wezterm": map[string]interface{}{
				"source": "wezterm/wezterm.lua",
				"link":   true,
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg).WithConfigRoot("/tmp/dotfiles")

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:wezterm:config", steps[0].ID().String())
}

func TestProvider_Compile_GhosttyConfig(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"ghostty": map[string]interface{}{
				"settings": map[string]interface{}{
					"font-family": "JetBrains Mono",
					"font-size":   14,
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:ghostty:config", steps[0].ID().String())
}

func TestProvider_Compile_HyperConfig(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"hyper": map[string]interface{}{
				"source": "hyper/.hyper.js",
				"link":   true,
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg).WithConfigRoot("/tmp/dotfiles")

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:hyper:config", steps[0].ID().String())
}

func TestProvider_Compile_WindowsTerminalConfig(t *testing.T) {
	p := NewProviderWithDiscovery(
		mocks.NewFileSystem(),
		mocks.NewCommandRunner(),
		NewDiscovery(),
		"windows", // Simulate Windows
	)

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"windows_terminal": map[string]interface{}{
				"settings": map[string]interface{}{
					"copyOnSelect": true,
				},
				"profiles": []interface{}{
					map[string]interface{}{
						"name":         "PowerShell",
						"command_line": "pwsh.exe",
					},
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:windows-terminal:config", steps[0].ID().String())
}

func TestProvider_Compile_MultipleTerminals(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"alacritty": map[string]interface{}{
				"source": "alacritty/alacritty.toml",
			},
			"kitty": map[string]interface{}{
				"settings": map[string]interface{}{
					"font_size": 12,
				},
			},
			"ghostty": map[string]interface{}{
				"settings": map[string]interface{}{
					"font-size": 14,
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg).WithConfigRoot("/tmp/dotfiles")

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	assert.Len(t, steps, 3)

	// Verify step IDs
	stepIDs := make(map[string]bool)
	for _, step := range steps {
		stepIDs[step.ID().String()] = true
	}
	assert.True(t, stepIDs["terminal:alacritty:config"])
	assert.True(t, stepIDs["terminal:kitty:config"])
	assert.True(t, stepIDs["terminal:ghostty:config"])
}

func TestProvider_Compile_ITerm2OnMac(t *testing.T) {
	p := NewProviderWithDiscovery(
		mocks.NewFileSystem(),
		mocks.NewCommandRunner(),
		NewDiscovery(),
		"darwin", // Simulate macOS
	)

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"iterm2": map[string]interface{}{
				"settings": map[string]interface{}{
					"HideScrollbar": true,
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:iterm2:settings", steps[0].ID().String())
}

func TestProvider_Compile_ITerm2IgnoredOnLinux(t *testing.T) {
	p := NewProviderWithDiscovery(
		mocks.NewFileSystem(),
		mocks.NewCommandRunner(),
		NewDiscovery(),
		"linux", // Simulate Linux
	)

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"iterm2": map[string]interface{}{
				"settings": map[string]interface{}{
					"HideScrollbar": true,
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	assert.Empty(t, steps, "iTerm2 should be ignored on Linux")
}

func TestProvider_Compile_WindowsTerminalIgnoredOnMac(t *testing.T) {
	p := NewProviderWithDiscovery(
		mocks.NewFileSystem(),
		mocks.NewCommandRunner(),
		NewDiscovery(),
		"darwin", // Simulate macOS
	)

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"windows_terminal": map[string]interface{}{
				"settings": map[string]interface{}{
					"copyOnSelect": true,
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	assert.Empty(t, steps, "Windows Terminal should be ignored on macOS")
}
