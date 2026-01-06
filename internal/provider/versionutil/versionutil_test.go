package versionutil

import (
	"errors"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/stretchr/testify/require"
)

type stubResolver struct {
	resolution compiler.Resolution
}

func (s stubResolver) Resolve(_, _, _ string) compiler.Resolution {
	return s.resolution
}

func TestResolvePackageVersion_ReturnsLatestAsEmpty(t *testing.T) {
	ctx := compiler.NewCompileContext(nil).WithResolver(stubResolver{
		resolution: compiler.Resolution{
			Version: "latest",
		},
	})

	version, err := ResolvePackageVersion(ctx, "npm", "eslint", "")
	require.NoError(t, err)
	require.Equal(t, "", version)
}

func TestResolvePackageVersion_ReturnsResolvedVersion(t *testing.T) {
	ctx := compiler.NewCompileContext(nil).WithResolver(stubResolver{
		resolution: compiler.Resolution{
			Version: "1.2.3",
		},
	})

	version, err := ResolvePackageVersion(ctx, "npm", "eslint", "")
	require.NoError(t, err)
	require.Equal(t, "1.2.3", version)
}

func TestResolvePackageVersion_PropagatesError(t *testing.T) {
	expected := errors.New("lock missing")
	ctx := compiler.NewCompileContext(nil).WithResolver(stubResolver{
		resolution: compiler.Resolution{
			Failed: true,
			Error:  expected,
		},
	})

	_, err := ResolvePackageVersion(ctx, "npm", "eslint", "")
	require.ErrorIs(t, err, expected)
}
