package main

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for preflight.

To load completions:

Bash:
  $ source <(preflight completion bash)
  # To load completions for each session, add to ~/.bashrc:
  # source <(preflight completion bash)

Zsh:
  $ source <(preflight completion zsh)
  # To load completions for each session, add to ~/.zshrc:
  # source <(preflight completion zsh)
  # You may need to start a new shell for this to take effect.

Fish:
  $ preflight completion fish | source
  # To load completions for each session, run:
  $ preflight completion fish > ~/.config/fish/completions/preflight.fish

PowerShell:
  PS> preflight completion powershell | Out-String | Invoke-Expression
  # To load completions for each session, add the output to your profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(_ *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
