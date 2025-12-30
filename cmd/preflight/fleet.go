package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/felixgeelhaar/preflight/internal/domain/fleet/execution"
	"github.com/felixgeelhaar/preflight/internal/domain/fleet/targeting"
	"github.com/felixgeelhaar/preflight/internal/domain/fleet/transport"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	fleetInventoryFile string
	fleetTarget        string
	fleetExclude       string
	fleetStrategy      string
	fleetMaxParallel   int
	fleetTimeout       time.Duration
	fleetDryRun        bool
	fleetJSON          bool
	fleetStopOnError   bool
)

var fleetCmd = &cobra.Command{
	Use:   "fleet",
	Short: "Manage fleet of remote hosts",
	Long: `Fleet commands allow you to manage configuration across multiple remote hosts.

Use targeting syntax to select hosts:
  @all              - All hosts
  @groupname        - Hosts in a group
  tag:tagname       - Hosts with a tag
  host-*            - Glob pattern matching
  !pattern          - Exclude matching hosts

Examples:
  preflight fleet list
  preflight fleet ping --target @production
  preflight fleet apply --target "tag:darwin" --strategy rolling`,
}

var fleetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List hosts in the inventory",
	Long:  `List all hosts in the fleet inventory with their status, tags, and groups.`,
	RunE:  runFleetList,
}

var fleetPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Test connectivity to hosts",
	Long:  `Test SSH connectivity to selected hosts in the fleet.`,
	RunE:  runFleetPing,
}

var fleetPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show planned changes for fleet",
	Long:  `Generate and display the execution plan for the selected hosts without applying changes.`,
	RunE:  runFleetPlan,
}

var fleetApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration to fleet",
	Long:  `Apply the configuration to selected hosts in the fleet.`,
	RunE:  runFleetApply,
}

var fleetStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show fleet status",
	Long:  `Display the current status of all hosts in the fleet.`,
	RunE:  runFleetStatus,
}

func init() {
	// Common flags
	fleetCmd.PersistentFlags().StringVar(&fleetInventoryFile, "inventory", "fleet.yaml", "inventory file")
	fleetCmd.PersistentFlags().StringVarP(&fleetTarget, "target", "t", "@all", "target selector")
	fleetCmd.PersistentFlags().StringVar(&fleetExclude, "exclude", "", "exclude selector")
	fleetCmd.PersistentFlags().BoolVar(&fleetJSON, "json", false, "output as JSON")

	// Apply/Plan specific flags
	fleetApplyCmd.Flags().StringVar(&fleetStrategy, "strategy", "parallel", "execution strategy (parallel, rolling, canary)")
	fleetApplyCmd.Flags().IntVar(&fleetMaxParallel, "max-parallel", 10, "maximum parallel executions")
	fleetApplyCmd.Flags().DurationVar(&fleetTimeout, "timeout", 5*time.Minute, "per-host timeout")
	fleetApplyCmd.Flags().BoolVar(&fleetDryRun, "dry-run", false, "show what would be done")
	fleetApplyCmd.Flags().BoolVar(&fleetStopOnError, "stop-on-error", false, "stop on first error")

	fleetPlanCmd.Flags().DurationVar(&fleetTimeout, "timeout", 5*time.Minute, "per-host timeout")

	// Ping specific flags
	fleetPingCmd.Flags().DurationVar(&fleetTimeout, "timeout", 30*time.Second, "connection timeout")

	// Add subcommands
	fleetCmd.AddCommand(fleetListCmd)
	fleetCmd.AddCommand(fleetPingCmd)
	fleetCmd.AddCommand(fleetPlanCmd)
	fleetCmd.AddCommand(fleetApplyCmd)
	fleetCmd.AddCommand(fleetStatusCmd)

	// Add to root
	rootCmd.AddCommand(fleetCmd)
}

func loadFleetInventory() (*fleet.Inventory, error) {
	data, err := os.ReadFile(fleetInventoryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("inventory file not found: %s\nCreate one with 'preflight fleet init' or specify with --inventory", fleetInventoryFile)
		}
		return nil, fmt.Errorf("failed to read inventory: %w", err)
	}

	var raw FleetInventoryFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse inventory: %w", err)
	}

	return raw.ToInventory()
}

// FleetInventoryFile represents the YAML structure of fleet.yaml.
type FleetInventoryFile struct {
	Version  int                         `yaml:"version"`
	Hosts    map[string]FleetHostConfig  `yaml:"hosts"`
	Groups   map[string]FleetGroupConfig `yaml:"groups"`
	Defaults FleetDefaultsConfig         `yaml:"defaults"`
}

// FleetHostConfig represents a host in the inventory file.
type FleetHostConfig struct {
	Hostname  string   `yaml:"hostname"`
	User      string   `yaml:"user"`
	Port      int      `yaml:"port"`
	SSHKey    string   `yaml:"ssh_key"`
	ProxyJump string   `yaml:"proxy_jump"`
	Tags      []string `yaml:"tags"`
	Groups    []string `yaml:"groups"`
}

// FleetGroupConfig represents a group in the inventory file.
type FleetGroupConfig struct {
	Description string   `yaml:"description"`
	Hosts       []string `yaml:"hosts"`
	Policies    []string `yaml:"policies"`
	Inherit     []string `yaml:"inherit"`
}

// FleetDefaultsConfig represents default SSH settings.
type FleetDefaultsConfig struct {
	User       string        `yaml:"user"`
	Port       int           `yaml:"port"`
	SSHKey     string        `yaml:"ssh_key"`
	SSHTimeout time.Duration `yaml:"ssh_timeout"`
}

// ToInventory converts the file to a domain Inventory.
func (f *FleetInventoryFile) ToInventory() (*fleet.Inventory, error) {
	inv := fleet.NewInventory()

	// Set defaults
	if f.Defaults.Port != 0 || f.Defaults.User != "" {
		inv.SetDefaults(fleet.SSHConfig{
			Port:           f.Defaults.Port,
			User:           f.Defaults.User,
			IdentityFile:   f.Defaults.SSHKey,
			ConnectTimeout: f.Defaults.SSHTimeout,
		})
	}

	// Add hosts
	for name, cfg := range f.Hosts {
		hostID, err := fleet.NewHostID(name)
		if err != nil {
			return nil, fmt.Errorf("invalid host ID %q: %w", name, err)
		}

		sshCfg := fleet.SSHConfig{
			Hostname:     cfg.Hostname,
			User:         cfg.User,
			Port:         cfg.Port,
			IdentityFile: cfg.SSHKey,
			ProxyJump:    cfg.ProxyJump,
		}

		host, err := fleet.NewHost(hostID, sshCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create host %q: %w", name, err)
		}

		// Add tags
		for _, tagName := range cfg.Tags {
			tag, err := fleet.NewTag(tagName)
			if err != nil {
				return nil, fmt.Errorf("invalid tag %q for host %q: %w", tagName, name, err)
			}
			host.AddTag(tag)
		}

		// Add groups
		for _, groupName := range cfg.Groups {
			host.AddGroup(groupName)
		}

		if err := inv.AddHost(host); err != nil {
			return nil, fmt.Errorf("failed to add host %q: %w", name, err)
		}
	}

	// Add groups
	for name, cfg := range f.Groups {
		groupName, err := fleet.NewGroupName(name)
		if err != nil {
			return nil, fmt.Errorf("invalid group name %q: %w", name, err)
		}

		group := fleet.NewGroup(groupName)
		group.SetDescription(cfg.Description)
		group.SetHostPatterns(cfg.Hosts)
		group.SetPolicies(cfg.Policies)

		for _, inherit := range cfg.Inherit {
			parentName, err := fleet.NewGroupName(inherit)
			if err != nil {
				return nil, fmt.Errorf("invalid parent group %q: %w", inherit, err)
			}
			group.AddInherit(parentName)
		}

		if err := inv.AddGroup(group); err != nil {
			return nil, fmt.Errorf("failed to add group %q: %w", name, err)
		}
	}

	return inv, nil
}

func selectHosts(inv *fleet.Inventory) ([]*fleet.Host, error) {
	// Collect all selector patterns
	var patterns []string

	// Parse includes
	for _, pattern := range strings.Split(fleetTarget, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			patterns = append(patterns, pattern)
		}
	}

	// Parse excludes (prefix with ! for negation)
	if fleetExclude != "" {
		for _, pattern := range strings.Split(fleetExclude, ",") {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				// Add negation prefix if not already present
				if !strings.HasPrefix(pattern, "!") {
					pattern = "!" + pattern
				}
				patterns = append(patterns, pattern)
			}
		}
	}

	// Build target from all patterns
	target, err := targeting.NewTarget(patterns...)
	if err != nil {
		return nil, fmt.Errorf("invalid target expression: %w", err)
	}

	return target.Select(inv), nil
}

func runFleetList(_ *cobra.Command, _ []string) error {
	inv, err := loadFleetInventory()
	if err != nil {
		return err
	}

	hosts, err := selectHosts(inv)
	if err != nil {
		return err
	}

	if fleetJSON {
		return printHostsJSON(hosts)
	}

	return printHostsTable(hosts)
}

func printHostsTable(hosts []*fleet.Host) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	//nolint:errcheck // Tabwriter errors are captured by Flush
	fmt.Fprintln(w, "HOST\tHOSTNAME\tUSER\tPORT\tSTATUS\tTAGS\tGROUPS")

	for _, h := range hosts {
		summary := h.Summary()
		//nolint:errcheck // Tabwriter errors are captured by Flush
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			summary.ID,
			summary.Hostname,
			summary.User,
			summary.Port,
			summary.Status,
			strings.Join(summary.Tags, ","),
			strings.Join(summary.Groups, ","),
		)
	}

	return w.Flush()
}

func printHostsJSON(hosts []*fleet.Host) error {
	summaries := make([]fleet.HostSummary, len(hosts))
	for i, h := range hosts {
		summaries[i] = h.Summary()
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(summaries)
}

func runFleetPing(_ *cobra.Command, _ []string) error {
	inv, err := loadFleetInventory()
	if err != nil {
		return err
	}

	hosts, err := selectHosts(inv)
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		fmt.Println("No hosts selected")
		return nil
	}

	tr := transport.NewSSHTransport()
	tr.DefaultTimeout = fleetTimeout

	fmt.Printf("Pinging %d hosts...\n\n", len(hosts))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	//nolint:errcheck // Tabwriter errors are captured by Flush
	fmt.Fprintln(w, "HOST\tSTATUS\tLATENCY\tERROR")

	for _, h := range hosts {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), fleetTimeout)
		pingErr := tr.Ping(ctx, h)
		cancel()
		latency := time.Since(start)

		status := "OK"
		errMsg := ""
		if pingErr != nil {
			status = "FAILED"
			errMsg = pingErr.Error()
		}

		//nolint:errcheck // Tabwriter errors are captured by Flush
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			h.ID(),
			status,
			latency.Round(time.Millisecond),
			errMsg,
		)
	}

	return w.Flush()
}

func runFleetPlan(_ *cobra.Command, _ []string) error {
	inv, err := loadFleetInventory()
	if err != nil {
		return err
	}

	hosts, err := selectHosts(inv)
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		fmt.Println("No hosts selected")
		return nil
	}

	// For now, create sample steps - in real usage these would come from compiled config
	steps := []*execution.RemoteStep{
		execution.NewRemoteStep("sample:check", "echo 'Fleet plan ready'").
			WithCheck("true").
			WithDescription("Sample check step"),
	}

	tr := transport.NewSSHTransport()
	config := execution.ExecutorConfig{
		Timeout: fleetTimeout,
	}
	executor := execution.NewFleetExecutor(tr, config)

	fmt.Printf("Planning for %d hosts...\n\n", len(hosts))

	plan, err := executor.Plan(context.Background(), hosts, steps)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	if fleetJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(plan)
	}

	fmt.Printf("Plan Summary:\n")
	fmt.Printf("  Hosts with changes: %d\n", plan.HostsWithChanges())
	fmt.Printf("  Total changes: %d\n\n", plan.TotalChanges())

	for _, hp := range plan.Hosts {
		if hp.Error != nil {
			fmt.Printf("  %s: ERROR - %v\n", hp.HostID, hp.Error)
			continue
		}

		if !hp.HasChanges() {
			fmt.Printf("  %s: No changes\n", hp.HostID)
			continue
		}

		fmt.Printf("  %s:\n", hp.HostID)
		for _, sp := range hp.Steps {
			status := "  "
			if sp.Status == execution.StepStatusNeeds {
				status = "+ "
			}
			fmt.Printf("    %s%s: %s\n", status, sp.StepID, sp.Description)
		}
	}

	return nil
}

func runFleetApply(_ *cobra.Command, _ []string) error {
	inv, err := loadFleetInventory()
	if err != nil {
		return err
	}

	hosts, err := selectHosts(inv)
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		fmt.Println("No hosts selected")
		return nil
	}

	// Parse strategy
	var strategy execution.Strategy
	switch fleetStrategy {
	case "parallel":
		strategy = execution.StrategyParallel
	case "rolling":
		strategy = execution.StrategyRolling
	case "canary":
		strategy = execution.StrategyCanary
	default:
		return fmt.Errorf("invalid strategy: %s (use parallel, rolling, or canary)", fleetStrategy)
	}

	// For now, create sample steps - in real usage these would come from compiled config
	steps := []*execution.RemoteStep{
		execution.NewRemoteStep("sample:apply", "echo 'Applied'").
			WithCheck("false").
			WithDescription("Sample apply step"),
	}

	tr := transport.NewSSHTransport()
	config := execution.ExecutorConfig{
		Strategy:    strategy,
		MaxParallel: fleetMaxParallel,
		Timeout:     fleetTimeout,
		DryRun:      fleetDryRun,
		StopOnError: fleetStopOnError,
	}
	executor := execution.NewFleetExecutor(tr, config)

	if fleetDryRun {
		fmt.Printf("[DRY-RUN] Would apply to %d hosts\n\n", len(hosts))
	} else {
		fmt.Printf("Applying to %d hosts (strategy: %s)...\n\n", len(hosts), fleetStrategy)
	}

	result := executor.Execute(context.Background(), hosts, steps)

	if fleetJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result.Summary())
	}

	// Print results
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	//nolint:errcheck // Tabwriter errors are captured by Flush
	fmt.Fprintln(w, "HOST\tSTATUS\tSTEPS\tDURATION\tERROR")

	for _, hr := range result.HostResults {
		errMsg := ""
		if hr.Error != nil {
			errMsg = hr.Error.Error()
		}
		//nolint:errcheck // Tabwriter errors are captured by Flush
		fmt.Fprintf(w, "%s\t%s\t%d/%d\t%s\t%s\n",
			hr.HostID,
			hr.Status,
			hr.StepsApplied(),
			len(hr.StepResults),
			hr.Duration().Round(time.Millisecond),
			errMsg,
		)
	}
	_ = w.Flush()

	fmt.Printf("\nSummary: %d/%d successful, %d failed, %d skipped (total: %s)\n",
		result.SuccessfulHosts(),
		result.TotalHosts(),
		result.FailedHosts(),
		result.SkippedHosts(),
		result.Duration().Round(time.Millisecond),
	)

	if !result.AllSuccessful() {
		return fmt.Errorf("some hosts failed")
	}

	return nil
}

func runFleetStatus(_ *cobra.Command, _ []string) error {
	inv, err := loadFleetInventory()
	if err != nil {
		return err
	}

	summary := inv.Summary()

	if fleetJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}

	fmt.Printf("Fleet Status:\n")
	fmt.Printf("  Total hosts:  %d\n", summary.HostCount)
	fmt.Printf("  Total groups: %d\n", summary.GroupCount)
	fmt.Printf("\n")
	fmt.Printf("  Online:  %d\n", summary.OnlineCount)
	fmt.Printf("  Offline: %d\n", summary.OfflineCount)
	fmt.Printf("  Error:   %d\n", summary.ErrorCount)

	if len(summary.TagCounts) > 0 {
		fmt.Printf("\nTags:\n")
		for tag, count := range summary.TagCounts {
			fmt.Printf("  %s: %d\n", tag, count)
		}
	}

	return nil
}
