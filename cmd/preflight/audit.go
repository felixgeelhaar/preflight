package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/audit"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:     "audit",
	Aliases: []string{"log", "logs"},
	Short:   "View security audit logs",
	Long: `View and query the security audit log for plugin operations.

The audit log records all security-relevant plugin operations including:
  - Catalog installation, removal, and verification
  - Plugin installation, execution, and uninstallation
  - Trust changes (key additions/removals)
  - Capability grants and denials
  - Sandbox violations

Examples:
  preflight audit                           # Show recent events
  preflight audit --limit 50                # Show last 50 events
  preflight audit --type catalog_installed  # Filter by event type
  preflight audit --severity critical       # Filter by severity
  preflight audit --catalog my-catalog      # Filter by catalog
  preflight audit --plugin my-plugin        # Filter by plugin
  preflight audit --days 7                  # Events from last 7 days
  preflight audit --failures                # Show only failures
  preflight audit summary                   # Show summary statistics
  preflight audit security                  # Show security events only
  preflight audit --json                    # Output as JSON`,
}

var auditShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show audit events (default)",
	Long: `Display audit events matching the specified filters.

This is the default subcommand when running 'preflight audit' without arguments.`,
	RunE: runAuditShow,
}

var auditSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show audit log summary",
	Long: `Display a summary of audit events including:
  - Total event counts by type and severity
  - Events by catalog and plugin
  - Time range of logged events
  - Security event counts

Examples:
  preflight audit summary
  preflight audit summary --days 30`,
	RunE: runAuditSummary,
}

var auditSecurityCmd = &cobra.Command{
	Use:   "security",
	Short: "Show security-related events",
	Long: `Display only security-related audit events:
  - Capability denials
  - Sandbox violations
  - Signature verification failures
  - Security audit results

Examples:
  preflight audit security
  preflight audit security --days 7`,
	RunE: runAuditSecurity,
}

var auditCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up old audit logs",
	Long: `Remove audit log files older than the retention period.

The default retention period is 90 days.

Examples:
  preflight audit clean`,
	RunE: runAuditClean,
}

// Flags
var (
	auditLimit      int
	auditDays       int
	auditEventType  string
	auditSeverity   string
	auditCatalog    string
	auditPlugin     string
	auditUser       string
	auditFailures   bool
	auditSuccesses  bool
	auditOutputJSON bool
)

func init() {
	// Flags for show command (also inherited by audit root)
	auditCmd.PersistentFlags().IntVarP(&auditLimit, "limit", "n", 20, "Maximum number of events to show")
	auditCmd.PersistentFlags().IntVarP(&auditDays, "days", "d", 0, "Show events from last N days")
	auditCmd.PersistentFlags().StringVarP(&auditEventType, "type", "t", "", "Filter by event type")
	auditCmd.PersistentFlags().StringVarP(&auditSeverity, "severity", "s", "", "Filter by severity (info, warning, error, critical)")
	auditCmd.PersistentFlags().StringVarP(&auditCatalog, "catalog", "c", "", "Filter by catalog name")
	auditCmd.PersistentFlags().StringVarP(&auditPlugin, "plugin", "p", "", "Filter by plugin name")
	auditCmd.PersistentFlags().StringVarP(&auditUser, "user", "u", "", "Filter by user")
	auditCmd.PersistentFlags().BoolVar(&auditFailures, "failures", false, "Show only failed events")
	auditCmd.PersistentFlags().BoolVar(&auditSuccesses, "successes", false, "Show only successful events")
	auditCmd.PersistentFlags().BoolVarP(&auditOutputJSON, "json", "j", false, "Output as JSON")

	// Add subcommands
	auditCmd.AddCommand(auditShowCmd)
	auditCmd.AddCommand(auditSummaryCmd)
	auditCmd.AddCommand(auditSecurityCmd)
	auditCmd.AddCommand(auditCleanCmd)

	// Set default command to show
	auditCmd.RunE = runAuditShow

	rootCmd.AddCommand(auditCmd)
}

// getAuditService returns the audit service with file logger.
func getAuditService() (*audit.Service, error) {
	config := audit.DefaultFileLoggerConfig()
	logger, err := audit.NewFileLogger(config)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	return audit.NewService(logger), nil
}

// buildFilter constructs a QueryFilter from command flags.
func buildFilter() audit.QueryFilter {
	builder := audit.NewQuery()

	if auditLimit > 0 {
		builder.Limit(auditLimit)
	}

	if auditDays > 0 {
		builder.LastDays(auditDays)
	}

	if auditEventType != "" {
		builder.WithEventTypes(audit.EventType(auditEventType))
	}

	if auditSeverity != "" {
		builder.WithSeverities(audit.Severity(auditSeverity))
	}

	if auditCatalog != "" {
		builder.WithCatalog(auditCatalog)
	}

	if auditPlugin != "" {
		builder.WithPlugin(auditPlugin)
	}

	if auditUser != "" {
		builder.WithUser(auditUser)
	}

	if auditFailures {
		builder.FailuresOnly()
	}

	if auditSuccesses {
		builder.SuccessOnly()
	}

	return builder.Build()
}

func runAuditShow(_ *cobra.Command, _ []string) error {
	service, err := getAuditService()
	if err != nil {
		return err
	}
	defer func() { _ = service.Close() }()

	ctx := context.Background()
	filter := buildFilter()

	events, err := service.Query(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query audit log: %w", err)
	}

	if len(events) == 0 {
		fmt.Println("No audit events found matching the criteria.")
		return nil
	}

	if auditOutputJSON {
		return outputEventsJSON(events)
	}

	return outputEventsTable(events)
}

func runAuditSummary(_ *cobra.Command, _ []string) error {
	service, err := getAuditService()
	if err != nil {
		return err
	}
	defer func() { _ = service.Close() }()

	ctx := context.Background()

	// Build filter without limit for summary
	limitBackup := auditLimit
	auditLimit = 0
	filter := buildFilter()
	auditLimit = limitBackup

	summary, err := service.Summary(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get summary: %w", err)
	}

	if auditOutputJSON {
		return outputJSON(summary)
	}

	fmt.Println("Audit Log Summary")
	fmt.Println("=================")
	fmt.Println()

	fmt.Printf("Total Events:    %d\n", summary.TotalEvents)
	fmt.Printf("Successful:      %d\n", summary.SuccessCount)
	fmt.Printf("Failed:          %d\n", summary.FailureCount)
	fmt.Printf("Critical:        %d\n", summary.CriticalCount)
	fmt.Printf("Security Events: %d\n", summary.SecurityEvents)
	fmt.Println()

	if !summary.FirstEvent.IsZero() {
		fmt.Printf("Time Range: %s to %s\n",
			summary.FirstEvent.Format(time.RFC3339),
			summary.LastEvent.Format(time.RFC3339),
		)
		fmt.Println()
	}

	if len(summary.BySeverity) > 0 {
		fmt.Println("By Severity:")
		for severity, count := range summary.BySeverity {
			fmt.Printf("  %-10s %d\n", severity, count)
		}
		fmt.Println()
	}

	if len(summary.ByType) > 0 {
		fmt.Println("By Event Type:")
		for eventType, count := range summary.ByType {
			fmt.Printf("  %-25s %d\n", eventType, count)
		}
		fmt.Println()
	}

	if len(summary.ByCatalog) > 0 {
		fmt.Println("By Catalog:")
		for catalog, count := range summary.ByCatalog {
			fmt.Printf("  %-20s %d\n", catalog, count)
		}
		fmt.Println()
	}

	if len(summary.ByPlugin) > 0 {
		fmt.Println("By Plugin:")
		for plugin, count := range summary.ByPlugin {
			fmt.Printf("  %-20s %d\n", plugin, count)
		}
	}

	return nil
}

func runAuditSecurity(_ *cobra.Command, _ []string) error {
	service, err := getAuditService()
	if err != nil {
		return err
	}
	defer func() { _ = service.Close() }()

	ctx := context.Background()

	days := auditDays
	if days == 0 {
		days = 30 // Default to 30 days for security events
	}

	events, err := service.SecurityEvents(ctx, days)
	if err != nil {
		return fmt.Errorf("failed to query security events: %w", err)
	}

	// Apply limit if set
	if auditLimit > 0 && len(events) > auditLimit {
		events = events[:auditLimit]
	}

	if len(events) == 0 {
		fmt.Printf("No security events in the last %d days.\n", days)
		return nil
	}

	fmt.Printf("Security events (last %d days):\n\n", days)

	if auditOutputJSON {
		return outputEventsJSON(events)
	}

	return outputSecurityEventsTable(events)
}

func runAuditClean(_ *cobra.Command, _ []string) error {
	config := audit.DefaultFileLoggerConfig()
	logger, err := audit.NewFileLogger(config)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}
	defer func() { _ = logger.Close() }()

	if err := logger.Cleanup(); err != nil {
		return fmt.Errorf("failed to clean audit logs: %w", err)
	}

	fmt.Println("Audit log cleanup complete.")
	return nil
}

//nolint:unparam // error return reserved for future use
func outputEventsTable(events []audit.Event) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TIME\tEVENT\tSEVERITY\tSUBJECT\tSTATUS")

	for _, e := range events {
		subject := e.Catalog
		if subject == "" {
			subject = e.Plugin
		}
		if subject == "" {
			subject = "-"
		}

		status := "✓"
		if !e.Success {
			status = "✗"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.Timestamp.Format("2006-01-02 15:04"),
			e.Type,
			e.Severity,
			truncateStr(subject, 20),
			status,
		)
	}

	_ = w.Flush()
	fmt.Printf("\nShowing %d events\n", len(events))
	return nil
}

//nolint:unparam // error return reserved for future use
func outputSecurityEventsTable(events []audit.Event) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TIME\tEVENT\tSEVERITY\tSUBJECT\tDETAILS")

	for _, e := range events {
		subject := e.Catalog
		if subject == "" {
			subject = e.Plugin
		}
		if subject == "" {
			subject = "-"
		}

		details := ""
		if e.Error != "" {
			details = truncateStr(e.Error, 40)
		} else if len(e.CapabilitiesDenied) > 0 {
			details = "denied: " + strings.Join(e.CapabilitiesDenied, ", ")
		} else if v, ok := e.Details["violation"]; ok {
			details = truncateStr(fmt.Sprintf("%v", v), 40)
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.Timestamp.Format("2006-01-02 15:04"),
			e.Type,
			severityIcon(e.Severity),
			truncateStr(subject, 15),
			details,
		)
	}

	_ = w.Flush()
	fmt.Printf("\nShowing %d security events\n", len(events))
	return nil
}

func outputEventsJSON(events []audit.Event) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(events)
}

func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func severityIcon(s audit.Severity) string {
	switch s {
	case audit.SeverityCritical:
		return "⛔ critical"
	case audit.SeverityError:
		return "❌ error"
	case audit.SeverityWarning:
		return "⚠️  warning"
	default:
		return "ℹ️  info"
	}
}
