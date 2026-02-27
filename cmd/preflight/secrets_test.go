package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// SecretRef struct tests
// ---------------------------------------------------------------------------

func TestSecretRef_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	ref := SecretRef{
		Path:     "git.signing_key",
		Backend:  "1password",
		Key:      "GitHub/signing-key",
		Resolved: true,
	}

	data, err := json.Marshal(ref)
	require.NoError(t, err)

	var decoded SecretRef
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ref, decoded)
}

func TestSecretRef_JSONFields(t *testing.T) {
	t.Parallel()

	ref := SecretRef{
		Path:     "ssh.passphrase",
		Backend:  "keychain",
		Key:      "ssh-work",
		Resolved: false,
	}

	data, err := json.Marshal(ref)
	require.NoError(t, err)

	raw := string(data)
	assert.Contains(t, raw, `"path"`)
	assert.Contains(t, raw, `"backend"`)
	assert.Contains(t, raw, `"key"`)
	assert.Contains(t, raw, `"resolved"`)
}

func TestSecretRef_ZeroValue(t *testing.T) {
	t.Parallel()

	var ref SecretRef
	assert.Empty(t, ref.Path)
	assert.Empty(t, ref.Backend)
	assert.Empty(t, ref.Key)
	assert.False(t, ref.Resolved)
}

// ---------------------------------------------------------------------------
// findSecretRefs tests
// ---------------------------------------------------------------------------

func TestFindSecretRefs_SingleRef(t *testing.T) {
	t.Parallel()

	content := `git:
  signing_key: "secret://1password/GitHub/signing-key"
`
	tmpFile := writeTemp(t, content)

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	require.Len(t, refs, 1)

	assert.Equal(t, "signing_key", refs[0].Path)
	assert.Equal(t, "1password", refs[0].Backend)
	assert.Equal(t, "GitHub/signing-key", refs[0].Key)
	assert.False(t, refs[0].Resolved)
}

func TestFindSecretRefs_MultipleRefs(t *testing.T) {
	t.Parallel()

	content := `git:
  signing_key: "secret://1password/vault/signing-key"
ssh:
  passphrase: "secret://keychain/ssh-work-passphrase"
env:
  API_TOKEN: "secret://env/MY_API_TOKEN"
`
	tmpFile := writeTemp(t, content)

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	require.Len(t, refs, 3)

	backends := make(map[string]string)
	for _, ref := range refs {
		backends[ref.Backend] = ref.Key
	}

	assert.Equal(t, "vault/signing-key", backends["1password"])
	assert.Equal(t, "ssh-work-passphrase", backends["keychain"])
	assert.Equal(t, "MY_API_TOKEN", backends["env"])
}

func TestFindSecretRefs_NoSecretsInConfig(t *testing.T) {
	t.Parallel()

	content := `git:
  name: "John Doe"
  email: "john@example.com"
brew:
  formulae:
    - git
    - curl
`
	tmpFile := writeTemp(t, content)

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindSecretRefs_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpFile := writeTemp(t, "")

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindSecretRefs_NonexistentFile(t *testing.T) {
	t.Parallel()

	_, err := findSecretRefs("/nonexistent/path/preflight.yaml")
	require.Error(t, err)
}

func TestFindSecretRefs_UnquotedRef(t *testing.T) {
	t.Parallel()

	content := `git:
  signing_key: secret://env/GIT_SIGNING_KEY
`
	tmpFile := writeTemp(t, content)

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	require.Len(t, refs, 1)

	assert.Equal(t, "env", refs[0].Backend)
	assert.Equal(t, "GIT_SIGNING_KEY", refs[0].Key)
}

func TestFindSecretRefs_SingleQuotedRef(t *testing.T) {
	t.Parallel()

	content := `ssh:
  passphrase: 'secret://keychain/my-pass'
`
	tmpFile := writeTemp(t, content)

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	require.Len(t, refs, 1)

	assert.Equal(t, "keychain", refs[0].Backend)
	assert.Equal(t, "my-pass", refs[0].Key)
}

func TestFindSecretRefs_MultipleBackendTypes(t *testing.T) {
	t.Parallel()

	content := `secrets:
  op_key: "secret://1password/vault/item"
  bw_key: "secret://bitwarden/my-password"
  kc_key: "secret://keychain/kc-item"
  age_key: "secret://age/encrypted-secret"
  env_key: "secret://env/ENV_VAR"
`
	tmpFile := writeTemp(t, content)

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	require.Len(t, refs, 5)

	backendSet := make(map[string]bool)
	for _, ref := range refs {
		backendSet[ref.Backend] = true
	}

	assert.True(t, backendSet["1password"])
	assert.True(t, backendSet["bitwarden"])
	assert.True(t, backendSet["keychain"])
	assert.True(t, backendSet["age"])
	assert.True(t, backendSet["env"])
}

func TestFindSecretRefs_PathExtraction(t *testing.T) {
	t.Parallel()

	content := `  nested_key: "secret://env/VALUE"
`
	tmpFile := writeTemp(t, content)

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	require.Len(t, refs, 1)

	assert.Equal(t, "nested_key", refs[0].Path)
}

func TestFindSecretRefs_MultiSlashKey(t *testing.T) {
	t.Parallel()

	content := `key: "secret://1password/vault/item/field"
`
	tmpFile := writeTemp(t, content)

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	require.Len(t, refs, 1)

	// SplitN with n=2 means the key gets everything after the first slash
	assert.Equal(t, "1password", refs[0].Backend)
	assert.Equal(t, "vault/item/field", refs[0].Key)
}

// ---------------------------------------------------------------------------
// resolveSecret tests
// ---------------------------------------------------------------------------

func TestResolveSecret_EnvBackend_Found(t *testing.T) {
	t.Setenv("PREFLIGHT_TEST_SECRET", "super-secret-value")

	val, err := resolveSecret("env", "PREFLIGHT_TEST_SECRET")
	require.NoError(t, err)
	assert.Equal(t, "super-secret-value", val)
}

func TestResolveSecret_EnvBackend_NotSet(t *testing.T) {
	t.Setenv("PREFLIGHT_TEST_MISSING", "")

	val, err := resolveSecret("env", "PREFLIGHT_TEST_MISSING")
	require.NoError(t, err)
	assert.Empty(t, val)
}

func TestResolveSecret_EnvBackend_Unset(t *testing.T) {
	// os.Getenv returns "" for unset variables, same as empty
	val, err := resolveSecret("env", "PREFLIGHT_DEFINITELY_NOT_SET_VAR_XYZ_123")
	require.NoError(t, err)
	assert.Empty(t, val)
}

func TestResolveSecret_UnknownBackendError(t *testing.T) {
	t.Parallel()

	_, err := resolveSecret("nonexistent", "some-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestResolveSecret_UnknownBackend_TableDriven(t *testing.T) {
	t.Parallel()

	backends := []string{"vault", "aws-ssm", "gcp-secret", "azure-keyvault", ""}

	for _, backend := range backends {
		t.Run("backend_"+backend, func(t *testing.T) {
			t.Parallel()
			_, err := resolveSecret(backend, "key")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown backend")
		})
	}
}

// ---------------------------------------------------------------------------
// setSecret tests
// ---------------------------------------------------------------------------

func TestSetSecret_EnvBackend_ReturnsError(t *testing.T) {
	t.Parallel()

	err := setSecret("env", "my-secret", "my-value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set environment variables")
}

func TestSetSecret_UnknownBackend_ReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		backend string
	}{
		{"vault_backend", "vault"},
		{"aws_ssm", "aws-ssm"},
		{"1password", "1password"},
		{"bitwarden", "bitwarden"},
		{"age", "age"},
		{"empty_backend", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := setSecret(tt.backend, "name", "value")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not supported")
		})
	}
}

// ---------------------------------------------------------------------------
// runSecretsList tests
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsConfigPath and secretsJSON
func TestRunSecretsList_TextOutput(t *testing.T) {
	content := `git:
  signing_key: "secret://1password/vault/key"
ssh:
  passphrase: "secret://keychain/ssh-pass"
`
	tmpFile := writeTemp(t, content)

	savedConfigPath := secretsConfigPath
	savedJSON := secretsJSON
	secretsConfigPath = tmpFile
	secretsJSON = false
	defer func() {
		secretsConfigPath = savedConfigPath
		secretsJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := runSecretsList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Found 2 secret reference(s)")
	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "BACKEND")
	assert.Contains(t, output, "KEY")
	assert.Contains(t, output, "1password")
	assert.Contains(t, output, "keychain")
}

//nolint:tparallel // modifies global secretsConfigPath and secretsJSON
func TestRunSecretsList_JSONOutput(t *testing.T) {
	content := `git:
  signing_key: "secret://env/GIT_KEY"
`
	tmpFile := writeTemp(t, content)

	savedConfigPath := secretsConfigPath
	savedJSON := secretsJSON
	secretsConfigPath = tmpFile
	secretsJSON = true
	defer func() {
		secretsConfigPath = savedConfigPath
		secretsJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := runSecretsList(nil, nil)
		require.NoError(t, err)
	})

	var refs []SecretRef
	err := json.Unmarshal([]byte(output), &refs)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "env", refs[0].Backend)
	assert.Equal(t, "GIT_KEY", refs[0].Key)
}

//nolint:tparallel // modifies global secretsConfigPath
func TestRunSecretsList_NoSecrets(t *testing.T) {
	content := `git:
  name: "John Doe"
`
	tmpFile := writeTemp(t, content)

	savedConfigPath := secretsConfigPath
	secretsConfigPath = tmpFile
	defer func() {
		secretsConfigPath = savedConfigPath
	}()

	output := captureStdout(t, func() {
		err := runSecretsList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No secret references found")
}

//nolint:tparallel // modifies global secretsConfigPath
func TestRunSecretsList_FileNotFound(t *testing.T) {
	savedConfigPath := secretsConfigPath
	secretsConfigPath = "/nonexistent/path/preflight.yaml"
	defer func() {
		secretsConfigPath = savedConfigPath
	}()

	err := runSecretsList(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find secrets")
}

// ---------------------------------------------------------------------------
// runSecretsGet tests
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsBackend
func TestRunSecretsGet_EnvBackend_Success(t *testing.T) {
	t.Setenv("PREFLIGHT_GET_TEST", "fetched-value")

	savedBackend := secretsBackend
	secretsBackend = "env"
	defer func() {
		secretsBackend = savedBackend
	}()

	output := captureStdout(t, func() {
		err := runSecretsGet(nil, []string{"PREFLIGHT_GET_TEST"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "fetched-value")
}

//nolint:tparallel // modifies global secretsBackend
func TestRunSecretsGet_EnvBackend_NotFound(t *testing.T) {
	// Ensure the variable is truly empty
	t.Setenv("PREFLIGHT_MISSING_GET", "")

	savedBackend := secretsBackend
	secretsBackend = "env"
	defer func() {
		secretsBackend = savedBackend
	}()

	err := runSecretsGet(nil, []string{"PREFLIGHT_MISSING_GET"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

//nolint:tparallel // modifies global secretsBackend
func TestRunSecretsGet_UnknownBackend(t *testing.T) {
	savedBackend := secretsBackend
	secretsBackend = "nonexistent-backend"
	defer func() {
		secretsBackend = savedBackend
	}()

	err := runSecretsGet(nil, []string{"any-key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret")
}

//nolint:tparallel // modifies global secretsBackend
func TestRunSecretsGet_DefaultsToKeychain(t *testing.T) {
	// When secretsBackend is empty, it defaults to "keychain".
	// On CI or machines without macOS keychain, resolveKeychain will fail,
	// but we verify the error references keychain behavior (not "unknown backend").
	savedBackend := secretsBackend
	secretsBackend = ""
	defer func() {
		secretsBackend = savedBackend
	}()

	err := runSecretsGet(nil, []string{"nonexistent-secret"})
	// Should either succeed (unlikely) or fail with a keychain-related error,
	// but NOT with "unknown backend".
	if err != nil {
		assert.NotContains(t, err.Error(), "unknown backend")
	}
}

// ---------------------------------------------------------------------------
// runSecretsBackends tests
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsJSON
func TestRunSecretsBackends_TextOutput(t *testing.T) {
	savedJSON := secretsJSON
	secretsJSON = false
	defer func() {
		secretsJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := runSecretsBackends(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Available secret backends")
	assert.Contains(t, output, "BACKEND")
	assert.Contains(t, output, "DESCRIPTION")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "COMMAND")

	// All backend names should appear
	assert.Contains(t, output, "1password")
	assert.Contains(t, output, "bitwarden")
	assert.Contains(t, output, "keychain")
	assert.Contains(t, output, "age")
	assert.Contains(t, output, "env")

	// env is always available
	assert.Contains(t, output, "Environment variables")
}

//nolint:tparallel // modifies global secretsJSON
func TestRunSecretsBackends_JSONOutput(t *testing.T) {
	savedJSON := secretsJSON
	secretsJSON = true
	defer func() {
		secretsJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := runSecretsBackends(nil, nil)
		require.NoError(t, err)
	})

	var backends []struct {
		Name        string `json:"Name"`
		Description string `json:"Description"`
		Available   bool   `json:"Available"`
		Command     string `json:"Command"`
	}
	err := json.Unmarshal([]byte(output), &backends)
	require.NoError(t, err)
	require.Len(t, backends, 5)

	// Build lookup
	byName := make(map[string]struct {
		Description string
		Available   bool
		Command     string
	})
	for _, b := range backends {
		byName[b.Name] = struct {
			Description string
			Available   bool
			Command     string
		}{b.Description, b.Available, b.Command}
	}

	// env backend is always available and has no command
	envBackend, ok := byName["env"]
	require.True(t, ok, "env backend should be present")
	assert.True(t, envBackend.Available)
	assert.Empty(t, envBackend.Command)
	assert.Equal(t, "Environment variables", envBackend.Description)

	// 1password uses "op" command
	opBackend, ok := byName["1password"]
	require.True(t, ok)
	assert.Equal(t, "op", opBackend.Command)

	// bitwarden uses "bw" command
	bwBackend, ok := byName["bitwarden"]
	require.True(t, ok)
	assert.Equal(t, "bw", bwBackend.Command)

	// keychain uses "security" command
	kcBackend, ok := byName["keychain"]
	require.True(t, ok)
	assert.Equal(t, "security", kcBackend.Command)

	// age uses "age" command
	ageBackend, ok := byName["age"]
	require.True(t, ok)
	assert.Equal(t, "age", ageBackend.Command)
}

// ---------------------------------------------------------------------------
// runSecretsCheck tests
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsConfigPath
func TestRunSecretsCheck_NoSecrets(t *testing.T) {
	content := `brew:
  formulae:
    - git
`
	tmpFile := writeTemp(t, content)

	savedConfigPath := secretsConfigPath
	secretsConfigPath = tmpFile
	defer func() {
		secretsConfigPath = savedConfigPath
	}()

	output := captureStdout(t, func() {
		err := runSecretsCheck(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No secret references to check")
}

//nolint:tparallel // modifies global secretsConfigPath
func TestRunSecretsCheck_FileNotFound(t *testing.T) {
	savedConfigPath := secretsConfigPath
	secretsConfigPath = "/nonexistent/path/preflight.yaml"
	defer func() {
		secretsConfigPath = savedConfigPath
	}()

	err := runSecretsCheck(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find secrets")
}

// ---------------------------------------------------------------------------
// Command registration tests
// ---------------------------------------------------------------------------

func TestSecretsCmd_UseAndShort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "secrets", secretsCmd.Use)
	assert.NotEmpty(t, secretsCmd.Short)
	assert.NotEmpty(t, secretsCmd.Long)
}

func TestSecretsListCmd_UseAndShort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "list", secretsListCmd.Use)
	assert.NotEmpty(t, secretsListCmd.Short)
}

func TestSecretsCheckCmd_UseAndShort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "check", secretsCheckCmd.Use)
	assert.NotEmpty(t, secretsCheckCmd.Short)
}

func TestSecretsSetCmd_UseAndShort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "set <name>", secretsSetCmd.Use)
	assert.NotEmpty(t, secretsSetCmd.Short)
}

func TestSecretsGetCmd_UseAndShort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "get <name>", secretsGetCmd.Use)
	assert.NotEmpty(t, secretsGetCmd.Short)
}

func TestSecretsBackendsCmd_UseAndShort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "backends", secretsBackendsCmd.Use)
	assert.NotEmpty(t, secretsBackendsCmd.Short)
}

func TestSecretsCmd_AllSubcommands(t *testing.T) {
	t.Parallel()

	subCmds := secretsCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subCmds {
		names[cmd.Name()] = true
	}

	assert.True(t, names["list"], "should have list subcommand")
	assert.True(t, names["check"], "should have check subcommand")
	assert.True(t, names["set"], "should have set subcommand")
	assert.True(t, names["get"], "should have get subcommand")
	assert.True(t, names["backends"], "should have backends subcommand")
}

func TestSecretsCmd_PersistentFlags(t *testing.T) {
	t.Parallel()

	configFlag := secretsCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag)
	assert.Equal(t, "c", configFlag.Shorthand)
	assert.Equal(t, "preflight.yaml", configFlag.DefValue)

	backendFlag := secretsCmd.PersistentFlags().Lookup("backend")
	require.NotNil(t, backendFlag)
	assert.Empty(t, backendFlag.DefValue)

	jsonFlag := secretsCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)
}

func TestSecretsCmd_DefaultRunEIsSecretsList(t *testing.T) {
	t.Parallel()

	// The root secrets command has RunE set to runSecretsList
	assert.NotNil(t, secretsCmd.RunE)
}

func TestSecretsSetCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	// cobra.ExactArgs(1) is set; verify by checking Args is not nil
	assert.NotNil(t, secretsSetCmd.Args)

	cmd := &cobra.Command{
		Args: secretsSetCmd.Args,
	}
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.NoError(t, cmd.Args(cmd, []string{"one"}))
	assert.Error(t, cmd.Args(cmd, []string{"one", "two"}))
}

func TestSecretsGetCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, secretsGetCmd.Args)

	cmd := &cobra.Command{
		Args: secretsGetCmd.Args,
	}
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.NoError(t, cmd.Args(cmd, []string{"one"}))
	assert.Error(t, cmd.Args(cmd, []string{"one", "two"}))
}

// ---------------------------------------------------------------------------
// Backend availability check tests
// ---------------------------------------------------------------------------

func TestCheck1PasswordCLI_ReturnsBool(t *testing.T) {
	t.Parallel()

	// We simply verify it returns a boolean without panicking.
	// The result depends on whether "op" is installed.
	result := check1PasswordCLI()
	assert.IsType(t, true, result)
}

func TestCheckBitwardenCLI_ReturnsBool(t *testing.T) {
	t.Parallel()

	result := checkBitwardenCLI()
	assert.IsType(t, true, result)
}

func TestCheckKeychain_ReturnsBool(t *testing.T) {
	t.Parallel()

	result := checkKeychain()
	assert.IsType(t, true, result)
}

func TestCheckAgeCLI_ReturnsBool(t *testing.T) {
	t.Parallel()

	result := checkAgeCLI()
	assert.IsType(t, true, result)
}

// ---------------------------------------------------------------------------
// findSecretRefs edge cases (table-driven)
// ---------------------------------------------------------------------------

func TestFindSecretRefs_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		expectedLen  int
		expectBacken string // expected backend of first ref (if any)
		expectKey    string // expected key of first ref (if any)
	}{
		{
			name:        "no_secret_prefix",
			content:     "key: value\n",
			expectedLen: 0,
		},
		{
			name:         "double_quoted",
			content:      `key: "secret://env/MY_VAR"` + "\n",
			expectedLen:  1,
			expectBacken: "env",
			expectKey:    "MY_VAR",
		},
		{
			name:         "single_quoted",
			content:      `key: 'secret://keychain/pass'` + "\n",
			expectedLen:  1,
			expectBacken: "keychain",
			expectKey:    "pass",
		},
		{
			name:         "unquoted",
			content:      "key: secret://bitwarden/login\n",
			expectedLen:  1,
			expectBacken: "bitwarden",
			expectKey:    "login",
		},
		{
			name:         "tab_terminated",
			content:      "key: secret://age/mysecret\t# comment\n",
			expectedLen:  1,
			expectBacken: "age",
			expectKey:    "mysecret",
		},
		{
			name:         "space_terminated",
			content:      "key: secret://env/X some more text\n",
			expectedLen:  1,
			expectBacken: "env",
			expectKey:    "X",
		},
		{
			name: "multiple_on_same_line_only_first",
			// findSecretRefs uses strings.Index which finds only the first occurrence per line
			content:     `key: "secret://env/A" other: "secret://env/B"` + "\n",
			expectedLen: 1,
		},
		{
			name: "multiple_lines",
			content: `a: "secret://env/X"
b: "secret://env/Y"
c: "secret://env/Z"
`,
			expectedLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpFile := writeTemp(t, tt.content)
			refs, err := findSecretRefs(tmpFile)
			require.NoError(t, err)
			require.Len(t, refs, tt.expectedLen)

			if tt.expectedLen > 0 && tt.expectBacken != "" {
				assert.Equal(t, tt.expectBacken, refs[0].Backend)
				assert.Equal(t, tt.expectKey, refs[0].Key)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveSecret dispatching table-driven
// ---------------------------------------------------------------------------

func TestResolveSecret_KnownBackends_DoNotReturnUnknown(t *testing.T) {
	t.Parallel()

	// These backends are known and should never return "unknown backend" error.
	// They may fail for other reasons (CLI not installed, etc).
	knownBackends := []string{"1password", "bitwarden", "keychain", "age", "env"}

	for _, backend := range knownBackends {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			_, err := resolveSecret(backend, "test-key")
			if err != nil {
				assert.NotContains(t, err.Error(), "unknown backend")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setSecret dispatching table-driven
// ---------------------------------------------------------------------------

func TestSetSecret_KeychainBackend_IsAccepted(t *testing.T) {
	t.Parallel()

	// setSecret("keychain", ...) calls setKeychainSecret which invokes the
	// "security" command. It should not return "not supported" error.
	err := setSecret("keychain", "test-name", "test-value")
	if err != nil {
		// If it fails, it should be a system error, not "not supported"
		assert.NotContains(t, err.Error(), "not supported")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeTemp creates a temporary file with the given content and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "preflight.yaml")
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	return path
}
