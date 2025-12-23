package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information set by build flags.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, _ []string) {
		out := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(out, "preflight %s\n", version)
		_, _ = fmt.Fprintf(out, "  commit: %s\n", commit)
		_, _ = fmt.Fprintf(out, "  built:  %s\n", buildDate)
	},
}
