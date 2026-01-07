package gem

import (
	"context"
	"os/exec"
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

func TestStep_Check_Installed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"list", "-i", "rails"}, ports.CommandResult{
		Stdout:   "true",
		ExitCode: 0,
	})

	step := NewStep(Gem{Name: "rails"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestStep_Check_NotInstalled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"list", "-i", "rails"}, ports.CommandResult{
		Stdout:   "false",
		ExitCode: 1,
	})

	step := NewStep(Gem{Name: "rails"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"install", "rails"}, ports.CommandResult{
		Stdout:   "Successfully installed rails-7.0.0",
		ExitCode: 0,
	})

	step := NewStep(Gem{Name: "rails"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestStep_Apply_WithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"install", "bundler", "-v", "2.4.0"}, ports.CommandResult{
		Stdout:   "Successfully installed bundler-2.4.0",
		ExitCode: 0,
	})

	step := NewStep(Gem{Name: "bundler", Version: "2.4.0"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestStep_Plan(t *testing.T) {
	step := NewStep(Gem{Name: "rails", Version: "7.0.0"}, nil, nil)
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

func TestStep_Explain(t *testing.T) {
	step := NewStep(Gem{Name: "rails", Version: "7.0.0"}, nil, nil)
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

func TestStep_DependsOn(t *testing.T) {
	deps := []compiler.StepID{compiler.MustNewStepID("brew:formula:ruby")}
	step := NewStep(Gem{Name: "rails"}, nil, deps)

	got := step.DependsOn()
	if len(got) != 1 {
		t.Fatalf("DependsOn() len = %d, want 1", len(got))
	}
	if got[0].String() != "brew:formula:ruby" {
		t.Errorf("DependsOn()[0] = %q, want %q", got[0].String(), "brew:formula:ruby")
	}
}

func TestStep_DependsOn_Empty(t *testing.T) {
	step := NewStep(Gem{Name: "rails"}, nil, nil)

	got := step.DependsOn()
	if got != nil {
		t.Errorf("DependsOn() = %v, want nil", got)
	}
}

func TestStep_LockInfo(t *testing.T) {
	step := NewStep(Gem{Name: "rails", Version: "7.0.0"}, nil, nil)

	info, ok := step.LockInfo()
	if !ok {
		t.Fatal("LockInfo() ok = false, want true")
	}
	if info.Provider != "gem" {
		t.Errorf("LockInfo().Provider = %q, want %q", info.Provider, "gem")
	}
	if info.Name != "rails" {
		t.Errorf("LockInfo().Name = %q, want %q", info.Name, "rails")
	}
	if info.Version != "7.0.0" {
		t.Errorf("LockInfo().Version = %q, want %q", info.Version, "7.0.0")
	}
}

func TestStep_InstalledVersion_Found(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"list", "rails", "--exact"}, ports.CommandResult{
		Stdout:   "rails (7.0.4, 7.0.3)",
		ExitCode: 0,
	})

	step := NewStep(Gem{Name: "rails"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if !found {
		t.Error("InstalledVersion() found = false, want true")
	}
	if version != "7.0.4" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "7.0.4")
	}
}

func TestStep_InstalledVersion_NotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"list", "rails", "--exact"}, ports.CommandResult{
		Stdout:   "",
		ExitCode: 0,
	})

	step := NewStep(Gem{Name: "rails"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if found {
		t.Error("InstalledVersion() found = true, want false")
	}
	if version != "" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "")
	}
}

func TestStep_InstalledVersion_SingleVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"list", "bundler", "--exact"}, ports.CommandResult{
		Stdout:   "bundler (2.4.0)",
		ExitCode: 0,
	})

	step := NewStep(Gem{Name: "bundler"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if !found {
		t.Error("InstalledVersion() found = false, want true")
	}
	if version != "2.4.0" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "2.4.0")
	}
}

func TestStep_Check_GemNotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("gem", []string{"list", "-i", "rails"}, &commandNotFoundError{cmd: "gem"})

	step := NewStep(Gem{Name: "rails"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err == nil {
		t.Fatal("Check() error = nil, want error for command not found with no deps")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusUnknown)
	}
}

func TestStep_Check_GemNotFound_WithDeps(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("gem", []string{"list", "-i", "rails"}, &commandNotFoundError{cmd: "gem"})

	deps := []compiler.StepID{compiler.MustNewStepID("brew:formula:ruby")}
	step := NewStep(Gem{Name: "rails"}, runner, deps)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestStep_Apply_Failure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("gem", []string{"install", "nonexistent"}, ports.CommandResult{
		Stderr:   "ERROR: Could not find a valid gem 'nonexistent'",
		ExitCode: 1,
	})

	step := NewStep(Gem{Name: "nonexistent"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err == nil {
		t.Error("Apply() error = nil, want error for failed install")
	}
}

func TestStep_Apply_GemNotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("gem", []string{"install", "rails"}, &commandNotFoundError{cmd: "gem"})

	step := NewStep(Gem{Name: "rails"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err == nil {
		t.Error("Apply() error = nil, want error for gem not found")
	}
}

func TestStep_Plan_NoVersion(t *testing.T) {
	step := NewStep(Gem{Name: "rails"}, nil, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.NewValue() != "latest" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "latest")
	}
}

// commandNotFoundError implements exec.Error for testing command not found scenarios.
type commandNotFoundError struct {
	cmd string
}

func (e *commandNotFoundError) Error() string {
	return "exec: " + e.cmd + ": executable file not found in $PATH"
}

func (e *commandNotFoundError) Unwrap() error {
	return exec.ErrNotFound
}

func TestParseGem_Map(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    Gem
		wantErr bool
	}{
		{
			name:  "map with name only",
			input: map[string]interface{}{"name": "rails"},
			want:  Gem{Name: "rails"},
		},
		{
			name:  "map with name and version",
			input: map[string]interface{}{"name": "bundler", "version": "2.4.0"},
			want:  Gem{Name: "bundler", Version: "2.4.0"},
		},
		{
			name:    "map missing name",
			input:   map[string]interface{}{"version": "1.0.0"},
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGem(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseGem() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
