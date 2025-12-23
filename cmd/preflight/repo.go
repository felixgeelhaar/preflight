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
  preflight repo init                    # Initialize repo
  preflight repo init --remote <url>     # With remote origin`,
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

var (
	repoRemote string
	repoForce  bool
	repoBranch string
)

func init() {
	repoInitCmd.Flags().StringVar(&repoRemote, "remote", "", "Remote repository URL")
	repoInitCmd.Flags().StringVar(&repoBranch, "branch", "main", "Branch name")
	repoPushCmd.Flags().BoolVar(&repoForce, "force", false, "Force push")

	repoCmd.AddCommand(repoInitCmd)
	repoCmd.AddCommand(repoStatusCmd)
	repoCmd.AddCommand(repoPushCmd)
	repoCmd.AddCommand(repoPullCmd)

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
