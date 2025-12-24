package winget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{}
	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	assert.Empty(t, cfg.Packages)
}

func TestParseConfig_PackagesSimple(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			"Microsoft.VisualStudioCode",
			"Git.Git",
			"7zip.7zip",
		},
	}

	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Packages, 3)
	assert.Equal(t, "Microsoft.VisualStudioCode", cfg.Packages[0].ID)
	assert.Equal(t, "Git.Git", cfg.Packages[1].ID)
	assert.Equal(t, "7zip.7zip", cfg.Packages[2].ID)
	// Simple strings should have no version or source
	assert.Empty(t, cfg.Packages[0].Version)
	assert.Empty(t, cfg.Packages[0].Source)
}

func TestParseConfig_PackagesWithOptions(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			map[string]interface{}{
				"id":      "Microsoft.VisualStudioCode",
				"version": "1.85.0",
				"source":  "winget",
			},
		},
	}

	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.Equal(t, "Microsoft.VisualStudioCode", cfg.Packages[0].ID)
	assert.Equal(t, "1.85.0", cfg.Packages[0].Version)
	assert.Equal(t, "winget", cfg.Packages[0].Source)
}

func TestParseConfig_MixedPackages(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			"Git.Git",
			map[string]interface{}{
				"id":      "Microsoft.VisualStudioCode",
				"version": "1.85.0",
			},
		},
	}

	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Packages, 2)
	assert.Equal(t, "Git.Git", cfg.Packages[0].ID)
	assert.Empty(t, cfg.Packages[0].Version)
	assert.Equal(t, "Microsoft.VisualStudioCode", cfg.Packages[1].ID)
	assert.Equal(t, "1.85.0", cfg.Packages[1].Version)
}

func TestParseConfig_InvalidPackagesNotList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": "not-a-list",
	}

	_, err := ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "packages must be a list")
}

func TestParseConfig_PackageWithoutID(t *testing.T) {
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
	assert.Contains(t, err.Error(), "package must have an id")
}

func TestParseConfig_InvalidPackageType(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			123, // invalid type
		},
	}

	_, err := ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package must be a string or object")
}
