//go:build windows

package keychain

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// WindowsKeychain implements Keychain using Windows Credential Manager via cmdkey.
type WindowsKeychain struct{}

// NewPlatformKeychain creates a keychain adapter for Windows.
func NewPlatformKeychain() *WindowsKeychain {
	return &WindowsKeychain{}
}

// targetName builds the credential target name.
func targetName(service, account string) string {
	return fmt.Sprintf("preflight:%s:%s", service, account)
}

// Get retrieves a secret from Windows Credential Manager.
func (k *WindowsKeychain) Get(service, account string) (string, error) {
	target := targetName(service, account)
	cmd := exec.Command("cmdkey", "/list:"+target)
	out, err := cmd.Output()
	if err != nil || !strings.Contains(string(out), target) {
		return "", ports.ErrKeychainItemNotFound
	}
	// Windows Credential Manager doesn't expose passwords via cmdkey /list.
	// Use PowerShell to retrieve the actual credential.
	psCmd := fmt.Sprintf(
		`[Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR((Get-StoredCredential -Target '%s').Password))`,
		target,
	)
	cmd = exec.Command("powershell", "-Command", psCmd)
	out, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("credential manager get failed: %w", err)
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", ports.ErrKeychainItemNotFound
	}
	return result, nil
}

// Set stores a secret in Windows Credential Manager.
func (k *WindowsKeychain) Set(service, account, secret string) error {
	target := targetName(service, account)
	cmd := exec.Command("cmdkey",
		"/generic:"+target,
		"/user:"+account,
		"/pass:"+secret,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("credential manager set failed: %w", err)
	}
	return nil
}

// Delete removes a secret from Windows Credential Manager.
func (k *WindowsKeychain) Delete(service, account string) error {
	target := targetName(service, account)
	cmd := exec.Command("cmdkey", "/delete:"+target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("credential manager delete failed: %w", err)
	}
	return nil
}

// Available returns true if cmdkey is available.
func (k *WindowsKeychain) Available() bool {
	_, err := exec.LookPath("cmdkey")
	return err == nil
}

// Compile-time interface check.
var _ ports.Keychain = (*WindowsKeychain)(nil)
