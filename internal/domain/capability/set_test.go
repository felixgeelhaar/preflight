package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSet(t *testing.T) {
	t.Parallel()

	s := NewSet()
	assert.NotNil(t, s)
	assert.True(t, s.IsEmpty())
	assert.Equal(t, 0, s.Count())
}

func TestNewSetFrom(t *testing.T) {
	t.Parallel()

	s := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})
	assert.Equal(t, 2, s.Count())
	assert.True(t, s.Has(CapFilesRead))
	assert.True(t, s.Has(CapFilesWrite))
}

func TestParseSet(t *testing.T) {
	t.Parallel()

	s, err := ParseSet([]string{"files:read", "files:write"})
	require.NoError(t, err)
	assert.Equal(t, 2, s.Count())

	// Invalid capability
	_, err = ParseSet([]string{"invalid"})
	assert.Error(t, err)
}

func TestSet_AddRemove(t *testing.T) {
	t.Parallel()

	s := NewSet()

	s.Add(CapFilesRead)
	assert.True(t, s.Has(CapFilesRead))
	assert.Equal(t, 1, s.Count())

	s.Add(CapFilesRead) // Duplicate
	assert.Equal(t, 1, s.Count())

	s.Remove(CapFilesRead)
	assert.False(t, s.Has(CapFilesRead))
	assert.Equal(t, 0, s.Count())
}

func TestSet_AddZero(t *testing.T) {
	t.Parallel()

	s := NewSet()
	var zero Capability
	s.Add(zero)
	assert.Equal(t, 0, s.Count())
}

func TestSet_HasAny(t *testing.T) {
	t.Parallel()

	s := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})

	assert.True(t, s.HasAny(CapFilesRead, CapPackagesBrew))
	assert.True(t, s.HasAny(CapPackagesBrew, CapFilesWrite))
	assert.False(t, s.HasAny(CapPackagesBrew, CapPackagesApt))
}

func TestSet_HasAll(t *testing.T) {
	t.Parallel()

	s := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})

	assert.True(t, s.HasAll(CapFilesRead, CapFilesWrite))
	assert.False(t, s.HasAll(CapFilesRead, CapPackagesBrew))
}

func TestSet_Matches(t *testing.T) {
	t.Parallel()

	s := NewSet()
	s.Add(MustParseCapability("files:*"))

	assert.True(t, s.Matches(CapFilesRead))
	assert.True(t, s.Matches(CapFilesWrite))
	assert.False(t, s.Matches(CapPackagesBrew))
}

func TestSet_List(t *testing.T) {
	t.Parallel()

	s := NewSetFrom([]Capability{CapFilesWrite, CapFilesRead})
	list := s.List()

	assert.Len(t, list, 2)
	// Should be sorted
	assert.Equal(t, "files:read", list[0].String())
	assert.Equal(t, "files:write", list[1].String())
}

func TestSet_Strings(t *testing.T) {
	t.Parallel()

	s := NewSetFrom([]Capability{CapFilesRead, CapPackagesBrew})
	strs := s.Strings()

	assert.Len(t, strs, 2)
	assert.Contains(t, strs, "files:read")
	assert.Contains(t, strs, "packages:brew")
}

func TestSet_DangerousCapabilities(t *testing.T) {
	t.Parallel()

	s := NewSetFrom([]Capability{
		CapFilesRead,
		CapShellExecute,
		CapSecretsRead,
	})

	dangerous := s.DangerousCapabilities()
	assert.Len(t, dangerous, 2)
	assert.True(t, s.HasDangerous())

	safe := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})
	assert.False(t, safe.HasDangerous())
}

func TestSet_Union(t *testing.T) {
	t.Parallel()

	a := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})
	b := NewSetFrom([]Capability{CapFilesWrite, CapPackagesBrew})

	union := a.Union(b)
	assert.Equal(t, 3, union.Count())
	assert.True(t, union.Has(CapFilesRead))
	assert.True(t, union.Has(CapFilesWrite))
	assert.True(t, union.Has(CapPackagesBrew))
}

func TestSet_UnionNil(t *testing.T) {
	t.Parallel()

	a := NewSetFrom([]Capability{CapFilesRead})
	union := a.Union(nil)
	assert.Equal(t, 1, union.Count())
}

func TestSet_Intersection(t *testing.T) {
	t.Parallel()

	a := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})
	b := NewSetFrom([]Capability{CapFilesWrite, CapPackagesBrew})

	inter := a.Intersection(b)
	assert.Equal(t, 1, inter.Count())
	assert.True(t, inter.Has(CapFilesWrite))
}

func TestSet_IntersectionNil(t *testing.T) {
	t.Parallel()

	a := NewSetFrom([]Capability{CapFilesRead})
	inter := a.Intersection(nil)
	assert.Equal(t, 0, inter.Count())
}

func TestSet_Difference(t *testing.T) {
	t.Parallel()

	a := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})
	b := NewSetFrom([]Capability{CapFilesWrite, CapPackagesBrew})

	diff := a.Difference(b)
	assert.Equal(t, 1, diff.Count())
	assert.True(t, diff.Has(CapFilesRead))
}

func TestSet_DifferenceNil(t *testing.T) {
	t.Parallel()

	a := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})
	diff := a.Difference(nil)
	assert.Equal(t, 2, diff.Count())
}

func TestSet_ByCategory(t *testing.T) {
	t.Parallel()

	s := NewSetFrom([]Capability{
		CapFilesRead,
		CapFilesWrite,
		CapPackagesBrew,
	})

	byCategory := s.ByCategory()
	assert.Len(t, byCategory[CategoryFiles], 2)
	assert.Len(t, byCategory[CategoryPackages], 1)
}

func TestSet_Clone(t *testing.T) {
	t.Parallel()

	original := NewSetFrom([]Capability{CapFilesRead, CapFilesWrite})
	clone := original.Clone()

	assert.Equal(t, original.Count(), clone.Count())

	// Modify original
	original.Add(CapPackagesBrew)
	assert.Equal(t, 3, original.Count())
	assert.Equal(t, 2, clone.Count()) // Clone unchanged
}
