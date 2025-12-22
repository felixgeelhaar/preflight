package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "preflight",
	Short: "A deterministic workstation compiler",
	Long: `Preflight compiles declarative configuration into a reproducible workstation setup.

It turns intent (targets, layers, capabilities) into a reproducible,
explainable local setup using the compiler model:
  Intent → Merge → Plan → Apply → Verify`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
