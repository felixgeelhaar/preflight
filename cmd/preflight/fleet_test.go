package main

import (
	"bytes"
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
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runFleetList(nil, nil)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "server01")
	assert.Contains(t, output, "10.0.0.1")
	assert.Contains(t, output, "admin")
}

//nolint:tparallel // Test modifies global state (flags and os.Stdout)
func TestFleetCmd_List_JSON(t *testing.T) {
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

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runFleetList(nil, nil)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	output := buf.String()
	assert.True(t, strings.HasPrefix(output, "["))
	assert.Contains(t, output, "\"id\": \"server01\"")
}

//nolint:tparallel // Test modifies global state (flags and os.Stdout)
func TestFleetCmd_Status(t *testing.T) {
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

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runFleetStatus(nil, nil)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Total hosts:  2")
	assert.Contains(t, output, "Total groups: 1")
}

//nolint:tparallel // Test modifies global state (flags)
func TestFleetCmd_MissingInventory(t *testing.T) {
	origInventoryFile := fleetInventoryFile
	defer func() {
		fleetInventoryFile = origInventoryFile
	}()

	fleetInventoryFile = "/nonexistent/path/fleet.yaml"

	err := runFleetList(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}
