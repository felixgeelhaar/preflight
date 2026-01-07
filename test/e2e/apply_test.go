//go:build e2e
// +build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2E_Apply_DryRun_GitConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Create a simple git config
	manifest := `targets:
  default:
    - base
`
	layer := `name: base
git:
  user:
    name: "Test User"
    email: "test@example.com"
  core:
    editor: nvim
`
	h.CreateConfigFile("preflight.yaml", manifest)
	h.CreateConfigFile("layers/base.yaml", layer)

	// Run plan first
	planOutput := h.Plan()
	assert.Contains(t, planOutput, "git")

	// Run apply with dry-run
	applyOutput := h.Apply(true)
	assert.Contains(t, applyOutput, "Dry run")
}

func TestE2E_Apply_FilesProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Create source file that will be linked
	sourceContent := "# My custom config\nexport FOO=bar\n"
	h.CreateConfigFile("dotfiles/.custom_rc", sourceContent)

	// Create config that links the file
	manifest := `targets:
  default:
    - base
`
	layer := `name: base
files:
  - source: dotfiles/.custom_rc
    destination: ~/.custom_rc
`
	h.CreateConfigFile("preflight.yaml", manifest)
	h.CreateConfigFile("layers/base.yaml", layer)

	// Run plan
	planOutput := h.Plan()
	t.Log("Plan output:", planOutput)

	// Run apply dry-run
	applyOutput := h.Apply(true)
	t.Log("Apply output:", applyOutput)
}

func TestE2E_Plan_InvalidConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Create invalid config (no targets)
	invalidManifest := `defaults:
  mode: intent
`
	h.CreateConfigFile("preflight.yaml", invalidManifest)

	// Plan should fail
	exitCode := h.Run("plan", "--config", h.ConfigDir+"/preflight.yaml")
	assert.NotEqual(t, 0, exitCode)
}

func TestE2E_Plan_MissingLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Create config referencing non-existent layer
	manifest := `targets:
  default:
    - nonexistent
`
	h.CreateConfigFile("preflight.yaml", manifest)

	// Plan should fail or handle gracefully
	exitCode := h.Run("plan", "--config", h.ConfigDir+"/preflight.yaml")
	// This should either fail or report the missing layer
	t.Logf("Exit code: %d, Output: %s, Error: %s", exitCode, h.LastOutput, h.LastError)
}

func TestE2E_Validate_ValidConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Create valid config
	manifest := `targets:
  default:
    - base
`
	layer := `name: base
git:
  user:
    name: "Test User"
`
	h.CreateConfigFile("preflight.yaml", manifest)
	h.CreateConfigFile("layers/base.yaml", layer)

	// Validate should succeed
	output := h.Validate()
	t.Log("Validate output:", output)
}

func TestE2E_Doctor_AfterInit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Init with balanced preset
	h.Init("git:minimal")

	// Doctor should run without error
	output := h.Doctor()
	t.Log("Doctor output:", output)
}

func TestE2E_FullWorkflow_Init_Plan_Apply(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Step 1: Init
	initOutput := h.Init("git:minimal")
	assert.Contains(t, initOutput, "Configuration created")

	// Step 2: Plan
	planOutput := h.Plan()
	t.Log("Plan output:", planOutput)

	// Step 3: Apply (dry-run)
	applyOutput := h.Apply(true)
	assert.Contains(t, applyOutput, "Dry run")

	// Step 4: Doctor
	doctorOutput := h.Doctor()
	t.Log("Doctor output:", doctorOutput)
}

func TestE2E_Apply_WithExistingFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Create existing gitconfig in home
	h.CreateFile(".gitconfig", "[user]\n  name = Existing User\n")

	// Create config that would modify gitconfig
	manifest := `targets:
  default:
    - base
`
	layer := `name: base
git:
  user:
    name: "New User"
    email: "new@example.com"
`
	h.CreateConfigFile("preflight.yaml", manifest)
	h.CreateConfigFile("layers/base.yaml", layer)

	// Plan should work
	planOutput := h.Plan()
	t.Log("Plan output:", planOutput)

	// Apply dry-run should work
	applyOutput := h.Apply(true)
	t.Log("Apply output:", applyOutput)

	// Original file should be unchanged after dry-run
	content := h.ReadFile(".gitconfig")
	assert.Contains(t, content, "Existing User")
}

func TestE2E_MultiTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Create multi-target config
	manifest := `targets:
  default:
    - base
  work:
    - base
    - work
`
	baseLayer := `name: base
git:
  core:
    editor: nvim
`
	workLayer := `name: work
git:
  user:
    name: "Work User"
    email: "work@company.com"
`
	h.CreateConfigFile("preflight.yaml", manifest)
	h.CreateConfigFile("layers/base.yaml", baseLayer)
	h.CreateConfigFile("layers/work.yaml", workLayer)

	// Plan with default target
	defaultPlan := h.Plan("--target", "default")
	t.Log("Default plan:", defaultPlan)

	// Plan with work target
	workPlan := h.Plan("--target", "work")
	t.Log("Work plan:", workPlan)
}
