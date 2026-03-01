package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFleetInventoryFile_ToInventory(t *testing.T) {
	t.Parallel()

	file := FleetInventoryFile{
		Version: 1,
		Hosts: map[string]FleetHostConfig{
			"server01": {
				Hostname: "10.0.0.1",
				User:     "admin",
				Port:     22,
				Tags:     []string{"production", "darwin"},
				Groups:   []string{"web"},
			},
			"server02": {
				Hostname: "10.0.0.2",
				User:     "root",
				Port:     2222,
				Tags:     []string{"staging", "linux"},
			},
		},
		Groups: map[string]FleetGroupConfig{
			"web": {
				Description: "Web servers",
				Hosts:       []string{"web-*"},
				Policies:    []string{"require-approval"},
			},
		},
		Defaults: FleetDefaultsConfig{
			User: "deploy",
			Port: 22,
		},
	}

	inv, err := file.ToInventory()
	require.NoError(t, err)

	assert.Equal(t, 2, inv.HostCount())
	assert.Equal(t, 1, inv.GroupCount())

	host, ok := inv.GetHost("server01")
	require.True(t, ok)
	assert.Equal(t, "10.0.0.1", host.SSH().Hostname)
	assert.Equal(t, "admin", host.SSH().User)
}

func TestFleetInventoryFile_ToInventory_InvalidHost(t *testing.T) {
	t.Parallel()

	file := FleetInventoryFile{
		Hosts: map[string]FleetHostConfig{
			"123invalid": { // Starts with number
				Hostname: "10.0.0.1",
			},
		},
	}

	_, err := file.ToInventory()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid host ID")
}

func TestFleetInventoryFile_ToInventory_InvalidTag(t *testing.T) {
	t.Parallel()

	file := FleetInventoryFile{
		Hosts: map[string]FleetHostConfig{
			"server01": {
				Hostname: "10.0.0.1",
				Tags:     []string{"Invalid Tag"}, // Contains space
			},
		},
	}

	_, err := file.ToInventory()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tag")
}

func TestFleetInventoryFile_ToInventory_InvalidGroup(t *testing.T) {
	t.Parallel()

	file := FleetInventoryFile{
		Groups: map[string]FleetGroupConfig{
			"123invalid": { // Starts with number
				Description: "Invalid group",
			},
		},
	}

	_, err := file.ToInventory()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid group name")
}

//nolint:tparallel // Test modifies global state (flags and os.Stdout)
func TestFleetCmd_List(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	// Create temp inventory file
	tmpDir := t.TempDir()
	invPath := filepath.Join(tmpDir, "fleet.yaml")

	invContent := `version: 1
hosts:
  server01:
    hostname: 10.0.0.1
    user: admin
    port: 22
    tags:
      - production
      - darwin
`
	err := os.WriteFile(invPath, []byte(invContent), 0o644)
	require.NoError(t, err)

	// Save and restore original flag values
	origInventoryFile := fleetInventoryFile
	origTarget := fleetTarget
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInventoryFile
		fleetTarget = origTarget
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invPath
	fleetTarget = "@all"
	fleetJSON = false

	// Capture output
	output := captureStdout(t, func() {
		err = runFleetList(nil, nil)
	})
	require.NoError(t, err)
	assert.Contains(t, output, "server01")
	assert.Contains(t, output, "10.0.0.1")
	assert.Contains(t, output, "admin")
}

//nolint:tparallel // Test modifies global state (flags and os.Stdout)
func TestFleetCmd_List_JSON(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	tmpDir := t.TempDir()
	invPath := filepath.Join(tmpDir, "fleet.yaml")

	invContent := `version: 1
hosts:
  server01:
    hostname: 10.0.0.1
    user: admin
    port: 22
`
	err := os.WriteFile(invPath, []byte(invContent), 0o644)
	require.NoError(t, err)

	origInventoryFile := fleetInventoryFile
	origTarget := fleetTarget
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInventoryFile
		fleetTarget = origTarget
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invPath
	fleetTarget = "@all"
	fleetJSON = true

	output := captureStdout(t, func() {
		err = runFleetList(nil, nil)
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(output, "["))
	assert.Contains(t, output, "\"id\": \"server01\"")
}

//nolint:tparallel // Test modifies global state (flags and os.Stdout)
func TestFleetCmd_Status(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	tmpDir := t.TempDir()
	invPath := filepath.Join(tmpDir, "fleet.yaml")

	invContent := `version: 1
hosts:
  server01:
    hostname: 10.0.0.1
    tags: [darwin]
  server02:
    hostname: 10.0.0.2
    tags: [linux]
groups:
  production:
    description: Production servers
`
	err := os.WriteFile(invPath, []byte(invContent), 0o644)
	require.NoError(t, err)

	origInventoryFile := fleetInventoryFile
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInventoryFile
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invPath
	fleetJSON = false

	output := captureStdout(t, func() {
		err = runFleetStatus(nil, nil)
	})
	require.NoError(t, err)
	assert.Contains(t, output, "Total hosts:  2")
	assert.Contains(t, output, "Total groups: 1")
}

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_MissingInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	origInventoryFile := fleetInventoryFile
	defer func() {
		fleetInventoryFile = origInventoryFile
	}()

	fleetInventoryFile = "/nonexistent/path/fleet.yaml"

	err := runFleetList(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

// ---------------------------------------------------------------------------
// runFleetPing tests
// ---------------------------------------------------------------------------

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Ping_MissingInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	origInventoryFile := fleetInventoryFile
	defer func() { fleetInventoryFile = origInventoryFile }()

	fleetInventoryFile = "/nonexistent/path/fleet.yaml"
	err := runFleetPing(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Ping_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runFleetPing(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

// ---------------------------------------------------------------------------
// runFleetPlan tests
// ---------------------------------------------------------------------------

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Plan_MissingInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	origInventoryFile := fleetInventoryFile
	defer func() { fleetInventoryFile = origInventoryFile }()

	fleetInventoryFile = "/nonexistent/path/fleet.yaml"
	err := runFleetPlan(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Plan_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runFleetPlan(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

// ---------------------------------------------------------------------------
// runFleetApply tests
// ---------------------------------------------------------------------------

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Apply_MissingInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	origInventoryFile := fleetInventoryFile
	defer func() { fleetInventoryFile = origInventoryFile }()

	fleetInventoryFile = "/nonexistent/path/fleet.yaml"
	err := runFleetApply(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Apply_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runFleetApply(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Apply_InvalidStrategy(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	tmpDir := t.TempDir()
	invPath := filepath.Join(tmpDir, "fleet.yaml")

	invContent := `version: 1
hosts:
  server01:
    hostname: 10.0.0.1
`
	err := os.WriteFile(invPath, []byte(invContent), 0o644)
	require.NoError(t, err)

	origInventoryFile := fleetInventoryFile
	origTarget := fleetTarget
	origStrategy := fleetStrategy
	defer func() {
		fleetInventoryFile = origInventoryFile
		fleetTarget = origTarget
		fleetStrategy = origStrategy
	}()

	fleetInventoryFile = invPath
	fleetTarget = "@all"
	fleetStrategy = "invalid-strategy"

	err = runFleetApply(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid strategy")
}

// ---------------------------------------------------------------------------
// runFleetStatus tests
// ---------------------------------------------------------------------------

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Status_MissingInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	origInventoryFile := fleetInventoryFile
	defer func() { fleetInventoryFile = origInventoryFile }()

	fleetInventoryFile = "/nonexistent/path/fleet.yaml"
	err := runFleetStatus(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_Status_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runFleetStatus(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

//nolint:tparallel // Test modifies global state (flags and os.Stdout)
func TestFleetCmd_Status_JSON(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	tmpDir := t.TempDir()
	invPath := filepath.Join(tmpDir, "fleet.yaml")

	invContent := `version: 1
hosts:
  server01:
    hostname: 10.0.0.1
    tags: [darwin]
`
	err := os.WriteFile(invPath, []byte(invContent), 0o644)
	require.NoError(t, err)

	origInventoryFile := fleetInventoryFile
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInventoryFile
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invPath
	fleetJSON = true

	output := captureStdout(t, func() {
		err = runFleetStatus(nil, nil)
	})
	require.NoError(t, err)
	assert.Contains(t, output, "\"host_count\"")
	assert.Contains(t, output, "\"group_count\"")
}

// ---------------------------------------------------------------------------
// Subcommand structure tests
// ---------------------------------------------------------------------------

func TestFleetCmd_SubcommandStructure(t *testing.T) {
	t.Parallel()

	subcommands := fleetCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "list")
	assert.Contains(t, names, "ping")
	assert.Contains(t, names, "plan")
	assert.Contains(t, names, "apply")
	assert.Contains(t, names, "status")
}

func TestFleetCmd_PersistentFlags(t *testing.T) {
	t.Parallel()

	invFlag := fleetCmd.PersistentFlags().Lookup("inventory")
	require.NotNil(t, invFlag)
	assert.Equal(t, "fleet.yaml", invFlag.DefValue)

	targetFlag := fleetCmd.PersistentFlags().Lookup("target")
	require.NotNil(t, targetFlag)
	assert.Equal(t, "@all", targetFlag.DefValue)

	excludeFlag := fleetCmd.PersistentFlags().Lookup("exclude")
	require.NotNil(t, excludeFlag)
	assert.Equal(t, "", excludeFlag.DefValue)

	jsonFlag := fleetCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)
}

func TestFleetCmd_ApplyFlags(t *testing.T) {
	t.Parallel()

	strategyFlag := fleetApplyCmd.Flags().Lookup("strategy")
	require.NotNil(t, strategyFlag)
	assert.Equal(t, "parallel", strategyFlag.DefValue)

	maxParallelFlag := fleetApplyCmd.Flags().Lookup("max-parallel")
	require.NotNil(t, maxParallelFlag)
	assert.Equal(t, "10", maxParallelFlag.DefValue)

	dryRunFlag := fleetApplyCmd.Flags().Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
	assert.Equal(t, "false", dryRunFlag.DefValue)

	stopOnErrorFlag := fleetApplyCmd.Flags().Lookup("stop-on-error")
	require.NotNil(t, stopOnErrorFlag)
	assert.Equal(t, "false", stopOnErrorFlag.DefValue)
}

// ---------------------------------------------------------------------------
// loadFleetInventory tests
// ---------------------------------------------------------------------------

//nolint:tparallel // Test modifies global state (flags)
func TestLoadFleetInventory_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	invPath := filepath.Join(tmpDir, "fleet.yaml")

	err := os.WriteFile(invPath, []byte("invalid: [yaml: {broken"), 0o644)
	require.NoError(t, err)

	origInventoryFile := fleetInventoryFile
	defer func() { fleetInventoryFile = origInventoryFile }()

	fleetInventoryFile = invPath

	_, err = loadFleetInventory()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse inventory")
}

// ---------------------------------------------------------------------------
// selectHosts tests
// ---------------------------------------------------------------------------

//nolint:tparallel // Test modifies global state (flags)
func TestSelectHosts_WithExclude(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	file := FleetInventoryFile{
		Version: 1,
		Hosts: map[string]FleetHostConfig{
			"web-server": {
				Hostname: "10.0.0.1",
				Tags:     []string{"production"},
			},
			"db-server": {
				Hostname: "10.0.0.2",
				Tags:     []string{"production"},
			},
		},
	}

	inv, err := file.ToInventory()
	require.NoError(t, err)

	origTarget := fleetTarget
	origExclude := fleetExclude
	defer func() {
		fleetTarget = origTarget
		fleetExclude = origExclude
	}()

	fleetTarget = "@all"
	fleetExclude = "db-server"

	hosts, err := selectHosts(inv)
	require.NoError(t, err)
	assert.Len(t, hosts, 1)
	assert.Equal(t, "web-server", string(hosts[0].ID()))
}

// ---------------------------------------------------------------------------
// FleetInventoryFile.ToInventory additional tests
// ---------------------------------------------------------------------------

func TestFleetInventoryFile_ToInventory_WithInheritGroup(t *testing.T) {
	t.Parallel()

	file := FleetInventoryFile{
		Version: 1,
		Hosts: map[string]FleetHostConfig{
			"server01": {
				Hostname: "10.0.0.1",
			},
		},
		Groups: map[string]FleetGroupConfig{
			"base": {
				Description: "Base group",
			},
			"web": {
				Description: "Web servers",
				Inherit:     []string{"base"},
			},
		},
	}

	inv, err := file.ToInventory()
	require.NoError(t, err)
	assert.Equal(t, 2, inv.GroupCount())
}

func TestFleetInventoryFile_ToInventory_InvalidInheritGroup(t *testing.T) {
	t.Parallel()

	file := FleetInventoryFile{
		Groups: map[string]FleetGroupConfig{
			"web": {
				Description: "Web servers",
				Inherit:     []string{"123invalid"},
			},
		},
	}

	_, err := file.ToInventory()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parent group")
}

func TestFleetInventoryFile_ToInventory_WithSSHConfig(t *testing.T) {
	t.Parallel()

	file := FleetInventoryFile{
		Hosts: map[string]FleetHostConfig{
			"server01": {
				Hostname:  "10.0.0.1",
				User:      "admin",
				Port:      2222,
				SSHKey:    "~/.ssh/id_ed25519",
				ProxyJump: "bastion",
			},
		},
	}

	inv, err := file.ToInventory()
	require.NoError(t, err)

	host, ok := inv.GetHost("server01")
	require.True(t, ok)
	assert.Equal(t, "10.0.0.1", host.SSH().Hostname)
	assert.Equal(t, "admin", host.SSH().User)
	assert.Equal(t, 2222, host.SSH().Port)
	assert.Equal(t, "~/.ssh/id_ed25519", host.SSH().IdentityFile)
	assert.Equal(t, "bastion", host.SSH().ProxyJump)
}
