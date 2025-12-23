package lock

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// PackageLock errors.
var (
	ErrEmptyProvider      = errors.New("provider cannot be empty")
	ErrEmptyName          = errors.New("name cannot be empty")
	ErrEmptyVersion       = errors.New("version cannot be empty")
	ErrMissingIntegrity   = errors.New("integrity is required")
	ErrInvalidInstalledAt = errors.New("installed time cannot be zero")
	ErrInvalidPackageKey  = errors.New("invalid package key format")
)

// PackageLock represents a locked package version with integrity verification.
// It is an immutable value object.
type PackageLock struct {
	provider    string
	name        string
	version     string
	integrity   Integrity
	installedAt time.Time
}

// NewPackageLock creates a new PackageLock value object.
// Returns an error if any required field is empty or invalid.
func NewPackageLock(provider, name, version string, integrity Integrity, installedAt time.Time) (PackageLock, error) {
	if provider == "" {
		return PackageLock{}, ErrEmptyProvider
	}

	if name == "" {
		return PackageLock{}, ErrEmptyName
	}

	if version == "" {
		return PackageLock{}, ErrEmptyVersion
	}

	if integrity.IsZero() {
		return PackageLock{}, ErrMissingIntegrity
	}

	if installedAt.IsZero() {
		return PackageLock{}, ErrInvalidInstalledAt
	}

	return PackageLock{
		provider:    provider,
		name:        name,
		version:     version,
		integrity:   integrity,
		installedAt: installedAt,
	}, nil
}

// Provider returns the package provider (e.g., "brew", "apt").
func (p PackageLock) Provider() string {
	return p.provider
}

// Name returns the package name.
func (p PackageLock) Name() string {
	return p.name
}

// Version returns the locked version.
func (p PackageLock) Version() string {
	return p.version
}

// Integrity returns the content integrity hash.
func (p PackageLock) Integrity() Integrity {
	return p.integrity
}

// InstalledAt returns when this version was installed.
func (p PackageLock) InstalledAt() time.Time {
	return p.installedAt
}

// Key returns the unique key for this package (provider:name).
func (p PackageLock) Key() string {
	return p.provider + ":" + p.name
}

// String returns a human-readable representation (provider:name@version).
func (p PackageLock) String() string {
	return fmt.Sprintf("%s:%s@%s", p.provider, p.name, p.version)
}

// IsZero returns true if this is a zero-value PackageLock.
func (p PackageLock) IsZero() bool {
	return p.provider == "" && p.name == "" && p.version == ""
}

// WithVersion creates a new PackageLock with an updated version.
// The provider and name remain unchanged.
func (p PackageLock) WithVersion(version string, integrity Integrity, installedAt time.Time) (PackageLock, error) {
	return NewPackageLock(p.provider, p.name, version, integrity, installedAt)
}

// MatchesVersion returns true if the given version matches this lock's version.
func (p PackageLock) MatchesVersion(version string) bool {
	return p.version == version && version != ""
}

// ParsePackageKey parses a package key in the format "provider:name".
// Returns provider, name, and any error.
func ParsePackageKey(key string) (provider, name string, err error) {
	if key == "" {
		return "", "", fmt.Errorf("%w: empty key", ErrInvalidPackageKey)
	}

	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("%w: missing colon separator", ErrInvalidPackageKey)
	}

	if parts[0] == "" {
		return "", "", fmt.Errorf("%w: empty provider", ErrInvalidPackageKey)
	}

	if parts[1] == "" {
		return "", "", fmt.Errorf("%w: empty name", ErrInvalidPackageKey)
	}

	return parts[0], parts[1], nil
}
