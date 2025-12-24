package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Defaults(t *testing.T) {
	raw := map[string]interface{}{}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)

	assert.True(t, cfg.Install)
	assert.True(t, cfg.Compose)
	assert.False(t, cfg.Kubernetes)
	assert.True(t, cfg.BuildKit)
	assert.Nil(t, cfg.ResourceLimits)
	assert.Empty(t, cfg.Registries)
	assert.Empty(t, cfg.Contexts)
}

func TestParseConfig_AllOptions(t *testing.T) {
	raw := map[string]interface{}{
		"install":    true,
		"compose":    true,
		"kubernetes": true,
		"buildkit":   false,
		"resource_limits": map[string]interface{}{
			"cpus":   4,
			"memory": "8GB",
			"swap":   "2GB",
			"disk":   "100GB",
		},
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)

	assert.True(t, cfg.Install)
	assert.True(t, cfg.Compose)
	assert.True(t, cfg.Kubernetes)
	assert.False(t, cfg.BuildKit)
	require.NotNil(t, cfg.ResourceLimits)
	assert.Equal(t, 4, cfg.ResourceLimits.CPUs)
	assert.Equal(t, "8GB", cfg.ResourceLimits.Memory)
	assert.Equal(t, "2GB", cfg.ResourceLimits.Swap)
	assert.Equal(t, "100GB", cfg.ResourceLimits.Disk)
}

func TestParseConfig_Registries_String(t *testing.T) {
	raw := map[string]interface{}{
		"registries": []interface{}{
			"ghcr.io",
			"quay.io",
		},
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)

	assert.Len(t, cfg.Registries, 2)
	assert.Equal(t, "ghcr.io", cfg.Registries[0].URL)
	assert.Equal(t, "quay.io", cfg.Registries[1].URL)
}

func TestParseConfig_Registries_Object(t *testing.T) {
	raw := map[string]interface{}{
		"registries": []interface{}{
			map[string]interface{}{
				"url":      "registry.example.com",
				"username": "admin",
				"insecure": true,
			},
		},
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)

	assert.Len(t, cfg.Registries, 1)
	assert.Equal(t, "registry.example.com", cfg.Registries[0].URL)
	assert.Equal(t, "admin", cfg.Registries[0].Username)
	assert.True(t, cfg.Registries[0].Insecure)
}

func TestParseConfig_Registries_MissingURL(t *testing.T) {
	raw := map[string]interface{}{
		"registries": []interface{}{
			map[string]interface{}{
				"username": "admin",
			},
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "url")
}

func TestParseConfig_Registries_InvalidType(t *testing.T) {
	raw := map[string]interface{}{
		"registries": []interface{}{
			123, // invalid type
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "string or object")
}

func TestParseConfig_Registries_NotList(t *testing.T) {
	raw := map[string]interface{}{
		"registries": "not-a-list",
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a list")
}

func TestParseConfig_Contexts(t *testing.T) {
	raw := map[string]interface{}{
		"contexts": []interface{}{
			map[string]interface{}{
				"name":        "production",
				"description": "Production Docker host",
				"host":        "ssh://admin@prod.example.com",
				"default":     true,
			},
			map[string]interface{}{
				"name": "staging",
				"host": "ssh://admin@staging.example.com",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)

	assert.Len(t, cfg.Contexts, 2)
	assert.Equal(t, "production", cfg.Contexts[0].Name)
	assert.Equal(t, "Production Docker host", cfg.Contexts[0].Description)
	assert.Equal(t, "ssh://admin@prod.example.com", cfg.Contexts[0].Host)
	assert.True(t, cfg.Contexts[0].Default)
	assert.Equal(t, "staging", cfg.Contexts[1].Name)
	assert.False(t, cfg.Contexts[1].Default)
}

func TestParseConfig_Contexts_MissingName(t *testing.T) {
	raw := map[string]interface{}{
		"contexts": []interface{}{
			map[string]interface{}{
				"host": "ssh://admin@example.com",
			},
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestParseConfig_Contexts_MissingHost(t *testing.T) {
	raw := map[string]interface{}{
		"contexts": []interface{}{
			map[string]interface{}{
				"name": "production",
			},
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host")
}

func TestParseConfig_Contexts_InvalidType(t *testing.T) {
	raw := map[string]interface{}{
		"contexts": []interface{}{
			"not-an-object",
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be an object")
}

func TestParseConfig_Contexts_NotList(t *testing.T) {
	raw := map[string]interface{}{
		"contexts": "not-a-list",
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a list")
}

func TestParseConfig_ResourceLimits_InvalidType(t *testing.T) {
	raw := map[string]interface{}{
		"resource_limits": "not-an-object",
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be an object")
}

func TestParseConfig_DisabledInstall(t *testing.T) {
	raw := map[string]interface{}{
		"install": false,
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)

	assert.False(t, cfg.Install)
}
