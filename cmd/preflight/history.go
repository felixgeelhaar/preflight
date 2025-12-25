package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show history of applied changes",
	Long: `Display the history of changes applied by preflight.

The history tracks all apply, rollback, and doctor --fix operations,
providing an audit trail of system modifications.

History is stored locally in ~/.preflight/history/ and includes:
  - Timestamp of each operation
  - Command executed
  - Changes made (files, packages, etc.)
  - Success/failure status

Examples:
  preflight history                   # Show recent history
  preflight history --limit 50        # Show more entries
  preflight history --since 7d        # Last 7 days
  preflight history --json            # JSON output
  preflight history --provider brew   # Filter by provider
  preflight history clear             # Clear history`,
	RunE: runHistory,
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear history",
	Long:  `Remove all history entries. This cannot be undone.`,
	RunE:  runHistoryClear,
}

var (
	historyLimit    int
	historySince    string
	historyJSON     bool
	historyProvider string
	historyVerbose  bool
)

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.AddCommand(historyClearCmd)

	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "Maximum entries to show")
	historyCmd.Flags().StringVar(&historySince, "since", "", "Show entries since (e.g., 1h, 7d, 2w)")
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "Output as JSON")
	historyCmd.Flags().StringVar(&historyProvider, "provider", "", "Filter by provider")
	historyCmd.Flags().BoolVarP(&historyVerbose, "verbose", "v", false, "Show detailed output")
}

// HistoryEntry represents a single history entry
type HistoryEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Command   string    `json:"command"`
	Target    string    `json:"target,omitempty"`
	Status    string    `json:"status"` // "success", "failed", "partial"
	Duration  string    `json:"duration,omitempty"`
	Changes   []Change  `json:"changes,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// Change represents a single change made
type Change struct {
	Provider string `json:"provider"`
	Action   string `json:"action"` // "install", "remove", "update", "create", "modify", "delete"
	Item     string `json:"item"`
	Details  string `json:"details,omitempty"`
}

func runHistory(_ *cobra.Command, _ []string) error {
	entries, err := loadHistory()
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Filter by time
	if historySince != "" {
		since, err := parseDuration(historySince)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		cutoff := time.Now().Add(-since)
		var filtered []HistoryEntry
		for _, e := range entries {
			if e.Timestamp.After(cutoff) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// Filter by provider
	if historyProvider != "" {
		var filtered []HistoryEntry
		for _, e := range entries {
			for _, c := range e.Changes {
				if c.Provider == historyProvider {
					filtered = append(filtered, e)
					break
				}
			}
		}
		entries = filtered
	}

	// Sort by timestamp descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	// Apply limit
	if historyLimit > 0 && len(entries) > historyLimit {
		entries = entries[:historyLimit]
	}

	if len(entries) == 0 {
		fmt.Println("No history entries found.")
		return nil
	}

	if historyJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	outputHistoryText(entries)
	return nil
}

func runHistoryClear(_ *cobra.Command, _ []string) error {
	historyDir := getHistoryDir()

	if err := os.RemoveAll(historyDir); err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	fmt.Println("History cleared.")
	return nil
}

func getHistoryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".preflight", "history")
}

func loadHistory() ([]HistoryEntry, error) {
	historyDir := getHistoryDir()

	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(historyDir)
	if err != nil {
		return nil, err
	}

	entries := make([]HistoryEntry, 0, len(files))
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(historyDir, f.Name()))
		if err != nil {
			continue
		}

		var entry HistoryEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return 0, err
	}

	switch unit {
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit: %c (use h, d, w, m)", unit)
	}
}

func outputHistoryText(entries []HistoryEntry) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if historyVerbose {
		for i, e := range entries {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("─── %s ───\n", e.ID)
			fmt.Printf("  Time:     %s (%s ago)\n", e.Timestamp.Format("2006-01-02 15:04:05"), formatHistoryAge(e.Timestamp))
			fmt.Printf("  Command:  %s\n", e.Command)
			if e.Target != "" {
				fmt.Printf("  Target:   %s\n", e.Target)
			}
			fmt.Printf("  Status:   %s\n", formatStatus(e.Status))
			if e.Duration != "" {
				fmt.Printf("  Duration: %s\n", e.Duration)
			}
			if len(e.Changes) > 0 {
				fmt.Printf("  Changes:\n")
				for _, c := range e.Changes {
					fmt.Printf("    [%s] %s: %s\n", c.Provider, c.Action, c.Item)
				}
			}
			if e.Error != "" {
				fmt.Printf("  Error:    %s\n", e.Error)
			}
		}
	} else {
		_, _ = fmt.Fprintln(w, "TIME\tCOMMAND\tTARGET\tSTATUS\tCHANGES")
		for _, e := range entries {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
				formatHistoryAge(e.Timestamp),
				e.Command,
				e.Target,
				formatStatus(e.Status),
				len(e.Changes),
			)
		}
		_ = w.Flush()
	}

	fmt.Printf("\nShowing %d entries\n", len(entries))
}

func formatStatus(status string) string {
	switch status {
	case "success":
		return "✓ success"
	case "failed":
		return "✗ failed"
	case "partial":
		return "~ partial"
	default:
		return status
	}
}

func formatHistoryAge(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	if d < 30*24*time.Hour {
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	}
	return t.Format("Jan 2")
}

// SaveHistoryEntry saves a history entry (called by other commands)
func SaveHistoryEntry(entry HistoryEntry) error {
	historyDir := getHistoryDir()

	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return err
	}

	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.json", entry.ID)
	return os.WriteFile(filepath.Join(historyDir, filename), data, 0o644)
}
