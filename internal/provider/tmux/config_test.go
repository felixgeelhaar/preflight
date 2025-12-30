package tmux_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := tmux.ParseConfig(map[string]interface{}{})

	require.NoError(t, err)
	assert.Empty(t, cfg.Plugins)
	assert.Empty(t, cfg.Settings)
	assert.Empty(t, cfg.ConfigFile)
}

func TestParseConfig_WithPlugins(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugins": []interface{}{"tmux-plugins/tpm", "tmux-plugins/tmux-sensible"},
	}

	cfg, err := tmux.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Plugins, 2)
	assert.Contains(t, cfg.Plugins, "tmux-plugins/tpm")
	assert.Contains(t, cfg.Plugins, "tmux-plugins/tmux-sensible")
}

func TestParseConfig_WithSettings(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"settings": map[string]interface{}{
			"prefix":     "C-a",
			"mouse":      "on",
			"base-index": "1",
		},
	}

	cfg, err := tmux.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Settings, 3)
	assert.Equal(t, "C-a", cfg.Settings["prefix"])
	assert.Equal(t, "on", cfg.Settings["mouse"])
	assert.Equal(t, "1", cfg.Settings["base-index"])
}

func TestParseConfig_WithConfigFile(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"config_file": "~/.config/tmux/tmux.conf",
	}

	cfg, err := tmux.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "~/.config/tmux/tmux.conf", cfg.ConfigFile)
}

func TestParseConfig_Complete(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugins": []interface{}{"tmux-plugins/tpm"},
		"settings": map[string]interface{}{
			"prefix": "C-a",
		},
		"config_file": "~/.tmux.conf",
	}

	cfg, err := tmux.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Plugins, 1)
	assert.Equal(t, "tmux-plugins/tpm", cfg.Plugins[0])
	assert.Len(t, cfg.Settings, 1)
	assert.Equal(t, "C-a", cfg.Settings["prefix"])
	assert.Equal(t, "~/.tmux.conf", cfg.ConfigFile)
}

func TestParseConfig_InvalidPluginsList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugins": "not-a-list",
	}

	cfg, err := tmux.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugins must be a list")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidPluginItem(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugins": []interface{}{123},
	}

	cfg, err := tmux.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin must be a string")
	assert.Nil(t, cfg)
}

func TestParseConfig_SettingsWithNonStringValues(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"settings": map[string]interface{}{
			"prefix": "C-a",
			"number": 123, // Non-string values are silently ignored
		},
	}

	cfg, err := tmux.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Settings, 1)
	assert.Equal(t, "C-a", cfg.Settings["prefix"])
}
