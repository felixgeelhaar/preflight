package vscode_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/vscode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := vscode.ParseConfig(nil)
	require.NoError(t, err)
	assert.Empty(t, cfg.Extensions)
	assert.Empty(t, cfg.Settings)
}

func TestParseConfig_Extensions(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{
			"ms-python.python",
			"golang.go",
			"esbenp.prettier-vscode",
		},
	}

	cfg, err := vscode.ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.Extensions, 3)
	assert.Equal(t, "ms-python.python", cfg.Extensions[0])
	assert.Equal(t, "golang.go", cfg.Extensions[1])
	assert.Equal(t, "esbenp.prettier-vscode", cfg.Extensions[2])
}

func TestParseConfig_Settings(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"settings": map[string]interface{}{
			"editor.fontSize":     14,
			"editor.tabSize":      2,
			"editor.formatOnSave": true,
		},
	}

	cfg, err := vscode.ParseConfig(raw)
	require.NoError(t, err)
	assert.Equal(t, 14, cfg.Settings["editor.fontSize"])
	assert.Equal(t, 2, cfg.Settings["editor.tabSize"])
	assert.Equal(t, true, cfg.Settings["editor.formatOnSave"])
}

func TestParseConfig_Keybindings(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"keybindings": []interface{}{
			map[string]interface{}{
				"key":     "ctrl+shift+p",
				"command": "workbench.action.showCommands",
			},
			map[string]interface{}{
				"key":     "ctrl+b",
				"command": "workbench.action.toggleSidebarVisibility",
			},
		},
	}

	cfg, err := vscode.ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.Keybindings, 2)
	assert.Equal(t, "ctrl+shift+p", cfg.Keybindings[0].Key)
	assert.Equal(t, "workbench.action.showCommands", cfg.Keybindings[0].Command)
}

func TestParseConfig_Full(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
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
	}

	cfg, err := vscode.ParseConfig(raw)
	require.NoError(t, err)
	assert.Len(t, cfg.Extensions, 2)
	assert.Len(t, cfg.Settings, 1)
	assert.Len(t, cfg.Keybindings, 1)
}
