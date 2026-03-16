//go:build darwin

package keychain

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// DarwinKeychain implements Keychain using macOS Keychain Services via the security CLI.
type DarwinKeychain struct{}

// NewPlatformKeychain creates a keychain adapter for macOS.
func NewPlatformKeychain() *DarwinKeychain {
	return &DarwinKeychain{}
}

// Get retrieves a secret from the macOS Keychain.
func (k *DarwinKeychain) Get(service, account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", service,
		"-a", account,
		"-w",
	)
	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "could not be found") || cmd.ProcessState.ExitCode() == 44 {
			return "", ports.ErrKeychainItemNotFound
		}
		return "", fmt.Errorf("keychain get failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Set stores a secret in the macOS Keychain.
func (k *DarwinKeychain) Set(service, account, secret string) error {
	// Delete existing entry first (ignore errors)
	_ = k.Delete(service, account)

	cmd := exec.Command("security", "add-generic-password",
		"-s", service,
		"-a", account,
		"-w", secret,
		"-U", // Update if exists
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain set failed: %w", err)
	}
	return nil
}

// Delete removes a secret from the macOS Keychain.
func (k *DarwinKeychain) Delete(service, account string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", service,
		"-a", account,
	)
	if err := cmd.Run(); err != nil {
		if cmd.ProcessState.ExitCode() == 44 {
			return ports.ErrKeychainItemNotFound
		}
		return fmt.Errorf("keychain delete failed: %w", err)
	}
	return nil
}

// Available returns true if the macOS security CLI is accessible.
func (k *DarwinKeychain) Available() bool {
	_, err := exec.LookPath("security")
	return err == nil
}

// Compile-time interface check.
var _ ports.Keychain = (*DarwinKeychain)(nil)
