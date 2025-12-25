package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"gopkg.in/ini.v1"
)

// getAWSConfigPath returns the AWS configuration directory path.
func getAWSConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws")
}

// ProfileStep represents an AWS CLI profile configuration step.
type ProfileStep struct {
	profile Profile
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewProfileStep creates a new ProfileStep.
func NewProfileStep(profile Profile, runner ports.CommandRunner) *ProfileStep {
	id := compiler.MustNewStepID("aws:profile:" + profile.Name)
	return &ProfileStep{
		profile: profile,
		id:      id,
		runner:  runner,
	}
}

// ID returns the step identifier.
func (s *ProfileStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ProfileStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the profile exists.
func (s *ProfileStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := filepath.Join(getAWSConfigPath(), "config")

	cfg, err := ini.Load(configPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // File not existing means we need to apply
	}

	sectionName := "profile " + s.profile.Name
	if s.profile.Name == "default" {
		sectionName = "default"
	}

	section, err := cfg.GetSection(sectionName)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // Section not existing means we need to apply
	}

	// Check if key settings match
	if s.profile.Region != "" && section.Key("region").String() != s.profile.Region {
		return compiler.StatusNeedsApply, nil
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *ProfileStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "profile", s.profile.Name, "", s.profile.Region), nil
}

// Apply creates or updates the profile.
func (s *ProfileStep) Apply(ctx compiler.RunContext) error {
	// Use aws configure for each setting
	if s.profile.Region != "" {
		result, err := s.runner.Run(ctx.Context(), "aws", "configure", "set", "region", s.profile.Region, "--profile", s.profile.Name)
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("aws configure set region failed: %s", result.Stderr)
		}
	}

	if s.profile.Output != "" {
		result, err := s.runner.Run(ctx.Context(), "aws", "configure", "set", "output", s.profile.Output, "--profile", s.profile.Name)
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("aws configure set output failed: %s", result.Stderr)
		}
	}

	if s.profile.RoleArn != "" {
		result, err := s.runner.Run(ctx.Context(), "aws", "configure", "set", "role_arn", s.profile.RoleArn, "--profile", s.profile.Name)
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("aws configure set role_arn failed: %s", result.Stderr)
		}
	}

	if s.profile.SourceProfile != "" {
		result, err := s.runner.Run(ctx.Context(), "aws", "configure", "set", "source_profile", s.profile.SourceProfile, "--profile", s.profile.Name)
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("aws configure set source_profile failed: %s", result.Stderr)
		}
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *ProfileStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure AWS Profile",
		fmt.Sprintf("Creates or updates the %s AWS CLI profile", s.profile.Name),
		[]string{
			"https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html",
		},
	).WithTradeoffs([]string{
		"+ Enables profile-based credential management",
		"+ Supports role assumption and MFA",
		"- Credentials stored in ~/.aws/credentials (use secret refs)",
	})
}

// SSOStep represents an AWS SSO profile configuration step.
type SSOStep struct {
	sso    SSOConfig
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewSSOStep creates a new SSOStep.
func NewSSOStep(sso SSOConfig, runner ports.CommandRunner) *SSOStep {
	id := compiler.MustNewStepID("aws:sso:" + sso.ProfileName)
	return &SSOStep{
		sso:    sso,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *SSOStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *SSOStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the SSO profile exists.
func (s *SSOStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := filepath.Join(getAWSConfigPath(), "config")

	cfg, err := ini.Load(configPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // File not existing means we need to apply
	}

	sectionName := "profile " + s.sso.ProfileName
	section, err := cfg.GetSection(sectionName)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // Section not existing means we need to apply
	}

	// Check SSO configuration
	if section.Key("sso_start_url").String() != s.sso.SSOStartURL {
		return compiler.StatusNeedsApply, nil
	}

	return compiler.StatusSatisfied, nil
}

// Plan returns the diff for this step.
func (s *SSOStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "sso", s.sso.ProfileName, "", s.sso.SSOStartURL), nil
}

// Apply creates or updates the SSO profile.
func (s *SSOStep) Apply(ctx compiler.RunContext) error {
	profile := s.sso.ProfileName

	settings := map[string]string{
		"sso_start_url":  s.sso.SSOStartURL,
		"sso_region":     s.sso.SSORegion,
		"sso_account_id": s.sso.SSOAccountID,
		"sso_role_name":  s.sso.SSORoleName,
		"region":         s.sso.Region,
		"output":         s.sso.Output,
	}

	for key, value := range settings {
		if value == "" {
			continue
		}
		result, err := s.runner.Run(ctx.Context(), "aws", "configure", "set", key, value, "--profile", profile)
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("aws configure set %s failed: %s", key, result.Stderr)
		}
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *SSOStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure AWS SSO Profile",
		fmt.Sprintf("Creates SSO profile %s pointing to %s", s.sso.ProfileName, s.sso.SSOStartURL),
		[]string{
			"https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html",
		},
	).WithTradeoffs([]string{
		"+ No long-term credentials stored",
		"+ Integrated with corporate identity",
		"- Requires 'aws sso login' before use",
	})
}

// DefaultProfileStep sets the default AWS profile.
type DefaultProfileStep struct {
	profile string
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewDefaultProfileStep creates a new DefaultProfileStep.
func NewDefaultProfileStep(profile string, runner ports.CommandRunner) *DefaultProfileStep {
	return &DefaultProfileStep{
		profile: profile,
		id:      compiler.MustNewStepID("aws:default-profile"),
		runner:  runner,
	}
}

// ID returns the step identifier.
func (s *DefaultProfileStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *DefaultProfileStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the default profile is set.
func (s *DefaultProfileStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	// Check AWS_PROFILE environment variable or config
	if os.Getenv("AWS_PROFILE") == s.profile {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *DefaultProfileStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	currentProfile := os.Getenv("AWS_PROFILE")
	return compiler.NewDiff(compiler.DiffTypeModify, "default_profile", "AWS_PROFILE", currentProfile, s.profile), nil
}

// Apply sets the default profile (via environment file).
func (s *DefaultProfileStep) Apply(_ compiler.RunContext) error {
	// This would typically update a shell profile to export AWS_PROFILE
	// For now, we just log the recommendation
	fmt.Printf("To set default profile, add to your shell profile:\n  export AWS_PROFILE=%s\n", s.profile)
	return nil
}

// Explain provides a human-readable explanation.
func (s *DefaultProfileStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Set Default AWS Profile",
		fmt.Sprintf("Configures %s as the default AWS profile", s.profile),
		[]string{
			"https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html",
		},
	).WithTradeoffs([]string{
		"+ No need to specify --profile for every command",
	})
}

// DefaultRegionStep sets the default AWS region.
type DefaultRegionStep struct {
	region string
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewDefaultRegionStep creates a new DefaultRegionStep.
func NewDefaultRegionStep(region string, runner ports.CommandRunner) *DefaultRegionStep {
	return &DefaultRegionStep{
		region: region,
		id:     compiler.MustNewStepID("aws:default-region"),
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *DefaultRegionStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *DefaultRegionStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the default region is set.
func (s *DefaultRegionStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "aws", "configure", "get", "region")
	if err != nil {
		return compiler.StatusUnknown, err
	}

	currentRegion := strings.TrimSpace(result.Stdout)
	if currentRegion == s.region {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *DefaultRegionStep) Plan(ctx compiler.RunContext) (compiler.Diff, error) {
	result, _ := s.runner.Run(ctx.Context(), "aws", "configure", "get", "region")
	currentRegion := strings.TrimSpace(result.Stdout)
	return compiler.NewDiff(compiler.DiffTypeModify, "default_region", "region", currentRegion, s.region), nil
}

// Apply sets the default region.
func (s *DefaultRegionStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "aws", "configure", "set", "region", s.region)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("aws configure set region failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *DefaultRegionStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Set Default AWS Region",
		fmt.Sprintf("Configures %s as the default AWS region", s.region),
		[]string{
			"https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html#cli-configure-quickstart-region",
		},
	).WithTradeoffs([]string{
		"+ No need to specify --region for every command",
	})
}
