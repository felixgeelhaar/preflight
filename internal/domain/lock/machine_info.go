package lock

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"
)

// MachineInfo errors.
var (
	ErrUnsupportedOS   = errors.New("unsupported operating system")
	ErrUnsupportedArch = errors.New("unsupported architecture")
	ErrEmptyHostname   = errors.New("hostname cannot be empty")
	ErrInvalidSnapshot = errors.New("snapshot time cannot be zero")
)

// Supported operating systems.
var supportedOS = map[string]bool{
	"darwin": true,
	"linux":  true,
}

// Supported architectures.
var supportedArch = map[string]bool{
	"amd64": true,
	"arm64": true,
}

// MachineInfo represents a snapshot of machine information.
// It is an immutable value object used to track where a lockfile was generated.
type MachineInfo struct {
	os       string
	arch     string
	hostname string
	snapshot time.Time
}

// NewMachineInfo creates a new MachineInfo value object.
// Returns an error if the OS or architecture is unsupported,
// hostname is empty, or snapshot time is zero.
func NewMachineInfo(os, arch, hostname string, snapshot time.Time) (MachineInfo, error) {
	if !supportedOS[os] {
		return MachineInfo{}, fmt.Errorf("%w: %s", ErrUnsupportedOS, os)
	}

	if !supportedArch[arch] {
		return MachineInfo{}, fmt.Errorf("%w: %s", ErrUnsupportedArch, arch)
	}

	if hostname == "" {
		return MachineInfo{}, ErrEmptyHostname
	}

	if snapshot.IsZero() {
		return MachineInfo{}, ErrInvalidSnapshot
	}

	return MachineInfo{
		os:       os,
		arch:     arch,
		hostname: hostname,
		snapshot: snapshot,
	}, nil
}

// MachineInfoFromSystem captures the current system's machine information.
func MachineInfoFromSystem() MachineInfo {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return MachineInfo{
		os:       runtime.GOOS,
		arch:     runtime.GOARCH,
		hostname: hostname,
		snapshot: time.Now(),
	}
}

// OS returns the operating system.
func (m MachineInfo) OS() string {
	return m.os
}

// Arch returns the architecture.
func (m MachineInfo) Arch() string {
	return m.arch
}

// Hostname returns the machine hostname.
func (m MachineInfo) Hostname() string {
	return m.hostname
}

// Snapshot returns the time when this info was captured.
func (m MachineInfo) Snapshot() time.Time {
	return m.snapshot
}

// String returns a human-readable representation.
func (m MachineInfo) String() string {
	return fmt.Sprintf("%s/%s (%s)", m.os, m.arch, m.hostname)
}

// IsZero returns true if this is a zero-value MachineInfo.
func (m MachineInfo) IsZero() bool {
	return m.os == "" && m.arch == "" && m.hostname == "" && m.snapshot.IsZero()
}

// Matches returns true if OS and architecture match.
// This is used to determine if a lockfile was generated on a compatible machine.
func (m MachineInfo) Matches(other MachineInfo) bool {
	return m.os == other.os && m.arch == other.arch
}

// MatchesExact returns true if OS, architecture, and hostname all match.
// Snapshot time is ignored in comparison.
func (m MachineInfo) MatchesExact(other MachineInfo) bool {
	return m.os == other.os && m.arch == other.arch && m.hostname == other.hostname
}
