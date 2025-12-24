package scoop

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
	assert.Empty(t, cfg.Buckets)
	assert.Empty(t, cfg.Packages)
}

func TestParseConfig_BucketsSimple(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"buckets": []interface{}{
			"extras",
			"versions",
		},
	}

	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Buckets, 2)
	assert.Equal(t, "extras", cfg.Buckets[0].Name)
	assert.Equal(t, "versions", cfg.Buckets[1].Name)
	assert.Empty(t, cfg.Buckets[0].URL)
}

func TestParseConfig_BucketsWithURL(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"buckets": []interface{}{
			map[string]interface{}{
				"name": "custom",
				"url":  "https://github.com/user/scoop-bucket",
			},
		},
	}

	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Buckets, 1)
	assert.Equal(t, "custom", cfg.Buckets[0].Name)
	assert.Equal(t, "https://github.com/user/scoop-bucket", cfg.Buckets[0].URL)
}

func TestParseConfig_PackagesSimple(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			"git",
			"curl",
			"wget",
		},
	}

	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Packages, 3)
	assert.Equal(t, "git", cfg.Packages[0].Name)
	assert.Empty(t, cfg.Packages[0].Bucket)
	assert.Empty(t, cfg.Packages[0].Version)
}

func TestParseConfig_PackagesWithOptions(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			map[string]interface{}{
				"name":    "neovim",
				"bucket":  "extras",
				"version": "0.9.5",
			},
		},
	}

	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.Equal(t, "neovim", cfg.Packages[0].Name)
	assert.Equal(t, "extras", cfg.Packages[0].Bucket)
	assert.Equal(t, "0.9.5", cfg.Packages[0].Version)
}

func TestParseConfig_Full(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"buckets": []interface{}{"extras"},
		"packages": []interface{}{
			"git",
			map[string]interface{}{
				"name":   "neovim",
				"bucket": "extras",
			},
		},
	}

	cfg, err := ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Buckets, 1)
	assert.Len(t, cfg.Packages, 2)
}

func TestParseConfig_InvalidBucketsNotList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"buckets": "not-a-list",
	}

	_, err := ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "buckets must be a list")
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

func TestParseConfig_BucketWithoutName(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"buckets": []interface{}{
			map[string]interface{}{
				"url": "https://example.com",
			},
		},
	}

	_, err := ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket must have a name")
}

func TestParseConfig_PackageWithoutName(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			map[string]interface{}{
				"bucket": "extras",
			},
		},
	}

	_, err := ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package must have a name")
}

func TestParseConfig_InvalidBucketType(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"buckets": []interface{}{
			123,
		},
	}

	_, err := ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket must be a string or object")
}

func TestParseConfig_InvalidPackageType(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			123,
		},
	}

	_, err := ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package must be a string or object")
}

func TestPackage_FullName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pkg      Package
		expected string
	}{
		{Package{Name: "git"}, "git"},
		{Package{Name: "neovim", Bucket: "extras"}, "extras/neovim"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.pkg.FullName())
	}
}
