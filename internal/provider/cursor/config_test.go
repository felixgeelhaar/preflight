package cursor_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/cursor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := cursor.ParseConfig(map[string]interface{}{})

	require.NoError(t, err)
	assert.Empty(t, cfg.Extensions)
	assert.Empty(t, cfg.Settings)
	assert.Empty(t, cfg.Keybindings)
}

func TestParseConfig_WithExtensions(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{"ms-python.python", "golang.go"},
	}

	cfg, err := cursor.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Extensions, 2)
	assert.Contains(t, cfg.Extensions, "ms-python.python")
	assert.Contains(t, cfg.Extensions, "golang.go")
}

func TestParseConfig_WithSettings(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"settings": map[string]interface{}{
			"editor.fontSize":   14,
			"editor.tabSize":    4,
			"files.autoSave":    "afterDelay",
		},
	}

	cfg, err := cursor.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Settings, 3)
	assert.Equal(t, 14, cfg.Settings["editor.fontSize"])
}

func TestParseConfig_WithKeybindings(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"keybindings": []interface{}{
			map[string]interface{}{
				"key":     "ctrl+shift+p",
				"command": "workbench.action.showCommands",
			},
			map[string]interface{}{
				"key":     "ctrl+`",
				"command": "workbench.action.terminal.toggleTerminal",
				"when":    "editorFocus",
			},
		},
	}

	cfg, err := cursor.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Keybindings, 2)
	assert.Equal(t, "ctrl+shift+p", cfg.Keybindings[0].Key)
	assert.Equal(t, "workbench.action.showCommands", cfg.Keybindings[0].Command)
	assert.Empty(t, cfg.Keybindings[0].When)
	assert.Equal(t, "editorFocus", cfg.Keybindings[1].When)
}

func TestParseConfig_Complete(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{"ms-python.python"},
		"settings": map[string]interface{}{
			"editor.fontSize": 14,
		},
		"keybindings": []interface{}{
			map[string]interface{}{
				"key":     "ctrl+p",
				"command": "quickOpen",
			},
		},
	}

	cfg, err := cursor.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Extensions, 1)
	assert.Len(t, cfg.Settings, 1)
	assert.Len(t, cfg.Keybindings, 1)
}

func TestParseConfig_InvalidExtensionsList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": "not-a-list",
	}

	cfg, err := cursor.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extensions must be a list")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidExtensionItem(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{123},
	}

	cfg, err := cursor.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extension must be a string")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidKeybinding_NotObject(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"keybindings": []interface{}{
			"not-an-object",
		},
	}

	cfg, err := cursor.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "keybinding must be an object")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidKeybinding_NoKey(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"keybindings": []interface{}{
			map[string]interface{}{
				"command": "someCommand",
			},
		},
	}

	cfg, err := cursor.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "keybinding must have a key")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidKeybinding_NoCommand(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"keybindings": []interface{}{
			map[string]interface{}{
				"key": "ctrl+p",
			},
		},
	}

	cfg, err := cursor.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "keybinding must have a command")
	assert.Nil(t, cfg)
}
