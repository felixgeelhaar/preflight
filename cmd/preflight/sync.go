package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/felixgeelhaar/preflight/internal/adapters/lockfile"
	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
	"github.com/felixgeelhaar/preflight/internal/validation"
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

func runSync(cmd *cobra.Command, _ []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Find repository root
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	if err := validation.ValidateGitRemoteName(syncRemote); err != nil {
		return fmt.Errorf("invalid remote name: %w", err)
	}
	if syncBranch != "" {
		if err := validation.ValidateGitBranch(syncBranch); err != nil {
			return fmt.Errorf("invalid branch: %w", err)
		}
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
	if err := validation.ValidateGitBranch(branch); err != nil {
		return fmt.Errorf("invalid current branch: %w", err)
	}

	// Step 1.5: Check for lockfile conflicts
	lockPath := filepath.Join(repoRoot, "preflight.lock")
	conflicts, err := checkLockfileConflicts(ctx, repoRoot, lockPath, syncRemote, branch)
	if err != nil {
		// Non-fatal: lockfile may not exist yet
		if !errors.Is(err, lock.ErrLockfileNotFound) {
			fmt.Printf("   Warning: could not check lockfile conflicts: %v\n", err)
		}
	} else if conflicts != nil && conflicts.HasManualConflicts() {
		fmt.Println()
		fmt.Printf("⚠️  Lockfile conflict detected! %d package(s) have conflicting versions.\n", len(conflicts.ManualConflicts))
		fmt.Println("   Run 'preflight sync conflicts' to see details.")
		fmt.Println("   Run 'preflight sync resolve' to resolve conflicts before syncing.")
		if !syncForce {
			return fmt.Errorf("lockfile conflicts detected. Use --force to proceed anyway")
		}
		fmt.Println("   Proceeding with --force (local versions will be kept)...")
	}
	fmt.Println()

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
	if modeOverride, err := resolveModeOverride(cmd); err != nil {
		return err
	} else if modeOverride != nil {
		preflight.WithMode(*modeOverride)
	}

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
			if app.RequiresBootstrapConfirmation(plan) {
				steps := app.BootstrapSteps(plan)
				if !confirmBootstrap(steps) {
					return fmt.Errorf("aborted bootstrap steps")
				}
			}
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
	// #nosec G204 -- repoRoot is derived from git and validated by caller.
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func gitFetch(repoRoot, remote string) error {
	// #nosec G204 -- repoRoot and remote are validated by caller.
	cmd := exec.Command("git", "-C", repoRoot, "fetch", remote)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getCommitDiff(repoRoot, remote, branch string) (behind, ahead int, err error) {
	// Get behind count
	// #nosec G204 -- repoRoot, remote, and branch are validated by caller.
	cmd := exec.Command("git", "-C", repoRoot, "rev-list", "--count", fmt.Sprintf("HEAD..%s/%s", remote, branch))
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	_, _ = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &behind)

	// Get ahead count
	// #nosec G204 -- repoRoot, remote, and branch are validated by caller.
	cmd = exec.Command("git", "-C", repoRoot, "rev-list", "--count", fmt.Sprintf("%s/%s..HEAD", remote, branch))
	output, err = cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	_, _ = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &ahead)

	return behind, ahead, nil
}

func gitPull(repoRoot, remote, branch string) error {
	// #nosec G204 -- repoRoot, remote, and branch are validated by caller.
	cmd := exec.Command("git", "-C", repoRoot, "pull", remote, branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitPush(repoRoot, remote, branch string) error {
	// #nosec G204 -- repoRoot, remote, and branch are validated by caller.
	cmd := exec.Command("git", "-C", repoRoot, "push", remote, branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// checkLockfileConflicts compares local and remote lockfiles for conflicts.
// Returns nil if no lockfile exists or if there are no conflicts.
// The branch parameter is unused because getRemoteLockfilePath computes it internally.
func checkLockfileConflicts(ctx context.Context, repoRoot, lockPath, remote, _ string) (*sync.SyncResult, error) {
	repo := lockfile.NewYAMLRepository()

	// Load local lockfile
	localLock, err := repo.Load(ctx, lockPath)
	if err != nil {
		return nil, err
	}

	// Get remote lockfile via git show
	remoteLockPath, err := getRemoteLockfilePath(repoRoot, remote, filepath.Base(lockPath))
	if err != nil {
		return nil, err
	}
	if remoteLockPath == "" {
		// No remote lockfile - no conflicts possible
		return nil, nil
	}
	defer func() { _ = os.Remove(remoteLockPath) }()

	remoteLock, err := repo.Load(ctx, remoteLockPath)
	if err != nil {
		return nil, err
	}

	// Convert to sync domain types
	adapter := lockfile.NewSyncAdapter()
	localState := adapter.ToLockfileState(localLock)
	remoteState := adapter.ToLockfileState(remoteLock)

	// Check if merge is needed
	if !adapter.NeedsMerge(localLock, remoteLock) {
		return nil, nil
	}

	// Run sync engine to detect conflicts
	engine := sync.NewSyncEngine()
	result, err := engine.Sync(sync.SyncInput{
		Local:  localState,
		Remote: remoteState,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
