package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/adapters/lockfile"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
	"github.com/spf13/cobra"
)

// ConflictJSON represents a conflict in JSON output format.
type ConflictJSON struct {
	PackageKey    string `json:"package_key"`
	Type          string `json:"type"`
	LocalVersion  string `json:"local_version"`
	RemoteVersion string `json:"remote_version"`
	Resolvable    bool   `json:"auto_resolvable"`
}

// ConflictsOutputJSON is the JSON output for sync conflicts command.
type ConflictsOutputJSON struct {
	Relation        string         `json:"relation"`
	TotalConflicts  int            `json:"total_conflicts"`
	AutoResolvable  int            `json:"auto_resolvable"`
	ManualConflicts []ConflictJSON `json:"manual_conflicts"`
	NeedsMerge      bool           `json:"needs_merge"`
}

var syncConflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "Show lockfile conflicts between local and remote",
	Long: `Detect and display conflicts between local and remote lockfile states.

Conflicts occur when both local and remote machines have modified the same
package independently. This command shows:
  - Which packages are in conflict
  - The local and remote versions
  - The conflict type (both modified, version mismatch, etc.)
  - Whether the conflict can be auto-resolved

Use 'preflight sync resolve' to resolve conflicts interactively.

Examples:
  preflight sync conflicts                  # Show all conflicts
  preflight sync conflicts --json           # Output as JSON
  preflight sync conflicts --auto-resolve   # Auto-resolve where possible`,
	RunE: runSyncConflicts,
}

var syncResolveCmd = &cobra.Command{
	Use:   "resolve [package-key]",
	Short: "Resolve lockfile conflicts",
	Long: `Interactively resolve conflicts between local and remote lockfiles.

When called without arguments, resolves all conflicts interactively.
When called with a package key, resolves that specific conflict.

Resolution strategies:
  --local   Keep local version
  --remote  Take remote version
  --newest  Take the most recently modified version (default auto)
  --skip    Skip this package (keep local, don't sync)

Examples:
  preflight sync resolve                       # Resolve all interactively
  preflight sync resolve brew:ripgrep          # Resolve specific package
  preflight sync resolve brew:ripgrep --local  # Keep local version
  preflight sync resolve --remote              # Take remote for all`,
	RunE: runSyncResolve,
}

var (
	conflictsJSON       bool
	conflictsAutoRes    bool
	resolveLocal        bool
	resolveRemote       bool
	resolveNewest       bool
	resolveSkip         bool
	conflictsLockPath   string
	conflictsRemotePath string
)

func init() {
	syncCmd.AddCommand(syncConflictsCmd)
	syncCmd.AddCommand(syncResolveCmd)

	// Conflicts flags
	syncConflictsCmd.Flags().BoolVar(&conflictsJSON, "json", false, "Output as JSON for CI/automation")
	syncConflictsCmd.Flags().BoolVar(&conflictsAutoRes, "auto-resolve", false, "Auto-resolve resolvable conflicts")
	syncConflictsCmd.Flags().StringVar(&conflictsLockPath, "lockfile", "preflight.lock", "Path to local lockfile")
	syncConflictsCmd.Flags().StringVar(&conflictsRemotePath, "remote-lockfile", "", "Path to remote lockfile (defaults to fetched version)")

	// Resolve flags
	syncResolveCmd.Flags().BoolVar(&resolveLocal, "local", false, "Keep local version for all conflicts")
	syncResolveCmd.Flags().BoolVar(&resolveRemote, "remote", false, "Take remote version for all conflicts")
	syncResolveCmd.Flags().BoolVar(&resolveNewest, "newest", false, "Take newest version for all conflicts")
	syncResolveCmd.Flags().BoolVar(&resolveSkip, "skip", false, "Skip all conflicts (keep local, don't sync)")
}

func runSyncConflicts(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Find repository root
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load local lockfile
	localPath := filepath.Join(repoRoot, conflictsLockPath)
	repo := lockfile.NewYAMLRepository()

	localLock, err := repo.Load(ctx, localPath)
	if err != nil {
		if errors.Is(err, lock.ErrLockfileNotFound) {
			fmt.Println("No local lockfile found. Nothing to compare.")
			return nil
		}
		return fmt.Errorf("failed to load local lockfile: %w", err)
	}

	// Determine remote lockfile path
	remotePath := conflictsRemotePath
	remoteTempFile := false
	if remotePath == "" {
		// Get from git stash of remote version
		remotePath, err = getRemoteLockfilePath(repoRoot, syncRemote, conflictsLockPath)
		if err != nil {
			return fmt.Errorf("failed to get remote lockfile: %w", err)
		}
		if remotePath == "" {
			fmt.Println("No remote lockfile found. Nothing to compare.")
			return nil
		}
		remoteTempFile = true
	}

	// Clean up temp file if we created it
	if remoteTempFile {
		defer func() { _ = os.Remove(remotePath) }()
	}

	// Load remote lockfile
	remoteLock, err := repo.Load(ctx, remotePath)
	if err != nil {
		if errors.Is(err, lock.ErrLockfileNotFound) {
			fmt.Println("No remote lockfile found. Nothing to compare.")
			return nil
		}
		return fmt.Errorf("failed to load remote lockfile: %w", err)
	}

	// Create sync adapter and engine
	adapter := lockfile.NewSyncAdapter()
	localState := adapter.ToLockfileState(localLock)
	remoteState := adapter.ToLockfileState(remoteLock)

	// Check causal relationship
	relation := adapter.CompareStates(localLock, remoteLock)
	needsMerge := adapter.NeedsMerge(localLock, remoteLock)

	// Handle JSON output for no-merge cases
	if conflictsJSON && !needsMerge {
		output := ConflictsOutputJSON{
			Relation:        relationString(relation),
			TotalConflicts:  0,
			AutoResolvable:  0,
			ManualConflicts: []ConflictJSON{},
			NeedsMerge:      false,
		}
		return printJSONOutput(output)
	}

	if !needsMerge {
		fmt.Printf("Local vs Remote: %s\n", relationString(relation))
		fmt.Println()
		switch relation {
		case sync.Equal:
			fmt.Println("Lockfiles are identical. No conflicts.")
		case sync.After:
			fmt.Println("Local is ahead of remote. Push to sync.")
		case sync.Before:
			fmt.Println("Local is behind remote. Pull to sync.")
		case sync.Concurrent:
			// This case won't be reached since NeedsMerge returns true for Concurrent,
			// but we include it for exhaustiveness.
		}
		return nil
	}

	// Detect conflicts
	engine := sync.NewSyncEngine()
	result, err := engine.Sync(sync.SyncInput{
		Local:  localState,
		Remote: remoteState,
	})
	if err != nil {
		return fmt.Errorf("failed to detect conflicts: %w", err)
	}

	// Build conflict list for output
	manualConflicts := make([]ConflictJSON, 0, len(result.ManualConflicts))
	for _, c := range result.ManualConflicts {
		manualConflicts = append(manualConflicts, ConflictJSON{
			PackageKey:    c.PackageKey(),
			Type:          c.Type().String(),
			LocalVersion:  c.Local().Version(),
			RemoteVersion: c.Remote().Version(),
			Resolvable:    c.IsResolvable(),
		})
	}

	totalConflicts := len(result.ManualConflicts) + result.Stats.ConflictsAutoResolved

	// JSON output
	if conflictsJSON {
		output := ConflictsOutputJSON{
			Relation:        relationString(relation),
			TotalConflicts:  totalConflicts,
			AutoResolvable:  result.Stats.ConflictsAutoResolved,
			ManualConflicts: manualConflicts,
			NeedsMerge:      true,
		}
		return printJSONOutput(output)
	}

	// Human-readable output
	fmt.Printf("Local vs Remote: %s\n", relationString(relation))
	fmt.Println()

	if !result.HasManualConflicts() && len(result.Resolutions) == 0 {
		fmt.Println("No conflicts detected.")
		return nil
	}

	fmt.Printf("Found %d conflict(s), %d auto-resolvable\n\n",
		totalConflicts, result.Stats.ConflictsAutoResolved)

	if len(result.ManualConflicts) > 0 {
		fmt.Println("Manual conflicts (require resolution):")
		printConflicts(result.ManualConflicts)
	}

	if result.Stats.ConflictsAutoResolved > 0 {
		fmt.Printf("\nAuto-resolvable conflicts: %d\n", result.Stats.ConflictsAutoResolved)
		if conflictsAutoRes {
			fmt.Println("These will be automatically resolved during sync.")
		} else {
			fmt.Println("Use --auto-resolve to resolve these automatically.")
		}
	}

	return nil
}

func printJSONOutput(output ConflictsOutputJSON) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func runSyncResolve(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	// Find repository root
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load lockfiles
	localPath := filepath.Join(repoRoot, conflictsLockPath)
	repo := lockfile.NewYAMLRepository()

	localLock, err := repo.Load(ctx, localPath)
	if err != nil {
		return fmt.Errorf("failed to load local lockfile: %w", err)
	}

	remotePath := conflictsRemotePath
	remoteTempFile := false
	if remotePath == "" {
		remotePath, err = getRemoteLockfilePath(repoRoot, syncRemote, conflictsLockPath)
		if err != nil {
			return fmt.Errorf("failed to get remote lockfile: %w", err)
		}
		remoteTempFile = true
	}

	// Clean up temp file if we created it
	if remoteTempFile {
		defer func() { _ = os.Remove(remotePath) }()
	}

	remoteLock, err := repo.Load(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("failed to load remote lockfile: %w", err)
	}

	// Create sync adapter and engine
	adapter := lockfile.NewSyncAdapter()
	localState := adapter.ToLockfileState(localLock)
	remoteState := adapter.ToLockfileState(remoteLock)

	// Get machine ID for the resolution
	machineID, err := sync.GetMachineID()
	if err != nil {
		return fmt.Errorf("failed to get machine ID: %w", err)
	}

	hostname, _ := os.Hostname()
	engine := sync.NewSyncEngine(sync.WithMachineID(machineID, hostname))

	// Detect conflicts
	result, err := engine.Sync(sync.SyncInput{
		Local:  localState,
		Remote: remoteState,
	})
	if err != nil {
		return fmt.Errorf("failed to detect conflicts: %w", err)
	}

	if !result.HasManualConflicts() {
		fmt.Println("No conflicts to resolve.")
		return nil
	}

	// Determine resolution strategy
	var choice sync.ResolutionChoice
	var useNewestStrategy bool
	switch {
	case resolveLocal:
		choice = sync.ChooseLocal
	case resolveRemote:
		choice = sync.ChooseRemote
	case resolveNewest:
		// Will determine choice per-conflict based on timestamps
		useNewestStrategy = true
	case resolveSkip:
		choice = sync.ChooseSkip
	default:
		// No strategy specified - show conflicts and guidance
		fmt.Printf("Found %d conflict(s) requiring resolution:\n\n", len(result.ManualConflicts))
		printConflicts(result.ManualConflicts)
		fmt.Println()

		fmt.Println("Resolution strategies:")
		fmt.Println("  --local   Keep your local versions (discard remote changes)")
		fmt.Println("  --remote  Accept remote versions (discard local changes)")
		fmt.Println("  --newest  Automatically choose the most recently modified version")
		fmt.Println("  --skip    Skip syncing these packages (keep local, exclude from sync)")
		fmt.Println()

		fmt.Println("Examples:")
		if len(result.ManualConflicts) > 0 {
			examplePkg := result.ManualConflicts[0].PackageKey()
			fmt.Printf("  preflight sync resolve --newest              # Auto-resolve all by timestamp\n")
			fmt.Printf("  preflight sync resolve %s --local   # Keep local for specific package\n", examplePkg)
			fmt.Printf("  preflight sync resolve --remote              # Accept all remote versions\n")
		} else {
			fmt.Println("  preflight sync resolve --newest              # Auto-resolve all by timestamp")
			fmt.Println("  preflight sync resolve brew:pkg --local      # Keep local for specific package")
		}
		return nil
	}

	// Filter conflicts if specific package requested
	conflictsToResolve := result.ManualConflicts
	if len(args) > 0 {
		targetKey := args[0]
		var filtered []sync.LockConflict
		for _, c := range conflictsToResolve {
			if c.PackageKey() == targetKey {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("no conflict found for package: %s", targetKey)
		}
		conflictsToResolve = filtered
	}

	// Resolve conflicts
	fmt.Println("Resolving conflicts:")
	for _, conflict := range conflictsToResolve {
		resolveChoice := choice
		chosenVersion := ""
		if useNewestStrategy {
			// Compare timestamps to determine newest
			localTime := conflict.Local().ModifiedAt()
			remoteTime := conflict.Remote().ModifiedAt()
			if localTime.After(remoteTime) {
				resolveChoice = sync.ChooseLocal
				chosenVersion = conflict.Local().Version()
			} else {
				resolveChoice = sync.ChooseRemote
				chosenVersion = conflict.Remote().Version()
			}
		} else {
			switch resolveChoice {
			case sync.ChooseLocal, sync.ChooseSkip:
				chosenVersion = conflict.Local().Version()
			case sync.ChooseRemote:
				chosenVersion = conflict.Remote().Version()
			case sync.ChooseBase:
				// ChooseBase not used in CLI resolution; falls back to local
				chosenVersion = conflict.Local().Version()
			}
		}
		if err := engine.ResolveManualConflict(result, conflict, resolveChoice); err != nil {
			return fmt.Errorf("failed to resolve %s: %w", conflict.PackageKey(), err)
		}
		fmt.Printf("  ✓ %s: %s -> %s (chose %s)\n",
			conflict.PackageKey(), conflict.Local().Version(), conflict.Remote().Version(), chosenVersion)
	}

	// Apply merged result back to lockfile (with remote for ChangeAdded support)
	mergedLock, err := adapter.ApplyMergeResultWithRemote(localLock, &sync.MergeResult{
		State:   result.Merged,
		Changes: nil, // Calculated during merge
	}, remoteLock)
	if err != nil {
		return fmt.Errorf("failed to apply resolution: %w", err)
	}

	// Save the merged lockfile
	if err := repo.Save(ctx, localPath, mergedLock); err != nil {
		return fmt.Errorf("failed to save lockfile: %w", err)
	}

	fmt.Printf("\nResolved %d conflict(s). Lockfile updated.\n", len(conflictsToResolve))
	fmt.Println("Run 'preflight sync --push' to share changes with remote.")
	return nil
}

func printConflicts(conflicts []sync.LockConflict) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PACKAGE\tTYPE\tLOCAL\tREMOTE\tRESOLVABLE")

	for _, c := range conflicts {
		resolvable := "no"
		if c.IsResolvable() {
			resolvable = "auto"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			c.PackageKey(),
			c.Type().String(),
			c.Local().Version(),
			c.Remote().Version(),
			resolvable)
	}
	_ = w.Flush()
}

func relationString(r sync.CausalRelation) string {
	switch r {
	case sync.Equal:
		return "equal (in sync)"
	case sync.Before:
		return "behind (pull needed)"
	case sync.After:
		return "ahead (push needed)"
	case sync.Concurrent:
		return "concurrent (merge needed)"
	default:
		return "unknown"
	}
}

func getRemoteLockfilePath(repoRoot, remote, lockPath string) (string, error) {
	// Get the current branch
	branch, err := getCurrentBranch(repoRoot)
	if err != nil {
		return "", err
	}

	// Create a temp file to store the remote lockfile
	tmpFile, err := os.CreateTemp("", "preflight-remote-*.lock")
	if err != nil {
		return "", err
	}
	defer func() { _ = tmpFile.Close() }()

	// Get the remote lockfile content using git show
	cmd := exec.Command("git", "-C", repoRoot, "show",
		fmt.Sprintf("%s/%s:%s", remote, branch, lockPath))
	output, err := cmd.Output()
	if err != nil {
		// Remote lockfile doesn't exist
		_ = os.Remove(tmpFile.Name())
		return "", nil
	}

	if _, err := tmpFile.Write(output); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}
