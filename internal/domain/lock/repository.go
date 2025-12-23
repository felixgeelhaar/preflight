package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
)

// Repository errors.
var (
	ErrLockfileNotFound = errors.New("lockfile not found")
	ErrLockfileCorrupt  = errors.New("lockfile is corrupt")
	ErrSaveFailed       = errors.New("failed to save lockfile")
)

// Repository is the port for lockfile persistence.
// Implementations handle the actual file I/O and serialization.
type Repository interface {
	// Load reads a lockfile from the given path.
	// Returns ErrLockfileNotFound if the file doesn't exist.
	// Returns ErrLockfileCorrupt if the file exists but is invalid.
	Load(ctx context.Context, path string) (*Lockfile, error)

	// Save writes a lockfile to the given path.
	// Creates the file if it doesn't exist, overwrites if it does.
	Save(ctx context.Context, path string, lockfile *Lockfile) error

	// Exists returns true if a lockfile exists at the given path.
	Exists(ctx context.Context, path string) bool
}

// LockfileDTO is a data transfer object for lockfile serialization.
// It maps between the domain Lockfile and the persisted format.
type LockfileDTO struct {
	Version     int                   `yaml:"version"`
	Mode        string                `yaml:"mode"`
	MachineInfo MachineInfoDTO        `yaml:"machine_info"`
	Packages    map[string]PackageDTO `yaml:"packages,omitempty"`
}

// MachineInfoDTO is the serializable representation of MachineInfo.
type MachineInfoDTO struct {
	OS       string `yaml:"os"`
	Arch     string `yaml:"arch"`
	Hostname string `yaml:"hostname"`
	Snapshot string `yaml:"snapshot"` // RFC3339 format
}

// PackageDTO is the serializable representation of PackageLock.
type PackageDTO struct {
	Version     string `yaml:"version"`
	Integrity   string `yaml:"integrity"`    // "algorithm:hash" format
	InstalledAt string `yaml:"installed_at"` // RFC3339 format
}

// LockfileToDTO converts a Lockfile to its serializable DTO representation.
func LockfileToDTO(l *Lockfile) LockfileDTO {
	dto := LockfileDTO{
		Version: l.Version(),
		Mode:    string(l.Mode()),
		MachineInfo: MachineInfoDTO{
			OS:       l.MachineInfo().OS(),
			Arch:     l.MachineInfo().Arch(),
			Hostname: l.MachineInfo().Hostname(),
			Snapshot: l.MachineInfo().Snapshot().Format(time.RFC3339),
		},
		Packages: make(map[string]PackageDTO),
	}

	for key, pkg := range l.Packages() {
		dto.Packages[key] = PackageDTO{
			Version:     pkg.Version(),
			Integrity:   pkg.Integrity().String(),
			InstalledAt: pkg.InstalledAt().Format(time.RFC3339),
		}
	}

	return dto
}

// LockfileFromDTO converts a DTO to a Lockfile domain object.
// Returns an error if any field is invalid.
func LockfileFromDTO(dto LockfileDTO) (*Lockfile, error) {
	// Parse machine info
	snapshot, err := time.Parse(time.RFC3339, dto.MachineInfo.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("invalid machine info snapshot: %w", err)
	}

	machineInfo, err := NewMachineInfo(
		dto.MachineInfo.OS,
		dto.MachineInfo.Arch,
		dto.MachineInfo.Hostname,
		snapshot,
	)
	if err != nil {
		return nil, err
	}

	// Create lockfile with mode
	mode := config.ReproducibilityMode(dto.Mode)
	lockfile := NewLockfile(mode, machineInfo)

	// Add packages
	for key, pkgDTO := range dto.Packages {
		provider, name, err := ParsePackageKey(key)
		if err != nil {
			return nil, err
		}

		integrity, err := ParseIntegrity(pkgDTO.Integrity)
		if err != nil {
			return nil, err
		}

		installedAt, err := time.Parse(time.RFC3339, pkgDTO.InstalledAt)
		if err != nil {
			return nil, fmt.Errorf("invalid installed_at for %s: %w", key, err)
		}

		pkg, err := NewPackageLock(provider, name, pkgDTO.Version, integrity, installedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid package %s: %w", key, err)
		}

		if err := lockfile.AddPackage(pkg); err != nil {
			return nil, err
		}
	}

	return lockfile, nil
}
