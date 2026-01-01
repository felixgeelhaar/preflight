package main

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/domain/advisor/anthropic"
	"github.com/felixgeelhaar/preflight/internal/domain/advisor/gemini"
	"github.com/felixgeelhaar/preflight/internal/domain/advisor/openai"
	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new preflight configuration",
	Long: `Initialize a new preflight configuration using an interactive wizard.

The wizard will guide you through:
  - Selecting providers (nvim, shell, git, brew, etc.)
  - Choosing presets for each provider
  - Configuring targets and layers

Examples:
  preflight init                    # Interactive wizard
  preflight init --provider nvim    # Start with nvim provider
  preflight init --preset balanced  # Use balanced preset
  preflight init --yes              # Accept defaults`,
	RunE: runInit,
}

var (
	initProvider    string
	initPreset      string
	initSkipWelcome bool
	initYes         bool
	initNoAI        bool
)

func init() {
	initCmd.Flags().StringVar(&initProvider, "provider", "", "Pre-select a provider")
	initCmd.Flags().StringVar(&initPreset, "preset", "", "Pre-select a preset")
	initCmd.Flags().BoolVar(&initSkipWelcome, "skip-welcome", false, "Skip the welcome screen")
	initCmd.Flags().BoolVarP(&initYes, "yes", "y", false, "Accept all defaults")
	initCmd.Flags().BoolVar(&initNoAI, "no-ai", false, "Skip AI-guided interview")

	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	// Check if config already exists
	if _, err := os.Stat("preflight.yaml"); err == nil {
		fmt.Println("preflight.yaml already exists.")
		fmt.Println("Use 'preflight plan' to review your configuration.")
		return nil
	}

	// Build wizard options
	opts := tui.NewInitWizardOptions()

	if initProvider != "" {
		opts = opts.WithPreselectedProvider(initProvider)
	}
	if initPreset != "" {
		opts = opts.WithPreselectedPreset(initPreset)
	}
	if initSkipWelcome || initYes {
		opts = opts.WithSkipWelcome(true)
	}
	if initNoAI {
		opts = opts.WithSkipInterview(true)
	}

	// Detect AI provider from environment
	aiProvider := detectAIProvider()
	if aiProvider != nil && !initNoAI {
		opts = opts.WithAdvisor(aiProvider)
	}

	// Run the wizard
	ctx := context.Background()
	result, err := tui.RunInitWizard(ctx, opts)
	if err != nil {
		return fmt.Errorf("init wizard failed: %w", err)
	}

	if result.Cancelled {
		fmt.Println("Initialization cancelled.")
		return nil
	}

	fmt.Printf("Configuration created: %s\n", result.ConfigPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  preflight plan   - Review the execution plan")
	fmt.Println("  preflight apply  - Apply the configuration")

	return nil
}

// detectAIProvider returns an available AI provider from environment.
// Priority: explicit --ai-provider flag > ANTHROPIC_API_KEY > GEMINI_API_KEY > OPENAI_API_KEY
func detectAIProvider() advisor.AIProvider {
	// If --ai-provider flag is set, use that specific provider
	if aiProvider != "" {
		return getProviderByName(aiProvider)
	}

	// Check for Anthropic API key first (preferred)
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		provider := anthropic.NewProvider(apiKey)
		if provider.Available() {
			return provider
		}
	}

	// Check for Gemini API key (GEMINI_API_KEY or GOOGLE_API_KEY)
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		provider := gemini.NewProvider(apiKey)
		if provider.Available() {
			return provider
		}
	}
	if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
		provider := gemini.NewProvider(apiKey)
		if provider.Available() {
			return provider
		}
	}

	// Check for OpenAI API key
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider := openai.NewProvider(apiKey)
		if provider.Available() {
			return provider
		}
	}

	return nil
}

// getProviderByName returns a specific provider by name.
func getProviderByName(name string) advisor.AIProvider {
	switch name {
	case "anthropic":
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			return anthropic.NewProvider(apiKey)
		}
	case "gemini":
		if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
			return gemini.NewProvider(apiKey)
		}
		if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
			return gemini.NewProvider(apiKey)
		}
	case "openai":
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			return openai.NewProvider(apiKey)
		}
	}
	return nil
}
