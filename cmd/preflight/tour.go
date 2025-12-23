package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tourCmd = &cobra.Command{
	Use:   "tour [topic]",
	Short: "Interactive guided walkthroughs",
	Long: `Tour provides interactive guided walkthroughs of preflight features.

Available tours:
  basics      - Introduction to preflight concepts
  config      - Understanding configuration structure
  layers      - Working with layers and composition
  providers   - Available providers and their options
  ai          - Using AI-powered suggestions

Examples:
  preflight tour            # List available tours
  preflight tour basics     # Start the basics tour
  preflight tour providers  # Learn about providers`,
	RunE: runTour,
}

func init() {
	rootCmd.AddCommand(tourCmd)
}

func runTour(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		fmt.Println("Available tours:")
		fmt.Println()
		fmt.Println("  basics      Introduction to preflight concepts")
		fmt.Println("  config      Understanding configuration structure")
		fmt.Println("  layers      Working with layers and composition")
		fmt.Println("  providers   Available providers and their options")
		fmt.Println("  ai          Using AI-powered suggestions")
		fmt.Println()
		fmt.Println("Run 'preflight tour <name>' to start a tour.")
		return nil
	}

	topic := args[0]

	switch topic {
	case "basics":
		fmt.Println("Welcome to the Preflight Basics Tour!")
		fmt.Println()
		fmt.Println("Preflight is a deterministic workstation compiler.")
		fmt.Println("It turns declarative configuration into reproducible setups.")
		fmt.Println()
		fmt.Println("Key concepts:")
		fmt.Println("  - Manifest: Your main preflight.yaml file")
		fmt.Println("  - Layers: Composable configuration overlays")
		fmt.Println("  - Targets: Different machine profiles")
		fmt.Println("  - Providers: Integrations (brew, git, nvim, etc.)")
		fmt.Println()
		fmt.Println("Run 'preflight init' to get started!")

	case "config":
		fmt.Println("Configuration Structure Tour")
		fmt.Println()
		fmt.Println("preflight.yaml is your main configuration file:")
		fmt.Println("  version: 1")
		fmt.Println("  targets:")
		fmt.Println("    - name: default")
		fmt.Println("      layers: [base, work]")
		fmt.Println()
		fmt.Println("Layers live in layers/*.yaml and are merged together.")

	case "layers":
		fmt.Println("Layers Tour")
		fmt.Println()
		fmt.Println("Layers provide composition and reuse:")
		fmt.Println("  base.yaml    - Common configuration")
		fmt.Println("  work.yaml    - Work-specific settings")
		fmt.Println("  personal.yaml - Personal overrides")
		fmt.Println()
		fmt.Println("Merge semantics:")
		fmt.Println("  - Scalars: last wins")
		fmt.Println("  - Maps: deep merge")
		fmt.Println("  - Lists: set union with add/remove")

	case "providers":
		fmt.Println("Providers Tour")
		fmt.Println()
		fmt.Println("Available providers:")
		fmt.Println("  brew    - Homebrew packages (macOS)")
		fmt.Println("  apt     - APT packages (Linux)")
		fmt.Println("  files   - Dotfile management")
		fmt.Println("  git     - Git configuration")
		fmt.Println("  ssh     - SSH config generation")
		fmt.Println("  nvim    - Neovim setup")
		fmt.Println("  vscode  - VS Code extensions")
		fmt.Println("  runtime - Language version management")
		fmt.Println("  shell   - Shell configuration")

	case "ai":
		fmt.Println("AI Features Tour")
		fmt.Println()
		fmt.Println("Preflight supports AI-powered suggestions (BYOK).")
		fmt.Println()
		fmt.Println("Set your API key:")
		fmt.Println("  export PREFLIGHT_OPENAI_API_KEY=sk-...")
		fmt.Println("  export PREFLIGHT_ANTHROPIC_API_KEY=sk-ant-...")
		fmt.Println("  export PREFLIGHT_OLLAMA_ENDPOINT=http://localhost:11434")
		fmt.Println()
		fmt.Println("AI can help with:")
		fmt.Println("  - Preset recommendations")
		fmt.Println("  - Configuration explanations")
		fmt.Println("  - Troubleshooting suggestions")

	default:
		return fmt.Errorf("unknown tour: %s", topic)
	}

	return nil
}
