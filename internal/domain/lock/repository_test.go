package lock

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockfileToDTO(t *testing.T) {
	t.Parallel()

	machineInfo, _ := NewMachineInfo("darwin", "arm64", "macbook.local",
		time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))
	lockfile := NewLockfile(config.ModeLocked, machineInfo)

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)
	pkg, _ := NewPackageLock("brew", "ripgrep", "14.1.0", integrity,
		time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC))
	_ = lockfile.AddPackage(pkg)

	dto := LockfileToDTO(lockfile)

	assert.Equal(t, LockfileVersion, dto.Version)
	assert.Equal(t, "locked", dto.Mode)
	assert.Equal(t, "darwin", dto.MachineInfo.OS)
	assert.Equal(t, "arm64", dto.MachineInfo.Arch)
	assert.Equal(t, "macbook.local", dto.MachineInfo.Hostname)
	assert.Equal(t, "2025-01-15T10:30:00Z", dto.MachineInfo.Snapshot)

	pkgDTO, exists := dto.Packages["brew:ripgrep"]
	require.True(t, exists)
	assert.Equal(t, "14.1.0", pkgDTO.Version)
	assert.Equal(t, "sha256:"+hash, pkgDTO.Integrity)
	assert.Equal(t, "2025-01-15T11:00:00Z", pkgDTO.InstalledAt)
}

func TestLockfileFromDTO(t *testing.T) {
	t.Parallel()

	dto := LockfileDTO{
		Version: 1,
		Mode:    "frozen",
		MachineInfo: MachineInfoDTO{
			OS:       "linux",
			Arch:     "amd64",
			Hostname: "server.local",
			Snapshot: "2025-01-15T10:30:00Z",
		},
		Packages: map[string]PackageDTO{
			"brew:ripgrep": {
				Version:     "14.1.0",
				Integrity:   "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				InstalledAt: "2025-01-15T11:00:00Z",
			},
		},
	}

	lockfile, err := LockfileFromDTO(dto)

	require.NoError(t, err)
	assert.Equal(t, LockfileVersion, lockfile.Version())
	assert.Equal(t, config.ModeFrozen, lockfile.Mode())
	assert.Equal(t, "linux", lockfile.MachineInfo().OS())
	assert.Equal(t, "amd64", lockfile.MachineInfo().Arch())
	assert.Equal(t, "server.local", lockfile.MachineInfo().Hostname())

	pkg, exists := lockfile.GetPackage("brew", "ripgrep")
	require.True(t, exists)
	assert.Equal(t, "14.1.0", pkg.Version())
	assert.Equal(t, "sha256", pkg.Integrity().Algorithm())
}

func TestLockfileFromDTO_InvalidMachineInfo(t *testing.T) {
	t.Parallel()

	dto := LockfileDTO{
		Version: 1,
		Mode:    "locked",
		MachineInfo: MachineInfoDTO{
			OS:       "windows", // unsupported
			Arch:     "amd64",
			Hostname: "server",
			Snapshot: "2025-01-15T10:30:00Z",
		},
	}

	_, err := LockfileFromDTO(dto)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedOS)
}

func TestLockfileFromDTO_InvalidPackage(t *testing.T) {
	t.Parallel()

	dto := LockfileDTO{
		Version: 1,
		Mode:    "locked",
		MachineInfo: MachineInfoDTO{
			OS:       "darwin",
			Arch:     "arm64",
			Hostname: "macbook",
			Snapshot: "2025-01-15T10:30:00Z",
		},
		Packages: map[string]PackageDTO{
			"invalidkey": { // missing colon
				Version:     "14.1.0",
				Integrity:   "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				InstalledAt: "2025-01-15T11:00:00Z",
			},
		},
	}

	_, err := LockfileFromDTO(dto)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPackageKey)
}

func TestLockfileFromDTO_InvalidIntegrity(t *testing.T) {
	t.Parallel()

	dto := LockfileDTO{
		Version: 1,
		Mode:    "locked",
		MachineInfo: MachineInfoDTO{
			OS:       "darwin",
			Arch:     "arm64",
			Hostname: "macbook",
			Snapshot: "2025-01-15T10:30:00Z",
		},
		Packages: map[string]PackageDTO{
			"brew:ripgrep": {
				Version:     "14.1.0",
				Integrity:   "invalid", // missing colon
				InstalledAt: "2025-01-15T11:00:00Z",
			},
		},
	}

	_, err := LockfileFromDTO(dto)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidHash)
}

func TestLockfileFromDTO_InvalidTimestamp(t *testing.T) {
	t.Parallel()

	dto := LockfileDTO{
		Version: 1,
		Mode:    "locked",
		MachineInfo: MachineInfoDTO{
			OS:       "darwin",
			Arch:     "arm64",
			Hostname: "macbook",
			Snapshot: "not-a-timestamp",
		},
	}

	_, err := LockfileFromDTO(dto)

	require.Error(t, err)
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	// Create original lockfile
	machineInfo, _ := NewMachineInfo("darwin", "arm64", "macbook.local",
		time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))
	original := NewLockfile(config.ModeLocked, machineInfo)

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := NewIntegrity("sha256", hash)
	pkg, _ := NewPackageLock("brew", "ripgrep", "14.1.0", integrity,
		time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC))
	_ = original.AddPackage(pkg)

	// Convert to DTO and back
	dto := LockfileToDTO(original)
	restored, err := LockfileFromDTO(dto)

	require.NoError(t, err)
	assert.Equal(t, original.Version(), restored.Version())
	assert.Equal(t, original.Mode(), restored.Mode())
	assert.Equal(t, original.MachineInfo().OS(), restored.MachineInfo().OS())
	assert.Equal(t, original.MachineInfo().Arch(), restored.MachineInfo().Arch())
	assert.Equal(t, original.MachineInfo().Hostname(), restored.MachineInfo().Hostname())
	assert.Equal(t, original.PackageCount(), restored.PackageCount())

	origPkg, _ := original.GetPackage("brew", "ripgrep")
	restPkg, _ := restored.GetPackage("brew", "ripgrep")
	assert.Equal(t, origPkg.Version(), restPkg.Version())
	assert.Equal(t, origPkg.Integrity().String(), restPkg.Integrity().String())
}
