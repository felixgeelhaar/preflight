// Package platform provides platform detection and utilities for cross-platform support.
package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// OS represents the operating system type.
type OS string

const (
	// OSDarwin is macOS.
	OSDarwin OS = "darwin"
	// OSLinux is Linux (native or WSL).
	OSLinux OS = "linux"
	// OSWindows is Windows.
	OSWindows OS = "windows"
	// OSUnknown is an unsupported OS.
	OSUnknown OS = "unknown"
)

// Environment represents the execution environment.
type Environment string

const (
	// EnvNative is a native OS environment.
	EnvNative Environment = "native"
	// EnvWSL1 is Windows Subsystem for Linux version 1.
	EnvWSL1 Environment = "wsl1"
	// EnvWSL2 is Windows Subsystem for Linux version 2.
	EnvWSL2 Environment = "wsl2"
	// EnvDocker is running inside a Docker container.
	EnvDocker Environment = "docker"
	// EnvUnknown is an unknown environment.
	EnvUnknown Environment = "unknown"
)

// Platform contains detected platform information.
type Platform struct {
	os          OS
	arch        string
	environment Environment
	wslDistro   string
	windowsPath string // Path to Windows from WSL (e.g., /mnt/c)
}

var (
	detected     *Platform
	detectOnce   sync.Once
	detectedErr  error
	testPlatform *Platform // For testing
)

// Detect returns the current platform information.
// Results are cached after the first call.
func Detect() (*Platform, error) {
	if testPlatform != nil {
		return testPlatform, nil
	}

	detectOnce.Do(func() {
		detected, detectedErr = detect()
	})
	return detected, detectedErr
}

// SetTestPlatform sets a mock platform for testing.
// Pass nil to reset to actual detection.
func SetTestPlatform(p *Platform) {
	testPlatform = p
}

//nolint:unparam // error return kept for future expansion
func detect() (*Platform, error) {
	p := &Platform{
		arch:        runtime.GOARCH,
		environment: EnvNative,
	}

	switch runtime.GOOS {
	case "darwin":
		p.os = OSDarwin
	case "linux":
		p.os = OSLinux
		p.detectLinuxEnvironment()
	case "windows":
		p.os = OSWindows
	default:
		p.os = OSUnknown
	}

	return p, nil
}

// detectLinuxEnvironment checks if running in WSL or Docker.
func (p *Platform) detectLinuxEnvironment() {
	// Check for WSL
	if p.isWSL() {
		p.detectWSLVersion()
		p.detectWSLDistro()
		p.detectWindowsPath()
		return
	}

	// Check for Docker
	if p.isDocker() {
		p.environment = EnvDocker
		return
	}
}

// isWSL checks if running in Windows Subsystem for Linux.
func (p *Platform) isWSL() bool {
	// Check /proc/version for Microsoft or WSL indicators
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}

	version := strings.ToLower(string(data))
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}

// detectWSLVersion determines WSL 1 or WSL 2.
func (p *Platform) detectWSLVersion() {
	// WSL 2 uses a real Linux kernel and has /run/WSL
	if _, err := os.Stat("/run/WSL"); err == nil {
		p.environment = EnvWSL2
		return
	}

	// Check for WSL interop which is present in both but structured differently
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		p.environment = EnvWSL1
		return
	}

	version := strings.ToLower(string(data))
	// WSL 2 kernels typically have higher version numbers and different naming
	if strings.Contains(version, "wsl2") {
		p.environment = EnvWSL2
	} else {
		// Default to WSL1 for older versions
		p.environment = EnvWSL1
	}
}

// detectWSLDistro detects the WSL distribution name.
func (p *Platform) detectWSLDistro() {
	// Check WSL_DISTRO_NAME environment variable
	if distro := os.Getenv("WSL_DISTRO_NAME"); distro != "" {
		p.wslDistro = distro
		return
	}

	// Try to read from /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			p.wslDistro = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			return
		}
	}
}

// detectWindowsPath finds the Windows mount point.
func (p *Platform) detectWindowsPath() {
	// Default WSL mount points
	candidates := []string{"/mnt/c", "/mnt/d", "/c", "/d"}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			// Verify it's a Windows filesystem by checking for typical Windows folders
			if _, err := os.Stat(path + "/Windows"); err == nil {
				p.windowsPath = path
				return
			}
			if _, err := os.Stat(path + "/Users"); err == nil {
				p.windowsPath = path
				return
			}
		}
	}

	// Default to /mnt/c even if not verified
	p.windowsPath = "/mnt/c"
}

// isDocker checks if running inside a Docker container.
func (p *Platform) isDocker() bool {
	// Check for Docker-specific files
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}

	return strings.Contains(string(data), "docker") ||
		strings.Contains(string(data), "containerd")
}

// OS returns the operating system.
func (p *Platform) OS() OS {
	return p.os
}

// Arch returns the architecture.
func (p *Platform) Arch() string {
	return p.arch
}

// Environment returns the execution environment.
func (p *Platform) Environment() Environment {
	return p.environment
}

// WSLDistro returns the WSL distribution name (empty if not WSL).
func (p *Platform) WSLDistro() string {
	return p.wslDistro
}

// WindowsPath returns the Windows mount point in WSL (e.g., /mnt/c).
func (p *Platform) WindowsPath() string {
	return p.windowsPath
}

// IsWindows returns true if running on Windows (native).
func (p *Platform) IsWindows() bool {
	return p.os == OSWindows
}

// IsMacOS returns true if running on macOS.
func (p *Platform) IsMacOS() bool {
	return p.os == OSDarwin
}

// IsLinux returns true if running on Linux (native or WSL).
func (p *Platform) IsLinux() bool {
	return p.os == OSLinux
}

// IsWSL returns true if running in WSL (1 or 2).
func (p *Platform) IsWSL() bool {
	return p.environment == EnvWSL1 || p.environment == EnvWSL2
}

// IsWSL2 returns true if running specifically in WSL 2.
func (p *Platform) IsWSL2() bool {
	return p.environment == EnvWSL2
}

// IsDocker returns true if running in a Docker container.
func (p *Platform) IsDocker() bool {
	return p.environment == EnvDocker
}

// IsNative returns true if running in a native environment.
func (p *Platform) IsNative() bool {
	return p.environment == EnvNative
}

// CanAccessWindows returns true if Windows filesystem is accessible.
func (p *Platform) CanAccessWindows() bool {
	return p.IsWSL() && p.windowsPath != ""
}

// HasCommand checks if a command is available in PATH.
func (p *Platform) HasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// String returns a human-readable description.
func (p *Platform) String() string {
	var parts []string
	parts = append(parts, string(p.os))
	parts = append(parts, p.arch)

	if p.environment != EnvNative {
		parts = append(parts, string(p.environment))
	}

	if p.wslDistro != "" {
		parts = append(parts, p.wslDistro)
	}

	return strings.Join(parts, "/")
}

// New creates a Platform with specified values (for testing).
func New(os OS, arch string, env Environment) *Platform {
	return &Platform{
		os:          os,
		arch:        arch,
		environment: env,
	}
}

// NewWSL creates a Platform for WSL testing.
func NewWSL(version Environment, distro, windowsPath string) *Platform {
	return &Platform{
		os:          OSLinux,
		arch:        "amd64",
		environment: version,
		wslDistro:   distro,
		windowsPath: windowsPath,
	}
}
