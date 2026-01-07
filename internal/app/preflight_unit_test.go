package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/stretchr/testify/require"
)

func TestPreflight_WithMode(t *testing.T) {
	t.Parallel()

	p := New(io.Discard)
	require.Same(t, p, p.WithMode(config.ModeFrozen))
}

func TestPreflight_LoadMergedConfigAndManifest(t *testing.T) {
	t.Parallel()

	p := New(io.Discard)
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "preflight.yaml")
	configYAML := `
version: "1"
defaults:
  target: base
targets:
  base:
    - base
`
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "layers"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "layers", "base.yaml"), []byte("name: base\n"), 0o644))
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o644))

	merged, err := p.LoadMergedConfig(context.Background(), configPath, "base")
	require.NoError(t, err)
	require.NotEmpty(t, merged)

	manifest, err := p.LoadManifest(context.Background(), configPath)
	require.NoError(t, err)
	require.Contains(t, manifest.Targets, "base")
}

func TestVersionResolverAdapterResolve(t *testing.T) {
	t.Parallel()

	data := []byte("content")
	integrity := lock.IntegrityFromData(lock.AlgorithmSHA256, data)
	pkg, err := lock.NewPackageLock("brew", "git", "1.2.3", integrity, time.Now())
	require.NoError(t, err)

	lockfile := lock.NewLockfile(config.ModeLocked, lock.MachineInfoFromSystem())
	require.NoError(t, lockfile.AddPackage(pkg))

	resolver := lock.NewResolver(lockfile)
	adapter := versionResolverAdapter{resolver: resolver}

	res := adapter.Resolve("brew", "git", "1.2.3")
	require.Equal(t, compiler.ResolutionSourceLockfile, res.Source)
	require.True(t, res.Locked)
	require.Equal(t, "1.2.3", res.Version)
}
