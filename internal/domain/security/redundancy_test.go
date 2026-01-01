package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedundancies_ByType(t *testing.T) {
	t.Parallel()

	redundancies := Redundancies{
		{Type: RedundancyDuplicate, Packages: []string{"go", "go@1.24"}},
		{Type: RedundancyOverlap, Packages: []string{"grype", "trivy"}},
		{Type: RedundancyDuplicate, Packages: []string{"python", "python@3.12"}},
		{Type: RedundancyOrphan, Packages: []string{"libpng", "zlib"}},
	}

	t.Run("filter duplicates", func(t *testing.T) {
		t.Parallel()
		result := redundancies.ByType(RedundancyDuplicate)
		assert.Len(t, result, 2)
	})

	t.Run("filter overlaps", func(t *testing.T) {
		t.Parallel()
		result := redundancies.ByType(RedundancyOverlap)
		assert.Len(t, result, 1)
	})

	t.Run("filter orphans", func(t *testing.T) {
		t.Parallel()
		result := redundancies.ByType(RedundancyOrphan)
		assert.Len(t, result, 1)
	})
}

func TestRedundancies_TotalRemovable(t *testing.T) {
	t.Parallel()

	redundancies := Redundancies{
		{Type: RedundancyDuplicate, Remove: []string{"go@1.24"}},
		{Type: RedundancyOverlap, Remove: []string{"trivy"}},
		{Type: RedundancyOrphan, Remove: []string{"libpng", "zlib", "jpeg"}},
	}

	assert.Equal(t, 5, redundancies.TotalRemovable())
}

func TestRedundancyResult_Summary(t *testing.T) {
	t.Parallel()

	result := &RedundancyResult{
		Redundancies: Redundancies{
			{Type: RedundancyDuplicate, Remove: []string{"go@1.24"}},
			{Type: RedundancyDuplicate, Remove: []string{"python@3.12"}},
			{Type: RedundancyOverlap, Remove: []string{"trivy"}},
			{Type: RedundancyOrphan, Remove: []string{"libpng", "zlib"}},
		},
	}

	summary := result.Summary()

	assert.Equal(t, 4, summary.Total)
	assert.Equal(t, 2, summary.Duplicates)
	assert.Equal(t, 1, summary.Overlaps)
	assert.Equal(t, 1, summary.Orphans)
	assert.Equal(t, 5, summary.Removable) // 1+1+1+2 = 5 removable packages
}

func TestBrewRedundancyChecker_Name(t *testing.T) {
	t.Parallel()
	checker := NewBrewRedundancyChecker()
	assert.Equal(t, "brew", checker.Name())
}

func TestBrewRedundancyChecker_detectDuplicates(t *testing.T) {
	t.Parallel()

	checker := NewBrewRedundancyChecker()

	tests := []struct {
		name       string
		packages   []string
		ignore     map[string]bool
		keep       map[string]bool
		expectLen  int
		expectPkgs []string
	}{
		{
			name:       "no duplicates",
			packages:   []string{"git", "curl", "jq"},
			ignore:     nil,
			keep:       nil,
			expectLen:  0,
			expectPkgs: nil,
		},
		{
			name:       "go duplicates",
			packages:   []string{"go", "go@1.24", "go@1.23"},
			ignore:     nil,
			keep:       nil,
			expectLen:  1,
			expectPkgs: []string{"go@1.24", "go@1.23"}, // Remove versioned
		},
		{
			name:       "python duplicates",
			packages:   []string{"python@3.12", "python@3.14"},
			ignore:     nil,
			keep:       nil,
			expectLen:  1,
			expectPkgs: []string{"python@3.12"}, // Keep higher version
		},
		{
			name:       "ignore package",
			packages:   []string{"go", "go@1.24"},
			ignore:     map[string]bool{"go@1.24": true},
			keep:       nil,
			expectLen:  0,
			expectPkgs: nil,
		},
		{
			name:       "keep specific version",
			packages:   []string{"go", "go@1.24"},
			ignore:     nil,
			keep:       map[string]bool{"go@1.24": true},
			expectLen:  1,
			expectPkgs: []string{"go"}, // Remove unversioned since versioned is kept
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ignore := tt.ignore
			if ignore == nil {
				ignore = make(map[string]bool)
			}
			keep := tt.keep
			if keep == nil {
				keep = make(map[string]bool)
			}

			result := checker.detectDuplicates(tt.packages, ignore, keep)
			assert.Len(t, result, tt.expectLen)

			if tt.expectLen > 0 && tt.expectPkgs != nil {
				assert.ElementsMatch(t, tt.expectPkgs, result[0].Remove)
			}
		})
	}
}

func TestBrewRedundancyChecker_detectOverlaps(t *testing.T) {
	t.Parallel()

	checker := NewBrewRedundancyChecker()

	tests := []struct {
		name      string
		packages  []string
		expectLen int
		category  string
	}{
		{
			name:      "no overlaps",
			packages:  []string{"git", "curl", "jq"},
			expectLen: 0,
		},
		{
			name:      "security scanners overlap",
			packages:  []string{"grype", "trivy", "git"},
			expectLen: 1,
			category:  "security_scanners",
		},
		{
			name:      "node package managers overlap",
			packages:  []string{"npm", "yarn", "pnpm"},
			expectLen: 1,
			category:  "node_package_managers",
		},
		{
			name:      "multiple overlaps",
			packages:  []string{"grype", "trivy", "npm", "yarn"},
			expectLen: 2,
		},
		{
			name:      "single tool in category",
			packages:  []string{"grype", "git"},
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := checker.detectOverlaps(tt.packages, make(map[string]bool))
			assert.Len(t, result, tt.expectLen)

			if tt.expectLen == 1 && tt.category != "" {
				assert.Equal(t, tt.category, result[0].Category)
			}
		})
	}
}

func TestDefaultToolCategories(t *testing.T) {
	t.Parallel()

	categories := DefaultToolCategories()

	assert.NotEmpty(t, categories)

	// Verify some expected categories exist
	categoryNames := make(map[string]bool)
	for _, cat := range categories {
		categoryNames[cat.Name] = true
		assert.NotEmpty(t, cat.Tools, "category %s should have tools", cat.Name)
		assert.NotEmpty(t, cat.Description, "category %s should have description", cat.Name)
	}

	assert.True(t, categoryNames["security_scanners"])
	assert.True(t, categoryNames["node_package_managers"])
	assert.True(t, categoryNames["container_runtimes"])
}

func TestRedundancyCheckerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("register and get", func(t *testing.T) {
		t.Parallel()
		registry := NewRedundancyCheckerRegistry()

		brewChecker := NewBrewRedundancyChecker()
		registry.Register(brewChecker)

		// Get should return checker if available
		checker := registry.Get("brew")
		if brewChecker.Available() {
			assert.NotNil(t, checker)
		}
	})

	t.Run("get nonexistent returns nil", func(t *testing.T) {
		t.Parallel()
		registry := NewRedundancyCheckerRegistry()
		assert.Nil(t, registry.Get("nonexistent"))
	})

	t.Run("all returns available only", func(t *testing.T) {
		t.Parallel()
		registry := NewRedundancyCheckerRegistry()
		registry.Register(NewBrewRedundancyChecker())

		all := registry.All()
		for _, c := range all {
			assert.True(t, c.Available())
		}
	})
}

// mockRedundancyChecker is a test double for RedundancyChecker.
type mockRedundancyChecker struct {
	name         string
	available    bool
	redundancies Redundancies
	err          error
}

func (m *mockRedundancyChecker) Name() string {
	return m.name
}

func (m *mockRedundancyChecker) Available() bool {
	return m.available
}

func (m *mockRedundancyChecker) Check(_ context.Context, _ RedundancyOptions) (*RedundancyResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &RedundancyResult{
		Checker:      m.name,
		Redundancies: m.redundancies,
	}, nil
}

func TestRedundancyCheckerRegistry_WithMock(t *testing.T) {
	t.Parallel()

	registry := NewRedundancyCheckerRegistry()

	mockAvailable := &mockRedundancyChecker{
		name:      "mock-available",
		available: true,
		redundancies: Redundancies{
			{Type: RedundancyDuplicate, Packages: []string{"go", "go@1.24"}},
		},
	}

	mockUnavailable := &mockRedundancyChecker{
		name:      "mock-unavailable",
		available: false,
	}

	registry.Register(mockAvailable)
	registry.Register(mockUnavailable)

	t.Run("get available returns checker", func(t *testing.T) {
		t.Parallel()
		checker := registry.Get("mock-available")
		require.NotNil(t, checker)
		assert.Equal(t, "mock-available", checker.Name())
	})

	t.Run("get unavailable returns nil", func(t *testing.T) {
		t.Parallel()
		checker := registry.Get("mock-unavailable")
		assert.Nil(t, checker)
	})

	t.Run("check returns redundancies", func(t *testing.T) {
		t.Parallel()
		checker := registry.Get("mock-available")
		require.NotNil(t, checker)

		result, err := checker.Check(context.Background(), RedundancyOptions{})
		require.NoError(t, err)
		assert.Len(t, result.Redundancies, 1)
	})
}

func TestVersionedPackageRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
		matches  bool
	}{
		{"go@1.24", "go", true},
		{"python@3.12", "python", true},
		{"node@20.10.0", "node", true},
		{"go", "", false},
		{"git", "", false},
		{"@types/node", "", false}, // npm-style package
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			matches := versionedPackageRegex.FindStringSubmatch(tt.input)
			if tt.matches {
				require.Len(t, matches, 2)
				assert.Equal(t, tt.expected, matches[1])
			} else {
				assert.Empty(t, matches)
			}
		})
	}
}

func TestRedundancyResult_ToJSON(t *testing.T) {
	t.Parallel()

	result := &RedundancyResult{
		Checker: "brew",
		Redundancies: Redundancies{
			{
				Type:           RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Category:       "go",
				Recommendation: "Keep go (tracks latest)",
				Keep:           []string{"go"},
				Remove:         []string{"go@1.24"},
			},
		},
	}

	jsonBytes, err := result.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), "duplicate")
	assert.Contains(t, string(jsonBytes), "go@1.24")
}
