package lock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMachineInfo_Valid(t *testing.T) {
	t.Parallel()

	now := time.Now()
	info, err := NewMachineInfo("darwin", "arm64", "macbook.local", now)

	require.NoError(t, err)
	assert.Equal(t, "darwin", info.OS())
	assert.Equal(t, "arm64", info.Arch())
	assert.Equal(t, "macbook.local", info.Hostname())
	assert.Equal(t, now, info.Snapshot())
}

func TestNewMachineInfo_AllSupportedOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		os string
	}{
		{"darwin"},
		{"linux"},
	}

	for _, tt := range tests {
		t.Run(tt.os, func(t *testing.T) {
			t.Parallel()
			info, err := NewMachineInfo(tt.os, "amd64", "host", time.Now())
			require.NoError(t, err)
			assert.Equal(t, tt.os, info.OS())
		})
	}
}

func TestNewMachineInfo_AllSupportedArch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		arch string
	}{
		{"amd64"},
		{"arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			t.Parallel()
			info, err := NewMachineInfo("darwin", tt.arch, "host", time.Now())
			require.NoError(t, err)
			assert.Equal(t, tt.arch, info.Arch())
		})
	}
}

func TestNewMachineInfo_UnsupportedOS(t *testing.T) {
	t.Parallel()

	_, err := NewMachineInfo("windows", "amd64", "host", time.Now())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedOS)
}

func TestNewMachineInfo_UnsupportedArch(t *testing.T) {
	t.Parallel()

	_, err := NewMachineInfo("darwin", "386", "host", time.Now())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedArch)
}

func TestNewMachineInfo_EmptyHostname(t *testing.T) {
	t.Parallel()

	_, err := NewMachineInfo("darwin", "arm64", "", time.Now())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyHostname)
}

func TestNewMachineInfo_ZeroTime(t *testing.T) {
	t.Parallel()

	_, err := NewMachineInfo("darwin", "arm64", "host", time.Time{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSnapshot)
}

func TestMachineInfoFromSystem(t *testing.T) {
	t.Parallel()

	info := MachineInfoFromSystem()

	// Should capture current system info
	assert.NotEmpty(t, info.OS())
	assert.NotEmpty(t, info.Arch())
	assert.NotEmpty(t, info.Hostname())
	assert.False(t, info.Snapshot().IsZero())

	// Snapshot should be recent (within last second)
	assert.WithinDuration(t, time.Now(), info.Snapshot(), time.Second)
}

func TestMachineInfo_String(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	info, err := NewMachineInfo("darwin", "arm64", "macbook.local", now)
	require.NoError(t, err)

	expected := "darwin/arm64 (macbook.local)"
	assert.Equal(t, expected, info.String())
}

func TestMachineInfo_IsZero(t *testing.T) {
	t.Parallel()

	var zero MachineInfo
	assert.True(t, zero.IsZero())

	nonZero, _ := NewMachineInfo("darwin", "arm64", "host", time.Now())
	assert.False(t, nonZero.IsZero())
}

func TestMachineInfo_Matches(t *testing.T) {
	t.Parallel()

	now := time.Now()
	info1, _ := NewMachineInfo("darwin", "arm64", "host1", now)
	info2, _ := NewMachineInfo("darwin", "arm64", "host2", now.Add(time.Hour))
	info3, _ := NewMachineInfo("linux", "arm64", "host1", now)
	info4, _ := NewMachineInfo("darwin", "amd64", "host1", now)

	// Same OS/arch matches regardless of hostname or time
	assert.True(t, info1.Matches(info2))

	// Different OS doesn't match
	assert.False(t, info1.Matches(info3))

	// Different arch doesn't match
	assert.False(t, info1.Matches(info4))
}

func TestMachineInfo_MatchesExact(t *testing.T) {
	t.Parallel()

	now := time.Now()
	info1, _ := NewMachineInfo("darwin", "arm64", "host", now)
	info2, _ := NewMachineInfo("darwin", "arm64", "host", now.Add(time.Hour))
	info3, _ := NewMachineInfo("darwin", "arm64", "different", now)

	// Same OS/arch/hostname matches (time ignored)
	assert.True(t, info1.MatchesExact(info2))

	// Different hostname doesn't match exactly
	assert.False(t, info1.MatchesExact(info3))
}
