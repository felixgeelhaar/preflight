//go:build linux

package keychain

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// LinuxKeychain implements Keychain using secret-tool (libsecret D-Bus interface).
type LinuxKeychain struct{}

// NewPlatformKeychain creates a keychain adapter for Linux.
func NewPlatformKeychain() *LinuxKeychain {
	return &LinuxKeychain{}
}

// Get retrieves a secret from the Linux secret service.
func (k *LinuxKeychain) Get(service, account string) (string, error) {
	cmd := exec.Command("secret-tool", "lookup",
		"service", service,
		"account", account,
	)
	out, err := cmd.Output()
	if err != nil {
		// secret-tool returns exit code 1 when not found
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return "", ports.ErrKeychainItemNotFound
		}
		return "", fmt.Errorf("secret-tool lookup failed: %w", err)
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", ports.ErrKeychainItemNotFound
	}
	return result, nil
}

// Set stores a secret in the Linux secret service.
func (k *LinuxKeychain) Set(service, account, secret string) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", fmt.Sprintf("preflight:%s:%s", service, account),
		"service", service,
		"account", account,
	)
	cmd.Stdin = strings.NewReader(secret)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("secret-tool store failed: %w", err)
	}
	return nil
}

// Delete removes a secret from the Linux secret service.
func (k *LinuxKeychain) Delete(service, account string) error {
	cmd := exec.Command("secret-tool", "clear",
		"service", service,
		"account", account,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("secret-tool clear failed: %w", err)
	}
	return nil
}

// Available returns true if secret-tool is installed.
func (k *LinuxKeychain) Available() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

// Compile-time interface check.
var _ ports.Keychain = (*LinuxKeychain)(nil)
