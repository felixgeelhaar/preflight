package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration without applying",
	Long: `Validate checks your configuration for errors without making changes.

This command is designed for CI/CD pipelines to catch configuration
issues before deployment. It supports both allow/deny policies and
org policies with required/forbidden patterns.

Exit codes:
  0 - Valid configuration
  1 - Validation errors or policy violations found
  2 - Could not read configuration

Examples:
  preflight validate
  preflight validate --config custom.yaml
  preflight validate --json
  preflight validate --target work
  preflight validate --org-policy org-policy.yaml`,
	RunE: runValidate,
}

var (
	validateConfigPath    string
	validateTarget        string
	validateJSON          bool
	validateStrict        bool
	validatePolicyFile    string
	validateOrgPolicyFile string
)

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVarP(&validateConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	validateCmd.Flags().StringVarP(&validateTarget, "target", "t", "default", "Target to validate")
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output results as JSON")
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "Treat warnings as errors")
	validateCmd.Flags().StringVar(&validatePolicyFile, "policy", "", "Path to policy YAML file (allow/deny rules)")
	validateCmd.Flags().StringVar(&validateOrgPolicyFile, "org-policy", "", "Path to org policy YAML file (required/forbidden)")
}

func runValidate(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create the application
	preflight := app.New(os.Stdout)
	if modeOverride, err := resolveModeOverride(cmd); err != nil {
		return err
	} else if modeOverride != nil {
		preflight.WithMode(*modeOverride)
	}

	// Configure validation options
	opts := app.ValidateOptions{
		PolicyFile:    validatePolicyFile,
		OrgPolicyFile: validateOrgPolicyFile,
	}

	// Validate the configuration
	result, err := preflight.ValidateWithOptions(ctx, validateConfigPath, validateTarget, opts)
	if err != nil {
		if validateJSON {
			outputValidationJSON(nil, err)
		} else {
			printError(err)
		}
		os.Exit(2)
	}

	// Determine if validation passed
	hasErrors := len(result.Errors) > 0
	hasWarnings := len(result.Warnings) > 0
	hasPolicyViolations := len(result.PolicyViolations) > 0
	failed := hasErrors || hasPolicyViolations || (validateStrict && hasWarnings)

	// Output results
	if validateJSON {
		outputValidationJSON(result, nil)
	} else {
		outputValidationText(result)
	}

	if failed {
		os.Exit(1)
	}

	return nil
}

func outputValidationJSON(result *app.ValidationResult, err error) {
	output := struct {
		Valid            bool     `json:"valid"`
		Errors           []string `json:"errors,omitempty"`
		Warnings         []string `json:"warnings,omitempty"`
		PolicyViolations []string `json:"policy_violations,omitempty"`
		Info             []string `json:"info,omitempty"`
		Error            string   `json:"error,omitempty"`
	}{}

	if err != nil {
		output.Valid = false
		output.Error = err.Error()
	} else if result != nil {
		output.Valid = len(result.Errors) == 0 && len(result.PolicyViolations) == 0
		output.Errors = result.Errors
		output.Warnings = result.Warnings
		output.PolicyViolations = result.PolicyViolations
		output.Info = result.Info
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(output)
}

func outputValidationText(result *app.ValidationResult) {
	hasIssues := len(result.Errors) > 0 || len(result.Warnings) > 0 || len(result.PolicyViolations) > 0

	if !hasIssues {
		fmt.Println("✓ Configuration is valid")
		for _, info := range result.Info {
			fmt.Printf("  • %s\n", info)
		}
		return
	}

	if len(result.Errors) > 0 {
		fmt.Println("✗ Validation errors:")
		for _, e := range result.Errors {
			fmt.Printf("  ✗ %s\n", e)
		}
	}

	if len(result.PolicyViolations) > 0 {
		fmt.Println("⛔ Policy violations:")
		for _, v := range result.PolicyViolations {
			fmt.Printf("  ⛔ %s\n", v)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("⚠ Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}

	if len(result.Info) > 0 {
		fmt.Println("ℹ Info:")
		for _, i := range result.Info {
			fmt.Printf("  • %s\n", i)
		}
	}
}
