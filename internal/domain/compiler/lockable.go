package compiler

// LockInfo describes a package version that can be persisted to the lockfile.
type LockInfo struct {
	Provider string
	Name     string
	Version  string
}

// LockableStep exposes lockable package information for deterministic runs.
type LockableStep interface {
	LockInfo() (LockInfo, bool)
}

// VersionedStep provides an installed version for lockfile updates.
type VersionedStep interface {
	InstalledVersion(ctx RunContext) (string, bool, error)
}
