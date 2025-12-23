package lock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPackageLock_Valid(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)
	now := time.Now()

	lock, err := NewPackageLock("brew", "ripgrep", "14.1.0", integrity, now)

	require.NoError(t, err)
	assert.Equal(t, "brew", lock.Provider())
	assert.Equal(t, "ripgrep", lock.Name())
	assert.Equal(t, "14.1.0", lock.Version())
	assert.Equal(t, integrity, lock.Integrity())
	assert.Equal(t, now, lock.InstalledAt())
}

func TestNewPackageLock_EmptyProvider(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)

	_, err := NewPackageLock("", "ripgrep", "14.1.0", integrity, time.Now())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyProvider)
}

func TestNewPackageLock_EmptyName(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)

	_, err := NewPackageLock("brew", "", "14.1.0", integrity, time.Now())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyName)
}

func TestNewPackageLock_EmptyVersion(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)

	_, err := NewPackageLock("brew", "ripgrep", "", integrity, time.Now())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyVersion)
}

func TestNewPackageLock_ZeroIntegrity(t *testing.T) {
	t.Parallel()

	_, err := NewPackageLock("brew", "ripgrep", "14.1.0", Integrity{}, time.Now())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingIntegrity)
}

func TestNewPackageLock_ZeroInstalledAt(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)

	_, err := NewPackageLock("brew", "ripgrep", "14.1.0", integrity, time.Time{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInstalledAt)
}

func TestPackageLock_Key(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)
	lock, _ := NewPackageLock("brew", "ripgrep", "14.1.0", integrity, time.Now())

	assert.Equal(t, "brew:ripgrep", lock.Key())
}

func TestPackageLock_String(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)
	lock, _ := NewPackageLock("brew", "ripgrep", "14.1.0", integrity, time.Now())

	expected := "brew:ripgrep@14.1.0"
	assert.Equal(t, expected, lock.String())
}

func TestPackageLock_IsZero(t *testing.T) {
	t.Parallel()

	var zero PackageLock
	assert.True(t, zero.IsZero())

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)
	nonZero, _ := NewPackageLock("brew", "ripgrep", "14.1.0", integrity, time.Now())
	assert.False(t, nonZero.IsZero())
}

func TestPackageLock_WithVersion(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)
	original, _ := NewPackageLock("brew", "ripgrep", "14.0.0", integrity, time.Now())

	newHash := "a3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	newIntegrity, _ := NewIntegrity("sha256", newHash)
	newTime := time.Now().Add(time.Hour)

	updated, err := original.WithVersion("14.1.0", newIntegrity, newTime)

	require.NoError(t, err)
	// Original unchanged
	assert.Equal(t, "14.0.0", original.Version())

	// New version updated
	assert.Equal(t, "14.1.0", updated.Version())
	assert.Equal(t, newIntegrity, updated.Integrity())
	assert.Equal(t, newTime, updated.InstalledAt())

	// Provider and name preserved
	assert.Equal(t, original.Provider(), updated.Provider())
	assert.Equal(t, original.Name(), updated.Name())
}

func TestPackageLock_MatchesVersion(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)
	lock, _ := NewPackageLock("brew", "ripgrep", "14.1.0", integrity, time.Now())

	assert.True(t, lock.MatchesVersion("14.1.0"))
	assert.False(t, lock.MatchesVersion("14.0.0"))
	assert.False(t, lock.MatchesVersion(""))
}

func TestParsePackageKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		key          string
		wantProvider string
		wantName     string
		wantErr      bool
	}{
		{"valid", "brew:ripgrep", "brew", "ripgrep", false},
		{"valid with dots", "brew:go@1.21", "brew", "go@1.21", false},
		{"empty", "", "", "", true},
		{"no colon", "brewripgrep", "", "", true},
		{"empty provider", ":ripgrep", "", "", true},
		{"empty name", "brew:", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, name, err := ParsePackageKey(tt.key)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantProvider, provider)
				assert.Equal(t, tt.wantName, name)
			}
		})
	}
}
