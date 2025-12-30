package zed_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/zed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := zed.ParseConfig(map[string]interface{}{})

	require.NoError(t, err)
	assert.Empty(t, cfg.Extensions)
	assert.Empty(t, cfg.Settings)
	assert.Empty(t, cfg.Keymap)
	assert.Empty(t, cfg.Theme)
}

func TestParseConfig_WithExtensions(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{"python", "go", "rust"},
	}

	cfg, err := zed.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Extensions, 3)
	assert.Contains(t, cfg.Extensions, "python")
	assert.Contains(t, cfg.Extensions, "go")
	assert.Contains(t, cfg.Extensions, "rust")
}

func TestParseConfig_WithSettings(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"settings": map[string]interface{}{
			"tab_size":       4,
			"format_on_save": "on",
			"vim_mode":       true,
		},
	}

	cfg, err := zed.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Settings, 3)
	assert.Equal(t, 4, cfg.Settings["tab_size"])
	assert.Equal(t, true, cfg.Settings["vim_mode"])
}

func TestParseConfig_WithKeymap(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"keymap": []interface{}{
			map[string]interface{}{
				"context": "Editor",
				"bindings": map[string]interface{}{
					"ctrl-p": "file_finder::Toggle",
					"ctrl-s": "workspace::Save",
				},
			},
			map[string]interface{}{
				"bindings": map[string]interface{}{
					"ctrl-q": "zed::Quit",
				},
			},
		},
	}

	cfg, err := zed.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Keymap, 2)
	assert.Equal(t, "Editor", cfg.Keymap[0].Context)
	assert.Len(t, cfg.Keymap[0].Bindings, 2)
	assert.Empty(t, cfg.Keymap[1].Context)
}

func TestParseConfig_WithTheme(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"theme": "one-dark",
	}

	cfg, err := zed.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "one-dark", cfg.Theme)
}

func TestParseConfig_Complete(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
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
	}

	cfg, err := zed.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Extensions, 1)
	assert.Len(t, cfg.Settings, 1)
	assert.Len(t, cfg.Keymap, 1)
	assert.Equal(t, "one-dark", cfg.Theme)
}

func TestParseConfig_InvalidExtensionsList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": "not-a-list",
	}

	cfg, err := zed.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extensions must be a list")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidExtensionItem(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{123},
	}

	cfg, err := zed.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extension must be a string")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidKeybinding_NotObject(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"keymap": []interface{}{
			"not-an-object",
		},
	}

	cfg, err := zed.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "keybinding must be an object")
	assert.Nil(t, cfg)
}
