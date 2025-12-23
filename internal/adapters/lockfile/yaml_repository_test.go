package lockfile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLRepository_SaveAndLoad(t *testing.T) {
	t.Parallel()

	repo := NewYAMLRepository()
	ctx := context.Background()

	// Create temp directory
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "preflight.lock")

	// Create a lockfile
	machineInfo := lock.MachineInfoFromSystem()
	lockfile := lock.NewLockfile(config.ModeIntent, machineInfo)

	// Add a package - use IntegrityFromData which doesn't error
	integrity := lock.IntegrityFromData(lock.AlgorithmSHA256, []byte("test data"))
	pkg, err := lock.NewPackageLock(
		"brew",
		"git",
		"2.43.0",
		integrity,
		time.Now(),
	)
	require.NoError(t, err)
	require.NoError(t, lockfile.AddPackage(pkg))

	// Save
	err = repo.Save(ctx, lockPath, lockfile)
	require.NoError(t, err)

	// Verify file exists
	assert.True(t, repo.Exists(ctx, lockPath))

	// Load
	loaded, err := repo.Load(ctx, lockPath)
	require.NoError(t, err)

	// Verify loaded data
	assert.Equal(t, lockfile.Version(), loaded.Version())
	assert.Equal(t, lockfile.Mode(), loaded.Mode())
	assert.Equal(t, lockfile.PackageCount(), loaded.PackageCount())

	loadedPkg, found := loaded.GetPackage("brew", "git")
	assert.True(t, found)
	assert.Equal(t, "2.43.0", loadedPkg.Version())
}

func TestYAMLRepository_LoadNotFound(t *testing.T) {
	t.Parallel()

	repo := NewYAMLRepository()
	ctx := context.Background()

	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "nonexistent.lock")

	_, err := repo.Load(ctx, lockPath)
	assert.ErrorIs(t, err, lock.ErrLockfileNotFound)
}

func TestYAMLRepository_LoadCorrupt(t *testing.T) {
	t.Parallel()

	repo := NewYAMLRepository()
	ctx := context.Background()

	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "corrupt.lock")

	// Write invalid YAML
	err := os.WriteFile(lockPath, []byte("invalid: yaml: content: ["), 0o644)
	require.NoError(t, err)

	_, err = repo.Load(ctx, lockPath)
	assert.ErrorIs(t, err, lock.ErrLockfileCorrupt)
}

func TestYAMLRepository_Exists(t *testing.T) {
	t.Parallel()

	repo := NewYAMLRepository()
	ctx := context.Background()

	tmpDir := t.TempDir()

	// File doesn't exist
	assert.False(t, repo.Exists(ctx, filepath.Join(tmpDir, "missing.lock")))

	// Create file
	existingPath := filepath.Join(tmpDir, "exists.lock")
	err := os.WriteFile(existingPath, []byte("version: 1"), 0o644)
	require.NoError(t, err)

	assert.True(t, repo.Exists(ctx, existingPath))
}

func TestYAMLRepository_SaveCreatesDirectory(t *testing.T) {
	t.Parallel()

	repo := NewYAMLRepository()
	ctx := context.Background()

	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "nested", "dir", "preflight.lock")

	machineInfo := lock.MachineInfoFromSystem()
	lockfile := lock.NewLockfile(config.ModeIntent, machineInfo)

	err := repo.Save(ctx, lockPath, lockfile)
	require.NoError(t, err)

	assert.True(t, repo.Exists(ctx, lockPath))
}

func TestYAMLRepository_SaveModes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode config.ReproducibilityMode
	}{
		{"intent", config.ModeIntent},
		{"locked", config.ModeLocked},
		{"frozen", config.ModeFrozen},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := NewYAMLRepository()
			ctx := context.Background()

			tmpDir := t.TempDir()
			lockPath := filepath.Join(tmpDir, "preflight.lock")

			machineInfo := lock.MachineInfoFromSystem()
			lockfile := lock.NewLockfile(tc.mode, machineInfo)

			err := repo.Save(ctx, lockPath, lockfile)
			require.NoError(t, err)

			loaded, err := repo.Load(ctx, lockPath)
			require.NoError(t, err)

			assert.Equal(t, tc.mode, loaded.Mode())
		})
	}
}
