package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestCleanCmd_Exists and TestCleanCmd_HasFlags are in helpers_test.go

func TestCleanCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"config default", "config", "preflight.yaml"},
		{"target default", "target", "default"},
		{"apply default", "apply", "false"},
		{"providers default", "providers", ""},
		{"ignore default", "ignore", ""},
		{"json default", "json", "false"},
		{"force default", "force", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := cleanCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f, "flag %s should exist", tt.flag)
			if f != nil {
				assert.Equal(t, tt.expected, f.DefValue)
			}
		})
	}
}

func TestCleanCmd_ConfigShorthand(t *testing.T) {
	t.Parallel()

	f := cleanCmd.Flags().Lookup("config")
	assert.NotNil(t, f)
	assert.Equal(t, "c", f.Shorthand)
}

func TestCleanCmd_TargetShorthand(t *testing.T) {
	t.Parallel()

	f := cleanCmd.Flags().Lookup("target")
	assert.NotNil(t, f)
	assert.Equal(t, "t", f.Shorthand)
}

func TestCleanCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "clean" {
			found = true
			break
		}
	}
	assert.True(t, found, "clean should be a subcommand of root")
}

// Note: TestOrphanedItemFields is in helpers_test.go
// Note: TestShouldCheckProvider, TestIsIgnored, TestOutputOrphansText are in helpers_test.go

// ---------------------------------------------------------------------------
// findOrphans additional tests
// ---------------------------------------------------------------------------

func TestFindOrphans_EmptySystemState(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "curl"},
		},
	}
	systemState := map[string]interface{}{}

	orphans := findOrphans(config, systemState, nil, nil)
	assert.Empty(t, orphans)
}

func TestFindOrphans_NoOrphans(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "curl"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "curl"},
		},
	}

	orphans := findOrphans(config, systemState, nil, nil)
	assert.Empty(t, orphans)
}

func TestFindOrphans_BrewFormulaeOrphan(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "htop"},
		},
	}

	orphans := findOrphans(config, systemState, nil, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "brew", orphans[0].Provider)
	assert.Equal(t, "formula", orphans[0].Type)
	assert.Equal(t, "htop", orphans[0].Name)
}

func TestFindOrphans_BrewCaskOrphan(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"casks": []interface{}{"firefox"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"casks": []interface{}{"firefox", "slack"},
		},
	}

	orphans := findOrphans(config, systemState, nil, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "cask", orphans[0].Type)
	assert.Equal(t, "slack", orphans[0].Name)
}

func TestFindOrphans_VSCodeOrphan(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.go"},
		},
	}
	systemState := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.go", "eamodio.gitlens"},
		},
	}

	orphans := findOrphans(config, systemState, nil, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "vscode", orphans[0].Provider)
	assert.Equal(t, "extension", orphans[0].Type)
	assert.Equal(t, "eamodio.gitlens", orphans[0].Name)
}

func TestFindOrphans_ProviderFilter(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"htop"},
		},
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"eamodio.gitlens"},
		},
	}

	// Only check brew
	orphans := findOrphans(config, systemState, []string{"brew"}, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "brew", orphans[0].Provider)
}

func TestFindOrphans_IgnoreList(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"htop", "curl"},
		},
	}

	orphans := findOrphans(config, systemState, nil, []string{"htop"})
	assert.Len(t, orphans, 1)
	assert.Equal(t, "curl", orphans[0].Name)
}

func TestFindFileOrphans_ReturnsNil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, findFileOrphans(nil, nil, nil))
}

// ---------------------------------------------------------------------------
// removeOrphans tests
// ---------------------------------------------------------------------------

func TestRemoveOrphans_BrewFormula(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "formula", Name: "htop"},
	}

	output := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		assert.Equal(t, 1, removed)
		assert.Equal(t, 0, failed)
	})

	assert.Contains(t, output, "brew uninstall htop")
	assert.Contains(t, output, "Removed brew htop")
}

func TestRemoveOrphans_BrewCask(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "cask", Name: "slack"},
	}

	output := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		assert.Equal(t, 1, removed)
		assert.Equal(t, 0, failed)
	})

	assert.Contains(t, output, "brew uninstall --cask slack")
	assert.Contains(t, output, "Removed brew slack")
}

func TestRemoveOrphans_VSCode(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "vscode", Type: "extension", Name: "eamodio.gitlens"},
	}

	output := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		assert.Equal(t, 1, removed)
		assert.Equal(t, 0, failed)
	})

	assert.Contains(t, output, "code --uninstall-extension eamodio.gitlens")
	assert.Contains(t, output, "Removed vscode eamodio.gitlens")
}

func TestRemoveOrphans_Empty(t *testing.T) {
	removed, failed := removeOrphans(context.Background(), nil)
	assert.Equal(t, 0, removed)
	assert.Equal(t, 0, failed)
}

func TestRemoveOrphans_Mixed(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "formula", Name: "htop"},
		{Provider: "brew", Type: "cask", Name: "slack"},
		{Provider: "vscode", Type: "extension", Name: "gitlens"},
	}

	output := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		assert.Equal(t, 3, removed)
		assert.Equal(t, 0, failed)
	})

	assert.Contains(t, output, "Removed brew htop")
	assert.Contains(t, output, "Removed brew slack")
	assert.Contains(t, output, "Removed vscode gitlens")
}

// ---------------------------------------------------------------------------
// runBrewUninstall tests
// ---------------------------------------------------------------------------

func TestRunBrewUninstall_Formula(t *testing.T) {
	output := captureStdout(t, func() {
		err := runBrewUninstall("htop", false)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "brew uninstall htop")
	assert.NotContains(t, output, "--cask")
}

func TestRunBrewUninstall_Cask(t *testing.T) {
	output := captureStdout(t, func() {
		err := runBrewUninstall("slack", true)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "brew uninstall --cask slack")
}

// ---------------------------------------------------------------------------
// runVSCodeUninstall tests
// ---------------------------------------------------------------------------

func TestRunVSCodeUninstall(t *testing.T) {
	output := captureStdout(t, func() {
		err := runVSCodeUninstall("eamodio.gitlens")
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "code --uninstall-extension eamodio.gitlens")
}
