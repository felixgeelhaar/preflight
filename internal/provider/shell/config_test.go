package shell_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_EmptyMap(t *testing.T) {
	t.Parallel()

	cfg, err := shell.ParseConfig(map[string]interface{}{})

	require.NoError(t, err)
	assert.Empty(t, cfg.Shells)
}

func TestParseConfig_SingleShell(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"shells": []interface{}{
			map[string]interface{}{
				"name":      "zsh",
				"framework": "oh-my-zsh",
				"theme":     "robbyrussell",
			},
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Shells, 1)
	assert.Equal(t, "zsh", cfg.Shells[0].Name)
	assert.Equal(t, "oh-my-zsh", cfg.Shells[0].Framework)
	assert.Equal(t, "robbyrussell", cfg.Shells[0].Theme)
}

func TestParseConfig_ShellWithPlugins(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"shells": []interface{}{
			map[string]interface{}{
				"name":      "zsh",
				"framework": "oh-my-zsh",
				"plugins": []interface{}{
					"git",
					"docker",
					"kubectl",
				},
			},
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Shells, 1)
	assert.ElementsMatch(t, []string{"git", "docker", "kubectl"}, cfg.Shells[0].Plugins)
}

func TestParseConfig_ShellWithCustomPlugins(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"shells": []interface{}{
			map[string]interface{}{
				"name":      "zsh",
				"framework": "oh-my-zsh",
				"custom_plugins": []interface{}{
					map[string]interface{}{
						"name": "zsh-autosuggestions",
						"repo": "zsh-users/zsh-autosuggestions",
					},
					map[string]interface{}{
						"name": "zsh-syntax-highlighting",
						"repo": "zsh-users/zsh-syntax-highlighting",
					},
				},
			},
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Shells, 1)
	require.Len(t, cfg.Shells[0].CustomPlugins, 2)
	assert.Equal(t, "zsh-autosuggestions", cfg.Shells[0].CustomPlugins[0].Name)
	assert.Equal(t, "zsh-users/zsh-autosuggestions", cfg.Shells[0].CustomPlugins[0].Repo)
}

func TestParseConfig_FishWithFisher(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"shells": []interface{}{
			map[string]interface{}{
				"name":      "fish",
				"framework": "fisher",
				"plugins": []interface{}{
					"jorgebucaran/autopair.fish",
					"PatrickF1/fzf.fish",
				},
			},
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Shells, 1)
	assert.Equal(t, "fish", cfg.Shells[0].Name)
	assert.Equal(t, "fisher", cfg.Shells[0].Framework)
	assert.Len(t, cfg.Shells[0].Plugins, 2)
}

func TestParseConfig_Starship(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"starship": map[string]interface{}{
			"enabled": true,
			"preset":  "nerd-font-symbols",
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	assert.True(t, cfg.Starship.Enabled)
	assert.Equal(t, "nerd-font-symbols", cfg.Starship.Preset)
}

func TestParseConfig_MultipleShells(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"shells": []interface{}{
			map[string]interface{}{
				"name":      "zsh",
				"framework": "oh-my-zsh",
			},
			map[string]interface{}{
				"name":      "fish",
				"framework": "fisher",
			},
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Shells, 2)
	assert.Equal(t, "zsh", cfg.Shells[0].Name)
	assert.Equal(t, "fish", cfg.Shells[1].Name)
}

func TestParseConfig_DefaultShell(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"default": "zsh",
		"shells": []interface{}{
			map[string]interface{}{
				"name": "zsh",
			},
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "zsh", cfg.Default)
}

func TestParseConfig_EnvVars(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"env": map[string]interface{}{
			"EDITOR": "nvim",
			"PAGER":  "less",
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "nvim", cfg.Env["EDITOR"])
	assert.Equal(t, "less", cfg.Env["PAGER"])
}

func TestParseConfig_Aliases(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"aliases": map[string]interface{}{
			"ll":  "ls -la",
			"vim": "nvim",
		},
	}

	cfg, err := shell.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "ls -la", cfg.Aliases["ll"])
	assert.Equal(t, "nvim", cfg.Aliases["vim"])
}

func TestShellConfig_ConfigPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		shell    string
		expected string
	}{
		{"zsh", "zsh", "~/.zshrc"},
		{"bash", "bash", "~/.bashrc"},
		{"fish", "fish", "~/.config/fish/config.fish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sc := shell.Entry{Name: tt.shell}
			assert.Equal(t, tt.expected, sc.ConfigPath())
		})
	}
}
