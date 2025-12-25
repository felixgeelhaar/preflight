package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/policy"
	"github.com/spf13/cobra"
)

var complianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Generate org policy compliance reports",
	Long: `Generate detailed compliance reports against organizational policies.

This command evaluates your configuration against org policy rules and produces
a structured report showing compliance status, violations, warnings, and applied
overrides.

Compliance reports are useful for:
  - Security audits and governance reviews
  - CI/CD pipeline gates that require policy compliance
  - Tracking compliance score improvements over time
  - Identifying expiring overrides that need renewal

Exit codes:
  0 - Compliant (no blocking violations)
  1 - Non-compliant (blocking violations found)
  2 - Could not load configuration or policy

Examples:
  preflight compliance
  preflight compliance --policy org-policy.yaml
  preflight compliance --json
  preflight compliance --strict`,
	RunE: runCompliance,
}

var (
	complianceConfigPath string
	complianceTarget     string
	compliancePolicyFile string
	complianceJSON       bool
	complianceStrict     bool
	complianceShowItems  bool
)

func init() {
	rootCmd.AddCommand(complianceCmd)

	complianceCmd.Flags().StringVarP(&complianceConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	complianceCmd.Flags().StringVarP(&complianceTarget, "target", "t", "default", "Target to evaluate")
	complianceCmd.Flags().StringVar(&compliancePolicyFile, "policy", "", "Path to org policy YAML file")
	complianceCmd.Flags().BoolVar(&complianceJSON, "json", false, "Output report as JSON")
	complianceCmd.Flags().BoolVar(&complianceStrict, "strict", false, "Treat warnings as errors (exit 1)")
	complianceCmd.Flags().BoolVar(&complianceShowItems, "show-items", false, "Include evaluated items in report")
}

func runCompliance(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create the application
	preflight := app.New(os.Stdout)

	// Load configuration to get evaluated items
	result, err := preflight.ValidateWithOptions(ctx, complianceConfigPath, complianceTarget, app.ValidateOptions{
		OrgPolicyFile: compliancePolicyFile,
	})
	if err != nil {
		if complianceJSON {
			outputComplianceError(err)
		} else {
			printError(err)
		}
		os.Exit(2)
	}

	// Load the org policy for generating the full compliance report
	var orgPolicy *policy.OrgPolicy
	if compliancePolicyFile != "" {
		orgPolicy, err = policy.LoadOrgPolicyFromFile(compliancePolicyFile)
		if err != nil {
			if complianceJSON {
				outputComplianceError(err)
			} else {
				fmt.Fprintf(os.Stderr, "Error loading org policy: %v\n", err)
			}
			os.Exit(2)
		}
	}

	// Collect evaluated items from configuration
	evaluatedItems := collectEvaluatedItems(result)

	// Generate compliance report
	var report *policy.ComplianceReport
	if orgPolicy != nil {
		svc := policy.NewComplianceReportService(orgPolicy)
		report = svc.EvaluateAndReport(evaluatedItems)
	} else {
		// Create a minimal report when no policy is provided
		report = &policy.ComplianceReport{
			PolicyName:  "none",
			Enforcement: policy.EnforcementBlock,
			Summary: policy.ComplianceSummary{
				Status:          policy.ComplianceStatusCompliant,
				TotalChecks:     len(evaluatedItems),
				PassedChecks:    len(evaluatedItems),
				ComplianceScore: 100,
			},
		}
		if complianceShowItems {
			report.EvaluatedItems = evaluatedItems
		}
	}

	// Optionally include evaluated items
	if !complianceShowItems {
		report.EvaluatedItems = nil
	}

	// Output the report
	if complianceJSON {
		outputComplianceJSON(report)
	} else {
		outputComplianceText(report)
	}

	// Determine exit code
	if report.HasBlockingViolations() {
		os.Exit(1)
	}
	if complianceStrict && report.Summary.Status == policy.ComplianceStatusWarning {
		os.Exit(1)
	}

	return nil
}

// collectEvaluatedItems extracts all items that were evaluated from the validation result.
func collectEvaluatedItems(result *app.ValidationResult) []string {
	if result == nil {
		return nil
	}

	items := make([]string, 0, len(result.Info)+len(result.Errors))

	// Collect from info messages that represent evaluated items
	items = append(items, result.Info...)

	// Add any items from errors/warnings that reference specific config
	items = append(items, result.Errors...)

	return items
}

func outputComplianceError(err error) {
	output := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(output)
}

func outputComplianceJSON(report *policy.ComplianceReport) {
	data, err := report.ToJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting report: %v\n", err)
		os.Exit(2)
	}
	fmt.Println(string(data))
}

func outputComplianceText(report *policy.ComplianceReport) {
	fmt.Print(report.ToText())

	// Show expiring overrides warning
	expiring := report.ExpiringOverrides(7)
	if len(expiring) > 0 {
		fmt.Printf("\n⚠ %d override(s) expiring within 7 days\n", len(expiring))
	}
}
