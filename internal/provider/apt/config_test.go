package apt_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/apt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := apt.ParseConfig(map[string]interface{}{})
	require.NoError(t, err)
	assert.Empty(t, cfg.Packages)
	assert.Empty(t, cfg.PPAs)
}

func TestParseConfig_SimplePackages(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{"git", "curl", "wget"},
	}
	cfg, err := apt.ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 3)
	assert.Equal(t, "git", cfg.Packages[0].Name)
	assert.Equal(t, "curl", cfg.Packages[1].Name)
	assert.Equal(t, "wget", cfg.Packages[2].Name)
}

func TestParseConfig_PackageWithVersion(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			map[string]interface{}{
				"name":    "nodejs",
				"version": "18.0.0",
			},
		},
	}
	cfg, err := apt.ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.Equal(t, "nodejs", cfg.Packages[0].Name)
	assert.Equal(t, "18.0.0", cfg.Packages[0].Version)
}

func TestParseConfig_PPAs(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"ppas": []interface{}{
			"ppa:graphics-drivers/ppa",
			"ppa:deadsnakes/ppa",
		},
	}
	cfg, err := apt.ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.PPAs, 2)
	assert.Equal(t, "ppa:graphics-drivers/ppa", cfg.PPAs[0])
	assert.Equal(t, "ppa:deadsnakes/ppa", cfg.PPAs[1])
}

func TestParseConfig_Full(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"ppas": []interface{}{
			"ppa:git-core/ppa",
		},
		"packages": []interface{}{
			"git",
			map[string]interface{}{
				"name":    "build-essential",
				"version": "latest",
			},
		},
	}
	cfg, err := apt.ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.PPAs, 1)
	require.Len(t, cfg.Packages, 2)
	assert.Equal(t, "git", cfg.Packages[0].Name)
	assert.Equal(t, "build-essential", cfg.Packages[1].Name)
	assert.Equal(t, "latest", cfg.Packages[1].Version)
}

func TestParseConfig_InvalidPackages(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": "not-a-list",
	}
	_, err := apt.ParseConfig(raw)
	assert.Error(t, err)
}

func TestParseConfig_InvalidPPAs(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"ppas": "not-a-list",
	}
	_, err := apt.ParseConfig(raw)
	assert.Error(t, err)
}

func TestPackage_FullName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pkg      apt.Package
		expected string
	}{
		{
			name:     "simple package",
			pkg:      apt.Package{Name: "git"},
			expected: "git",
		},
		{
			name:     "package with version",
			pkg:      apt.Package{Name: "nodejs", Version: "18.0.0"},
			expected: "nodejs=18.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.pkg.FullName())
		})
	}
}
