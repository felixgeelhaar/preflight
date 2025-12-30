package sync

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVersionVector(t *testing.T) {
	t.Parallel()

	v := NewVersionVector()
	assert.True(t, v.IsEmpty())
	assert.Equal(t, 0, v.Size())
}

func TestVersionVector_FromMap(t *testing.T) {
	t.Parallel()

	m := map[string]uint64{
		"machine-a": 5,
		"machine-b": 3,
	}
	v := FromMap(m)

	assert.Equal(t, uint64(5), v.Get("machine-a"))
	assert.Equal(t, uint64(3), v.Get("machine-b"))
	assert.Equal(t, 2, v.Size())

	// Verify it's a copy (immutability)
	m["machine-a"] = 100
	assert.Equal(t, uint64(5), v.Get("machine-a"))
}

func TestVersionVector_Get(t *testing.T) {
	t.Parallel()

	v := FromMap(map[string]uint64{"machine-a": 5})

	assert.Equal(t, uint64(5), v.Get("machine-a"))
	assert.Equal(t, uint64(0), v.Get("nonexistent"))
}

func TestVersionVector_Set(t *testing.T) {
	t.Parallel()

	v := NewVersionVector()
	v2 := v.Set("machine-a", 5)

	// Original should be unchanged (immutability)
	assert.Equal(t, uint64(0), v.Get("machine-a"))
	assert.Equal(t, uint64(5), v2.Get("machine-a"))
}

func TestVersionVector_Increment(t *testing.T) {
	t.Parallel()

	machineID := NewMachineID()
	v := NewVersionVector()

	v1 := v.Increment(machineID)
	assert.Equal(t, uint64(1), v1.GetByMachineID(machineID))

	v2 := v1.Increment(machineID)
	assert.Equal(t, uint64(2), v2.GetByMachineID(machineID))

	v3 := v2.Increment(machineID)
	assert.Equal(t, uint64(3), v3.GetByMachineID(machineID))

	// Original should be unchanged
	assert.Equal(t, uint64(0), v.GetByMachineID(machineID))
}

func TestVersionVector_IncrementByString(t *testing.T) {
	t.Parallel()

	v := NewVersionVector()
	v = v.IncrementByString("machine-a")
	v = v.IncrementByString("machine-a")
	v = v.IncrementByString("machine-b")

	assert.Equal(t, uint64(2), v.Get("machine-a"))
	assert.Equal(t, uint64(1), v.Get("machine-b"))
}

func TestVersionVector_Merge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v1       map[string]uint64
		v2       map[string]uint64
		expected map[string]uint64
	}{
		{
			name:     "both empty",
			v1:       map[string]uint64{},
			v2:       map[string]uint64{},
			expected: map[string]uint64{},
		},
		{
			name:     "one empty",
			v1:       map[string]uint64{"a": 5},
			v2:       map[string]uint64{},
			expected: map[string]uint64{"a": 5},
		},
		{
			name:     "disjoint keys",
			v1:       map[string]uint64{"a": 5},
			v2:       map[string]uint64{"b": 3},
			expected: map[string]uint64{"a": 5, "b": 3},
		},
		{
			name:     "overlapping keys take max",
			v1:       map[string]uint64{"a": 5, "b": 3},
			v2:       map[string]uint64{"a": 3, "b": 7},
			expected: map[string]uint64{"a": 5, "b": 7},
		},
		{
			name:     "mixed overlap",
			v1:       map[string]uint64{"a": 5, "b": 3},
			v2:       map[string]uint64{"b": 7, "c": 2},
			expected: map[string]uint64{"a": 5, "b": 7, "c": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v1 := FromMap(tt.v1)
			v2 := FromMap(tt.v2)
			merged := v1.Merge(v2)
			assert.Equal(t, tt.expected, merged.ToMap())
		})
	}
}

func TestVersionVector_Merge_Commutativity(t *testing.T) {
	t.Parallel()

	// Property: Merge(a, b) == Merge(b, a)
	v1 := FromMap(map[string]uint64{"a": 5, "b": 3, "c": 1})
	v2 := FromMap(map[string]uint64{"a": 3, "b": 7, "d": 2})

	merged1 := v1.Merge(v2)
	merged2 := v2.Merge(v1)

	assert.True(t, merged1.Equals(merged2))
}

func TestVersionVector_Merge_Associativity(t *testing.T) {
	t.Parallel()

	// Property: Merge(Merge(a, b), c) == Merge(a, Merge(b, c))
	v1 := FromMap(map[string]uint64{"a": 5, "b": 3})
	v2 := FromMap(map[string]uint64{"b": 7, "c": 2})
	v3 := FromMap(map[string]uint64{"a": 8, "c": 1, "d": 4})

	merged1 := v1.Merge(v2).Merge(v3)
	merged2 := v1.Merge(v2.Merge(v3))

	assert.True(t, merged1.Equals(merged2))
}

func TestVersionVector_Merge_Idempotence(t *testing.T) {
	t.Parallel()

	// Property: Merge(a, a) == a
	v := FromMap(map[string]uint64{"a": 5, "b": 3})
	merged := v.Merge(v)

	assert.True(t, v.Equals(merged))
}

func TestVersionVector_Compare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v1       map[string]uint64
		v2       map[string]uint64
		expected CausalRelation
	}{
		{
			name:     "both empty",
			v1:       map[string]uint64{},
			v2:       map[string]uint64{},
			expected: Equal,
		},
		{
			name:     "identical non-empty",
			v1:       map[string]uint64{"a": 5, "b": 3},
			v2:       map[string]uint64{"a": 5, "b": 3},
			expected: Equal,
		},
		{
			name:     "v1 before v2 - all less",
			v1:       map[string]uint64{"a": 3, "b": 2},
			v2:       map[string]uint64{"a": 5, "b": 3},
			expected: Before,
		},
		{
			name:     "v1 before v2 - some equal",
			v1:       map[string]uint64{"a": 3, "b": 3},
			v2:       map[string]uint64{"a": 5, "b": 3},
			expected: Before,
		},
		{
			name:     "v1 before v2 - missing key",
			v1:       map[string]uint64{"a": 3},
			v2:       map[string]uint64{"a": 3, "b": 1},
			expected: Before,
		},
		{
			name:     "v1 after v2 - all greater",
			v1:       map[string]uint64{"a": 5, "b": 4},
			v2:       map[string]uint64{"a": 3, "b": 2},
			expected: After,
		},
		{
			name:     "v1 after v2 - some equal",
			v1:       map[string]uint64{"a": 5, "b": 3},
			v2:       map[string]uint64{"a": 3, "b": 3},
			expected: After,
		},
		{
			name:     "v1 after v2 - extra key",
			v1:       map[string]uint64{"a": 3, "b": 1},
			v2:       map[string]uint64{"a": 3},
			expected: After,
		},
		{
			name:     "concurrent - crossover",
			v1:       map[string]uint64{"a": 5, "b": 2},
			v2:       map[string]uint64{"a": 3, "b": 4},
			expected: Concurrent,
		},
		{
			name:     "concurrent - disjoint keys",
			v1:       map[string]uint64{"a": 5},
			v2:       map[string]uint64{"b": 3},
			expected: Concurrent,
		},
		{
			name:     "concurrent - mixed with disjoint",
			v1:       map[string]uint64{"a": 5, "c": 1},
			v2:       map[string]uint64{"a": 3, "b": 2},
			expected: Concurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v1 := FromMap(tt.v1)
			v2 := FromMap(tt.v2)
			result := v1.Compare(v2)
			assert.Equal(t, tt.expected, result, "expected %s but got %s", tt.expected, result)
		})
	}
}

func TestVersionVector_Compare_Symmetry(t *testing.T) {
	t.Parallel()

	// Property: If Compare(a, b) == Before then Compare(b, a) == After
	v1 := FromMap(map[string]uint64{"a": 3, "b": 2})
	v2 := FromMap(map[string]uint64{"a": 5, "b": 3})

	assert.Equal(t, Before, v1.Compare(v2))
	assert.Equal(t, After, v2.Compare(v1))
}

func TestVersionVector_Compare_ConcurrentIsSymmetric(t *testing.T) {
	t.Parallel()

	// Property: If Compare(a, b) == Concurrent then Compare(b, a) == Concurrent
	v1 := FromMap(map[string]uint64{"a": 5, "b": 2})
	v2 := FromMap(map[string]uint64{"a": 3, "b": 4})

	assert.Equal(t, Concurrent, v1.Compare(v2))
	assert.Equal(t, Concurrent, v2.Compare(v1))
}

func TestVersionVector_Dominates(t *testing.T) {
	t.Parallel()

	v1 := FromMap(map[string]uint64{"a": 5, "b": 3})
	v2 := FromMap(map[string]uint64{"a": 3, "b": 2})
	v3 := FromMap(map[string]uint64{"a": 5, "b": 3})

	assert.True(t, v1.Dominates(v2))
	assert.True(t, v1.Dominates(v3))
	assert.False(t, v2.Dominates(v1))
}

func TestVersionVector_IsConcurrentWith(t *testing.T) {
	t.Parallel()

	v1 := FromMap(map[string]uint64{"a": 5, "b": 2})
	v2 := FromMap(map[string]uint64{"a": 3, "b": 4})
	v3 := FromMap(map[string]uint64{"a": 5, "b": 3})

	assert.True(t, v1.IsConcurrentWith(v2))
	assert.False(t, v1.IsConcurrentWith(v3))
}

func TestVersionVector_IsEmpty(t *testing.T) {
	t.Parallel()

	empty := NewVersionVector()
	assert.True(t, empty.IsEmpty())

	nonEmpty := FromMap(map[string]uint64{"a": 1})
	assert.False(t, nonEmpty.IsEmpty())
}

func TestVersionVector_MachineIDs(t *testing.T) {
	t.Parallel()

	v := FromMap(map[string]uint64{"c": 1, "a": 2, "b": 3})
	ids := v.MachineIDs()

	// Should be sorted
	assert.Equal(t, []string{"a", "b", "c"}, ids)
}

func TestVersionVector_Sum(t *testing.T) {
	t.Parallel()

	v := FromMap(map[string]uint64{"a": 5, "b": 3, "c": 2})
	assert.Equal(t, uint64(10), v.Sum())

	empty := NewVersionVector()
	assert.Equal(t, uint64(0), empty.Sum())
}

func TestVersionVector_ToMap(t *testing.T) {
	t.Parallel()

	original := map[string]uint64{"a": 5, "b": 3}
	v := FromMap(original)
	result := v.ToMap()

	assert.Equal(t, original, result)

	// Verify it's a copy
	result["a"] = 100
	assert.Equal(t, uint64(5), v.Get("a"))
}

func TestVersionVector_String(t *testing.T) {
	t.Parallel()

	empty := NewVersionVector()
	assert.Equal(t, "{}", empty.String())

	v := FromMap(map[string]uint64{"aaaaaaaa-1234": 5, "bbbbbbbb-5678": 3})
	s := v.String()
	assert.Contains(t, s, "aaaaaaaa:5")
	assert.Contains(t, s, "bbbbbbbb:3")
}

func TestVersionVector_JSON(t *testing.T) {
	t.Parallel()

	original := FromMap(map[string]uint64{"machine-a": 5, "machine-b": 3})

	// Marshal
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var restored VersionVector
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.True(t, original.Equals(restored))
}

func TestVersionVector_Clone(t *testing.T) {
	t.Parallel()

	v := FromMap(map[string]uint64{"a": 5, "b": 3})
	clone := v.Clone()

	assert.True(t, v.Equals(clone))

	// Modify clone shouldn't affect original
	// (Clone returns new VersionVector, Set returns new one too, so this is testing immutability)
	clone2 := clone.Set("a", 100)
	assert.Equal(t, uint64(5), v.Get("a"))
	assert.Equal(t, uint64(100), clone2.Get("a"))
}

func TestVersionVector_Max(t *testing.T) {
	t.Parallel()

	v := FromMap(map[string]uint64{"a": 5, "b": 10, "c": 3})
	assert.Equal(t, uint64(10), v.Max())

	empty := NewVersionVector()
	assert.Equal(t, uint64(0), empty.Max())
}

func TestVersionVector_ContainsMachine(t *testing.T) {
	t.Parallel()

	v := FromMap(map[string]uint64{"a": 5, "b": 3})

	assert.True(t, v.ContainsMachine("a"))
	assert.True(t, v.ContainsMachine("b"))
	assert.False(t, v.ContainsMachine("c"))
}

func TestVersionVector_WithoutMachine(t *testing.T) {
	t.Parallel()

	v := FromMap(map[string]uint64{"a": 5, "b": 3, "c": 1})
	v2 := v.WithoutMachine("b")

	assert.Equal(t, 3, v.Size())
	assert.Equal(t, 2, v2.Size())
	assert.True(t, v2.ContainsMachine("a"))
	assert.False(t, v2.ContainsMachine("b"))
	assert.True(t, v2.ContainsMachine("c"))
}

func TestVersionVector_Diff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v1       map[string]uint64
		v2       map[string]uint64
		expected []string
	}{
		{
			name:     "identical",
			v1:       map[string]uint64{"a": 5, "b": 3},
			v2:       map[string]uint64{"a": 5, "b": 3},
			expected: []string{},
		},
		{
			name:     "different values",
			v1:       map[string]uint64{"a": 5, "b": 3},
			v2:       map[string]uint64{"a": 5, "b": 7},
			expected: []string{"b"},
		},
		{
			name:     "extra in v1",
			v1:       map[string]uint64{"a": 5, "b": 3},
			v2:       map[string]uint64{"a": 5},
			expected: []string{"b"},
		},
		{
			name:     "extra in v2",
			v1:       map[string]uint64{"a": 5},
			v2:       map[string]uint64{"a": 5, "b": 3},
			expected: []string{"b"},
		},
		{
			name:     "completely different",
			v1:       map[string]uint64{"a": 5},
			v2:       map[string]uint64{"b": 3},
			expected: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v1 := FromMap(tt.v1)
			v2 := FromMap(tt.v2)
			diff := v1.Diff(v2)
			assert.Equal(t, tt.expected, diff)
		})
	}
}

func TestCausalRelation_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "equal", Equal.String())
	assert.Equal(t, "before", Before.String())
	assert.Equal(t, "after", After.String())
	assert.Equal(t, "concurrent", Concurrent.String())
}

// TestVersionVector_RealWorldScenario tests a realistic multi-machine sync scenario.
func TestVersionVector_RealWorldScenario(t *testing.T) {
	t.Parallel()

	// Scenario: Machine A and B start synced, then make independent changes

	// Initial state: both machines have empty vectors
	machineA := "machine-a-uuid"
	machineB := "machine-b-uuid"

	// Machine A makes 3 changes
	vA := NewVersionVector().
		IncrementByString(machineA).
		IncrementByString(machineA).
		IncrementByString(machineA)

	// Machine B makes 2 changes (independently)
	vB := NewVersionVector().
		IncrementByString(machineB).
		IncrementByString(machineB)

	// These are concurrent changes
	assert.Equal(t, Concurrent, vA.Compare(vB))

	// Now A syncs from B (merge)
	vAMerged := vA.Merge(vB).IncrementByString(machineA)

	// A's merged vector dominates both originals
	assert.True(t, vAMerged.Dominates(vA))
	assert.True(t, vAMerged.Dominates(vB))

	// vAMerged should have: A=4, B=2
	assert.Equal(t, uint64(4), vAMerged.Get(machineA))
	assert.Equal(t, uint64(2), vAMerged.Get(machineB))

	// B also syncs (gets A's merged state)
	vBMerged := vB.Merge(vAMerged).IncrementByString(machineB)

	// Now B dominates the previous merged state
	assert.True(t, vBMerged.Dominates(vAMerged))

	// vBMerged should have: A=4, B=3
	assert.Equal(t, uint64(4), vBMerged.Get(machineA))
	assert.Equal(t, uint64(3), vBMerged.Get(machineB))
}
