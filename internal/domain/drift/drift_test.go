package drift

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestType_String(t *testing.T) {
	tests := []struct {
		driftType Type
		expected  string
	}{
		{TypeManual, "manual"},
		{TypeExternal, "external"},
		{TypeUnknown, "unknown"},
		{TypeNone, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.driftType))
		})
	}
}

func TestNewDrift(t *testing.T) {
	now := time.Now()
	drift := NewDrift("/path/to/file", "abc123", "def456", now, TypeManual)

	assert.Equal(t, "/path/to/file", drift.Path)
	assert.Equal(t, "abc123", drift.CurrentHash)
	assert.Equal(t, "def456", drift.ExpectedHash)
	assert.Equal(t, now, drift.LastApplied)
	assert.Equal(t, TypeManual, drift.Type)
}

func TestDrift_HasDrift(t *testing.T) {
	tests := []struct {
		name         string
		currentHash  string
		expectedHash string
		hasDrift     bool
	}{
		{
			name:         "no drift when hashes match",
			currentHash:  "abc123",
			expectedHash: "abc123",
			hasDrift:     false,
		},
		{
			name:         "drift when hashes differ",
			currentHash:  "abc123",
			expectedHash: "def456",
			hasDrift:     true,
		},
		{
			name:         "no drift when both empty",
			currentHash:  "",
			expectedHash: "",
			hasDrift:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drift := NewDrift("/path", tt.currentHash, tt.expectedHash, time.Now(), TypeManual)
			assert.Equal(t, tt.hasDrift, drift.HasDrift())
		})
	}
}

func TestDrift_Description(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		driftType Type
		contains  string
	}{
		{
			name:      "manual drift description",
			driftType: TypeManual,
			contains:  "manually modified",
		},
		{
			name:      "external drift description",
			driftType: TypeExternal,
			contains:  "external tool",
		},
		{
			name:      "unknown drift description",
			driftType: TypeUnknown,
			contains:  "unknown source",
		},
		{
			name:      "no drift description",
			driftType: TypeNone,
			contains:  "no drift",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drift := NewDrift("/path/to/file", "abc", "def", now, tt.driftType)
			desc := drift.Description()
			assert.Contains(t, desc, tt.contains)
		})
	}
}

func TestFileState(t *testing.T) {
	now := time.Now()
	state := FileState{
		Path:        "/path/to/file",
		AppliedHash: "abc123",
		AppliedAt:   now,
		SourceLayer: "base",
	}

	assert.Equal(t, "/path/to/file", state.Path)
	assert.Equal(t, "abc123", state.AppliedHash)
	assert.Equal(t, now, state.AppliedAt)
	assert.Equal(t, "base", state.SourceLayer)
}

func TestAppliedState_SetFile(t *testing.T) {
	state := NewAppliedState()
	now := time.Now()

	state.SetFile("/path/to/file", "abc123", "base", now)

	fileState, exists := state.GetFile("/path/to/file")
	assert.True(t, exists)
	assert.Equal(t, "/path/to/file", fileState.Path)
	assert.Equal(t, "abc123", fileState.AppliedHash)
	assert.Equal(t, "base", fileState.SourceLayer)
	assert.Equal(t, now, fileState.AppliedAt)
}

func TestAppliedState_GetFile_NotFound(t *testing.T) {
	state := NewAppliedState()

	_, exists := state.GetFile("/nonexistent")
	assert.False(t, exists)
}

func TestAppliedState_RemoveFile(t *testing.T) {
	state := NewAppliedState()
	now := time.Now()

	state.SetFile("/path/to/file", "abc123", "base", now)
	state.RemoveFile("/path/to/file")

	_, exists := state.GetFile("/path/to/file")
	assert.False(t, exists)
}

func TestAppliedState_ListFiles(t *testing.T) {
	state := NewAppliedState()
	now := time.Now()

	state.SetFile("/path/a", "hash1", "base", now)
	state.SetFile("/path/b", "hash2", "work", now)
	state.SetFile("/path/c", "hash3", "base", now)

	files := state.ListFiles()
	assert.Len(t, files, 3)

	// Check all paths are present
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	assert.ElementsMatch(t, []string{"/path/a", "/path/b", "/path/c"}, paths)
}

func TestAppliedState_Clear(t *testing.T) {
	state := NewAppliedState()
	now := time.Now()

	state.SetFile("/path/a", "hash1", "base", now)
	state.SetFile("/path/b", "hash2", "work", now)

	state.Clear()

	assert.Empty(t, state.ListFiles())
}

func TestNewAppliedState(t *testing.T) {
	state := NewAppliedState()
	assert.NotNil(t, state)
	assert.NotNil(t, state.Files)
	assert.Empty(t, state.Files)
}
