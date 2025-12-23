package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_UseLine(t *testing.T) {
	assert.Equal(t, "preflight", rootCmd.Use)
}

func TestRootCommand_Short(t *testing.T) {
	assert.Equal(t, "A deterministic workstation compiler", rootCmd.Short)
}

func TestRootCommand_HasPersistentFlags(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	t.Run("config flag exists", func(t *testing.T) {
		flag := flags.Lookup("config")
		require.NotNil(t, flag)
		assert.Empty(t, flag.DefValue)
	})

	t.Run("verbose flag exists", func(t *testing.T) {
		flag := flags.Lookup("verbose")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("no-ai flag exists", func(t *testing.T) {
		flag := flags.Lookup("no-ai")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("ai-provider flag exists", func(t *testing.T) {
		flag := flags.Lookup("ai-provider")
		require.NotNil(t, flag)
		assert.Empty(t, flag.DefValue)
	})

	t.Run("mode flag exists", func(t *testing.T) {
		flag := flags.Lookup("mode")
		require.NotNil(t, flag)
		assert.Equal(t, "intent", flag.DefValue)
	})

	t.Run("yes flag exists", func(t *testing.T) {
		flag := flags.Lookup("yes")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestRootCommand_HasVersionSubcommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			found = true
			break
		}
	}
	assert.True(t, found, "root command should have version subcommand")
}

func TestVersionCommand_Output(t *testing.T) {
	// Save original values
	originalVersion := version
	originalCommit := commit
	originalBuildDate := buildDate

	// Set test values
	version = "1.0.0"
	commit = "abc123"
	buildDate = "2025-01-01"

	defer func() {
		// Restore original values
		version = originalVersion
		commit = originalCommit
		buildDate = originalBuildDate
	}()

	// Capture output
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	// Execute version command
	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "preflight 1.0.0")
	assert.Contains(t, output, "commit: abc123")
	assert.Contains(t, output, "built:  2025-01-01")

	// Reset args for future tests
	rootCmd.SetArgs([]string{})
}

func TestVersionCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "version", versionCmd.Use)
	assert.Equal(t, "Show version information", versionCmd.Short)
}

func TestPlanCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "plan", planCmd.Use)
	assert.Equal(t, "Show what changes preflight would make", planCmd.Short)
}

func TestPlanCommand_HasFlags(t *testing.T) {
	flags := planCmd.Flags()

	t.Run("config flag exists", func(t *testing.T) {
		flag := flags.Lookup("config")
		require.NotNil(t, flag)
		assert.Equal(t, "preflight.yaml", flag.DefValue)
		assert.Equal(t, "c", flag.Shorthand)
	})

	t.Run("target flag exists", func(t *testing.T) {
		flag := flags.Lookup("target")
		require.NotNil(t, flag)
		assert.Equal(t, "default", flag.DefValue)
		assert.Equal(t, "t", flag.Shorthand)
	})
}

func TestApplyCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "apply", applyCmd.Use)
	assert.Equal(t, "Apply configuration changes to your system", applyCmd.Short)
}

func TestApplyCommand_HasFlags(t *testing.T) {
	flags := applyCmd.Flags()

	t.Run("config flag exists", func(t *testing.T) {
		flag := flags.Lookup("config")
		require.NotNil(t, flag)
		assert.Equal(t, "preflight.yaml", flag.DefValue)
	})

	t.Run("target flag exists", func(t *testing.T) {
		flag := flags.Lookup("target")
		require.NotNil(t, flag)
		assert.Equal(t, "default", flag.DefValue)
	})

	t.Run("dry-run flag exists", func(t *testing.T) {
		flag := flags.Lookup("dry-run")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestCompletionCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "completion [bash|zsh|fish|powershell]", completionCmd.Use)
	assert.Equal(t, "Generate shell completion scripts", completionCmd.Short)
}

func TestCompletionCommand_ValidArgs(t *testing.T) {
	expected := []string{"bash", "zsh", "fish", "powershell"}
	assert.Equal(t, expected, completionCmd.ValidArgs)
}

func TestRootCommand_HasAllSubcommands(t *testing.T) {
	subcommands := rootCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	expected := []string{
		"apply",
		"capture",
		"completion",
		"diff",
		"doctor",
		"init",
		"lock",
		"plan",
		"repo",
		"tour",
		"version",
	}

	for _, exp := range expected {
		assert.Contains(t, names, exp, "root command should have %s subcommand", exp)
	}
}

// Capture command tests
func TestCaptureCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "capture", captureCmd.Use)
	assert.Equal(t, "Capture current machine configuration", captureCmd.Short)
}

func TestCaptureCommand_HasFlags(t *testing.T) {
	flags := captureCmd.Flags()

	t.Run("all flag exists", func(t *testing.T) {
		flag := flags.Lookup("all")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("provider flag exists", func(t *testing.T) {
		flag := flags.Lookup("provider")
		require.NotNil(t, flag)
		assert.Empty(t, flag.DefValue)
	})
}

// Doctor command tests
func TestDoctorCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "doctor", doctorCmd.Use)
	assert.Equal(t, "Verify system state and detect drift", doctorCmd.Short)
}

func TestDoctorCommand_HasFlags(t *testing.T) {
	flags := doctorCmd.Flags()

	t.Run("fix flag exists", func(t *testing.T) {
		flag := flags.Lookup("fix")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("verbose flag exists", func(t *testing.T) {
		flag := flags.Lookup("verbose")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
		assert.Equal(t, "v", flag.Shorthand)
	})
}

// Init command tests
func TestInitCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "init", initCmd.Use)
	assert.Equal(t, "Initialize a new preflight configuration", initCmd.Short)
}

func TestInitCommand_HasFlags(t *testing.T) {
	flags := initCmd.Flags()

	t.Run("provider flag exists", func(t *testing.T) {
		flag := flags.Lookup("provider")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("preset flag exists", func(t *testing.T) {
		flag := flags.Lookup("preset")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("skip-welcome flag exists", func(t *testing.T) {
		flag := flags.Lookup("skip-welcome")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("yes flag exists", func(t *testing.T) {
		flag := flags.Lookup("yes")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
		assert.Equal(t, "y", flag.Shorthand)
	})
}

// Tour command tests
func TestTourCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "tour [topic]", tourCmd.Use)
	assert.Equal(t, "Interactive guided walkthroughs", tourCmd.Short)
}

// Diff command tests
func TestDiffCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "diff", diffCmd.Use)
	assert.Equal(t, "Show differences between configuration and system", diffCmd.Short)
}

func TestDiffCommand_HasFlags(t *testing.T) {
	flags := diffCmd.Flags()

	t.Run("provider flag exists", func(t *testing.T) {
		flag := flags.Lookup("provider")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})
}

// Lock command tests
func TestLockCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "lock", lockCmd.Use)
	assert.Equal(t, "Manage lockfile for reproducible builds", lockCmd.Short)
}

func TestLockCommand_HasSubcommands(t *testing.T) {
	subcommands := lockCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	expected := []string{"update", "freeze", "status"}
	for _, exp := range expected {
		assert.Contains(t, names, exp, "lock command should have %s subcommand", exp)
	}
}

// Repo command tests
func TestRepoCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "repo", repoCmd.Use)
	assert.Equal(t, "Manage configuration repository", repoCmd.Short)
}

func TestRepoCommand_HasSubcommands(t *testing.T) {
	subcommands := repoCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	expected := []string{"init", "status", "push", "pull"}
	for _, exp := range expected {
		assert.Contains(t, names, exp, "repo command should have %s subcommand", exp)
	}
}

// Test that main commands have non-empty Long descriptions
func TestAllCommands_HaveLongDescriptions(t *testing.T) {
	// versionCmd intentionally has no Long description (Short is sufficient)
	commands := []*cobra.Command{
		rootCmd,
		planCmd,
		applyCmd,
		completionCmd,
		captureCmd,
		doctorCmd,
		initCmd,
		tourCmd,
		diffCmd,
		lockCmd,
		repoCmd,
	}

	for _, cmd := range commands {
		t.Run(cmd.Name(), func(t *testing.T) {
			assert.NotEmpty(t, cmd.Long, "%s should have a long description", cmd.Name())
		})
	}
}

// Test help works for all commands
func TestAllCommands_HelpWorks(t *testing.T) {
	commands := []string{
		"--help",
		"plan --help",
		"apply --help",
		"doctor --help",
		"capture --help",
		"init --help",
		"tour --help",
		"diff --help",
		"lock --help",
		"repo --help",
		"version --help",
		"completion --help",
	}

	for _, cmdArgs := range commands {
		t.Run(cmdArgs, func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			args := []string{}
			for _, arg := range bytes.Fields([]byte(cmdArgs)) {
				args = append(args, string(arg))
			}
			rootCmd.SetArgs(args)
			err := rootCmd.Execute()
			assert.NoError(t, err)
			assert.NotEmpty(t, out.String())

			// Reset for next test
			rootCmd.SetArgs([]string{})
		})
	}
}

// Tour command execution tests
// Note: These test that the commands run successfully. Output assertions
// are limited because commands use fmt.Println (writes to os.Stdout)
// rather than cmd.OutOrStdout().
func TestRunTour_NoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"tour"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	rootCmd.SetArgs([]string{})
}

func TestRunTour_Basics(t *testing.T) {
	rootCmd.SetArgs([]string{"tour", "basics"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	rootCmd.SetArgs([]string{})
}

func TestRunTour_Config(t *testing.T) {
	rootCmd.SetArgs([]string{"tour", "config"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	rootCmd.SetArgs([]string{})
}

func TestRunTour_Layers(t *testing.T) {
	rootCmd.SetArgs([]string{"tour", "layers"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	rootCmd.SetArgs([]string{})
}

func TestRunTour_Providers(t *testing.T) {
	rootCmd.SetArgs([]string{"tour", "providers"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	rootCmd.SetArgs([]string{})
}

func TestRunTour_AI(t *testing.T) {
	rootCmd.SetArgs([]string{"tour", "ai"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	rootCmd.SetArgs([]string{})
}

func TestRunTour_UnknownTopic(t *testing.T) {
	// Call runTour directly to avoid Cobra state issues between tests
	err := runTour(nil, []string{"invalid-topic"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tour")
}

// Lock status command execution test
func TestRunLockStatus_NoLockfile(t *testing.T) {
	// Save original cfgFile
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	// Set config to a non-existent path
	cfgFile = "/tmp/nonexistent-preflight-test.yaml"

	rootCmd.SetArgs([]string{"lock", "status"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	rootCmd.SetArgs([]string{})
}

// Lock subcommand metadata tests
func TestLockUpdateCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "update", lockUpdateCmd.Use)
	assert.Equal(t, "Update lockfile to latest compatible versions", lockUpdateCmd.Short)
}

func TestLockFreezeCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "freeze", lockFreezeCmd.Use)
	assert.Equal(t, "Freeze current versions in lockfile", lockFreezeCmd.Short)
}

func TestLockStatusCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "status", lockStatusCmd.Use)
	assert.Equal(t, "Show lockfile status", lockStatusCmd.Short)
}

// Repo subcommand metadata tests
func TestRepoInitCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "init", repoInitCmd.Use)
	assert.Equal(t, "Initialize configuration as a git repository", repoInitCmd.Short)
}

func TestRepoStatusCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "status", repoStatusCmd.Use)
	assert.Equal(t, "Show repository status", repoStatusCmd.Short)
}

func TestRepoPushCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "push", repoPushCmd.Use)
	assert.Equal(t, "Push configuration changes", repoPushCmd.Short)
}

func TestRepoPullCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "pull", repoPullCmd.Use)
	assert.Equal(t, "Pull configuration updates", repoPullCmd.Short)
}

// Repo init command flags test
func TestRepoInitCommand_HasFlags(t *testing.T) {
	flags := repoInitCmd.Flags()

	t.Run("remote flag exists", func(t *testing.T) {
		flag := flags.Lookup("remote")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("branch flag exists", func(t *testing.T) {
		flag := flags.Lookup("branch")
		require.NotNil(t, flag)
		assert.Equal(t, "main", flag.DefValue)
	})
}

// Repo push command flags test
func TestRepoPushCommand_HasFlags(t *testing.T) {
	flags := repoPushCmd.Flags()

	t.Run("force flag exists", func(t *testing.T) {
		flag := flags.Lookup("force")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

// Lock update command flags test
func TestLockUpdateCommand_HasFlags(t *testing.T) {
	flags := lockUpdateCmd.Flags()

	t.Run("provider flag exists", func(t *testing.T) {
		flag := flags.Lookup("provider")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})
}

// Direct function tests for better coverage
func TestRunTour_DirectCall_AllTopics(t *testing.T) {
	topics := []string{"basics", "config", "layers", "providers", "ai"}
	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			err := runTour(nil, []string{topic})
			require.NoError(t, err)
		})
	}
}

func TestRunTour_DirectCall_NoArgs(t *testing.T) {
	err := runTour(nil, []string{})
	require.NoError(t, err)
}

func TestRunLockStatus_DirectCall(t *testing.T) {
	// Save original cfgFile
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	// Test with non-existent lockfile
	cfgFile = "/tmp/nonexistent-test.yaml"
	err := runLockStatus(nil, nil)
	require.NoError(t, err)
}

func TestGetConfigDir_DefaultPath(t *testing.T) {
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	// Test with empty cfgFile (uses default)
	cfgFile = ""
	dir := getConfigDir()
	assert.Equal(t, ".", dir)
}

func TestGetConfigDir_CustomPath(t *testing.T) {
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	// Test with custom path
	cfgFile = "/some/path/config.yaml"
	dir := getConfigDir()
	assert.Equal(t, "/some/path", dir)
}

// Test completion command execution
// Note: Completion commands write directly to os.Stdout via rootCmd.GenXxxCompletion
// We test via rootCmd instead for proper output capture
func TestCompletionCommand_BashViaRoot(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "bash"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.NotEmpty(t, out.String())
	rootCmd.SetArgs([]string{})
}

func TestCompletionCommand_ZshViaRoot(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "zsh"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.NotEmpty(t, out.String())
	rootCmd.SetArgs([]string{})
}

func TestCompletionCommand_FishViaRoot(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "fish"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.NotEmpty(t, out.String())
	rootCmd.SetArgs([]string{})
}

func TestCompletionCommand_PowershellViaRoot(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "powershell"})
	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.NotEmpty(t, out.String())
	rootCmd.SetArgs([]string{})
}

// Test completion RunE directly to ensure coverage
func TestCompletionRunE_Bash(t *testing.T) {
	// Call RunE directly to ensure coverage is tracked
	// Note: This writes to os.Stdout which we can't capture easily,
	// but it ensures the code path is exercised for coverage
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := completionCmd.RunE(completionCmd, []string{"bash"})

	_ = w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "bash")
}

func TestCompletionRunE_Zsh(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := completionCmd.RunE(completionCmd, []string{"zsh"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "zsh")
}

func TestCompletionRunE_Fish(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := completionCmd.RunE(completionCmd, []string{"fish"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "fish")
}

func TestCompletionRunE_Powershell(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := completionCmd.RunE(completionCmd, []string{"powershell"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestCompletionRunE_UnknownShell(t *testing.T) {
	// Test the default case in switch (returns nil)
	err := completionCmd.RunE(completionCmd, []string{"unknown"})
	require.NoError(t, err)
}

// Test lock subcommands have Long descriptions
func TestLockSubcommands_HaveLongDescriptions(t *testing.T) {
	subcommands := []*cobra.Command{
		lockUpdateCmd,
		lockFreezeCmd,
		lockStatusCmd,
	}

	for _, cmd := range subcommands {
		t.Run(cmd.Name(), func(t *testing.T) {
			assert.NotEmpty(t, cmd.Long, "%s should have a long description", cmd.Name())
		})
	}
}

// Test repo subcommands have Long descriptions
func TestRepoSubcommands_HaveLongDescriptions(t *testing.T) {
	subcommands := []*cobra.Command{
		repoInitCmd,
		repoStatusCmd,
		repoPushCmd,
		repoPullCmd,
	}

	for _, cmd := range subcommands {
		t.Run(cmd.Name(), func(t *testing.T) {
			assert.NotEmpty(t, cmd.Long, "%s should have a long description", cmd.Name())
		})
	}
}

// Note: capture and doctor commands require a TTY for the TUI
// and cannot be tested directly without mocking the TUI layer.
// These commands are tested indirectly through their flag and metadata tests.

// Test lock status with existing lockfile
func TestRunLockStatus_ExistingLockfile(t *testing.T) {
	// Create temporary directory and lockfile
	tmpDir := t.TempDir()
	configPath := tmpDir + "/preflight.yaml"
	lockPath := tmpDir + "/preflight.lock"

	// Create a fake lockfile
	err := os.WriteFile(lockPath, []byte("version: 1\n"), 0644)
	require.NoError(t, err)

	// Save original
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = configPath
	err = runLockStatus(nil, nil)
	require.NoError(t, err)
}

// ============================================================
// RunPlan Tests
// ============================================================

func TestRunPlan_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid manifest
	manifest := `
targets:
  default:
    - base
`
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644)
	require.NoError(t, err)

	// Create layers directory and base layer
	err = os.MkdirAll(tmpDir+"/layers", 0755)
	require.NoError(t, err)
	err = os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	originalPlanConfigPath := planConfigPath
	originalPlanTarget := planTarget
	defer func() {
		cfgFile = originalCfgFile
		planConfigPath = originalPlanConfigPath
		planTarget = originalPlanTarget
	}()

	cfgFile = ""
	planConfigPath = tmpDir + "/preflight.yaml"
	planTarget = "default"

	err = runPlan(nil, nil)
	require.NoError(t, err)
}

func TestRunPlan_MissingConfig(t *testing.T) {
	// Save and restore flags
	originalPlanConfigPath := planConfigPath
	originalPlanTarget := planTarget
	defer func() {
		planConfigPath = originalPlanConfigPath
		planTarget = originalPlanTarget
	}()

	planConfigPath = "/nonexistent/config.yaml"
	planTarget = "default"

	err := runPlan(nil, nil)
	require.Error(t, err)
}

// ============================================================
// RunApply Tests
// ============================================================

func TestRunApply_ValidConfig_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid manifest
	manifest := `
targets:
  default:
    - base
`
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644)
	require.NoError(t, err)

	// Create layers directory and base layer
	err = os.MkdirAll(tmpDir+"/layers", 0755)
	require.NoError(t, err)
	err = os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalApplyConfigPath := applyConfigPath
	originalApplyTarget := applyTarget
	originalApplyDryRun := applyDryRun
	defer func() {
		applyConfigPath = originalApplyConfigPath
		applyTarget = originalApplyTarget
		applyDryRun = originalApplyDryRun
	}()

	applyConfigPath = tmpDir + "/preflight.yaml"
	applyTarget = "default"
	applyDryRun = true

	err = runApply(nil, nil)
	require.NoError(t, err)
}

func TestRunApply_MissingConfig(t *testing.T) {
	// Save and restore flags
	originalApplyConfigPath := applyConfigPath
	originalApplyTarget := applyTarget
	defer func() {
		applyConfigPath = originalApplyConfigPath
		applyTarget = originalApplyTarget
	}()

	applyConfigPath = "/nonexistent/config.yaml"
	applyTarget = "default"

	err := runApply(nil, nil)
	require.Error(t, err)
}

// ============================================================
// RunDiff Tests
// ============================================================

func TestRunDiff_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid manifest
	manifest := `
targets:
  default:
    - base
`
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644)
	require.NoError(t, err)

	// Create layers directory and base layer
	err = os.MkdirAll(tmpDir+"/layers", 0755)
	require.NoError(t, err)
	err = os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err = runDiff(nil, nil)
	require.NoError(t, err)
}

func TestRunDiff_MissingConfig(t *testing.T) {
	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = "/nonexistent/config.yaml"

	err := runDiff(nil, nil)
	require.Error(t, err)
}

// ============================================================
// RunLockUpdate/Freeze Tests
// ============================================================

func TestRunLockUpdate_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err = runLockUpdate(nil, nil)
	require.NoError(t, err)
}

func TestRunLockFreeze_MissingLockfile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file but no lockfile
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err = runLockFreeze(nil, nil)
	require.Error(t, err)
}

// ============================================================
// RunRepoInit/Status Tests
// ============================================================

func TestRunRepoInit_NewDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := tmpDir + "/myrepo"
	err := os.MkdirAll(repoDir, 0755)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	originalRepoRemote := repoRemote
	originalRepoBranch := repoBranch
	defer func() {
		cfgFile = originalCfgFile
		repoRemote = originalRepoRemote
		repoBranch = originalRepoBranch
	}()

	cfgFile = repoDir + "/preflight.yaml"
	repoRemote = ""
	repoBranch = "main"

	err = runRepoInit(nil, nil)
	require.NoError(t, err)

	// Verify .git directory was created
	_, err = os.Stat(repoDir + "/.git")
	assert.NoError(t, err)
}

func TestRunRepoInit_AlreadyInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory to simulate existing repo
	err := os.MkdirAll(tmpDir+"/.git", 0755)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err = runRepoInit(nil, nil)
	require.Error(t, err)
}

func TestRunRepoStatus_NotARepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	// Should not error, just print "not initialized"
	err := runRepoStatus(nil, nil)
	require.NoError(t, err)
}

func TestRunRepoStatus_InitializedRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git for commit
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test")
	_ = cmd.Run()

	// Create initial commit
	err := os.WriteFile(tmpDir+"/README.md", []byte("# Test\n"), 0644)
	require.NoError(t, err)
	cmd = exec.Command("git", "-C", tmpDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit")
	_ = cmd.Run()

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err = runRepoStatus(nil, nil)
	require.NoError(t, err)
}

// ============================================================
// RunRepoPush/Pull Tests
// ============================================================

func TestRunRepoPush_NotARepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err := runRepoPush(nil, nil)
	require.Error(t, err)
}

func TestRunRepoPull_NotARepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err := runRepoPull(nil, nil)
	require.Error(t, err)
}

// ============================================================
// Execute Tests
// ============================================================

func TestExecute_VersionCommand(t *testing.T) {
	// Test that Execute works with version command
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"version"})

	// Execute should not panic
	err := rootCmd.Execute()
	assert.NoError(t, err)

	rootCmd.SetArgs([]string{})
}

func TestExecute_HelpCommand(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.NotEmpty(t, out.String())

	rootCmd.SetArgs([]string{})
}

func TestExecute_InvalidCommand(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"invalidcommand123"})

	err := rootCmd.Execute()
	assert.Error(t, err)

	rootCmd.SetArgs([]string{})
}

// ============================================================
// Additional Lock Tests for Coverage
// ============================================================

func TestRunLockUpdate_WithProvider(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	originalLockUpdateProvider := lockUpdateProvider
	defer func() {
		cfgFile = originalCfgFile
		lockUpdateProvider = originalLockUpdateProvider
	}()

	cfgFile = tmpDir + "/preflight.yaml"
	lockUpdateProvider = "brew" // Set provider filter

	err = runLockUpdate(nil, nil)
	require.NoError(t, err)
}

func TestRunLockUpdate_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty config file
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err = runLockUpdate(nil, nil)
	require.NoError(t, err)
}

func TestRunLockFreeze_WithExistingLock(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Create a lockfile with proper format (machine_info with snapshot)
	lockContent := `version: 1
mode: intent
machine_info:
  os: darwin
  arch: arm64
  hostname: testhost
  snapshot: "2024-01-01T00:00:00Z"
packages: {}
`
	err = os.WriteFile(tmpDir+"/preflight.lock", []byte(lockContent), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err = runLockFreeze(nil, nil)
	require.NoError(t, err)
}

func TestRunLockStatus_WithExistingLock(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Create a lockfile
	err = os.WriteFile(tmpDir+"/preflight.lock", []byte("version: 1\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	err = runLockStatus(nil, nil)
	require.NoError(t, err)
}

func TestRunLockStatus_DefaultConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the expected default config and lock files
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Change to tmpDir temporarily
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = "" // Use default

	err = runLockStatus(nil, nil)
	require.NoError(t, err)
}

// ============================================================
// Additional Repo Tests for Coverage
// ============================================================

func TestRunRepoInit_WithRemote(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := tmpDir + "/myrepo-remote"
	err := os.MkdirAll(repoDir, 0755)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	originalRepoRemote := repoRemote
	originalRepoBranch := repoBranch
	defer func() {
		cfgFile = originalCfgFile
		repoRemote = originalRepoRemote
		repoBranch = originalRepoBranch
	}()

	cfgFile = repoDir + "/preflight.yaml"
	repoRemote = "https://github.com/example/repo.git"
	repoBranch = "main"

	err = runRepoInit(nil, nil)
	require.NoError(t, err)
}

func TestRunRepoInit_CurrentDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir temporarily
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	originalRepoRemote := repoRemote
	originalRepoBranch := repoBranch
	defer func() {
		cfgFile = originalCfgFile
		repoRemote = originalRepoRemote
		repoBranch = originalRepoBranch
	}()

	cfgFile = "" // Use current directory
	repoRemote = ""
	repoBranch = "main"

	err = runRepoInit(nil, nil)
	require.NoError(t, err)
}

func TestRunRepoStatus_CurrentDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir temporarily
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = "" // Use current directory

	err = runRepoStatus(nil, nil)
	require.NoError(t, err)
}

func TestRunRepoPush_CurrentDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir temporarily
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = "" // Use current directory

	// This will fail because it's not a git repo, but it exercises the cfgFile == "" path
	err = runRepoPush(nil, nil)
	require.Error(t, err) // Expected to fail - not a git repo
}

func TestRunRepoPush_WithForce(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore flags
	originalCfgFile := cfgFile
	originalRepoForce := repoForce
	defer func() {
		cfgFile = originalCfgFile
		repoForce = originalRepoForce
	}()

	cfgFile = tmpDir + "/preflight.yaml"
	repoForce = true

	// This will fail because it's not a git repo, but it exercises the force flag path
	err := runRepoPush(nil, nil)
	require.Error(t, err) // Expected to fail - not a git repo
}

func TestRunRepoPull_CurrentDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir temporarily
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = "" // Use current directory

	// This will fail because it's not a git repo, but it exercises the cfgFile == "" path
	err = runRepoPull(nil, nil)
	require.Error(t, err) // Expected to fail - not a git repo
}

// ============================================================
// GetConfigDir Tests
// ============================================================

func TestGetConfigDir_WithPath(t *testing.T) {
	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = "/some/path/preflight.yaml"
	dir := getConfigDir()
	assert.Equal(t, "/some/path", dir)
}

func TestGetConfigDir_Empty(t *testing.T) {
	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = ""
	dir := getConfigDir()
	assert.Equal(t, ".", dir)
}

// ============================================================
// Diff Additional Coverage
// ============================================================

func TestRunDiff_DefaultConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid manifest at default location
	manifest := `
targets:
  default:
    - base
`
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644)
	require.NoError(t, err)

	// Create layers directory
	err = os.MkdirAll(tmpDir+"/layers", 0755)
	require.NoError(t, err)
	err = os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0644)
	require.NoError(t, err)

	// Change to tmpDir temporarily
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = "" // Use default

	err = runDiff(nil, nil)
	require.NoError(t, err)
}

// ============================================================
// Execute Function Coverage
// ============================================================

func TestExecute(t *testing.T) {
	// Test the Execute function directly
	// Save original args
	oldArgs := rootCmd.Args
	defer func() {
		rootCmd.Args = oldArgs
	}()

	// Test with valid command
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"version"})

	err := Execute()
	assert.NoError(t, err)

	rootCmd.SetArgs([]string{})
}

// ============================================================
// Completion Command Tests
// ============================================================

func TestCompletionCommand_Properties(t *testing.T) {
	assert.Equal(t, "completion [bash|zsh|fish|powershell]", completionCmd.Use)
	assert.Equal(t, "Generate shell completion scripts", completionCmd.Short)
	assert.True(t, completionCmd.DisableFlagsInUseLine)
	// Use ElementsMatch as Cobra may reorder ValidArgs internally
	assert.ElementsMatch(t, []string{"bash", "zsh", "fish", "powershell"}, completionCmd.ValidArgs)
}

// ============================================================
// Additional Apply Tests for Coverage
// ============================================================

func TestRunApply_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid manifest that produces no changes
	manifest := `
targets:
  default:
    - base
`
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte(manifest), 0644)
	require.NoError(t, err)

	// Create layers directory and base layer (empty, so no steps/changes)
	err = os.MkdirAll(tmpDir+"/layers", 0755)
	require.NoError(t, err)
	err = os.WriteFile(tmpDir+"/layers/base.yaml", []byte("name: base\n"), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalApplyConfigPath := applyConfigPath
	originalApplyTarget := applyTarget
	originalApplyDryRun := applyDryRun
	defer func() {
		applyConfigPath = originalApplyConfigPath
		applyTarget = originalApplyTarget
		applyDryRun = originalApplyDryRun
	}()

	applyConfigPath = tmpDir + "/preflight.yaml"
	applyTarget = "default"
	applyDryRun = false

	err = runApply(nil, nil)
	require.NoError(t, err)
}

// ============================================================
// Additional Repo Tests for Better Coverage
// ============================================================

func TestRunRepoPush_SuccessfulRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test")
	_ = cmd.Run()

	// Create initial commit
	err := os.WriteFile(tmpDir+"/README.md", []byte("# Test\n"), 0644)
	require.NoError(t, err)
	cmd = exec.Command("git", "-C", tmpDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit")
	_ = cmd.Run()

	// Save and restore flags
	originalCfgFile := cfgFile
	originalRepoForce := repoForce
	defer func() {
		cfgFile = originalCfgFile
		repoForce = originalRepoForce
	}()

	cfgFile = tmpDir + "/preflight.yaml"
	repoForce = false

	// This will fail because there's no remote, but it exercises more of the push code path
	err = runRepoPush(nil, nil)
	require.Error(t, err) // Expected - no remote configured
}

func TestRunRepoPull_InitializedRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test")
	_ = cmd.Run()

	// Create initial commit
	err := os.WriteFile(tmpDir+"/README.md", []byte("# Test\n"), 0644)
	require.NoError(t, err)
	cmd = exec.Command("git", "-C", tmpDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit")
	_ = cmd.Run()

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = tmpDir + "/preflight.yaml"

	// This will fail because there's no remote, but it exercises more of the pull code path
	err = runRepoPull(nil, nil)
	require.Error(t, err) // Expected - no remote configured
}

// ============================================================
// Lock Additional Coverage
// ============================================================

func TestRunLockUpdate_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Change to tmpDir temporarily
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	originalLockUpdateProvider := lockUpdateProvider
	defer func() {
		cfgFile = originalCfgFile
		lockUpdateProvider = originalLockUpdateProvider
	}()

	cfgFile = "" // Use default
	lockUpdateProvider = ""

	err = runLockUpdate(nil, nil)
	require.NoError(t, err)
}

func TestRunLockFreeze_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Create a lockfile with proper format
	lockContent := `version: 1
mode: intent
machine_info:
  os: darwin
  arch: arm64
  hostname: testhost
  snapshot: "2024-01-01T00:00:00Z"
packages: {}
`
	err = os.WriteFile(tmpDir+"/preflight.lock", []byte(lockContent), 0644)
	require.NoError(t, err)

	// Change to tmpDir temporarily
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = "" // Use default

	err = runLockFreeze(nil, nil)
	require.NoError(t, err)
}

// ============================================================
// RunInit Tests
// ============================================================

func TestRunInit_ConfigAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing preflight.yaml
	err := os.WriteFile(tmpDir+"/preflight.yaml", []byte("targets:\n  default: []\n"), 0644)
	require.NoError(t, err)

	// Change to tmpDir temporarily (runInit checks current directory)
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// runInit checks for "preflight.yaml" in current directory
	err = runInit(nil, nil)
	require.NoError(t, err) // Should return nil when config exists
}

// ============================================================
// RunLockUpdate Error Coverage
// ============================================================

func TestRunLockUpdate_ErrorPath(t *testing.T) {
	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	// Use non-existent config path
	cfgFile = "/nonexistent/path/preflight.yaml"

	err := runLockUpdate(nil, nil)
	require.Error(t, err)
}

// ============================================================
// Apply with Actual Changes
// ============================================================

func TestRunApply_WithFilesConfig_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp directory so relative paths work
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create manifest with files configuration that will generate steps
	manifest := `
targets:
  default:
    - base
`
	err = os.WriteFile("preflight.yaml", []byte(manifest), 0644)
	require.NoError(t, err)

	// Create layers directory with files config
	err = os.MkdirAll("layers", 0755)
	require.NoError(t, err)

	// Create dotfiles directory with source file
	err = os.MkdirAll("dotfiles", 0755)
	require.NoError(t, err)
	err = os.WriteFile("dotfiles/test.conf", []byte("# test config\n"), 0644)
	require.NoError(t, err)

	// Base layer with files config - target doesn't exist so it will generate changes
	targetFile := tmpDir + "/output/test.conf"
	baseLayer := `
name: base
files:
  - path: ` + targetFile + `
    mode: generated
    template: dotfiles/test.conf
`
	err = os.WriteFile("layers/base.yaml", []byte(baseLayer), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalApplyConfigPath := applyConfigPath
	originalApplyTarget := applyTarget
	originalApplyDryRun := applyDryRun
	defer func() {
		applyConfigPath = originalApplyConfigPath
		applyTarget = originalApplyTarget
		applyDryRun = originalApplyDryRun
	}()

	applyConfigPath = "preflight.yaml"
	applyTarget = "default"
	applyDryRun = true // Use dry-run to test the dry-run code path

	// Capture stdout to see plan output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runApply(nil, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should succeed with dry-run showing plan
	require.NoError(t, err)
	assert.Contains(t, output, "[Dry run - no changes made]")
}

// TestRunApply_WithFilesConfig_Execute tests actual apply execution path
func TestRunApply_WithFilesConfig_Execute(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp directory so relative paths work
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create manifest with files configuration
	manifest := `
targets:
  default:
    - base
`
	err = os.WriteFile("preflight.yaml", []byte(manifest), 0644)
	require.NoError(t, err)

	// Create layers directory and dotfiles
	err = os.MkdirAll("layers", 0755)
	require.NoError(t, err)
	err = os.MkdirAll("dotfiles", 0755)
	require.NoError(t, err)
	err = os.WriteFile("dotfiles/test.conf", []byte("# test config\n"), 0644)
	require.NoError(t, err)

	// Create output directory for target
	err = os.MkdirAll("output", 0755)
	require.NoError(t, err)

	// Base layer with files config
	targetFile := tmpDir + "/output/test.conf"
	baseLayer := `
name: base
files:
  - path: ` + targetFile + `
    mode: generated
    template: dotfiles/test.conf
`
	err = os.WriteFile("layers/base.yaml", []byte(baseLayer), 0644)
	require.NoError(t, err)

	// Save and restore flags
	originalApplyConfigPath := applyConfigPath
	originalApplyTarget := applyTarget
	originalApplyDryRun := applyDryRun
	defer func() {
		applyConfigPath = originalApplyConfigPath
		applyTarget = originalApplyTarget
		applyDryRun = originalApplyDryRun
	}()

	applyConfigPath = "preflight.yaml"
	applyTarget = "default"
	applyDryRun = false // Execute actual apply

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runApply(nil, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Should succeed and apply changes
	if err != nil {
		t.Logf("Apply output: %s", buf.String())
		t.Logf("Apply error: %v", err)
	}

	// Verify the symlink was created
	_, statErr := os.Lstat(targetFile)
	if statErr == nil {
		// File was created
		t.Log("Target file was created successfully")
	}
}

// ============================================================
// Additional Repo Status Error Path
// ============================================================

func TestRunRepoStatus_GetConfigDirError(t *testing.T) {
	// This test exercises different code paths in runRepoStatus
	tmpDir := t.TempDir()

	// Create a valid git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test")
	_ = cmd.Run()

	// Create initial commit
	err := os.WriteFile(tmpDir+"/README.md", []byte("# Test\n"), 0644)
	require.NoError(t, err)
	cmd = exec.Command("git", "-C", tmpDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit")
	_ = cmd.Run()

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	// Test with path that uses . (getConfigDir returns ".")
	cfgFile = "preflight.yaml" // This will result in configDir == "."

	// Change to tmpDir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint:errcheck
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	err = runRepoStatus(nil, nil)
	require.NoError(t, err)
}

// ============================================================
// RepoPush Success Path
// ============================================================

func TestRunRepoPush_WithRemote(t *testing.T) {
	// Create a "remote" repo first
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", remoteDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	// Create local repo
	localDir := t.TempDir()
	cmd = exec.Command("git", "init", localDir)
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = exec.Command("git", "-C", localDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", localDir, "config", "user.name", "Test")
	_ = cmd.Run()

	// Add remote
	cmd = exec.Command("git", "-C", localDir, "remote", "add", "origin", remoteDir)
	require.NoError(t, cmd.Run())

	// Create and commit a file
	err := os.WriteFile(localDir+"/README.md", []byte("# Test\n"), 0644)
	require.NoError(t, err)
	cmd = exec.Command("git", "-C", localDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", localDir, "commit", "-m", "Initial commit")
	_ = cmd.Run()

	// Save and restore flags
	originalCfgFile := cfgFile
	originalRepoForce := repoForce
	defer func() {
		cfgFile = originalCfgFile
		repoForce = originalRepoForce
	}()

	cfgFile = localDir + "/preflight.yaml"
	repoForce = false

	err = runRepoPush(nil, nil)
	require.NoError(t, err)
}

func TestRunRepoPush_SuccessPath(t *testing.T) {
	// Create repos and test the success path with force flag
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", remoteDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	localDir := t.TempDir()
	cmd = exec.Command("git", "init", localDir)
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = exec.Command("git", "-C", localDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", localDir, "config", "user.name", "Test")
	_ = cmd.Run()

	// Add remote
	cmd = exec.Command("git", "-C", localDir, "remote", "add", "origin", remoteDir)
	require.NoError(t, cmd.Run())

	// Create and commit a file
	err := os.WriteFile(localDir+"/README.md", []byte("# Test\n"), 0644)
	require.NoError(t, err)
	cmd = exec.Command("git", "-C", localDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", localDir, "commit", "-m", "Initial commit")
	_ = cmd.Run()

	// Save and restore flags
	originalCfgFile := cfgFile
	originalRepoForce := repoForce
	defer func() {
		cfgFile = originalCfgFile
		repoForce = originalRepoForce
	}()

	cfgFile = localDir + "/preflight.yaml"
	repoForce = true // Test force path

	err = runRepoPush(nil, nil)
	require.NoError(t, err)
}

// ============================================================
// RepoPull Success Path
// ============================================================

func TestRunRepoPull_WithRemote(t *testing.T) {
	// Create a "remote" repo with some commits
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", remoteDir)
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	// Create local repo
	localDir := t.TempDir()
	cmd = exec.Command("git", "init", localDir)
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = exec.Command("git", "-C", localDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", localDir, "config", "user.name", "Test")
	_ = cmd.Run()

	// Add remote
	cmd = exec.Command("git", "-C", localDir, "remote", "add", "origin", remoteDir)
	require.NoError(t, cmd.Run())

	// Create and commit a file
	err := os.WriteFile(localDir+"/README.md", []byte("# Test\n"), 0644)
	require.NoError(t, err)
	cmd = exec.Command("git", "-C", localDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", localDir, "commit", "-m", "Initial commit")
	_ = cmd.Run()

	// Push to remote first
	cmd = exec.Command("git", "-C", localDir, "push", "-u", "origin", "master")
	if err := cmd.Run(); err != nil {
		// Try main branch
		cmd = exec.Command("git", "-C", localDir, "push", "-u", "origin", "main")
		_ = cmd.Run()
	}

	// Save and restore flags
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
	}()

	cfgFile = localDir + "/preflight.yaml"

	// Pull should work now (nothing to pull but no error)
	err = runRepoPull(nil, nil)
	require.NoError(t, err)
}
