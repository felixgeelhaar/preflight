package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	cfgFile    string
	verbose    bool
	noAI       bool
	aiProvider string
	mode       string
	yesFlag    bool
)

var rootCmd = &cobra.Command{
	Use:   "preflight",
	Short: "A deterministic workstation compiler",
	Long: `Preflight compiles declarative configuration into a reproducible workstation setup.

It turns intent (targets, layers, capabilities) into a reproducible,
explainable local setup using the compiler model:
  Intent → Merge → Plan → Apply → Verify`,
	SilenceErrors: true, // We handle error formatting ourselves
	SilenceUsage:  true, // Don't show usage on error
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: preflight.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noAI, "no-ai", false, "disable AI features")
	rootCmd.PersistentFlags().StringVar(&aiProvider, "ai-provider", "", "AI provider (openai, anthropic, ollama)")
	rootCmd.PersistentFlags().StringVar(&mode, "mode", "intent", "reproducibility mode (intent, locked, frozen)")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "auto-confirm all prompts")

	// Register flag completions
	registerFlagCompletions()

	rootCmd.AddCommand(versionCmd)
}

// formatError returns a user-friendly error message.
// With verbose=false: shows only the user message and suggestion.
// With verbose=true: also shows the underlying technical error.
func formatError(err error) string {
	var userErr *config.UserError
	if errors.As(err, &userErr) {
		msg := userErr.Message
		if userErr.Context != "" {
			msg += fmt.Sprintf(" (at %s)", userErr.Context)
		}
		if userErr.Suggestion != "" {
			msg += fmt.Sprintf("\n\nSuggestion: %s", userErr.Suggestion)
		}
		if verbose && userErr.Underlying != nil {
			msg += fmt.Sprintf("\n\nTechnical details: %v", userErr.Underlying)
		}
		return msg
	}
	return err.Error()
}

// printError prints an error message to stderr with proper formatting.
func printError(err error) {
	printErrorTo(os.Stderr, err)
}

// printErrorTo prints an error message to the given writer.
func printErrorTo(w io.Writer, err error) {
	_, _ = fmt.Fprintf(w, "Error: %s\n", formatError(err))
}

// registerFlagCompletions sets up custom completions for global flags.
func registerFlagCompletions() {
	// Complete --config with YAML files
	_ = rootCmd.RegisterFlagCompletionFunc("config", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
	})

	// Complete --ai-provider with known providers
	_ = rootCmd.RegisterFlagCompletionFunc("ai-provider", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"openai\tOpenAI GPT models",
			"anthropic\tAnthropic Claude models",
			"ollama\tLocal Ollama models",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	// Complete --mode with reproducibility modes
	_ = rootCmd.RegisterFlagCompletionFunc("mode", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"intent\tInstall latest compatible versions",
			"locked\tPrefer lockfile, update intentionally",
			"frozen\tFail if resolution differs from lock",
		}, cobra.ShellCompDirectiveNoFileComp
	})
}
