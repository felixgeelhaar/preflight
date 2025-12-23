//go:build e2e

package scenarios

import (
	"testing"

	"github.com/felixgeelhaar/preflight/e2e/framework"
)

func TestVersion_ShowsVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	scenario := framework.NewScenario(t)

	scenario.
		Given("the preflight binary is built", func(env *framework.Environment) {
			// Binary is automatically built by NewEnvironment
		}).
		When("I run preflight version", func(r *framework.Runner) *framework.Result {
			return r.Version()
		}).
		Then("the command succeeds", func(t *testing.T, r *framework.Result) {
			framework.AssertSuccess(t, r)
		}).
		And("the output shows version information", func(t *testing.T, r *framework.Result) {
			framework.AssertStdoutContains(t, r, "preflight")
		})
}

func TestPlan_WithEmptyConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	scenario := framework.NewScenario(t)

	scenario.
		Given("an empty config file", func(env *framework.Environment) {
			env.WriteConfig(`
name: test
targets:
  default: []
`)
		}).
		When("I run preflight plan", func(r *framework.Runner) *framework.Result {
			return r.Plan()
		}).
		Then("the command succeeds", func(t *testing.T, r *framework.Result) {
			framework.AssertSuccess(t, r)
		}).
		And("the output shows no changes needed", func(t *testing.T, r *framework.Result) {
			framework.AssertStdoutContains(t, r, "No changes")
		})
}

func TestPlan_WithLayerFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	scenario := framework.NewScenario(t)

	scenario.
		Given("a config with a layer file", func(env *framework.Environment) {
			// Create the main config referencing a layer
			env.WriteConfig(`
name: test-layers
targets:
  default:
    - base
`)
			// Create the layer file
			env.WriteLayer("base", `
name: base
`)
		}).
		When("I run preflight plan", func(r *framework.Runner) *framework.Result {
			return r.Plan()
		}).
		Then("the command succeeds", func(t *testing.T, r *framework.Result) {
			framework.AssertSuccess(t, r)
		}).
		And("the output shows no changes", func(t *testing.T, r *framework.Result) {
			framework.AssertStdoutContains(t, r, "No changes")
		})
}

func TestApply_DryRun_NoChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	scenario := framework.NewScenario(t)

	scenario.
		Given("an empty config file", func(env *framework.Environment) {
			env.WriteConfig(`
name: test
targets:
  default: []
`)
		}).
		When("I run preflight apply --dry-run", func(r *framework.Runner) *framework.Result {
			return r.ApplyDryRun()
		}).
		Then("the command succeeds", func(t *testing.T, r *framework.Result) {
			framework.AssertSuccess(t, r)
		}).
		And("the output indicates dry run mode", func(t *testing.T, r *framework.Result) {
			// Dry run should not apply anything
			framework.AssertStdoutContains(t, r, "No changes")
		})
}

func TestDiff_ShowsDifferences(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	scenario := framework.NewScenario(t)

	scenario.
		Given("a config file", func(env *framework.Environment) {
			env.WriteConfig(`
name: test
targets:
  default: []
`)
		}).
		When("I run preflight diff", func(r *framework.Runner) *framework.Result {
			return r.Diff()
		}).
		Then("the command succeeds", func(t *testing.T, r *framework.Result) {
			framework.AssertSuccess(t, r)
		})
}
