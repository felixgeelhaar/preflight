package nvim_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/nvim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := nvim.ParseConfig(map[string]interface{}{})
	require.NoError(t, err)
	assert.Empty(t, cfg.Preset)
	assert.Empty(t, cfg.PluginManager)
}

func TestParseConfig_Preset(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"preset": "lazyvim",
	}
	cfg, err := nvim.ParseConfig(raw)
	require.NoError(t, err)
	assert.Equal(t, "lazyvim", cfg.Preset)
}

func TestParseConfig_PluginManager(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugin_manager": "lazy",
	}
	cfg, err := nvim.ParseConfig(raw)
	require.NoError(t, err)
	assert.Equal(t, "lazy", cfg.PluginManager)
}

func TestParseConfig_ConfigRepo(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"config_repo": "https://github.com/user/nvim-config",
	}
	cfg, err := nvim.ParseConfig(raw)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/nvim-config", cfg.ConfigRepo)
}

func TestParseConfig_Full(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"preset":         "lazyvim",
		"plugin_manager": "lazy",
		"config_repo":    "https://github.com/user/nvim-config",
		"ensure_install": true,
	}
	cfg, err := nvim.ParseConfig(raw)
	require.NoError(t, err)
	assert.Equal(t, "lazyvim", cfg.Preset)
	assert.Equal(t, "lazy", cfg.PluginManager)
	assert.Equal(t, "https://github.com/user/nvim-config", cfg.ConfigRepo)
	assert.True(t, cfg.EnsureInstall)
}

func TestParseConfig_PresetTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		preset string
		valid  bool
	}{
		{"lazyvim", "lazyvim", true},
		{"nvchad", "nvchad", true},
		{"astronvim", "astronvim", true},
		{"kickstart", "kickstart", true},
		{"custom", "custom", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := map[string]interface{}{
				"preset": tt.preset,
			}
			cfg, err := nvim.ParseConfig(raw)
			require.NoError(t, err)
			assert.Equal(t, tt.preset, cfg.Preset)
		})
	}
}
