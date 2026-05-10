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
