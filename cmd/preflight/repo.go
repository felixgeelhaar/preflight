package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage configuration repository",
	Long: `Repo manages your configuration as a git repository.

This enables versioning, sharing, and syncing configurations across machines.

Subcommands:
  init    - Initialize configuration as a git repository
  status  - Show repository status
  push    - Push configuration changes
  pull    - Pull configuration updates`,
}

var repoInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration as a git repository",
	Long: `Initialize your preflight configuration directory as a git repository.

This enables version control and sharing of your configuration.

Examples:
  preflight repo init                        # Initialize local repo
  preflight repo init --remote <url>         # With remote origin
  preflight repo init --github               # Create GitHub repo (requires gh CLI)
  preflight repo init --github --public      # Create public GitHub repo`,
	RunE: runRepoInit,
}

var repoStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository status",
	Long: `Show the current status of your configuration repository.

Examples:
  preflight repo status`,
	RunE: runRepoStatus,
}

var repoPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push configuration changes",
	Long: `Push your configuration changes to the remote repository.

Examples:
  preflight repo push           # Push to origin
  preflight repo push --force   # Force push`,
	RunE: runRepoPush,
}

var repoPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull configuration updates",
	Long: `Pull configuration updates from the remote repository.

Examples:
  preflight repo pull`,
	RunE: runRepoPull,
}

var repoCloneCmd = &cobra.Command{
	Use:   "clone <url> [path]",
	Short: "Clone and bootstrap a configuration repository",
	Long: `Clone a preflight configuration repository and optionally apply it.

This is the recommended way to bootstrap a new machine from an existing
configuration repository.

Examples:
  preflight repo clone git@github.com:user/dotfiles.git
  preflight repo clone git@github.com:user/dotfiles.git ~/dotfiles
  preflight repo clone git@github.com:user/dotfiles.git --apply
  preflight repo clone git@github.com:user/dotfiles.git --yes`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRepoClone,
}

var (
	repoRemote      string
	repoForce       bool
	repoBranch      string
	repoGitHub      bool
	repoPrivate     bool
	repoName        string
	repoApply       bool
	repoAutoConfirm bool
	repoTarget      string
)

func init() {
	repoInitCmd.Flags().StringVar(&repoRemote, "remote", "", "Remote repository URL")
	repoInitCmd.Flags().StringVar(&repoBranch, "branch", "main", "Branch name")
	repoInitCmd.Flags().BoolVar(&repoGitHub, "github", false, "Create GitHub repository (requires gh CLI)")
	repoInitCmd.Flags().BoolVar(&repoPrivate, "private", true, "Create private repository (default: true)")
	repoInitCmd.Flags().StringVar(&repoName, "name", "", "Repository name (default: directory name)")
	repoPushCmd.Flags().BoolVar(&repoForce, "force", false, "Force push")
	repoCloneCmd.Flags().BoolVar(&repoApply, "apply", false, "Apply configuration after cloning")
	repoCloneCmd.Flags().BoolVarP(&repoAutoConfirm, "yes", "y", false, "Skip confirmation prompts")
	repoCloneCmd.Flags().StringVarP(&repoTarget, "target", "t", "", "Target configuration to apply")

	repoCmd.AddCommand(repoInitCmd)
	repoCmd.AddCommand(repoStatusCmd)
	repoCmd.AddCommand(repoPushCmd)
	repoCmd.AddCommand(repoPullCmd)
	repoCmd.AddCommand(repoCloneCmd)

	rootCmd.AddCommand(repoCmd)
}

func getConfigDir() string {
	configPath := cfgFile
	if configPath == "" {
		configPath = "preflight.yaml"
	}
	return filepath.Dir(configPath)
}

func runRepoInit(_ *cobra.Command, _ []string) error {
	configDir := getConfigDir()
	if configDir == "" || configDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		configDir = cwd
	}

	ctx := context.Background()
	preflight := app.New(os.Stdout)

	// Handle GitHub repository creation
	if repoGitHub {
		name := repoName
		if name == "" {
			name = filepath.Base(configDir)
		}

		opts := app.GitHubRepoOptions{
			Path:        configDir,
			Name:        name,
			Description: "Dotfiles and machine configuration managed by preflight",
			Private:     repoPrivate,
			Branch:      repoBranch,
		}

		if err := preflight.RepoInitGitHub(ctx, opts); err != nil {
			return fmt.Errorf("GitHub repo init failed: %w", err)
		}

		return nil
	}

	// Standard git init
	opts := app.NewRepoOptions(configDir).
		WithBranch(repoBranch)

	if repoRemote != "" {
		opts = opts.WithRemote(repoRemote)
	}

	if err := preflight.RepoInit(ctx, opts); err != nil {
		return fmt.Errorf("repo init failed: %w", err)
	}

	return nil
}

func runRepoStatus(_ *cobra.Command, _ []string) error {
	configDir := getConfigDir()
	if configDir == "" || configDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		configDir = cwd
	}

	ctx := context.Background()
	preflight := app.New(os.Stdout)

	status, err := preflight.RepoStatus(ctx, configDir)
	if err != nil {
		return fmt.Errorf("repo status failed: %w", err)
	}

	preflight.PrintRepoStatus(status)
	return nil
}

func runRepoPush(_ *cobra.Command, _ []string) error {
	configDir := getConfigDir()
	if configDir == "" || configDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		configDir = cwd
	}

	fmt.Println("Pushing configuration...")

	args := []string{"-C", configDir, "push"}
	if repoForce {
		args = append(args, "--force")
	}

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	fmt.Println("Configuration pushed successfully.")
	return nil
}

func runRepoPull(_ *cobra.Command, _ []string) error {
	configDir := getConfigDir()
	if configDir == "" || configDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		configDir = cwd
	}

	fmt.Println("Pulling configuration updates...")

	cmd := exec.Command("git", "-C", configDir, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	fmt.Println("Configuration updated.")
	return nil
}

func runRepoClone(cmd *cobra.Command, args []string) error {
	url := args[0]
	path := ""
	if len(args) > 1 {
		path = args[1]
	}

	ctx := context.Background()
	preflight := app.New(os.Stdout)
	if modeOverride, err := resolveModeOverride(cmd); err != nil {
		return err
	} else if modeOverride != nil {
		preflight.WithMode(*modeOverride)
	}

	opts := app.CloneOptions{
		URL:            url,
		Path:           path,
		Apply:          repoApply,
		AutoConfirm:    repoAutoConfirm,
		AllowBootstrap: allowBootstrapFlag,
		Target:         repoTarget,
	}

	result, err := preflight.RepoClone(ctx, opts)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	if result.Applied {
		fmt.Printf("\n✓ Configuration applied successfully from %s\n", result.Path)
	} else if result.ConfigFound {
		fmt.Printf("\nRepository cloned to %s\n", result.Path)
		fmt.Println("Run 'preflight apply' to apply the configuration.")
	}

	return nil
}
