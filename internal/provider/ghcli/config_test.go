package ghcli_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := ghcli.ParseConfig(map[string]interface{}{})

	require.NoError(t, err)
	assert.Empty(t, cfg.Extensions)
	assert.Empty(t, cfg.Aliases)
	assert.Empty(t, cfg.Config)
}

func TestParseConfig_WithExtensions(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{"dlvhdr/gh-dash", "github/gh-copilot"},
	}

	cfg, err := ghcli.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Extensions, 2)
	assert.Contains(t, cfg.Extensions, "dlvhdr/gh-dash")
	assert.Contains(t, cfg.Extensions, "github/gh-copilot")
}

func TestParseConfig_WithAliases(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"aliases": map[string]interface{}{
			"co":  "pr checkout",
			"prc": "pr create",
		},
	}

	cfg, err := ghcli.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Aliases, 2)
	assert.Equal(t, "pr checkout", cfg.Aliases["co"])
	assert.Equal(t, "pr create", cfg.Aliases["prc"])
}

func TestParseConfig_WithConfig(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"config": map[string]interface{}{
			"editor":      "nvim",
			"git_protocol": "ssh",
		},
	}

	cfg, err := ghcli.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Config, 2)
	assert.Equal(t, "nvim", cfg.Config["editor"])
	assert.Equal(t, "ssh", cfg.Config["git_protocol"])
}

func TestParseConfig_Complete(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{"dlvhdr/gh-dash"},
		"aliases": map[string]interface{}{
			"co": "pr checkout",
		},
		"config": map[string]interface{}{
			"editor": "nvim",
		},
	}

	cfg, err := ghcli.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Extensions, 1)
	assert.Len(t, cfg.Aliases, 1)
	assert.Len(t, cfg.Config, 1)
}

func TestParseConfig_InvalidExtensionsList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": "not-a-list",
	}

	cfg, err := ghcli.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extensions must be a list")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidExtensionItem(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"extensions": []interface{}{123},
	}

	cfg, err := ghcli.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extension must be a string")
	assert.Nil(t, cfg)
}
