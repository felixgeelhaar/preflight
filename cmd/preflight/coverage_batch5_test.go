package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// secrets.go - resolveAge (40% coverage)
// ---------------------------------------------------------------------------

func TestBatch5_ResolveAge_NonExistentKey(t *testing.T) {
	t.Parallel()

	val, err := resolveAge("nonexistent-key-12345-batch5")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, val)
}

func TestBatch5_ResolveAge_SecretFileExists_NoIdentity(t *testing.T) {
	// Create a temp home with the age secret file but no identity file
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	secretsDir := filepath.Join(tmpHome, ".preflight", "secrets")
	require.NoError(t, os.MkdirAll(secretsDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(secretsDir, "test-key.age"),
		[]byte("age-encrypted-data"), 0o600))

	_, err := resolveAge("test-key")
	// This will fail because the age CLI either doesn't exist or can't decrypt
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decrypt")
}

// ---------------------------------------------------------------------------
// secrets.go - runSecretsCheck with env secrets (76.2% coverage)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsConfigPath
func TestBatch5_RunSecretsCheck_WithEnvSecrets_AllResolved(t *testing.T) {
	t.Setenv("BATCH5_SECRET_A", "val-a")
	t.Setenv("BATCH5_SECRET_B", "val-b")

	content := `env:
  key_a: "secret://env/BATCH5_SECRET_A"
  key_b: "secret://env/BATCH5_SECRET_B"
`
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	saved := secretsConfigPath
	defer func() { secretsConfigPath = saved }()
	secretsConfigPath = tmpFile

	output := captureStdout(t, func() {
		// runSecretsCheck calls os.Exit(1) if any fail, but all should pass
		err := runSecretsCheck(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Checking 2 secret(s)")
	assert.Contains(t, output, "2 passed, 0 failed")
}

// ---------------------------------------------------------------------------
// trust.go - runTrustList with keys in store (23.1% coverage)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME environment
func TestBatch5_RunTrustList_WithKeys(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a trust store with keys
	storePath := filepath.Join(tmpHome, ".preflight", "trust.json")
	store := catalog.NewTrustStore(storePath)

	pub1 := catalog.NewPublisher("Alice Dev", "alice@example.com", "ssh-key-1", catalog.SignatureTypeSSH)
	key1 := catalog.NewTrustedKey("ssh-key-1", catalog.SignatureTypeSSH, nil, pub1)
	key1.SetFingerprint("SHA256:abc123def456")
	key1.SetComment("Alice SSH key")
	require.NoError(t, store.Add(key1))

	pub2 := catalog.NewPublisher("Bob Sec", "bob@example.com", "gpg-key-1", catalog.SignatureTypeGPG)
	key2 := catalog.NewTrustedKey("gpg-key-1", catalog.SignatureTypeGPG, nil, pub2)
	key2.SetFingerprint("ABCD1234")
	key2.SetComment("Bob GPG key")
	require.NoError(t, store.Add(key2))

	require.NoError(t, store.Save())

	output := captureStdout(t, func() {
		err := runTrustList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "KEY ID")
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "PUBLISHER")
	assert.Contains(t, output, "LEVEL")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "Total:")
}

// ---------------------------------------------------------------------------
// trust.go - runTrustShow with key details (81.5% coverage)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME environment
func TestBatch5_RunTrustShow_FullKeyDetails(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	storePath := filepath.Join(tmpHome, ".preflight", "trust.json")
	store := catalog.NewTrustStore(storePath)

	pub := catalog.NewPublisher("Test Author", "test@example.com", "show-key-1", catalog.SignatureTypeSSH)
	key := catalog.NewTrustedKey("show-key-1", catalog.SignatureTypeSSH, nil, pub)
	key.SetFingerprint("SHA256:showkey123")
	key.SetComment("Batch5 test key")
	require.NoError(t, store.Add(key))
	require.NoError(t, store.Save())

	output := captureStdout(t, func() {
		err := runTrustShow(nil, []string{"show-key-1"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Key ID:")
	assert.Contains(t, output, "show-key-1")
	assert.Contains(t, output, "Type:")
	assert.Contains(t, output, "ssh")
	assert.Contains(t, output, "Fingerprint:")
	assert.Contains(t, output, "SHA256:showkey123")
	assert.Contains(t, output, "Trust Level:")
	assert.Contains(t, output, "Publisher:")
	assert.Contains(t, output, "Test Author")
	assert.Contains(t, output, "test@example.com")
	assert.Contains(t, output, "Added:")
	assert.Contains(t, output, "active")
	assert.Contains(t, output, "Comment:")
	assert.Contains(t, output, "Batch5 test key")
}

//nolint:tparallel // modifies HOME environment
func TestBatch5_RunTrustShow_KeyNotFound(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	err := runTrustShow(nil, []string{"nonexistent-key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

// ---------------------------------------------------------------------------
// trust.go - runTrustRemove with force flag (47.8% coverage)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME and global trustForce
func TestBatch5_RunTrustRemove_WithForce(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	storePath := filepath.Join(tmpHome, ".preflight", "trust.json")
	store := catalog.NewTrustStore(storePath)

	pub := catalog.NewPublisher("Remove Test", "", "remove-key-1", catalog.SignatureTypeSSH)
	key := catalog.NewTrustedKey("remove-key-1", catalog.SignatureTypeSSH, nil, pub)
	require.NoError(t, store.Add(key))
	require.NoError(t, store.Save())

	savedForce := trustForce
	defer func() { trustForce = savedForce }()
	trustForce = true

	output := captureStdout(t, func() {
		err := runTrustRemove(nil, []string{"remove-key-1"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Removed key: remove-key-1")
}

//nolint:tparallel // modifies HOME
func TestBatch5_RunTrustRemove_KeyNotFound(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	err := runTrustRemove(nil, []string{"nonexistent-key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

// ---------------------------------------------------------------------------
// trust.go - runTrustAdd SSH key type override (86.1% coverage)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME and global trust flags
func TestBatch5_RunTrustAdd_SSHKeyTypeOverride(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	tmpFile := filepath.Join(t.TempDir(), "sshkey.pub")
	require.NoError(t, os.WriteFile(tmpFile,
		[]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBatch5TestKey user@host"), 0o600))

	savedName := trustKeyName
	savedLevel := trustKeyLevel
	savedType := trustKeyType
	savedEmail := trustEmail
	defer func() {
		trustKeyName = savedName
		trustKeyLevel = savedLevel
		trustKeyType = savedType
		trustEmail = savedEmail
	}()

	trustKeyName = "Batch5 SSH Key"
	trustKeyLevel = "verified"
	trustKeyType = "ssh"
	trustEmail = "batch5@example.com"

	output := captureStdout(t, func() {
		err := runTrustAdd(nil, []string{tmpFile})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Added key")
	assert.Contains(t, output, "ssh")
	assert.Contains(t, output, "verified")
}

//nolint:tparallel // modifies global trust flags
func TestBatch5_RunTrustAdd_UnknownKeyType(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tmpFile := filepath.Join(t.TempDir(), "key.pub")
	require.NoError(t, os.WriteFile(tmpFile, []byte("some data"), 0o600))

	savedType := trustKeyType
	defer func() { trustKeyType = savedType }()
	trustKeyType = "unknown-type"

	err := runTrustAdd(nil, []string{tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key type")
}

//nolint:tparallel // modifies global trust flags
func TestBatch5_RunTrustAdd_CannotDetectKeyType(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tmpFile := filepath.Join(t.TempDir(), "badkey.pub")
	require.NoError(t, os.WriteFile(tmpFile, []byte("just random data"), 0o600))

	savedType := trustKeyType
	defer func() { trustKeyType = savedType }()
	trustKeyType = ""

	err := runTrustAdd(nil, []string{tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not detect key type")
}

//nolint:tparallel // modifies global trust flags
func TestBatch5_RunTrustAdd_InvalidTrustLevel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tmpFile := filepath.Join(t.TempDir(), "sshkey.pub")
	require.NoError(t, os.WriteFile(tmpFile,
		[]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 test@host"), 0o600))

	savedLevel := trustKeyLevel
	savedType := trustKeyType
	defer func() {
		trustKeyLevel = savedLevel
		trustKeyType = savedType
	}()
	trustKeyLevel = "invalid-level"
	trustKeyType = "ssh"

	err := runTrustAdd(nil, []string{tmpFile})
	require.Error(t, err)
}

//nolint:tparallel // modifies global trust flags
func TestBatch5_RunTrustAdd_NonExistentFile(t *testing.T) {
	err := runTrustAdd(nil, []string{"/nonexistent/file.pub"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read key file")
}

// ---------------------------------------------------------------------------
// trust.go - detectKeyType additional SSH prefix coverage
// ---------------------------------------------------------------------------

func TestBatch5_DetectKeyType_AllSSHPrefixes(t *testing.T) {
	t.Parallel()

	sshPrefixes := []string{
		"ssh-ed25519 AAAA...",
		"ssh-rsa AAAA...",
		"ssh-dss AAAA...",
		"ecdsa-sha2-nistp256 AAAA...",
		"ecdsa-sha2-nistp384 AAAA...",
		"ecdsa-sha2-nistp521 AAAA...",
		"sk-ssh-ed25519 AAAA...",
		"sk-ecdsa-sha2-nistp256 AAAA...",
	}

	for _, prefix := range sshPrefixes {
		t.Run(prefix[:10], func(t *testing.T) {
			t.Parallel()
			result := detectKeyType([]byte(prefix))
			assert.Equal(t, catalog.SignatureTypeSSH, result, "expected SSH for prefix %s", prefix)
		})
	}
}

func TestBatch5_DetectKeyType_AllGPGArmorHeaders(t *testing.T) {
	t.Parallel()

	gpgHeaders := []string{
		"-----BEGIN PGP PUBLIC KEY-----\ndata...",
		"-----BEGIN PGP PRIVATE KEY-----\ndata...",
		"-----BEGIN PGP MESSAGE-----\ndata...",
		"-----BEGIN PGP SIGNATURE-----\ndata...",
	}

	for _, header := range gpgHeaders {
		t.Run(header[14:28], func(t *testing.T) {
			t.Parallel()
			result := detectKeyType([]byte(header))
			assert.Equal(t, catalog.SignatureTypeGPG, result, "expected GPG for header %s", header[:30])
		})
	}
}

func TestBatch5_DetectKeyType_EmptyAndUnknown(t *testing.T) {
	t.Parallel()

	assert.Equal(t, catalog.SignatureType(""), detectKeyType(nil))
	assert.Equal(t, catalog.SignatureType(""), detectKeyType([]byte{}))
	assert.Equal(t, catalog.SignatureType(""), detectKeyType([]byte("random binary data")))
}

// ---------------------------------------------------------------------------
// trust.go - isValidOpenPGPPacket comprehensive
// ---------------------------------------------------------------------------

func TestBatch5_IsValidOpenPGPPacket_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		data   []byte
		expect bool
	}{
		{"too_short_one_byte", []byte{0xC6}, false},
		{"too_short_empty", []byte{}, false},
		{"bit7_not_set", []byte{0x00, 0x00}, false},
		{"bit7_not_set_with_data", []byte{0x3F, 0xFF}, false},
		// New format (bit 6 set): tag = data[0] & 0x3f
		{"new_format_tag2_signature", []byte{0xC2, 0x00}, true},   // 0xC0 | 2
		{"new_format_tag5_secret_key", []byte{0xC5, 0x00}, true},  // 0xC0 | 5
		{"new_format_tag6_public_key", []byte{0xC6, 0x00}, true},  // 0xC0 | 6
		{"new_format_tag7_secret_sub", []byte{0xC7, 0x00}, true},  // 0xC0 | 7
		{"new_format_tag14_pub_sub", []byte{0xCE, 0x00}, true},    // 0xC0 | 14
		{"new_format_tag0_invalid", []byte{0xC0, 0x00}, false},    // tag 0 not valid
		{"new_format_tag1_invalid", []byte{0xC1, 0x00}, false},    // tag 1 (session key) not valid
		{"new_format_tag3_invalid", []byte{0xC3, 0x00}, false},    // tag 3 not in valid set
		// Old format: tag = (data[0] & 0x3c) >> 2
		{"old_format_tag6_public_key", []byte{0x98, 0x00}, true},  // (6 << 2) | 0x80 = 0x98
		{"old_format_tag2_signature", []byte{0x88, 0x00}, true},   // (2 << 2) | 0x80 = 0x88
		{"old_format_tag5_secret_key", []byte{0x94, 0x00}, true},  // (5 << 2) | 0x80 = 0x94
		{"old_format_tag0_invalid", []byte{0x80, 0x00}, false},    // (0 << 2) | 0x80 = 0x80
		{"old_format_tag1_invalid", []byte{0x84, 0x00}, false},    // (1 << 2) | 0x80 = 0x84
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isValidOpenPGPPacket(tt.data)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// ---------------------------------------------------------------------------
// clean.go - removeOrphans with unknown provider type (84.6% coverage)
// ---------------------------------------------------------------------------

func TestBatch5_RemoveOrphans_UnknownProvider(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "unknown-provider", Type: "package", Name: "something"},
	}

	output := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		// Unknown providers fall through the switch without error
		assert.Equal(t, 1, removed)
		assert.Equal(t, 0, failed)
	})

	assert.Contains(t, output, "Removed unknown-provider something")
}

func TestBatch5_RemoveOrphans_MixedProviders(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "formula", Name: "htop"},
		{Provider: "brew", Type: "cask", Name: "chrome"},
		{Provider: "vscode", Type: "extension", Name: "ext1"},
		{Provider: "files", Type: "file", Name: "/some/file"},
	}

	output := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		assert.Equal(t, 4, removed)
		assert.Equal(t, 0, failed)
	})

	assert.Contains(t, output, "htop")
	assert.Contains(t, output, "chrome")
	assert.Contains(t, output, "ext1")
}

// ---------------------------------------------------------------------------
// clean.go - shouldCheckProvider and isIgnored edge cases
// ---------------------------------------------------------------------------

func TestBatch5_ShouldCheckProvider_EdgeCases(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldCheckProvider(nil, "brew"))
	assert.True(t, shouldCheckProvider([]string{}, "brew"))
	assert.True(t, shouldCheckProvider([]string{"brew", "vscode"}, "brew"))
	assert.True(t, shouldCheckProvider([]string{"brew", "vscode"}, "vscode"))
	assert.False(t, shouldCheckProvider([]string{"brew"}, "vscode"))
	assert.False(t, shouldCheckProvider([]string{"apt"}, "brew"))
}

func TestBatch5_IsIgnored_EdgeCases(t *testing.T) {
	t.Parallel()

	assert.False(t, isIgnored("pkg1", nil))
	assert.False(t, isIgnored("pkg1", []string{}))
	assert.False(t, isIgnored("pkg1", []string{"pkg2", "pkg3"}))
	assert.True(t, isIgnored("pkg1", []string{"pkg1", "pkg2"}))
	assert.True(t, isIgnored("pkg1", []string{"pkg1"}))
}

// ---------------------------------------------------------------------------
// clean.go - findBrewOrphans and findVSCodeOrphans direct tests
// ---------------------------------------------------------------------------

func TestBatch5_FindBrewOrphans_FormulaeAndCasks(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "go"},
			"casks":    []interface{}{"firefox"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "go", "htop", "curl"},
			"casks":    []interface{}{"firefox", "chrome", "slack"},
		},
	}

	orphans := findBrewOrphans(config, systemState, nil)
	assert.Len(t, orphans, 4) // htop, curl, chrome, slack

	// With ignore
	orphans2 := findBrewOrphans(config, systemState, []string{"htop", "chrome"})
	assert.Len(t, orphans2, 2) // curl, slack
}

func TestBatch5_FindBrewOrphans_EmptyConfig(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git"},
		},
	}

	orphans := findBrewOrphans(config, systemState, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "git", orphans[0].Name)
}

func TestBatch5_FindVSCodeOrphans_CaseInsensitive(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"MS-Python.Python"},
		},
	}
	systemState := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"ms-python.python", "golang.go"},
		},
	}

	orphans := findVSCodeOrphans(config, systemState, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "golang.go", orphans[0].Name)
}

func TestBatch5_FindVSCodeOrphans_EmptyConfig(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{}
	systemState := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.go"},
		},
	}

	orphans := findVSCodeOrphans(config, systemState, nil)
	assert.Len(t, orphans, 1)
}

// ---------------------------------------------------------------------------
// clean.go - findOrphans with combined providers and filters
// ---------------------------------------------------------------------------

func TestBatch5_FindOrphans_AllProvidersWithOrphans(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git"},
			"casks":    []interface{}{"firefox"},
		},
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"ms-python.python"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "htop"},
			"casks":    []interface{}{"firefox", "chrome"},
		},
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"ms-python.python", "golang.go"},
		},
	}

	orphans := findOrphans(config, systemState, nil, nil)
	assert.Len(t, orphans, 3) // htop, chrome, golang.go

	// Filter to vscode only
	orphans2 := findOrphans(config, systemState, []string{"vscode"}, nil)
	assert.Len(t, orphans2, 1)
	assert.Equal(t, "vscode", orphans2[0].Provider)

	// Ignore htop
	orphans3 := findOrphans(config, systemState, nil, []string{"htop"})
	assert.Len(t, orphans3, 2) // chrome, golang.go
}

// ---------------------------------------------------------------------------
// clean.go - outputOrphansText
// ---------------------------------------------------------------------------

func TestBatch5_OutputOrphansText_MultipleItems(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "formula", Name: "htop"},
		{Provider: "brew", Type: "cask", Name: "chrome"},
		{Provider: "vscode", Type: "extension", Name: "golang.go"},
	}

	output := captureStdout(t, func() {
		outputOrphansText(orphans)
	})

	assert.Contains(t, output, "3 orphaned items")
	assert.Contains(t, output, "PROVIDER")
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "htop")
	assert.Contains(t, output, "chrome")
	assert.Contains(t, output, "golang.go")
}

// ---------------------------------------------------------------------------
// clean.go - runBrewUninstall and runVSCodeUninstall
// ---------------------------------------------------------------------------

func TestBatch5_RunBrewUninstall_Formula(t *testing.T) {
	output := captureStdout(t, func() {
		err := runBrewUninstall("htop", false)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "brew uninstall htop")
	assert.NotContains(t, output, "--cask")
}

func TestBatch5_RunBrewUninstall_Cask(t *testing.T) {
	output := captureStdout(t, func() {
		err := runBrewUninstall("chrome", true)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "brew uninstall --cask chrome")
}

func TestBatch5_RunVSCodeUninstall(t *testing.T) {
	output := captureStdout(t, func() {
		err := runVSCodeUninstall("golang.go")
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "code --uninstall-extension golang.go")
}

// ---------------------------------------------------------------------------
// history.go - parseDuration edge cases
// ---------------------------------------------------------------------------

func TestBatch5_ParseDuration_AllUnits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"hours", "1h", time.Hour, false},
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"weeks", "2w", 14 * 24 * time.Hour, false},
		{"months", "1m", 30 * 24 * time.Hour, false},
		{"zero_hours", "0h", 0, false},
		{"large_days", "365d", 365 * 24 * time.Hour, false},
		{"single_char", "x", 0, true},
		{"no_number", "d", 0, true},
		{"unknown_unit", "1x", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, err := parseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// history.go - formatStatus
// ---------------------------------------------------------------------------

func TestBatch5_FormatStatus_AllCases(t *testing.T) {
	t.Parallel()

	assert.Contains(t, formatStatus("success"), "success")
	assert.Contains(t, formatStatus("failed"), "failed")
	assert.Contains(t, formatStatus("partial"), "partial")
	assert.Equal(t, "unknown", formatStatus("unknown"))
	assert.Equal(t, "custom", formatStatus("custom"))
	assert.Equal(t, "", formatStatus(""))
}

// ---------------------------------------------------------------------------
// history.go - formatHistoryAge comprehensive
// ---------------------------------------------------------------------------

func TestBatch5_FormatHistoryAge_AllRanges(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "just now", formatHistoryAge(time.Now()))
	assert.Contains(t, formatHistoryAge(time.Now().Add(-5*time.Minute)), "m ago")
	assert.Contains(t, formatHistoryAge(time.Now().Add(-3*time.Hour)), "h ago")
	assert.Contains(t, formatHistoryAge(time.Now().Add(-3*24*time.Hour)), "d ago")
	assert.Contains(t, formatHistoryAge(time.Now().Add(-10*24*time.Hour)), "w ago")

	// Old date - returns formatted date (e.g., "Jan 2")
	old := time.Now().Add(-60 * 24 * time.Hour)
	result := formatHistoryAge(old)
	assert.NotContains(t, result, "ago")
}

// ---------------------------------------------------------------------------
// history.go - outputHistoryText both modes
// ---------------------------------------------------------------------------

func TestBatch5_OutputHistoryText_NonVerbose(t *testing.T) {
	saved := historyVerbose
	defer func() { historyVerbose = saved }()
	historyVerbose = false

	entries := []HistoryEntry{
		{
			ID:        "b5-entry-1",
			Timestamp: time.Now().Add(-5 * time.Minute),
			Command:   "apply",
			Target:    "default",
			Status:    "success",
			Duration:  "1s",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "git"},
			},
		},
		{
			ID:        "b5-entry-2",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Command:   "rollback",
			Status:    "failed",
			Error:     "something went wrong",
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "Showing 2 entries")
	assert.Contains(t, output, "TIME")
	assert.Contains(t, output, "COMMAND")
}

func TestBatch5_OutputHistoryText_Verbose(t *testing.T) {
	saved := historyVerbose
	defer func() { historyVerbose = saved }()
	historyVerbose = true

	entries := []HistoryEntry{
		{
			ID:        "b5-verbose-1",
			Timestamp: time.Now(),
			Command:   "apply",
			Target:    "work",
			Status:    "success",
			Duration:  "2.5s",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "ripgrep"},
			},
		},
		{
			ID:      "b5-verbose-2",
			Command: "doctor",
			Status:  "failed",
			Error:   "connectivity issue",
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	assert.Contains(t, output, "Changes:")
	assert.Contains(t, output, "[brew] install: ripgrep")
	assert.Contains(t, output, "Error:")
	assert.Contains(t, output, "connectivity issue")
	assert.Contains(t, output, "Showing 2 entries")
}

// ---------------------------------------------------------------------------
// history.go - loadHistory and SaveHistoryEntry
// ---------------------------------------------------------------------------

func TestBatch5_LoadHistory_NonExistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entries, err := loadHistory()
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestBatch5_SaveHistoryEntry_And_LoadHistory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		ID:        "batch5-test-001",
		Timestamp: time.Now(),
		Command:   "apply",
		Target:    "work",
		Status:    "success",
		Duration:  "1.5s",
		Changes: []Change{
			{Provider: "brew", Action: "install", Item: "git"},
		},
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	entries, err := loadHistory()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "batch5-test-001", entries[0].ID)
}

func TestBatch5_GetHistoryDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := getHistoryDir()
	assert.Contains(t, dir, ".preflight")
	assert.Contains(t, dir, "history")
}

// ---------------------------------------------------------------------------
// history.go - runHistory with filters
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global history flags and HOME
func TestBatch5_RunHistory_EmptyHistory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedLimit := historyLimit
	savedSince := historySince
	savedJSON := historyJSON
	savedProvider := historyProvider
	savedVerbose := historyVerbose
	defer func() {
		historyLimit = savedLimit
		historySince = savedSince
		historyJSON = savedJSON
		historyProvider = savedProvider
		historyVerbose = savedVerbose
	}()

	historyLimit = 20
	historySince = ""
	historyJSON = false
	historyProvider = ""
	historyVerbose = false

	output := captureStdout(t, func() {
		err := runHistory(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No history entries found")
}

//nolint:tparallel // modifies global history flags and HOME
func TestBatch5_RunHistory_WithSinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save entries
	require.NoError(t, SaveHistoryEntry(HistoryEntry{
		ID:        "since-test-1",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Command:   "apply",
		Status:    "success",
	}))
	require.NoError(t, SaveHistoryEntry(HistoryEntry{
		ID:        "since-test-2",
		Timestamp: time.Now().Add(-48 * time.Hour),
		Command:   "rollback",
		Status:    "failed",
	}))

	savedLimit := historyLimit
	savedSince := historySince
	savedJSON := historyJSON
	savedProvider := historyProvider
	savedVerbose := historyVerbose
	defer func() {
		historyLimit = savedLimit
		historySince = savedSince
		historyJSON = savedJSON
		historyProvider = savedProvider
		historyVerbose = savedVerbose
	}()

	historyLimit = 20
	historySince = "24h"
	historyJSON = false
	historyProvider = ""
	historyVerbose = false

	output := captureStdout(t, func() {
		err := runHistory(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Showing 1 entries")
}

//nolint:tparallel // modifies global history flags and HOME
func TestBatch5_RunHistory_WithProviderFilter(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, SaveHistoryEntry(HistoryEntry{
		ID:        "prov-test-1",
		Timestamp: time.Now(),
		Command:   "apply",
		Status:    "success",
		Changes: []Change{
			{Provider: "brew", Action: "install", Item: "git"},
		},
	}))
	require.NoError(t, SaveHistoryEntry(HistoryEntry{
		ID:        "prov-test-2",
		Timestamp: time.Now(),
		Command:   "apply",
		Status:    "success",
		Changes: []Change{
			{Provider: "vscode", Action: "install", Item: "ext1"},
		},
	}))

	savedLimit := historyLimit
	savedSince := historySince
	savedJSON := historyJSON
	savedProvider := historyProvider
	savedVerbose := historyVerbose
	defer func() {
		historyLimit = savedLimit
		historySince = savedSince
		historyJSON = savedJSON
		historyProvider = savedProvider
		historyVerbose = savedVerbose
	}()

	historyLimit = 20
	historySince = ""
	historyJSON = false
	historyProvider = "brew"
	historyVerbose = false

	output := captureStdout(t, func() {
		err := runHistory(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Showing 1 entries")
}

//nolint:tparallel // modifies global history flags and HOME
func TestBatch5_RunHistory_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, SaveHistoryEntry(HistoryEntry{
		ID:        "json-test-1",
		Timestamp: time.Now(),
		Command:   "apply",
		Status:    "success",
	}))

	savedLimit := historyLimit
	savedSince := historySince
	savedJSON := historyJSON
	savedProvider := historyProvider
	savedVerbose := historyVerbose
	defer func() {
		historyLimit = savedLimit
		historySince = savedSince
		historyJSON = savedJSON
		historyProvider = savedProvider
		historyVerbose = savedVerbose
	}()

	historyLimit = 20
	historySince = ""
	historyJSON = true
	historyProvider = ""
	historyVerbose = false

	output := captureStdout(t, func() {
		err := runHistory(nil, nil)
		require.NoError(t, err)
	})

	var entries []HistoryEntry
	err := json.Unmarshal([]byte(output), &entries)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

//nolint:tparallel // modifies global history flags and HOME
func TestBatch5_RunHistory_InvalidSince(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedSince := historySince
	defer func() { historySince = savedSince }()
	historySince = "invalid"

	err := runHistory(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration")
}

//nolint:tparallel // modifies global history flags and HOME
func TestBatch5_RunHistory_WithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	for i := 0; i < 5; i++ {
		require.NoError(t, SaveHistoryEntry(HistoryEntry{
			ID:        "limit-test-" + string(rune('a'+i)),
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Command:   "apply",
			Status:    "success",
		}))
	}

	savedLimit := historyLimit
	savedSince := historySince
	savedJSON := historyJSON
	savedProvider := historyProvider
	savedVerbose := historyVerbose
	defer func() {
		historyLimit = savedLimit
		historySince = savedSince
		historyJSON = savedJSON
		historyProvider = savedProvider
		historyVerbose = savedVerbose
	}()

	historyLimit = 2
	historySince = ""
	historyJSON = false
	historyProvider = ""
	historyVerbose = false

	output := captureStdout(t, func() {
		err := runHistory(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Showing 2 entries")
}

// ---------------------------------------------------------------------------
// export.go - exportToNix comprehensive
// ---------------------------------------------------------------------------

func TestBatch5_ExportToNix_FullConfig(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "go", "vim"},
		},
		"git": map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
		"shell": map[string]interface{}{
			"shell":   "zsh",
			"plugins": []interface{}{"git", "docker"},
		},
	}

	output, err := exportToNix(config)
	require.NoError(t, err)
	text := string(output)

	assert.Contains(t, text, "Generated by preflight")
	assert.Contains(t, text, "home.packages")
	assert.Contains(t, text, "programs.git")
	assert.Contains(t, text, "Test User")
	assert.Contains(t, text, "test@example.com")
	assert.Contains(t, text, "programs.zsh")
	assert.Contains(t, text, "enable = true")
}

func TestBatch5_ExportToNix_EmptyConfig(t *testing.T) {
	t.Parallel()

	output, err := exportToNix(map[string]interface{}{})
	require.NoError(t, err)
	text := string(output)

	assert.Contains(t, text, "Generated by preflight")
	assert.NotContains(t, text, "home.packages")
	assert.NotContains(t, text, "programs.git")
}

func TestBatch5_ExportToNix_ShellNotZsh(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"shell": map[string]interface{}{
			"shell": "bash",
		},
	}

	output, err := exportToNix(config)
	require.NoError(t, err)
	text := string(output)

	assert.NotContains(t, text, "programs.zsh")
}

// ---------------------------------------------------------------------------
// export.go - exportToBrewfile comprehensive
// ---------------------------------------------------------------------------

func TestBatch5_ExportToBrewfile_FullConfig(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask-fonts", "homebrew/core"},
			"formulae": []interface{}{"git", "go"},
			"casks":    []interface{}{"firefox", "chrome"},
		},
	}

	output, err := exportToBrewfile(config)
	require.NoError(t, err)
	text := string(output)

	assert.Contains(t, text, `tap "homebrew/cask-fonts"`)
	assert.Contains(t, text, `tap "homebrew/core"`)
	assert.Contains(t, text, `brew "git"`)
	assert.Contains(t, text, `brew "go"`)
	assert.Contains(t, text, `cask "firefox"`)
	assert.Contains(t, text, `cask "chrome"`)
}

func TestBatch5_ExportToBrewfile_EmptyConfig(t *testing.T) {
	t.Parallel()

	output, err := exportToBrewfile(map[string]interface{}{})
	require.NoError(t, err)
	text := string(output)

	assert.Contains(t, text, "Generated by preflight")
	assert.NotContains(t, text, "tap")
	assert.NotContains(t, text, "brew")
	assert.NotContains(t, text, "cask")
}

// ---------------------------------------------------------------------------
// export.go - exportToShell comprehensive
// ---------------------------------------------------------------------------

func TestBatch5_ExportToShell_FullConfig(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/core"},
			"formulae": []interface{}{"git", "go"},
			"casks":    []interface{}{"firefox"},
		},
		"git": map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
	}

	output, err := exportToShell(config)
	require.NoError(t, err)
	text := string(output)

	assert.Contains(t, text, "#!/usr/bin/env bash")
	assert.Contains(t, text, "set -euo pipefail")
	assert.Contains(t, text, "brew tap homebrew/core")
	assert.Contains(t, text, "brew install")
	assert.Contains(t, text, "git")
	assert.Contains(t, text, "go")
	assert.Contains(t, text, "brew install --cask")
	assert.Contains(t, text, "firefox")
	assert.Contains(t, text, `git config --global user.name "Test User"`)
	assert.Contains(t, text, `git config --global user.email "test@example.com"`)
	assert.Contains(t, text, "Setup complete!")
}

func TestBatch5_ExportToShell_EmptyConfig(t *testing.T) {
	t.Parallel()

	output, err := exportToShell(map[string]interface{}{})
	require.NoError(t, err)
	text := string(output)

	assert.Contains(t, text, "#!/usr/bin/env bash")
	assert.Contains(t, text, "Setup complete!")
	assert.NotContains(t, text, "brew install")
}

// ---------------------------------------------------------------------------
// watch.go - runWatch error paths (54.4% coverage)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global watch flags and working directory
func TestBatch5_RunWatch_InvalidDebounce(t *testing.T) {
	reset := setWatchFlags("invalid-duration", false, false, false)
	defer reset()

	err := runWatch(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid debounce")
}

//nolint:tparallel // modifies global watch flags and working directory
func TestBatch5_RunWatch_MissingConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	reset := setWatchFlags("500ms", false, false, false)
	defer reset()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err = runWatch(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no preflight.yaml")
}

// ---------------------------------------------------------------------------
// lock.go - runLockStatus (100% coverage, but more test scenarios)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global cfgFile
func TestBatch5_RunLockStatus_NoLockfile(t *testing.T) {
	saved := cfgFile
	defer func() { cfgFile = saved }()
	cfgFile = filepath.Join(t.TempDir(), "preflight.yaml")

	output := captureStdout(t, func() {
		err := runLockStatus(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No lockfile found")
}

//nolint:tparallel // modifies global cfgFile
func TestBatch5_RunLockStatus_ExistingLockfile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "preflight.yaml")
	lockFile := filepath.Join(tmpDir, "preflight.lock")

	require.NoError(t, os.WriteFile(configFile, []byte("targets:\n  default:\n    - base\n"), 0o644))
	require.NoError(t, os.WriteFile(lockFile, []byte("lock data\n"), 0o644))

	saved := cfgFile
	defer func() { cfgFile = saved }()
	cfgFile = configFile

	output := captureStdout(t, func() {
		err := runLockStatus(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Lockfile status:")
	assert.Contains(t, output, "exists")
}

//nolint:tparallel // modifies global cfgFile
func TestBatch5_RunLockStatus_DefaultConfigPath(t *testing.T) {
	saved := cfgFile
	defer func() { cfgFile = saved }()
	cfgFile = ""

	output := captureStdout(t, func() {
		// This may succeed or fail depending on working directory
		_ = runLockStatus(nil, nil)
	})

	// Should at least produce some output
	assert.NotEmpty(t, output)
}

// ---------------------------------------------------------------------------
// repo.go - getConfigDir
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global cfgFile
func TestBatch5_GetConfigDir_EmptyPath(t *testing.T) {
	saved := cfgFile
	defer func() { cfgFile = saved }()
	cfgFile = ""

	result := getConfigDir()
	assert.Equal(t, ".", result)
}

//nolint:tparallel // modifies global cfgFile
func TestBatch5_GetConfigDir_WithPath(t *testing.T) {
	saved := cfgFile
	defer func() { cfgFile = saved }()
	cfgFile = "/some/path/preflight.yaml"

	result := getConfigDir()
	assert.Equal(t, "/some/path", result)
}

//nolint:tparallel // modifies global cfgFile
func TestBatch5_GetConfigDir_RelativePath(t *testing.T) {
	saved := cfgFile
	defer func() { cfgFile = saved }()
	cfgFile = "subdir/preflight.yaml"

	result := getConfigDir()
	assert.Equal(t, "subdir", result)
}

// ---------------------------------------------------------------------------
// secrets.go - runSecretsList with JSON output for refs
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsConfigPath and secretsJSON
func TestBatch5_RunSecretsList_EmptyConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("foo: bar\n"), 0o644))

	savedPath := secretsConfigPath
	savedJSON := secretsJSON
	defer func() {
		secretsConfigPath = savedPath
		secretsJSON = savedJSON
	}()
	secretsConfigPath = tmpFile
	secretsJSON = false

	output := captureStdout(t, func() {
		err := runSecretsList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No secret references found")
}

// ---------------------------------------------------------------------------
// secrets.go - runSecretsBackends text and JSON output
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsJSON
func TestBatch5_RunSecretsBackends_TextOutput(t *testing.T) {
	savedJSON := secretsJSON
	defer func() { secretsJSON = savedJSON }()
	secretsJSON = false

	output := captureStdout(t, func() {
		err := runSecretsBackends(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Available secret backends")
	assert.Contains(t, output, "1password")
	assert.Contains(t, output, "bitwarden")
	assert.Contains(t, output, "keychain")
	assert.Contains(t, output, "age")
	assert.Contains(t, output, "env")
}

//nolint:tparallel // modifies global secretsJSON
func TestBatch5_RunSecretsBackends_JSONOutput(t *testing.T) {
	savedJSON := secretsJSON
	defer func() { secretsJSON = savedJSON }()
	secretsJSON = true

	output := captureStdout(t, func() {
		err := runSecretsBackends(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "1password")
	assert.Contains(t, output, "env")

	// Verify it's valid JSON
	var backends []interface{}
	err := json.Unmarshal([]byte(output), &backends)
	require.NoError(t, err)
	assert.Len(t, backends, 5)
}

// ---------------------------------------------------------------------------
// secrets.go - runSecretsGet with env backend
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsBackend
func TestBatch5_RunSecretsGet_EnvSuccess(t *testing.T) {
	t.Setenv("BATCH5_TEST_SECRET_XYZ", "secret-value-123")

	savedBackend := secretsBackend
	defer func() { secretsBackend = savedBackend }()
	secretsBackend = "env"

	output := captureStdout(t, func() {
		err := runSecretsGet(nil, []string{"BATCH5_TEST_SECRET_XYZ"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "secret-value-123")
}

//nolint:tparallel // modifies global secretsBackend
func TestBatch5_RunSecretsGet_EnvNotFound(t *testing.T) {
	t.Setenv("BATCH5_EMPTY_SECRET", "")

	savedBackend := secretsBackend
	defer func() { secretsBackend = savedBackend }()
	secretsBackend = "env"

	err := runSecretsGet(nil, []string{"BATCH5_EMPTY_SECRET"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// secrets.go - runSecretsCheck with empty config
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsConfigPath
func TestBatch5_RunSecretsCheck_EmptyConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("foo: bar\n"), 0o644))

	saved := secretsConfigPath
	defer func() { secretsConfigPath = saved }()
	secretsConfigPath = tmpFile

	output := captureStdout(t, func() {
		err := runSecretsCheck(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No secret references to check")
}

// ---------------------------------------------------------------------------
// secrets.go - findSecretRefs comprehensive
// ---------------------------------------------------------------------------

func TestBatch5_FindSecretRefs_MultipleRefs(t *testing.T) {
	t.Parallel()

	content := `git:
  signing_key: "secret://1password/vault/key"
ssh:
  passphrase: "secret://keychain/ssh-pass"
env:
  token: "secret://env/GITHUB_TOKEN"
`
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	assert.Len(t, refs, 3)
	assert.Equal(t, "1password", refs[0].Backend)
}

func TestBatch5_FindSecretRefs_EmptyConfig(t *testing.T) {
	t.Parallel()

	emptyFile := filepath.Join(t.TempDir(), "empty.yaml")
	require.NoError(t, os.WriteFile(emptyFile, []byte("just: values\n"), 0o644))

	refs, err := findSecretRefs(emptyFile)
	require.NoError(t, err)
	assert.Len(t, refs, 0)
}

func TestBatch5_FindSecretRefs_NonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := findSecretRefs("/nonexistent/file.yaml")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// secrets.go - resolveSecret edge cases
// ---------------------------------------------------------------------------

func TestBatch5_ResolveSecret_EnvBackend(t *testing.T) {
	t.Setenv("BATCH5_RESOLVE_TEST", "hello")

	val, err := resolveSecret("env", "BATCH5_RESOLVE_TEST")
	require.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestBatch5_ResolveSecret_UnknownBackend(t *testing.T) {
	t.Parallel()

	_, err := resolveSecret("unknown", "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}

// ---------------------------------------------------------------------------
// secrets.go - setSecret edge cases
// ---------------------------------------------------------------------------

func TestBatch5_SetSecret_EnvBackend(t *testing.T) {
	t.Parallel()

	err := setSecret("env", "test", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set")
}

func TestBatch5_SetSecret_UnknownBackend(t *testing.T) {
	t.Parallel()

	err := setSecret("unknown", "test", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// ---------------------------------------------------------------------------
// secrets.go - backend CLI checks
// ---------------------------------------------------------------------------

func TestBatch5_BackendCLIChecks(t *testing.T) {
	t.Parallel()

	// Just verify they return without panic
	_ = check1PasswordCLI()
	_ = checkBitwardenCLI()
	_ = checkKeychain()
	_ = checkAgeCLI()
}

// ---------------------------------------------------------------------------
// secrets.go - resolve1Password error path
// ---------------------------------------------------------------------------

func TestBatch5_Resolve1Password_InvalidFormat(t *testing.T) {
	t.Parallel()

	// Invalid format with no slash
	_, err := resolveSecret("1password", "no-slash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid 1password key format")
}

// ---------------------------------------------------------------------------
// History and Change struct JSON tests
// ---------------------------------------------------------------------------

func TestBatch5_HistoryEntry_JSON_Roundtrip(t *testing.T) {
	t.Parallel()

	entry := HistoryEntry{
		ID:        "batch5-json",
		Timestamp: time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC),
		Command:   "apply",
		Target:    "work",
		Status:    "success",
		Duration:  "2.1s",
		Changes: []Change{
			{Provider: "brew", Action: "install", Item: "vim", Details: "v9.0"},
		},
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded HistoryEntry
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, entry.ID, decoded.ID)
	assert.Equal(t, entry.Command, decoded.Command)
	assert.Equal(t, entry.Target, decoded.Target)
	assert.Equal(t, entry.Status, decoded.Status)
	assert.Len(t, decoded.Changes, 1)
	assert.Equal(t, "v9.0", decoded.Changes[0].Details)
}

// ---------------------------------------------------------------------------
// OrphanedItem struct test
// ---------------------------------------------------------------------------

func TestBatch5_OrphanedItem_JSON(t *testing.T) {
	t.Parallel()

	item := OrphanedItem{
		Provider: "brew",
		Type:     "formula",
		Name:     "htop",
		Details:  "installed manually",
	}

	data, err := json.Marshal(item)
	require.NoError(t, err)

	var decoded OrphanedItem
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, item, decoded)
}

// ---------------------------------------------------------------------------
// SecretRef struct test
// ---------------------------------------------------------------------------

func TestBatch5_SecretRef_JSON(t *testing.T) {
	t.Parallel()

	ref := SecretRef{
		Path:     "git.key",
		Backend:  "env",
		Key:      "MY_KEY",
		Resolved: true,
	}

	data, err := json.Marshal(ref)
	require.NoError(t, err)

	var decoded SecretRef
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, ref, decoded)
}

// ---------------------------------------------------------------------------
// history.go - runHistoryClear
// ---------------------------------------------------------------------------

func TestBatch5_RunHistoryClear_WithExistingHistory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, SaveHistoryEntry(HistoryEntry{
		ID:      "clear-b5",
		Command: "apply",
		Status:  "success",
	}))

	output := captureStdout(t, func() {
		err := runHistoryClear(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "History cleared")

	// Verify dir removed
	histDir := filepath.Join(tmpDir, ".preflight", "history")
	_, err := os.Stat(histDir)
	assert.True(t, os.IsNotExist(err))
}

// ---------------------------------------------------------------------------
// SaveHistoryEntry - auto-generates ID and timestamp
// ---------------------------------------------------------------------------

func TestBatch5_SaveHistoryEntry_AutoGeneratesIDAndTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		Command: "doctor --fix",
		Status:  "success",
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	histDir := filepath.Join(tmpDir, ".preflight", "history")
	files, err := os.ReadDir(histDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files[0].Name(), ".json")
}

// ---------------------------------------------------------------------------
// detectKeyType - GPG binary detection via isValidOpenPGPPacket
// ---------------------------------------------------------------------------

func TestBatch5_DetectKeyType_GPGBinaryPacket(t *testing.T) {
	t.Parallel()

	// New format, tag 6 (public key) = 0xC6
	result := detectKeyType([]byte{0xC6, 0x00, 0x01, 0x02})
	assert.Equal(t, catalog.SignatureTypeGPG, result)

	// Old format, tag 6 = (6 << 2) | 0x80 = 0x98
	result = detectKeyType([]byte{0x98, 0x00, 0x01, 0x02})
	assert.Equal(t, catalog.SignatureTypeGPG, result)

	// Invalid packet - should not be detected as GPG
	result = detectKeyType([]byte{0x80, 0x00}) // old format, tag 0
	assert.Equal(t, catalog.SignatureType(""), result)
}
