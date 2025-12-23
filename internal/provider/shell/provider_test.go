package shell_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/shell"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	assert.Equal(t, "shell", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptyShellSection(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"shell": map[string]interface{}{},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_SingleShellWithFramework(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"shell": map[string]interface{}{
			"shells": []interface{}{
				map[string]interface{}{
					"name":      "zsh",
					"framework": "oh-my-zsh",
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "shell:framework:zsh:oh-my-zsh", steps[0].ID().String())
}

func TestProvider_Compile_ShellWithPlugins(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"shell": map[string]interface{}{
			"shells": []interface{}{
				map[string]interface{}{
					"name":      "zsh",
					"framework": "oh-my-zsh",
					"plugins": []interface{}{
						"git",
						"docker",
					},
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// 1 framework step + 2 plugin steps
	require.Len(t, steps, 3)
	assert.Equal(t, "shell:framework:zsh:oh-my-zsh", steps[0].ID().String())
}

func TestProvider_Compile_ShellWithCustomPlugins(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"shell": map[string]interface{}{
			"shells": []interface{}{
				map[string]interface{}{
					"name":      "zsh",
					"framework": "oh-my-zsh",
					"custom_plugins": []interface{}{
						map[string]interface{}{
							"name": "zsh-autosuggestions",
							"repo": "zsh-users/zsh-autosuggestions",
						},
					},
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// 1 framework step + 1 custom plugin step
	require.Len(t, steps, 2)
}

func TestProvider_Compile_WithStarship(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"shell": map[string]interface{}{
			"starship": map[string]interface{}{
				"enabled": true,
				"preset":  "nerd-font-symbols",
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "shell:starship", steps[0].ID().String())
}

func TestProvider_Compile_WithEnvAndAliases(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"shell": map[string]interface{}{
			"shells": []interface{}{
				map[string]interface{}{
					"name": "zsh",
				},
			},
			"env": map[string]interface{}{
				"EDITOR": "nvim",
			},
			"aliases": map[string]interface{}{
				"ll": "ls -la",
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// env step + aliases step (no framework since not specified)
	require.Len(t, steps, 2)
}

func TestProvider_Compile_FishWithFisher(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	p := shell.NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"shell": map[string]interface{}{
			"shells": []interface{}{
				map[string]interface{}{
					"name":      "fish",
					"framework": "fisher",
					"plugins": []interface{}{
						"jorgebucaran/autopair.fish",
					},
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// 1 framework step + 1 fisher plugin step
	require.Len(t, steps, 2)
	assert.Equal(t, "shell:framework:fish:fisher", steps[0].ID().String())
}
