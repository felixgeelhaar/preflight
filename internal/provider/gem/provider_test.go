package gem

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(nil)
	if got := provider.Name(); got != "gem" {
		t.Errorf("Name() = %q, want %q", got, "gem")
	}
}

func TestProvider_Compile_Empty(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestProvider_Compile_NoGemSection(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"npm": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestProvider_Compile_Gems(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"gem": map[string]interface{}{
			"gems": []interface{}{"rails", "bundler"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Compile() len = %d, want 2", len(steps))
	}

	// Verify step IDs
	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	if !ids["gem:gem:rails"] {
		t.Error("Missing gem:gem:rails step")
	}
	if !ids["gem:gem:bundler"] {
		t.Error("Missing gem:gem:bundler step")
	}
}

func TestProvider_Compile_GemWithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"gem": map[string]interface{}{
			"gems": []interface{}{"bundler@2.4.0"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}
	if steps[0].ID().String() != "gem:gem:bundler" {
		t.Errorf("ID() = %q, want %q", steps[0].ID().String(), "gem:gem:bundler")
	}
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"gem": map[string]interface{}{
			"gems": "not-a-list",
		},
	})
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Error("Compile() should return error for invalid config")
	}
}

func TestGemStep_Check_Installed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"list", "-i", "rails"}, ports.CommandResult{
		Stdout:   "true",
		ExitCode: 0,
	})

	step := NewGemStep(Gem{Name: "rails"}, runner)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestGemStep_Check_NotInstalled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"list", "-i", "rails"}, ports.CommandResult{
		Stdout:   "false",
		ExitCode: 1,
	})

	step := NewGemStep(Gem{Name: "rails"}, runner)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestGemStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"install", "rails"}, ports.CommandResult{
		Stdout:   "Successfully installed rails-7.0.0",
		ExitCode: 0,
	})

	step := NewGemStep(Gem{Name: "rails"}, runner)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestGemStep_Apply_WithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"install", "bundler", "-v", "2.4.0"}, ports.CommandResult{
		Stdout:   "Successfully installed bundler-2.4.0",
		ExitCode: 0,
	})

	step := NewGemStep(Gem{Name: "bundler", Version: "2.4.0"}, runner)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestGemStep_Plan(t *testing.T) {
	step := NewGemStep(Gem{Name: "rails", Version: "7.0.0"}, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Resource() != "gem" {
		t.Errorf("Plan().Resource() = %q, want %q", diff.Resource(), "gem")
	}
	if diff.Name() != "rails" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "rails")
	}
	if diff.NewValue() != "7.0.0" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "7.0.0")
	}
}

func TestGemStep_Explain(t *testing.T) {
	step := NewGemStep(Gem{Name: "rails", Version: "7.0.0"}, nil)
	explainCtx := compiler.NewExplainContext()

	explanation := step.Explain(explainCtx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
	if explanation.Detail() == "" {
		t.Error("Explain().Detail() should not be empty")
	}
}

func TestParseGem_Simple(t *testing.T) {
	gem := parseGemString("rails")
	if gem.Name != "rails" {
		t.Errorf("Name = %q, want %q", gem.Name, "rails")
	}
	if gem.Version != "" {
		t.Errorf("Version = %q, want %q", gem.Version, "")
	}
}

func TestParseGem_WithVersion(t *testing.T) {
	gem := parseGemString("bundler@2.4.0")
	if gem.Name != "bundler" {
		t.Errorf("Name = %q, want %q", gem.Name, "bundler")
	}
	if gem.Version != "2.4.0" {
		t.Errorf("Version = %q, want %q", gem.Version, "2.4.0")
	}
}
