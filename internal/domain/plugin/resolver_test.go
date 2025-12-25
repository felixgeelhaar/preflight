package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDependencyResolver(t *testing.T) {
	t.Run("default mode", func(t *testing.T) {
		resolver := NewDependencyResolver(nil, "")
		assert.Equal(t, ResolutionLatest, resolver.mode)
	})

	t.Run("strict mode", func(t *testing.T) {
		resolver := NewDependencyResolver(nil, ResolutionStrict)
		assert.Equal(t, ResolutionStrict, resolver.mode)
	})
}

func TestDependencyResolver_Resolve(t *testing.T) {
	t.Run("nil manifest", func(t *testing.T) {
		resolver := NewDependencyResolver(nil, ResolutionLatest)
		_, err := resolver.Resolve(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("no dependencies", func(t *testing.T) {
		resolver := NewDependencyResolver(nil, ResolutionLatest)
		manifest := &Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
		}

		result, err := resolver.Resolve(context.Background(), manifest)
		require.NoError(t, err)
		assert.Empty(t, result.Resolved)
		assert.Empty(t, result.Missing)
		assert.Empty(t, result.Conflicts)
		assert.False(t, result.HasErrors())
	})

	t.Run("missing dependency", func(t *testing.T) {
		resolver := NewDependencyResolver(nil, ResolutionLatest)
		manifest := &Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
			Requires: []Dependency{
				{Name: "missing-plugin", Version: ">=1.0.0"},
			},
		}

		result, err := resolver.Resolve(context.Background(), manifest)
		require.NoError(t, err)
		assert.Len(t, result.Missing, 1)
		assert.Equal(t, "missing-plugin", result.Missing[0].Name)
		assert.True(t, result.HasErrors())
	})

	t.Run("dependency found in registry", func(t *testing.T) {
		registry := NewRegistry()
		depPlugin := &Plugin{
			Manifest: Manifest{
				Name:    "dep-plugin",
				Version: "1.0.0",
				Type:    TypeConfig,
			},
		}
		err := registry.Register(depPlugin)
		require.NoError(t, err)

		resolver := NewDependencyResolver(registry, ResolutionLatest)
		manifest := &Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
			Requires: []Dependency{
				{Name: "dep-plugin", Version: ">=1.0.0"},
			},
		}

		result, err := resolver.Resolve(context.Background(), manifest)
		require.NoError(t, err)
		assert.Len(t, result.Resolved, 1)
		assert.Equal(t, "dep-plugin", result.Resolved[0].Name)
		assert.False(t, result.HasErrors())
	})

	t.Run("transitive dependencies", func(t *testing.T) {
		registry := NewRegistry()

		// Plugin A depends on Plugin B
		pluginB := &Plugin{
			Manifest: Manifest{
				Name:    "plugin-b",
				Version: "1.0.0",
				Type:    TypeConfig,
			},
		}
		require.NoError(t, registry.Register(pluginB))

		pluginA := &Plugin{
			Manifest: Manifest{
				Name:    "plugin-a",
				Version: "1.0.0",
				Type:    TypeConfig,
				Requires: []Dependency{
					{Name: "plugin-b", Version: ">=1.0.0"},
				},
			},
		}
		require.NoError(t, registry.Register(pluginA))

		resolver := NewDependencyResolver(registry, ResolutionLatest)
		manifest := &Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
			Requires: []Dependency{
				{Name: "plugin-a", Version: ">=1.0.0"},
			},
		}

		result, err := resolver.Resolve(context.Background(), manifest)
		require.NoError(t, err)
		assert.Len(t, result.Resolved, 2)
		assert.False(t, result.HasErrors())

		// plugin-b should come before plugin-a in install order
		bIndex := -1
		aIndex := -1
		for i, name := range result.InstallOrder {
			if name == "plugin-b" {
				bIndex = i
			}
			if name == "plugin-a" {
				aIndex = i
			}
		}
		assert.Less(t, bIndex, aIndex, "plugin-b should be installed before plugin-a")
	})

	t.Run("version conflict strict mode", func(t *testing.T) {
		registry := NewRegistry()

		pluginDep := &Plugin{
			Manifest: Manifest{
				Name:    "shared-dep",
				Version: "1.0.0",
				Type:    TypeConfig,
			},
		}
		require.NoError(t, registry.Register(pluginDep))

		resolver := NewDependencyResolver(registry, ResolutionStrict)
		manifest := &Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
			Requires: []Dependency{
				{Name: "shared-dep", Version: "1.0.0"},
				{Name: "shared-dep", Version: "2.0.0"},
			},
		}

		result, err := resolver.Resolve(context.Background(), manifest)
		require.NoError(t, err)
		assert.Len(t, result.Conflicts, 1)
		assert.True(t, result.HasErrors())
	})

	t.Run("version conflict latest mode uses higher version", func(t *testing.T) {
		registry := NewRegistry()

		pluginDep := &Plugin{
			Manifest: Manifest{
				Name:    "shared-dep",
				Version: "2.0.0",
				Type:    TypeConfig,
			},
		}
		require.NoError(t, registry.Register(pluginDep))

		resolver := NewDependencyResolver(registry, ResolutionLatest)
		manifest := &Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
			Requires: []Dependency{
				{Name: "shared-dep", Version: ">=1.0.0"},
				{Name: "shared-dep", Version: ">=2.0.0"},
			},
		}

		result, err := resolver.Resolve(context.Background(), manifest)
		require.NoError(t, err)
		assert.Empty(t, result.Conflicts)
		assert.False(t, result.HasErrors())
	})
}

func TestTopologicalSort(t *testing.T) {
	t.Run("empty graph", func(t *testing.T) {
		graph := make(map[string][]string)
		result, err := topologicalSort(graph)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("single node", func(t *testing.T) {
		graph := map[string][]string{
			"a": {},
		}
		result, err := topologicalSort(graph)
		require.NoError(t, err)
		assert.Equal(t, []string{"a"}, result)
	})

	t.Run("linear chain", func(t *testing.T) {
		graph := map[string][]string{
			"a": {"b"},
			"b": {"c"},
			"c": {},
		}
		result, err := topologicalSort(graph)
		require.NoError(t, err)
		assert.Equal(t, []string{"c", "b", "a"}, result)
	})

	t.Run("diamond dependency", func(t *testing.T) {
		graph := map[string][]string{
			"a": {"b", "c"},
			"b": {"d"},
			"c": {"d"},
			"d": {},
		}
		result, err := topologicalSort(graph)
		require.NoError(t, err)

		// d must come before b and c, which must come before a
		indexOf := func(s string) int {
			for i, v := range result {
				if v == s {
					return i
				}
			}
			return -1
		}

		assert.Less(t, indexOf("d"), indexOf("b"))
		assert.Less(t, indexOf("d"), indexOf("c"))
		assert.Less(t, indexOf("b"), indexOf("a"))
		assert.Less(t, indexOf("c"), indexOf("a"))
	})

	t.Run("simple cycle", func(t *testing.T) {
		graph := map[string][]string{
			"a": {"b"},
			"b": {"a"},
		}
		_, err := topologicalSort(graph)
		assert.Error(t, err)
		assert.True(t, IsCyclicDependency(err))
	})

	t.Run("self-loop", func(t *testing.T) {
		graph := map[string][]string{
			"a": {"a"},
		}
		_, err := topologicalSort(graph)
		assert.Error(t, err)
		assert.True(t, IsCyclicDependency(err))
	})

	t.Run("complex cycle", func(t *testing.T) {
		graph := map[string][]string{
			"a": {"b"},
			"b": {"c"},
			"c": {"d"},
			"d": {"b"},
		}
		_, err := topologicalSort(graph)
		assert.Error(t, err)
		assert.True(t, IsCyclicDependency(err))
	})
}

func TestSatisfiesVersionConstraint(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		constraint string
		expected   bool
	}{
		{"empty constraint", "1.0.0", "", true},
		{"empty version", "", ">=1.0.0", true},
		{"exact match", "1.0.0", "1.0.0", true},
		{"exact match with =", "1.0.0", "=1.0.0", true},
		{"exact match with v prefix", "v1.0.0", "1.0.0", true},

		{"greater than", "2.0.0", ">1.0.0", true},
		{"greater than equal", "1.0.0", ">=1.0.0", true},
		{"greater than fail", "0.9.0", ">1.0.0", false},

		{"less than", "0.9.0", "<1.0.0", true},
		{"less than equal", "1.0.0", "<=1.0.0", true},
		{"less than fail", "2.0.0", "<1.0.0", false},

		{"caret major", "1.5.0", "^1.0.0", true},
		{"caret major boundary", "2.0.0", "^1.0.0", false},
		{"caret minor ok", "1.2.5", "^1.2.0", true},

		{"tilde minor", "1.2.5", "~1.2.0", true},
		{"tilde minor boundary", "1.3.0", "~1.2.0", false},
		{"tilde ok", "1.2.9", "~1.2.3", true},

		{"invalid version", "invalid", ">=1.0.0", false},
		{"invalid constraint", "1.0.0", ">=invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := satisfiesVersionConstraint(tt.version, tt.constraint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareVersionConstraints(t *testing.T) {
	tests := []struct {
		c1       string
		c2       string
		expected int // >0 if c1>c2, <0 if c1<c2, 0 if equal
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{">=2.0.0", ">=1.0.0", 1},
		{"^1.5.0", "~1.2.0", 1},
	}

	for _, tt := range tests {
		t.Run(tt.c1+" vs "+tt.c2, func(t *testing.T) {
			result := compareVersionConstraints(tt.c1, tt.c2)
			switch {
			case tt.expected > 0:
				assert.Positive(t, result)
			case tt.expected < 0:
				assert.Negative(t, result)
			default:
				assert.Equal(t, 0, result)
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		constraint string
		expected   string
	}{
		{"1.0.0", "1.0.0"},
		{">=1.0.0", "1.0.0"},
		{"<=2.0.0", "2.0.0"},
		{">1.5.0", "1.5.0"},
		{"<3.0.0", "3.0.0"},
		{"^1.2.0", "1.2.0"},
		{"~1.2.3", "1.2.3"},
		{"=1.0.0", "1.0.0"},
		{" 1.0.0 ", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			result := extractVersion(tt.constraint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCyclicDependencyError(t *testing.T) {
	err := &CyclicDependencyError{Cycle: []string{"a", "b", "c", "a"}}
	assert.Contains(t, err.Error(), "cyclic dependency")
	assert.Contains(t, err.Error(), "a -> b -> c -> a")
	assert.True(t, IsCyclicDependency(err))
}

func TestResolutionResult_HasErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		result := &ResolutionResult{}
		assert.False(t, result.HasErrors())
	})

	t.Run("has missing", func(t *testing.T) {
		result := &ResolutionResult{
			Missing: []Dependency{{Name: "test"}},
		}
		assert.True(t, result.HasErrors())
	})

	t.Run("has conflicts", func(t *testing.T) {
		result := &ResolutionResult{
			Conflicts: []DependencyConflict{{Name: "test"}},
		}
		assert.True(t, result.HasErrors())
	})
}

func TestSemverMinor(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"v1.2.3", "1.2"},
		{"v1.0.0", "1.0"},
		{"v10.20.30", "10.20"},
		{"v1", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := semverMinor(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}
