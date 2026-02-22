package aws_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/aws"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	okResult   = ports.CommandResult{ExitCode: 0}
	failResult = ports.CommandResult{ExitCode: 1, Stderr: "error"}
)

// --- ProfileStep ---

func TestProfileStep_ID(t *testing.T) {
	t.Parallel()
	step := aws.NewProfileStep(aws.Profile{Name: "dev"}, mocks.NewCommandRunner())
	assert.Equal(t, "aws:profile:dev", step.ID().String())
}

func TestProfileStep_DependsOn(t *testing.T) {
	t.Parallel()
	step := aws.NewProfileStep(aws.Profile{Name: "dev"}, mocks.NewCommandRunner())
	assert.Nil(t, step.DependsOn())
}

func TestProfileStep_Check_NoConfigFile(t *testing.T) {
	t.Parallel()
	step := aws.NewProfileStep(aws.Profile{Name: "dev", Region: "us-east-1"}, mocks.NewCommandRunner())
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestProfileStep_Plan(t *testing.T) {
	t.Parallel()
	step := aws.NewProfileStep(aws.Profile{Name: "dev", Region: "us-east-1"}, mocks.NewCommandRunner())
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestProfileStep_Apply_Region(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "region", "us-east-1", "--profile", "dev"}, okResult)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", Region: "us-east-1"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestProfileStep_Apply_RegionRunnerError(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddError("aws", []string{"configure", "set", "region", "us-east-1", "--profile", "dev"}, assert.AnError)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", Region: "us-east-1"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
}

func TestProfileStep_Apply_RegionCommandFailure(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "region", "us-east-1", "--profile", "dev"}, failResult)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", Region: "us-east-1"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aws configure set region failed")
}

func TestProfileStep_Apply_Output(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "output", "json", "--profile", "dev"}, okResult)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", Output: "json"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestProfileStep_Apply_OutputRunnerError(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddError("aws", []string{"configure", "set", "output", "json", "--profile", "dev"}, assert.AnError)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", Output: "json"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
}

func TestProfileStep_Apply_OutputCommandFailure(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "output", "json", "--profile", "dev"}, failResult)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", Output: "json"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aws configure set output failed")
}

func TestProfileStep_Apply_RoleArn(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "role_arn", "arn:aws:iam::123:role/Admin", "--profile", "dev"}, okResult)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", RoleArn: "arn:aws:iam::123:role/Admin"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestProfileStep_Apply_RoleArnRunnerError(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddError("aws", []string{"configure", "set", "role_arn", "arn:aws:iam::123:role/Admin", "--profile", "dev"}, assert.AnError)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", RoleArn: "arn:aws:iam::123:role/Admin"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
}

func TestProfileStep_Apply_RoleArnCommandFailure(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "role_arn", "arn:aws:iam::123:role/Admin", "--profile", "dev"}, failResult)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", RoleArn: "arn:aws:iam::123:role/Admin"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aws configure set role_arn failed")
}

func TestProfileStep_Apply_SourceProfile(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "source_profile", "base", "--profile", "dev"}, okResult)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", SourceProfile: "base"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestProfileStep_Apply_SourceProfileRunnerError(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddError("aws", []string{"configure", "set", "source_profile", "base", "--profile", "dev"}, assert.AnError)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", SourceProfile: "base"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
}

func TestProfileStep_Apply_SourceProfileCommandFailure(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "source_profile", "base", "--profile", "dev"}, failResult)

	step := aws.NewProfileStep(aws.Profile{Name: "dev", SourceProfile: "base"}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aws configure set source_profile failed")
}

func TestProfileStep_Apply_AllFields(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "region", "us-east-1", "--profile", "dev"}, okResult)
	runner.AddResult("aws", []string{"configure", "set", "output", "json", "--profile", "dev"}, okResult)
	runner.AddResult("aws", []string{"configure", "set", "role_arn", "arn:aws:iam::123:role/Admin", "--profile", "dev"}, okResult)
	runner.AddResult("aws", []string{"configure", "set", "source_profile", "base", "--profile", "dev"}, okResult)

	step := aws.NewProfileStep(aws.Profile{
		Name:          "dev",
		Region:        "us-east-1",
		Output:        "json",
		RoleArn:       "arn:aws:iam::123:role/Admin",
		SourceProfile: "base",
	}, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestProfileStep_Apply_EmptyProfile(t *testing.T) {
	t.Parallel()
	step := aws.NewProfileStep(aws.Profile{Name: "dev"}, mocks.NewCommandRunner())
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestProfileStep_Explain(t *testing.T) {
	t.Parallel()
	step := aws.NewProfileStep(aws.Profile{Name: "prod"}, mocks.NewCommandRunner())
	explanation := step.Explain(compiler.ExplainContext{})
	assert.Contains(t, explanation.Summary(), "Profile")
	assert.Contains(t, explanation.Detail(), "prod")
}

// --- SSOStep ---

func TestSSOStep_ID(t *testing.T) {
	t.Parallel()
	step := aws.NewSSOStep(aws.SSOConfig{ProfileName: "sso-dev"}, mocks.NewCommandRunner())
	assert.Equal(t, "aws:sso:sso-dev", step.ID().String())
}

func TestSSOStep_DependsOn(t *testing.T) {
	t.Parallel()
	step := aws.NewSSOStep(aws.SSOConfig{ProfileName: "sso-dev"}, mocks.NewCommandRunner())
	assert.Nil(t, step.DependsOn())
}

func TestSSOStep_Check_NoConfigFile(t *testing.T) {
	t.Parallel()
	step := aws.NewSSOStep(aws.SSOConfig{
		ProfileName: "sso-dev",
		SSOStartURL: "https://company.awsapps.com/start",
	}, mocks.NewCommandRunner())
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSSOStep_Plan(t *testing.T) {
	t.Parallel()
	step := aws.NewSSOStep(aws.SSOConfig{
		ProfileName: "sso-dev",
		SSOStartURL: "https://company.awsapps.com/start",
	}, mocks.NewCommandRunner())
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestSSOStep_Apply_AllSettings(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()

	sso := aws.SSOConfig{
		ProfileName:  "sso-dev",
		SSOStartURL:  "https://company.awsapps.com/start",
		SSORegion:    "us-east-1",
		SSOAccountID: "123456789012",
		SSORoleName:  "Developer",
		Region:       "us-west-2",
		Output:       "json",
	}

	runner.AddResult("aws", []string{"configure", "set", "sso_start_url", sso.SSOStartURL, "--profile", sso.ProfileName}, okResult)
	runner.AddResult("aws", []string{"configure", "set", "sso_region", sso.SSORegion, "--profile", sso.ProfileName}, okResult)
	runner.AddResult("aws", []string{"configure", "set", "sso_account_id", sso.SSOAccountID, "--profile", sso.ProfileName}, okResult)
	runner.AddResult("aws", []string{"configure", "set", "sso_role_name", sso.SSORoleName, "--profile", sso.ProfileName}, okResult)
	runner.AddResult("aws", []string{"configure", "set", "region", sso.Region, "--profile", sso.ProfileName}, okResult)
	runner.AddResult("aws", []string{"configure", "set", "output", sso.Output, "--profile", sso.ProfileName}, okResult)

	step := aws.NewSSOStep(sso, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestSSOStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()

	sso := aws.SSOConfig{
		ProfileName: "sso-dev",
		SSOStartURL: "https://company.awsapps.com/start",
	}

	runner.AddError("aws", []string{"configure", "set", "sso_start_url", sso.SSOStartURL, "--profile", sso.ProfileName}, assert.AnError)

	step := aws.NewSSOStep(sso, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
}

func TestSSOStep_Apply_CommandFailure(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()

	sso := aws.SSOConfig{
		ProfileName: "sso-dev",
		SSOStartURL: "https://company.awsapps.com/start",
	}

	runner.AddResult("aws", []string{"configure", "set", "sso_start_url", sso.SSOStartURL, "--profile", sso.ProfileName}, failResult)

	step := aws.NewSSOStep(sso, runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestSSOStep_Apply_EmptyValues(t *testing.T) {
	t.Parallel()
	step := aws.NewSSOStep(aws.SSOConfig{ProfileName: "sso-dev"}, mocks.NewCommandRunner())
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestSSOStep_Explain(t *testing.T) {
	t.Parallel()
	step := aws.NewSSOStep(aws.SSOConfig{
		ProfileName: "sso-dev",
		SSOStartURL: "https://company.awsapps.com/start",
	}, mocks.NewCommandRunner())
	explanation := step.Explain(compiler.ExplainContext{})
	assert.Contains(t, explanation.Summary(), "SSO")
	assert.Contains(t, explanation.Detail(), "sso-dev")
}

// --- DefaultProfileStep ---

func TestDefaultProfileStep_ID(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultProfileStep("dev", mocks.NewCommandRunner())
	assert.Equal(t, "aws:default-profile", step.ID().String())
}

func TestDefaultProfileStep_DependsOn(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultProfileStep("dev", mocks.NewCommandRunner())
	assert.Nil(t, step.DependsOn())
}

func TestDefaultProfileStep_Check_EnvNotSet(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultProfileStep("some-unlikely-profile-name", mocks.NewCommandRunner())
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestDefaultProfileStep_Plan(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultProfileStep("dev", mocks.NewCommandRunner())
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestDefaultProfileStep_Apply(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultProfileStep("dev", mocks.NewCommandRunner())
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestDefaultProfileStep_Explain(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultProfileStep("dev", mocks.NewCommandRunner())
	explanation := step.Explain(compiler.ExplainContext{})
	assert.Contains(t, explanation.Summary(), "Profile")
	assert.Contains(t, explanation.Detail(), "dev")
}

// --- DefaultRegionStep ---

func TestDefaultRegionStep_ID(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultRegionStep("us-east-1", mocks.NewCommandRunner())
	assert.Equal(t, "aws:default-region", step.ID().String())
}

func TestDefaultRegionStep_DependsOn(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultRegionStep("us-east-1", mocks.NewCommandRunner())
	assert.Nil(t, step.DependsOn())
}

func TestDefaultRegionStep_Check_Satisfied(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "get", "region"},
		ports.CommandResult{ExitCode: 0, Stdout: "us-east-1\n"})

	step := aws.NewDefaultRegionStep("us-east-1", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestDefaultRegionStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "get", "region"},
		ports.CommandResult{ExitCode: 0, Stdout: "us-west-2\n"})

	step := aws.NewDefaultRegionStep("us-east-1", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestDefaultRegionStep_Check_RunnerError(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddError("aws", []string{"configure", "get", "region"}, assert.AnError)

	step := aws.NewDefaultRegionStep("us-east-1", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestDefaultRegionStep_Plan(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "get", "region"},
		ports.CommandResult{ExitCode: 0, Stdout: "us-west-2\n"})

	step := aws.NewDefaultRegionStep("us-east-1", runner)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestDefaultRegionStep_Apply_Success(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "region", "us-east-1"}, okResult)

	step := aws.NewDefaultRegionStep("us-east-1", runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.NoError(t, err)
}

func TestDefaultRegionStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddError("aws", []string{"configure", "set", "region", "us-east-1"}, assert.AnError)

	step := aws.NewDefaultRegionStep("us-east-1", runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
}

func TestDefaultRegionStep_Apply_CommandFailure(t *testing.T) {
	t.Parallel()
	runner := mocks.NewCommandRunner()
	runner.AddResult("aws", []string{"configure", "set", "region", "us-east-1"}, failResult)

	step := aws.NewDefaultRegionStep("us-east-1", runner)
	err := step.Apply(compiler.NewRunContext(context.TODO()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aws configure set region failed")
}

func TestDefaultRegionStep_Explain(t *testing.T) {
	t.Parallel()
	step := aws.NewDefaultRegionStep("eu-west-1", mocks.NewCommandRunner())
	explanation := step.Explain(compiler.ExplainContext{})
	assert.Contains(t, explanation.Summary(), "Region")
	assert.Contains(t, explanation.Detail(), "eu-west-1")
}
