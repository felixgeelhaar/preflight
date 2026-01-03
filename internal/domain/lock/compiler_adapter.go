package lock

import "github.com/felixgeelhaar/preflight/internal/domain/compiler"

// CompilerAdapter adapts lock.Resolver to compiler.VersionResolver interface.
// This breaks the dependency from compiler → lock by having lock implement
// the interface defined in compiler.
type CompilerAdapter struct {
	resolver *Resolver
}

// NewCompilerAdapter creates a new adapter for the given resolver.
func NewCompilerAdapter(resolver *Resolver) *CompilerAdapter {
	return &CompilerAdapter{resolver: resolver}
}

// Resolve implements compiler.VersionResolver by delegating to lock.Resolver
// and converting the lock.Resolution to compiler.Resolution.
func (a *CompilerAdapter) Resolve(provider, name, latestVersion string) compiler.Resolution {
	res := a.resolver.Resolve(provider, name, latestVersion)
	return toCompilerResolution(res)
}

// toCompilerResolution converts lock.Resolution to compiler.Resolution.
func toCompilerResolution(res Resolution) compiler.Resolution {
	var source compiler.ResolutionSource
	switch res.Source {
	case ResolutionSourceNone:
		source = compiler.ResolutionSourceNone
	case ResolutionSourceLatest:
		source = compiler.ResolutionSourceLatest
	case ResolutionSourceLockfile:
		source = compiler.ResolutionSourceLockfile
	}

	return compiler.Resolution{
		Provider:         res.Provider,
		Name:             res.Name,
		Version:          res.Version,
		Source:           source,
		Locked:           res.Locked,
		LockedVersion:    res.LockedVersion,
		AvailableVersion: res.AvailableVersion,
		Drifted:          res.Drifted,
		Updated:          res.Updated,
		Failed:           res.Failed,
		Error:            res.Error,
	}
}
