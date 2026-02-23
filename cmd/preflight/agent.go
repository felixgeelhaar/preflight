package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/preflight/internal/adapters/ipc"
	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/agent"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage the Preflight background agent",
	Long: `Manage the Preflight background agent for scheduled reconciliation and drift detection.

The agent runs in the background and periodically checks your configuration for drift,
applying remediations according to your policy settings.`,
}

// Agent start command flags
var (
	agentForeground  bool
	agentSchedule    string
	agentRemediation string
	agentTarget      string
)

var agentStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the background agent",
	Long: `Start the Preflight background agent.

By default, the agent runs as a background daemon. Use --foreground to run
in the current terminal for debugging.

Examples:
  preflight agent start                          # Start with defaults
  preflight agent start --schedule 15m           # Check every 15 minutes
  preflight agent start --foreground             # Run in foreground
  preflight agent start --remediation auto       # Auto-apply safe remediations`,
	RunE: runAgentStart,
}

// Agent stop command flags
var agentStopForce bool

var agentStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background agent",
	Long: `Stop the running background agent.

This gracefully shuts down the agent, waiting for any in-progress
reconciliation to complete.

Examples:
  preflight agent stop           # Graceful stop
  preflight agent stop --force   # Force immediate stop`,
	RunE: runAgentStop,
}

// Agent status command flags
var (
	agentStatusJSON  bool
	agentStatusWatch bool
)

var agentStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show agent status",
	Long: `Display the current status of the background agent.

Shows whether the agent is running, its current state, and recent
reconciliation statistics.

Examples:
  preflight agent status         # Show status
  preflight agent status --json  # Output as JSON
  preflight agent status --watch # Continuously update`,
	RunE: runAgentStatus,
}

var agentInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install agent as a system service",
	Long: `Install the Preflight agent as a system service.

On macOS, this creates a LaunchAgent that starts automatically at login.
On Linux, this creates a systemd user service.

The installed service will:
  - Start automatically at login
  - Restart if it crashes
  - Run with your user privileges

Examples:
  preflight agent install                    # Install with defaults
  preflight agent install --schedule 30m    # Install with custom schedule`,
	RunE: runAgentInstall,
}

var agentUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the agent system service",
	Long: `Remove the Preflight agent system service.

This stops the agent if running and removes the service configuration.`,
	RunE: runAgentUninstall,
}

var agentApproveCmd = &cobra.Command{
	Use:   "approve <request-id>",
	Short: "Approve a pending remediation request",
	Long: `Approve a pending remediation request.

When the agent is running with --remediation approved, certain
operations require explicit approval before being applied.

Examples:
  preflight agent approve abc123   # Approve request abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentApprove,
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentStartCmd)
	agentCmd.AddCommand(agentStopCmd)
	agentCmd.AddCommand(agentStatusCmd)
	agentCmd.AddCommand(agentInstallCmd)
	agentCmd.AddCommand(agentUninstallCmd)
	agentCmd.AddCommand(agentApproveCmd)

	// Start command flags
	agentStartCmd.Flags().BoolVar(&agentForeground, "foreground", false, "Run in foreground (don't daemonize)")
	agentStartCmd.Flags().StringVar(&agentSchedule, "schedule", "30m", "Reconciliation schedule (e.g., 30m, 1h, or cron expression)")
	agentStartCmd.Flags().StringVar(&agentRemediation, "remediation", "notify", "Remediation policy: notify, auto, approved, safe")
	agentStartCmd.Flags().StringVar(&agentTarget, "target", "default", "Target to reconcile")

	// Stop command flags
	agentStopCmd.Flags().BoolVar(&agentStopForce, "force", false, "Force immediate stop")

	// Status command flags
	agentStatusCmd.Flags().BoolVar(&agentStatusJSON, "json", false, "Output as JSON")
	agentStatusCmd.Flags().BoolVar(&agentStatusWatch, "watch", false, "Continuously update status")

	// Install command inherits schedule and remediation flags
	agentInstallCmd.Flags().StringVar(&agentSchedule, "schedule", "30m", "Reconciliation schedule")
	agentInstallCmd.Flags().StringVar(&agentRemediation, "remediation", "notify", "Remediation policy")
	agentInstallCmd.Flags().StringVar(&agentTarget, "target", "default", "Target to reconcile")
}

func runAgentStart(_ *cobra.Command, _ []string) error {
	if err := requireExperimental("agent"); err != nil {
		return err
	}
	client := ipc.NewClient(ipc.ClientConfig{})

	// Check if already running
	if client.IsAgentRunning() {
		return fmt.Errorf("agent is already running (PID %d)", client.GetAgentPID())
	}

	// Parse schedule
	schedule, err := agent.ParseSchedule(agentSchedule)
	if err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	// Parse remediation policy
	policy, err := agent.ParseRemediationPolicy(agentRemediation)
	if err != nil {
		return fmt.Errorf("invalid remediation policy: %w", err)
	}

	if agentForeground {
		// Run in foreground
		fmt.Printf("Starting agent in foreground mode...\n")
		fmt.Printf("  Schedule: %s\n", schedule)
		fmt.Printf("  Remediation: %s\n", policy)
		fmt.Printf("  Target: %s\n", agentTarget)
		fmt.Println()

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		cfg := agent.DefaultConfig().
			WithSchedule(schedule).
			WithRemediation(policy).
			WithTarget(agentTarget)

		ag, err := agent.NewAgent(cfg)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		preflight := app.New(os.Stdout)
		ag.SetReconcileHandler(func(rctx context.Context) (*agent.ReconciliationResult, error) {
			return reconcile(rctx, preflight, cfg)
		})

		provider := &agentProvider{agent: ag}
		server := ipc.NewServer(ipc.ServerConfig{Version: version}, provider)
		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start IPC server: %w", err)
		}
		defer func() { _ = server.Stop() }()

		if err := ag.Start(ctx); err != nil {
			return fmt.Errorf("failed to start agent: %w", err)
		}

		fmt.Println("Agent is running. Press Ctrl+C to stop.")

		<-ctx.Done()

		fmt.Println("\nShutting down agent...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Timeouts.Shutdown)
		defer shutdownCancel()
		_ = ag.Stop(shutdownCtx)

		return nil
	}

	// Start as daemon — re-exec self with --foreground
	fmt.Println("Starting agent as background daemon...")

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	args := []string{"agent", "start", "--foreground",
		"--schedule", agentSchedule,
		"--remediation", agentRemediation,
		"--target", agentTarget,
	}

	// #nosec G204 -- arguments are validated flags from this CLI, not user-controlled input.
	cmd := exec.Command(execPath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = daemonProcAttr()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Printf("Agent started (PID %d)\n", cmd.Process.Pid)
	fmt.Printf("  Schedule: %s\n", schedule)
	fmt.Printf("  Remediation: %s\n", policy)
	fmt.Printf("  Target: %s\n", agentTarget)

	_ = cmd.Process.Release()
	return nil
}

func runAgentStop(_ *cobra.Command, _ []string) error {
	if err := requireExperimental("agent"); err != nil {
		return err
	}
	client := ipc.NewClient(ipc.ClientConfig{})

	if !client.IsAgentRunning() {
		fmt.Println("Agent is not running.")
		return nil
	}

	fmt.Println("Stopping agent...")

	timeout := 30 * time.Second
	if agentStopForce {
		timeout = 5 * time.Second
	}

	resp, err := client.Stop(agentStopForce, timeout)
	if err != nil {
		return fmt.Errorf("failed to stop agent: %w", err)
	}

	if resp.Success {
		fmt.Println("Agent stopped successfully.")
	} else {
		fmt.Printf("Agent stop failed: %s\n", resp.Message)
	}

	return nil
}

func runAgentStatus(_ *cobra.Command, _ []string) error {
	if err := requireExperimental("agent"); err != nil {
		return err
	}
	client := ipc.NewClient(ipc.ClientConfig{})

	if agentStatusWatch {
		return runAgentStatusWatch(client)
	}

	if !client.IsAgentRunning() {
		if agentStatusJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]interface{}{
				"running": false,
			})
		}
		fmt.Println("Agent is not running.")
		fmt.Println("")
		fmt.Println("Start the agent with:")
		fmt.Println("  preflight agent start")
		return nil
	}

	resp, err := client.Status()
	if err != nil {
		return fmt.Errorf("failed to get agent status: %w", err)
	}

	if agentStatusJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"running":         true,
			"pid":             resp.PID,
			"version":         resp.Version,
			"state":           resp.Status.State,
			"reconcileCount":  resp.Status.ReconcileCount,
			"lastReconcile":   resp.Status.LastReconcileAt,
			"nextReconcile":   resp.Status.NextReconcileAt,
			"health":          resp.Status.Health.Status,
			"pendingApproval": resp.Status.PendingApproval,
		})
	}

	// Human-readable output
	fmt.Printf("Agent Status\n")
	fmt.Println("────────────")
	fmt.Printf("Running:     yes (PID %d)\n", resp.PID)
	fmt.Printf("Version:     %s\n", resp.Version)
	fmt.Printf("State:       %s\n", resp.Status.State)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Health:\t%s\n", formatHealth(resp.Status.Health))
	_, _ = fmt.Fprintf(w, "Reconciliations:\t%d\n", resp.Status.ReconcileCount)

	if !resp.Status.LastReconcileAt.IsZero() {
		_, _ = fmt.Fprintf(w, "Last Reconcile:\t%s (%s ago)\n",
			resp.Status.LastReconcileAt.Format("2006-01-02 15:04:05"),
			formatDuration(time.Since(resp.Status.LastReconcileAt)))
	}
	if !resp.Status.NextReconcileAt.IsZero() {
		_, _ = fmt.Fprintf(w, "Next Reconcile:\t%s (in %s)\n",
			resp.Status.NextReconcileAt.Format("2006-01-02 15:04:05"),
			formatDuration(time.Until(resp.Status.NextReconcileAt)))
	}

	if resp.Status.PendingApproval != "" {
		_, _ = fmt.Fprintf(w, "Pending Approval:\t%s\n", resp.Status.PendingApproval)
	}

	_ = w.Flush()

	// Show drift info if available
	if len(resp.Status.DriftCount) > 0 {
		fmt.Println()
		fmt.Println("Detected Drift:")
		for severity, count := range resp.Status.DriftCount {
			fmt.Printf("  %s: %d\n", severity, count)
		}
	}

	return nil
}

func runAgentStatusWatch(client *ipc.Client) error {
	fmt.Println("Watching agent status (Ctrl+C to stop)...")
	fmt.Println()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		// Clear screen (basic implementation)
		fmt.Print("\033[H\033[2J")

		if !client.IsAgentRunning() {
			fmt.Println("Agent is not running.")
			fmt.Println("Waiting for agent to start...")
		} else {
			resp, err := client.Status()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Agent Status (updated %s)\n", time.Now().Format("15:04:05"))
				fmt.Println("────────────────────────────────")
				fmt.Printf("State:       %s\n", resp.Status.State)
				fmt.Printf("Health:      %s\n", formatHealth(resp.Status.Health))
				fmt.Printf("Reconciles:  %d\n", resp.Status.ReconcileCount)
				if !resp.Status.NextReconcileAt.IsZero() {
					fmt.Printf("Next Run:    in %s\n", formatDuration(time.Until(resp.Status.NextReconcileAt)))
				}
			}
		}

		<-ticker.C
	}
}

func runAgentInstall(_ *cobra.Command, _ []string) error {
	if err := requireExperimental("agent"); err != nil {
		return err
	}
	if _, err := agent.ParseSchedule(agentSchedule); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}
	if _, err := agent.ParseRemediationPolicy(agentRemediation); err != nil {
		return fmt.Errorf("invalid remediation policy: %w", err)
	}
	switch runtime.GOOS {
	case "darwin":
		return installLaunchAgent()
	case "linux":
		return installSystemdService()
	default:
		return fmt.Errorf("service installation not supported on %s", runtime.GOOS)
	}
}

func runAgentUninstall(_ *cobra.Command, _ []string) error {
	if err := requireExperimental("agent"); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return uninstallLaunchAgent()
	case "linux":
		return uninstallSystemdService()
	default:
		return fmt.Errorf("service uninstallation not supported on %s", runtime.GOOS)
	}
}

func runAgentApprove(_ *cobra.Command, args []string) error {
	if err := requireExperimental("agent"); err != nil {
		return err
	}
	requestID := args[0]

	client := ipc.NewClient(ipc.ClientConfig{})

	if !client.IsAgentRunning() {
		return fmt.Errorf("agent is not running")
	}

	resp, err := client.Approve(requestID)
	if err != nil {
		return fmt.Errorf("failed to approve request: %w", err)
	}

	if resp.Success {
		fmt.Printf("Approved request: %s\n", requestID)
	} else {
		fmt.Printf("Approval failed: %s\n", resp.Message)
	}

	return nil
}

// Helper functions

func formatHealth(health agent.HealthStatus) string {
	switch health.Status {
	case agent.HealthHealthy:
		return "healthy"
	case agent.HealthDegraded:
		return fmt.Sprintf("degraded (%s)", health.Message)
	case agent.HealthUnhealthy:
		return fmt.Sprintf("unhealthy (%s)", health.Message)
	default:
		return "unknown"
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
}

// Service installation stubs - these will be implemented in the next phase

func installLaunchAgent() error {
	fmt.Println("Installing LaunchAgent for Preflight...")

	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create LaunchAgent plist
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	plistPath := home + "/Library/LaunchAgents/com.preflight.agent.plist"
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.preflight.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>agent</string>
        <string>start</string>
        <string>--foreground</string>
        <string>--schedule</string>
        <string>%s</string>
        <string>--remediation</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/.preflight/agent.log</string>
    <key>StandardErrorPath</key>
    <string>%s/.preflight/agent.log</string>
</dict>
</plist>`, execPath, agentSchedule, agentRemediation, home, home)

	// Ensure LaunchAgents directory exists
	launchAgentsDir := home + "/Library/LaunchAgents"
	// #nosec G301 -- LaunchAgents must be readable by launchctl.
	if err := os.MkdirAll(launchAgentsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Write plist file
	// #nosec G306 -- LaunchAgent plist must be readable by launchctl.
	if err := os.WriteFile(plistPath, []byte(plistContent), 0o644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	// Load the agent
	// #nosec G204 -- plistPath is constructed from the user's home directory.
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load LaunchAgent: %w", err)
	}

	fmt.Printf("LaunchAgent installed: %s\n", plistPath)
	fmt.Println("Agent will start automatically at login.")
	return nil
}

func uninstallLaunchAgent() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	plistPath := home + "/Library/LaunchAgents/com.preflight.agent.plist"

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Println("LaunchAgent is not installed.")
		return nil
	}

	// Unload the agent
	cmd := exec.Command("launchctl", "unload", plistPath) // #nosec G204 -- plist path is deterministic and under user's control
	_ = cmd.Run()                                         // Ignore error if not loaded

	// Remove plist
	if err := os.Remove(plistPath); err != nil {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	fmt.Println("LaunchAgent uninstalled successfully.")
	return nil
}

func installSystemdService() error {
	fmt.Println("Installing systemd user service for Preflight...")

	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	serviceDir := configDir + "/systemd/user"
	servicePath := serviceDir + "/preflight-agent.service"

	serviceContent := fmt.Sprintf(`[Unit]
Description=Preflight Background Agent
After=network.target

[Service]
Type=simple
ExecStart=%s agent start --foreground --schedule %s --remediation %s
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
`, execPath, agentSchedule, agentRemediation)

	// Ensure service directory exists
	// #nosec G301 -- systemd user services must be readable by systemd.
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	// Write service file
	// #nosec G306 -- systemd service files must be readable by systemd.
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0o644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd and enable service
	cmds := [][]string{
		{"systemctl", "--user", "daemon-reload"},
		{"systemctl", "--user", "enable", "preflight-agent"},
		{"systemctl", "--user", "start", "preflight-agent"},
	}

	for _, cmdArgs := range cmds {
		// #nosec G204 -- command arguments are static and controlled.
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmdArgs, err)
		}
	}

	fmt.Printf("Systemd service installed: %s\n", servicePath)
	fmt.Println("Agent will start automatically at login.")
	return nil
}

// agentProvider bridges the running Agent to the IPC Server.
type agentProvider struct {
	agent *agent.Agent
}

func (p *agentProvider) Status() agent.Status {
	return p.agent.Status()
}

func (p *agentProvider) Stop(ctx context.Context) error {
	return p.agent.Stop(ctx)
}

func (p *agentProvider) Approve(_ string) error {
	return fmt.Errorf("approval not yet implemented")
}

// reconcile runs a single Plan→Apply cycle against the given preflight app.
func reconcile(ctx context.Context, pf *app.Preflight, cfg *agent.Config) (*agent.ReconciliationResult, error) {
	startedAt := time.Now()

	plan, err := pf.Plan(ctx, cfg.ConfigPath, cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("plan failed: %w", err)
	}

	result := &agent.ReconciliationResult{
		StartedAt:  startedAt,
		DriftCount: plan.Summary().NeedsApply,
	}
	result.DriftDetected = result.DriftCount > 0

	if plan.HasChanges() {
		if cfg.Remediation == agent.RemediationAuto || cfg.Remediation == agent.RemediationSafe {
			results, applyErr := pf.Apply(ctx, plan, false)
			if applyErr != nil {
				return nil, fmt.Errorf("apply failed: %w", applyErr)
			}

			applied := 0
			for i := range results {
				if results[i].Error() == nil {
					applied++
				}
			}
			result.RemediationApplied = applied > 0
			result.RemediationCount = applied
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startedAt)
	return result, nil
}

func uninstallSystemdService() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	servicePath := configDir + "/systemd/user/preflight-agent.service"

	// Check if service exists
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		fmt.Println("Systemd service is not installed.")
		return nil
	}

	// Stop and disable service
	cmds := [][]string{
		{"systemctl", "--user", "stop", "preflight-agent"},
		{"systemctl", "--user", "disable", "preflight-agent"},
	}

	for _, cmdArgs := range cmds {
		// #nosec G204 -- command arguments are static and controlled.
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		_ = cmd.Run() // Ignore errors
	}

	// Remove service file
	if err := os.Remove(servicePath); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd
	// #nosec G204 -- command arguments are static and controlled.
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	_ = cmd.Run()

	fmt.Println("Systemd service uninstalled successfully.")
	return nil
}
