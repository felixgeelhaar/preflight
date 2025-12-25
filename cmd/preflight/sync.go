package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync configuration with remote and apply",
	Long: `Synchronize your local configuration with a remote repository and apply changes.

Sync combines git pull and preflight apply into a single command, making it
easy to keep multiple machines in sync with your configuration repository.

The sync process:
  1. Pull latest changes from remote
  2. Show what would change (plan)
  3. Apply changes (with confirmation)
  4. Optionally push local changes

This is useful for:
  - Bootstrapping a new machine from your dotfiles repo
  - Keeping work and personal machines synchronized
  - Applying team configuration updates

Examples:
  preflight sync                      # Pull and apply
  preflight sync --push               # Also push local changes
  preflight sync --dry-run            # Show what would happen
  preflight sync --remote origin      # Specify remote
  preflight sync --branch main        # Specify branch`,
	RunE: runSync,
}

var (
	syncConfigPath string
	syncTarget     string
	syncRemote     string
	syncBranch     string
	syncPush       bool
	syncDryRun     bool
	syncForce      bool
)

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().StringVarP(&syncConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	syncCmd.Flags().StringVarP(&syncTarget, "target", "t", "default", "Target to apply")
	syncCmd.Flags().StringVar(&syncRemote, "remote", "origin", "Git remote name")
	syncCmd.Flags().StringVar(&syncBranch, "branch", "", "Git branch (default: current branch)")
	syncCmd.Flags().BoolVar(&syncPush, "push", false, "Push local changes after apply")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show what would happen without making changes")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "Force sync even with uncommitted changes")
}

func runSync(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Find repository root
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Check for uncommitted changes
	if !syncForce {
		hasChanges, err := hasUncommittedChanges(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to check git status: %w", err)
		}
		if hasChanges {
			return fmt.Errorf("uncommitted changes detected. Commit or stash them, or use --force")
		}
	}

	fmt.Println("Syncing configuration...")
	fmt.Println()

	// Step 1: Fetch from remote
	fmt.Printf("1. Fetching from %s...\n", syncRemote)
	if !syncDryRun {
		if err := gitFetch(repoRoot, syncRemote); err != nil {
			return fmt.Errorf("failed to fetch: %w", err)
		}
	}

	// Determine branch
	branch := syncBranch
	if branch == "" {
		branch, err = getCurrentBranch(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	// Check if we're behind
	behind, ahead, err := getCommitDiff(repoRoot, syncRemote, branch)
	if err != nil {
		return fmt.Errorf("failed to check commit status: %w", err)
	}

	fmt.Printf("   Branch: %s\n", branch)
	fmt.Printf("   Behind: %d commits, Ahead: %d commits\n", behind, ahead)
	fmt.Println()

	// Step 2: Pull changes
	if behind > 0 {
		fmt.Printf("2. Pulling %d commit(s)...\n", behind)
		if !syncDryRun {
			if err := gitPull(repoRoot, syncRemote, branch); err != nil {
				return fmt.Errorf("failed to pull: %w", err)
			}
		}
	} else {
		fmt.Println("2. Already up to date.")
	}
	fmt.Println()

	// Step 3: Plan changes
	fmt.Println("3. Planning changes...")
	preflight := app.New(os.Stdout)

	configPath := filepath.Join(repoRoot, syncConfigPath)
	plan, err := preflight.Plan(ctx, configPath, syncTarget)
	if err != nil {
		return fmt.Errorf("failed to plan: %w", err)
	}

	if plan == nil || plan.IsEmpty() {
		fmt.Println("   No changes needed.")
	} else {
		fmt.Printf("   %d step(s) to apply\n", plan.Len())
		for _, entry := range plan.Entries() {
			fmt.Printf("   - %s\n", entry.Step().ID())
		}
	}
	fmt.Println()

	// Step 4: Apply changes
	if plan != nil && !plan.IsEmpty() {
		if syncDryRun {
			fmt.Println("4. Would apply changes (dry-run mode)")
		} else {
			fmt.Println("4. Applying changes...")
			if _, err := preflight.Apply(ctx, plan, false); err != nil {
				return fmt.Errorf("failed to apply: %w", err)
			}
			fmt.Println("   Applied successfully.")
		}
	} else {
		fmt.Println("4. Nothing to apply.")
	}
	fmt.Println()

	// Step 5: Push changes (if requested)
	if syncPush && ahead > 0 {
		fmt.Printf("5. Pushing %d commit(s)...\n", ahead)
		if !syncDryRun {
			if err := gitPush(repoRoot, syncRemote, branch); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
		}
	} else if syncPush {
		fmt.Println("5. Nothing to push.")
	}

	fmt.Println()
	fmt.Println("Sync complete!")

	return nil
}

func findRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func hasUncommittedChanges(repoRoot string) (bool, error) {
	cmd := exec.Command("git", "-C", repoRoot, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func getCurrentBranch(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func gitFetch(repoRoot, remote string) error {
	cmd := exec.Command("git", "-C", repoRoot, "fetch", remote)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getCommitDiff(repoRoot, remote, branch string) (behind, ahead int, err error) {
	// Get behind count
	cmd := exec.Command("git", "-C", repoRoot, "rev-list", "--count", fmt.Sprintf("HEAD..%s/%s", remote, branch))
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	_, _ = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &behind)

	// Get ahead count
	cmd = exec.Command("git", "-C", repoRoot, "rev-list", "--count", fmt.Sprintf("%s/%s..HEAD", remote, branch))
	output, err = cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	_, _ = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &ahead)

	return behind, ahead, nil
}

func gitPull(repoRoot, remote, branch string) error {
	cmd := exec.Command("git", "-C", repoRoot, "pull", remote, branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitPush(repoRoot, remote, branch string) error {
	cmd := exec.Command("git", "-C", repoRoot, "push", remote, branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
