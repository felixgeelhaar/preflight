package starship_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/starship"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := starship.ParseConfig(map[string]interface{}{})

	require.NoError(t, err)
	assert.Empty(t, cfg.Settings)
	assert.Empty(t, cfg.Preset)
	assert.Empty(t, cfg.Shell)
}

func TestParseConfig_WithSettings(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"settings": map[string]interface{}{
			"add_newline": false,
			"character": map[string]interface{}{
				"success_symbol": "[➜](bold green)",
			},
		},
	}

	cfg, err := starship.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Settings, 2)
	assert.Equal(t, false, cfg.Settings["add_newline"])
}

func TestParseConfig_WithPreset(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"preset": "nerd-font-symbols",
	}

	cfg, err := starship.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "nerd-font-symbols", cfg.Preset)
}

func TestParseConfig_WithShell(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"shell": "zsh",
	}

	cfg, err := starship.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "zsh", cfg.Shell)
}

func TestParseConfig_Complete(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"settings": map[string]interface{}{
			"add_newline": true,
		},
		"preset": "tokyo-night",
		"shell":  "bash",
	}

	cfg, err := starship.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Settings, 1)
	assert.Equal(t, "tokyo-night", cfg.Preset)
	assert.Equal(t, "bash", cfg.Shell)
}
