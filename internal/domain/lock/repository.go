package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
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
	Sync        *SyncMetadataDTO      `yaml:"sync,omitempty"` // V2: Multi-machine sync
	Packages    map[string]PackageDTO `yaml:"packages,omitempty"`
}

// SyncMetadataDTO is the serializable representation of SyncMetadata.
type SyncMetadataDTO struct {
	Vector  map[string]uint64     `yaml:"vector"`
	Lineage map[string]LineageDTO `yaml:"lineage,omitempty"`
}

// LineageDTO is the serializable representation of MachineLineage.
type LineageDTO struct {
	Hostname string `yaml:"hostname"`
	LastSeen string `yaml:"last_seen"` // RFC3339 format
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

	// V2: Include sync metadata if present
	if !l.SyncMetadata().IsZero() {
		syncDTO := &SyncMetadataDTO{
			Vector:  l.SyncMetadata().Vector().ToMap(),
			Lineage: make(map[string]LineageDTO),
		}
		for machineID, lineage := range l.SyncMetadata().Lineage() {
			syncDTO.Lineage[machineID] = LineageDTO{
				Hostname: lineage.Hostname(),
				LastSeen: lineage.LastSeen().Format(time.RFC3339),
			}
		}
		dto.Sync = syncDTO
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

	// Create lockfile based on version
	mode := config.ReproducibilityMode(dto.Mode)
	var lockfile *Lockfile
	if dto.Version >= LockfileVersionV2 {
		lockfile = NewLockfileV2(mode, machineInfo)
	} else {
		lockfile = NewLockfile(mode, machineInfo)
	}

	// V2: Parse sync metadata if present
	if dto.Sync != nil {
		vector := sync.FromMap(dto.Sync.Vector)
		syncMeta := sync.NewSyncMetadata(vector)

		for machineID, lineageDTO := range dto.Sync.Lineage {
			lastSeen, err := time.Parse(time.RFC3339, lineageDTO.LastSeen)
			if err != nil {
				return nil, fmt.Errorf("invalid last_seen for machine %s: %w", machineID, err)
			}
			lineage := sync.NewMachineLineage(machineID, lineageDTO.Hostname, lastSeen)
			syncMeta = syncMeta.AddLineage(lineage)
		}

		lockfile = lockfile.WithSyncMetadata(syncMeta)
	}

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
