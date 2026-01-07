package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapturePackageHelpers(t *testing.T) {
	outputs := map[string]string{
		"choco":      "pkg-one|1.0.0\npkg-two|2.1.0\n",
		"scoop":      "Name      Version    Source    Updated\npkg-three  1.0.0     main      today\npkg-four   2.0.0     extras    today\n",
		"winget":     "Name             Id                       Version\n-----------------------------------------------------\nAppOne           Company.AppOne           1.0.0\nAppTwo           Other.AppTwo             2.0.0\n",
		"dpkg-query": "pkg-five\npkg-six\n",
	}

	restoreEnv := withFakeCommands(t, outputs)
	defer restoreEnv()

	p := New(io.Discard)
	now := time.Now()
	ctx := context.Background()

	choco := p.captureChocolateyPackages(ctx, now)
	require.Len(t, choco, 2)
	assert.Equal(t, "pkg-one", choco[0].Name)
	assert.Equal(t, "chocolatey", choco[0].Provider)

	scoop := p.captureScoopPackages(ctx, now)
	require.Len(t, scoop, 2)
	assert.Equal(t, "pkg-three", scoop[0].Name)
	assert.Equal(t, "scoop", scoop[0].Provider)

	winget := p.captureWingetPackages(ctx, now)
	require.Len(t, winget, 2)
	assert.Equal(t, "Company.AppOne", winget[0].Name)
	assert.Equal(t, "winget", winget[0].Provider)

	apt := p.captureAPTPackages(ctx, now)
	require.Len(t, apt, 2)
	assert.Equal(t, "pkg-five", apt[0].Name)
	assert.Equal(t, "apt", apt[0].Provider)
}

func TestPreflight_WithRollbackOnFailure(t *testing.T) {
	p := New(io.Discard)
	require.Same(t, p, p.WithRollbackOnFailure(true))
}

func withFakeCommands(t *testing.T, outputs map[string]string) func() {
	t.Helper()

	dir := t.TempDir()
	for name, output := range outputs {
		script := fmt.Sprintf("#!/bin/sh\ncat <<'EOF'\n%s\nEOF\nexit 0\n", output)
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	}

	origPath := os.Getenv("PATH")
	require.NoError(t, os.Setenv("PATH", dir+string(os.PathListSeparator)+origPath))

	return func() {
		_ = os.Setenv("PATH", origPath)
	}
}
