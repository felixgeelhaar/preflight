package chocolatey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := ParseConfig(map[string]interface{}{})
	require.NoError(t, err)
	assert.Empty(t, cfg.Sources)
	assert.Empty(t, cfg.Packages)
}

func TestParseConfig_Packages_StringList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			"git",
			"nodejs",
			"vscode",
		},
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 3)

	assert.Equal(t, "git", cfg.Packages[0].Name)
	assert.Equal(t, "nodejs", cfg.Packages[1].Name)
	assert.Equal(t, "vscode", cfg.Packages[2].Name)
}

func TestParseConfig_Packages_ObjectList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			map[string]interface{}{
				"name":    "git",
				"version": "2.40.0",
			},
			map[string]interface{}{
				"name":   "nodejs",
				"source": "internal",
				"pin":    true,
			},
			map[string]interface{}{
				"name": "vscode",
				"args": "/NoDesktopIcon",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 3)

	// git with version
	assert.Equal(t, "git", cfg.Packages[0].Name)
	assert.Equal(t, "2.40.0", cfg.Packages[0].Version)
	assert.False(t, cfg.Packages[0].Pin)

	// nodejs with source and pin
	assert.Equal(t, "nodejs", cfg.Packages[1].Name)
	assert.Equal(t, "internal", cfg.Packages[1].Source)
	assert.True(t, cfg.Packages[1].Pin)

	// vscode with args
	assert.Equal(t, "vscode", cfg.Packages[2].Name)
	assert.Equal(t, "/NoDesktopIcon", cfg.Packages[2].Args)
}

func TestParseConfig_Packages_MissingName(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			map[string]interface{}{
				"version": "1.0.0",
			},
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have a name")
}

func TestParseConfig_Packages_InvalidType(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			123, // Invalid type
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string or object")
}

func TestParseConfig_Packages_NotList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": "not-a-list",
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a list")
}

func TestParseConfig_Sources(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sources": []interface{}{
			map[string]interface{}{
				"name": "chocolatey",
				"url":  "https://community.chocolatey.org/api/v2/",
			},
			map[string]interface{}{
				"name":     "internal",
				"url":      "https://nuget.internal.com/v3/",
				"priority": 1,
				"disabled": true,
			},
		},
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.Sources, 2)

	// chocolatey source
	assert.Equal(t, "chocolatey", cfg.Sources[0].Name)
	assert.Equal(t, "https://community.chocolatey.org/api/v2/", cfg.Sources[0].URL)
	assert.Equal(t, 0, cfg.Sources[0].Priority)
	assert.False(t, cfg.Sources[0].Disabled)

	// internal source
	assert.Equal(t, "internal", cfg.Sources[1].Name)
	assert.Equal(t, "https://nuget.internal.com/v3/", cfg.Sources[1].URL)
	assert.Equal(t, 1, cfg.Sources[1].Priority)
	assert.True(t, cfg.Sources[1].Disabled)
}

func TestParseConfig_Sources_MissingName(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sources": []interface{}{
			map[string]interface{}{
				"url": "https://example.com/",
			},
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have a name")
}

func TestParseConfig_Sources_MissingURL(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sources": []interface{}{
			map[string]interface{}{
				"name": "test",
			},
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have a url")
}

func TestParseConfig_Sources_InvalidType(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sources": []interface{}{
			"not-an-object",
		},
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be an object")
}

func TestParseConfig_Sources_NotList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sources": "not-a-list",
	}

	_, err := ParseConfig(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a list")
}

func TestParseConfig_Full(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sources": []interface{}{
			map[string]interface{}{
				"name": "internal",
				"url":  "https://nuget.internal.com/v3/",
			},
		},
		"packages": []interface{}{
			"git",
			map[string]interface{}{
				"name":    "nodejs",
				"version": "18.0.0",
				"source":  "internal",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	require.NoError(t, err)

	assert.Len(t, cfg.Sources, 1)
	assert.Len(t, cfg.Packages, 2)
}
