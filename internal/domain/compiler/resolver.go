package compiler

// ResolutionSource indicates where a version resolution came from.
type ResolutionSource string

const (
	// ResolutionSourceNone indicates resolution failed.
	ResolutionSourceNone ResolutionSource = ""
	// ResolutionSourceLatest indicates version from latest available.
	ResolutionSourceLatest ResolutionSource = "latest"
	// ResolutionSourceLockfile indicates version from lockfile.
	ResolutionSourceLockfile ResolutionSource = "lockfile"
)

// Resolution represents the result of version resolution.
type Resolution struct {
	Provider         string
	Name             string
	Version          string
	Source           ResolutionSource
	Locked           bool
	LockedVersion    string
	AvailableVersion string
	Drifted          bool
	Updated          bool
	Failed           bool
	Error            error
}

// VersionResolver resolves package versions based on lockfile state.
type VersionResolver interface {
	// Resolve returns the resolved version for a package.
	// If no locked version exists, returns the latestVersion.
	Resolve(provider, name, latestVersion string) Resolution
}
