package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncConflictsCmd_Exists(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, syncConflictsCmd)
	assert.Equal(t, "conflicts", syncConflictsCmd.Use)
	assert.Contains(t, syncConflictsCmd.Short, "conflict")
}

func TestSyncResolveCmd_Exists(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, syncResolveCmd)
	assert.Equal(t, "resolve [package-key]", syncResolveCmd.Use)
	assert.Contains(t, syncResolveCmd.Short, "Resolve")
}

func TestSyncConflictsCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"json", "auto-resolve", "lockfile", "remote-lockfile"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := syncConflictsCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

func TestSyncResolveCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"local", "remote", "newest", "skip"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := syncResolveCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

func TestRelationString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		relation sync.CausalRelation
		expected string
	}{
		{
			name:     "equal",
			relation: sync.Equal,
			expected: "equal (in sync)",
		},
		{
			name:     "before",
			relation: sync.Before,
			expected: "behind (pull needed)",
		},
		{
			name:     "after",
			relation: sync.After,
			expected: "ahead (push needed)",
		},
		{
			name:     "concurrent",
			relation: sync.Concurrent,
			expected: "concurrent (merge needed)",
		},
		{
			name:     "unknown",
			relation: sync.CausalRelation(99),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := relationString(tt.relation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConflictJSONTypes(t *testing.T) {
	t.Parallel()

	// Test ConflictJSON struct can be created with all fields
	conflict := ConflictJSON{
		PackageKey:    "brew:ripgrep",
		Type:          "BothModified",
		LocalVersion:  "14.0.0",
		RemoteVersion: "14.1.0",
		Resolvable:    true,
	}

	assert.Equal(t, "brew:ripgrep", conflict.PackageKey)
	assert.Equal(t, "BothModified", conflict.Type)
	assert.Equal(t, "14.0.0", conflict.LocalVersion)
	assert.Equal(t, "14.1.0", conflict.RemoteVersion)
	assert.True(t, conflict.Resolvable)
}

func TestConflictsOutputJSONTypes(t *testing.T) {
	t.Parallel()

	// Test ConflictsOutputJSON struct can be created with all fields
	output := ConflictsOutputJSON{
		Relation:       "concurrent (merge needed)",
		TotalConflicts: 5,
		AutoResolvable: 3,
		ManualConflicts: []ConflictJSON{
			{
				PackageKey:    "brew:pkg1",
				Type:          "BothModified",
				LocalVersion:  "1.0.0",
				RemoteVersion: "2.0.0",
				Resolvable:    false,
			},
		},
		NeedsMerge: true,
	}

	assert.Equal(t, "concurrent (merge needed)", output.Relation)
	assert.Equal(t, 5, output.TotalConflicts)
	assert.Equal(t, 3, output.AutoResolvable)
	assert.True(t, output.NeedsMerge)
	require.Len(t, output.ManualConflicts, 1)
	assert.Equal(t, "brew:pkg1", output.ManualConflicts[0].PackageKey)
}

func TestSyncConflictsCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"lockfile default", "lockfile", "preflight.lock"},
		{"remote-lockfile default", "remote-lockfile", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := syncConflictsCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestSyncResolveCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	// All boolean flags should default to false
	boolFlags := []string{"local", "remote", "newest", "skip"}

	for _, flag := range boolFlags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := syncResolveCmd.Flags().Lookup(flag)
			require.NotNil(t, f)
			assert.Equal(t, "false", f.DefValue)
		})
	}
}

func TestSyncConflictsCmd_IsSubcommandOfSync(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range syncCmd.Commands() {
		if cmd.Use == "conflicts" {
			found = true
			break
		}
	}
	assert.True(t, found, "conflicts should be a subcommand of sync")
}

func TestSyncResolveCmd_IsSubcommandOfSync(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range syncCmd.Commands() {
		if cmd.Name() == "resolve" {
			found = true
			break
		}
	}
	assert.True(t, found, "resolve should be a subcommand of sync")
}
