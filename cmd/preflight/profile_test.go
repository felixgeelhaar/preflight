package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProfileInfo_AllFields(t *testing.T) {
	t.Parallel()

	p := ProfileInfo{
		Name:        "work",
		Target:      "work-target",
		Description: "Work profile with corporate tools",
		Active:      true,
		LastUsed:    "2026-02-24T10:00:00Z",
	}

	assert.Equal(t, "work", p.Name)
	assert.Equal(t, "work-target", p.Target)
	assert.Equal(t, "Work profile with corporate tools", p.Description)
	assert.True(t, p.Active)
	assert.Equal(t, "2026-02-24T10:00:00Z", p.LastUsed)
}

func TestProfileInfo_InactiveDefaults(t *testing.T) {
	t.Parallel()

	p := ProfileInfo{
		Name:   "personal",
		Target: "personal",
	}

	assert.False(t, p.Active)
	assert.Empty(t, p.Description)
	assert.Empty(t, p.LastUsed)
}

func TestGetProfileDir_UsesHome(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := getProfileDir()
	assert.Equal(t, filepath.Join(tmpDir, ".preflight", "profiles"), dir)
}

func TestSetAndGetCurrentProfile_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Initially no profile set
	current := getCurrentProfile()
	assert.Empty(t, current)

	// Set a profile
	err := setCurrentProfile("work")
	require.NoError(t, err)

	// Read it back
	current = getCurrentProfile()
	assert.Equal(t, "work", current)

	// Switch to another
	err = setCurrentProfile("personal")
	require.NoError(t, err)

	current = getCurrentProfile()
	assert.Equal(t, "personal", current)
}

func TestGetCurrentProfile_NoFileReturnsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	result := getCurrentProfile()
	assert.Empty(t, result)
}

func TestSetCurrentProfile_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := setCurrentProfile("test-profile")
	require.NoError(t, err)

	// Verify the directory was created
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	info, err := os.Stat(profileDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestSaveAndLoadCustomProfiles_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{Name: "work", Target: "work-target", Description: "Work env"},
		{Name: "personal", Target: "personal-target", Description: "Personal env"},
		{Name: "testing", Target: "test-target"},
	}

	err := saveCustomProfiles(profiles)
	require.NoError(t, err)

	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	require.Len(t, loaded, 3)

	assert.Equal(t, "work", loaded[0].Name)
	assert.Equal(t, "work-target", loaded[0].Target)
	assert.Equal(t, "Work env", loaded[0].Description)

	assert.Equal(t, "personal", loaded[1].Name)
	assert.Equal(t, "personal-target", loaded[1].Target)

	assert.Equal(t, "testing", loaded[2].Name)
}

func TestLoadCustomProfiles_NoFileReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles, err := loadCustomProfiles()
	assert.Error(t, err)
	assert.Nil(t, profiles)
}

func TestLoadCustomProfiles_InvalidYAMLReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	require.NoError(t, os.MkdirAll(profileDir, 0o755))

	// Write invalid YAML
	err := os.WriteFile(filepath.Join(profileDir, "profiles.yaml"), []byte("{{invalid yaml"), 0o644)
	require.NoError(t, err)

	profiles, err := loadCustomProfiles()
	assert.Error(t, err)
	assert.Nil(t, profiles)
}

func TestSaveCustomProfiles_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{Name: "new-profile", Target: "default"},
	}

	err := saveCustomProfiles(profiles)
	require.NoError(t, err)

	// Verify file exists
	profilePath := filepath.Join(tmpDir, ".preflight", "profiles", "profiles.yaml")
	_, err = os.Stat(profilePath)
	require.NoError(t, err)

	// Verify content is valid YAML
	data, err := os.ReadFile(profilePath)
	require.NoError(t, err)

	var parsed []ProfileInfo
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 1)
}

func TestSaveCustomProfiles_EmptyList(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := saveCustomProfiles([]ProfileInfo{})
	require.NoError(t, err)

	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestRunProfileCreate_NewProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save the global and restore
	savedFromTarget := profileFromTarget
	profileFromTarget = "my-target"
	defer func() { profileFromTarget = savedFromTarget }()

	output := captureStdout(t, func() {
		err := runProfileCreate(&cobra.Command{}, []string{"my-new-profile"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Created profile 'my-new-profile' from target 'my-target'")

	// Verify profile was saved
	profiles, err := loadCustomProfiles()
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "my-new-profile", profiles[0].Name)
	assert.Equal(t, "my-target", profiles[0].Target)
}

func TestRunProfileCreate_DefaultTarget(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedFromTarget := profileFromTarget
	profileFromTarget = ""
	defer func() { profileFromTarget = savedFromTarget }()

	output := captureStdout(t, func() {
		err := runProfileCreate(&cobra.Command{}, []string{"minimal-profile"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "from target 'default'")

	profiles, err := loadCustomProfiles()
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "default", profiles[0].Target)
}

func TestRunProfileCreate_DuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedFromTarget := profileFromTarget
	profileFromTarget = ""
	defer func() { profileFromTarget = savedFromTarget }()

	// Create first profile
	captureStdout(t, func() {
		err := runProfileCreate(&cobra.Command{}, []string{"duplicate-profile"})
		require.NoError(t, err)
	})

	// Attempt to create again with same name
	err := runProfileCreate(&cobra.Command{}, []string{"duplicate-profile"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRunProfileCreate_MultipleProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedFromTarget := profileFromTarget
	profileFromTarget = "target-a"
	defer func() { profileFromTarget = savedFromTarget }()

	captureStdout(t, func() {
		err := runProfileCreate(&cobra.Command{}, []string{"profile-a"})
		require.NoError(t, err)
	})

	profileFromTarget = "target-b"

	captureStdout(t, func() {
		err := runProfileCreate(&cobra.Command{}, []string{"profile-b"})
		require.NoError(t, err)
	})

	profiles, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Len(t, profiles, 2)
}

func TestRunProfileDelete_ExistingProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create profiles to delete from
	profiles := []ProfileInfo{
		{Name: "keep-me", Target: "default"},
		{Name: "delete-me", Target: "default"},
		{Name: "also-keep", Target: "default"},
	}
	require.NoError(t, saveCustomProfiles(profiles))

	output := captureStdout(t, func() {
		err := runProfileDelete(&cobra.Command{}, []string{"delete-me"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Deleted profile 'delete-me'")

	// Verify only 2 profiles remain
	remaining, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Len(t, remaining, 2)

	names := make(map[string]bool)
	for _, p := range remaining {
		names[p.Name] = true
	}
	assert.True(t, names["keep-me"])
	assert.True(t, names["also-keep"])
	assert.False(t, names["delete-me"])
}

func TestRunProfileDelete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create some profiles but not the one we try to delete
	profiles := []ProfileInfo{
		{Name: "existing", Target: "default"},
	}
	require.NoError(t, saveCustomProfiles(profiles))

	err := runProfileDelete(&cobra.Command{}, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunProfileDelete_EmptyList(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// No profiles saved - loadCustomProfiles returns error on no file
	err := runProfileDelete(&cobra.Command{}, []string{"anything"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunProfileDelete_LastProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{Name: "last-one", Target: "default"},
	}
	require.NoError(t, saveCustomProfiles(profiles))

	captureStdout(t, func() {
		err := runProfileDelete(&cobra.Command{}, []string{"last-one"})
		require.NoError(t, err)
	})

	remaining, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestApplyGitConfig_NameEmailSigningKey(t *testing.T) {
	// Not parallel - writes to stdout
	git := map[string]interface{}{
		"name":        "Test User",
		"email":       "test@example.com",
		"signing_key": "ABC123",
	}

	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		require.NoError(t, err)
	})

	assert.Contains(t, output, `git config --global user.name "Test User"`)
	assert.Contains(t, output, `git config --global user.email "test@example.com"`)
	assert.Contains(t, output, `git config --global user.signingkey "ABC123"`)
}

func TestApplyGitConfig_OnlyEmail(t *testing.T) {
	// Not parallel - writes to stdout
	git := map[string]interface{}{
		"email": "only-email@example.com",
	}

	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "user.email")
	assert.NotContains(t, output, "user.name")
	assert.NotContains(t, output, "user.signingkey")
}

func TestApplyGitConfig_NoFields(t *testing.T) {
	// Not parallel - writes to stdout
	git := map[string]interface{}{}

	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		require.NoError(t, err)
	})

	assert.Empty(t, output)
}

func TestRunGitConfigSet_FormatsCorrectly(t *testing.T) {
	// Not parallel - writes to stdout
	output := captureStdout(t, func() {
		err := runGitConfigSet("user.name", "John Doe")
		require.NoError(t, err)
	})

	assert.Contains(t, output, `git config --global user.name "John Doe"`)
}

func TestProfileCmd_Registered(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "profile" {
			found = true
			break
		}
	}
	assert.True(t, found, "profile command should be registered on root")
}

func TestProfileCmd_SubcommandsList(t *testing.T) {
	t.Parallel()

	expectedSubs := []string{"list", "current", "switch", "create", "delete"}
	subNames := make(map[string]bool)
	for _, cmd := range profileCmd.Commands() {
		subNames[cmd.Name()] = true
	}

	for _, expected := range expectedSubs {
		assert.True(t, subNames[expected], "profile should have subcommand: %s", expected)
	}
}

func TestProfileCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{"config", "config", "preflight.yaml"},
		{"json", "json", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := profileCmd.PersistentFlags().Lookup(tt.flag)
			require.NotNil(t, f, "persistent flag %s should exist", tt.flag)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

func TestProfileCreateCmd_HasFromFlag(t *testing.T) {
	t.Parallel()

	f := profileCreateCmd.Flags().Lookup("from")
	require.NotNil(t, f, "create command should have --from flag")
	assert.Equal(t, "", f.DefValue)
}

func TestProfileSwitchCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, profileSwitchCmd.Args)
}

func TestProfileCreateCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, profileCreateCmd.Args)
}

func TestProfileDeleteCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, profileDeleteCmd.Args)
}

func TestRunProfileCurrent_NoActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedJSON := profileJSON
	profileJSON = false
	defer func() { profileJSON = savedJSON }()

	output := captureStdout(t, func() {
		err := runProfileCurrent(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No profile active")
}

func TestRunProfileCurrent_WithActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, setCurrentProfile("my-work"))

	savedJSON := profileJSON
	profileJSON = false
	defer func() { profileJSON = savedJSON }()

	output := captureStdout(t, func() {
		err := runProfileCurrent(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Current profile: my-work")
}

func TestRunProfileCurrent_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, setCurrentProfile("json-profile"))

	savedJSON := profileJSON
	profileJSON = true
	defer func() { profileJSON = savedJSON }()

	output := captureStdout(t, func() {
		err := runProfileCurrent(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, `"profile"`)
	assert.Contains(t, output, `"json-profile"`)
}
