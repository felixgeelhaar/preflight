package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestOrganizeCommandGroups_AssignsCoreVerbs(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "preflight"}
	for _, name := range []string{"init", "plan", "apply", "doctor", "capture", "sync"} {
		root.AddCommand(&cobra.Command{Use: name})
	}
	organizeCommandGroups(root)

	for _, cmd := range root.Commands() {
		if cmd.GroupID != groupCore {
			t.Errorf("command %q GroupID = %q, want %q", cmd.Name(), cmd.GroupID, groupCore)
		}
		if cmd.Hidden {
			t.Errorf("core command %q must not be hidden", cmd.Name())
		}
	}
}

func TestOrganizeCommandGroups_HidesEnterprise(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "preflight"}
	for _, name := range []string{"fleet", "identity", "compliance", "marketplace", "mcp", "trust"} {
		root.AddCommand(&cobra.Command{Use: name})
	}
	organizeCommandGroups(root)

	for _, cmd := range root.Commands() {
		if cmd.GroupID != groupEnterprise {
			t.Errorf("command %q GroupID = %q, want %q", cmd.Name(), cmd.GroupID, groupEnterprise)
		}
		if !cmd.Hidden {
			t.Errorf("enterprise command %q should be hidden from default help", cmd.Name())
		}
	}
}

func TestOrganizeCommandGroups_LeavesUnknownsUngrouped(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "preflight"}
	root.AddCommand(&cobra.Command{Use: "version"})
	root.AddCommand(&cobra.Command{Use: "completion"})
	organizeCommandGroups(root)

	for _, cmd := range root.Commands() {
		if cmd.GroupID != "" {
			t.Errorf("non-categorized command %q got GroupID = %q", cmd.Name(), cmd.GroupID)
		}
		if cmd.Hidden {
			t.Errorf("non-categorized command %q must not be hidden", cmd.Name())
		}
	}
}

// TestOrganizeCommandGroups_RealRootIsFullyCategorized walks the actual
// rootCmd (after package init() registrations) and asserts every subcommand
// is either grouped or in the explicit allow-list of intentionally-ungrouped
// utilities. New top-level commands that forget to add themselves to a group
// will fail this test instead of silently appearing under "Additional".
func TestOrganizeCommandGroups_RealRootIsFullyCategorized(t *testing.T) {
	// Not parallel: organizeCommandGroups mutates rootCmd's groups state.
	allowUngrouped := map[string]struct{}{
		"version":    {},
		"completion": {},
		"help":       {},
	}

	organizeCommandGroups(rootCmd)

	for _, cmd := range rootCmd.Commands() {
		if _, ok := allowUngrouped[cmd.Name()]; ok {
			continue
		}
		if cmd.GroupID == "" {
			t.Errorf("command %q has no GroupID — add it to coreCommands / inspectCommands / configCommands / enterpriseCommands in root.go", cmd.Name())
		}
	}
}
