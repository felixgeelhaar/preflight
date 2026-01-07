// Package versionutil provides version resolution utilities for providers.
package versionutil

import "github.com/felixgeelhaar/preflight/internal/domain/compiler"

// ResolvePackageVersion resolves a version using the compile context resolver.
// An empty requested version means "latest".
func ResolvePackageVersion(ctx compiler.CompileContext, provider, name, requested string) (string, error) {
	desired := requested
	if desired == "" {
		desired = "latest"
	}

	resolution := ctx.ResolveVersion(provider, name, desired)
	if resolution.Failed {
		return "", resolution.Error
	}
	if resolution.Version == "latest" {
		return "", nil
	}
	return resolution.Version, nil
}
