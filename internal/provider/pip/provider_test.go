package pip

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(nil)
	if got := provider.Name(); got != "pip" {
		t.Errorf("Name() = %q, want %q", got, "pip")
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

func TestProvider_Compile_NoPipSection(t *testing.T) {
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

func TestProvider_Compile_Packages(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"pip": map[string]interface{}{
			"packages": []interface{}{"httpie", "black"},
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
	if !ids["pip:package:httpie"] {
		t.Error("Missing pip:package:httpie step")
	}
	if !ids["pip:package:black"] {
		t.Error("Missing pip:package:black step")
	}
}

func TestProvider_Compile_PackageWithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"pip": map[string]interface{}{
			"packages": []interface{}{"black==23.1.0"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}
	if steps[0].ID().String() != "pip:package:black" {
		t.Errorf("ID() = %q, want %q", steps[0].ID().String(), "pip:package:black")
	}
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"pip": map[string]interface{}{
			"packages": "not-a-list",
		},
	})
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Error("Compile() should return error for invalid config")
	}
}

func TestPackageStep_Check_Installed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("pip", []string{"show", "black"}, ports.CommandResult{
		Stdout:   "Name: black\nVersion: 23.1.0",
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "black"}, runner)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestPackageStep_Check_NotInstalled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("pip", []string{"show", "black"}, ports.CommandResult{
		Stderr:   "WARNING: Package(s) not found: black",
		ExitCode: 1,
	})

	step := NewPackageStep(Package{Name: "black"}, runner)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestPackageStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("pip", []string{"install", "--user", "black"}, ports.CommandResult{
		Stdout:   "Successfully installed black-23.1.0",
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "black"}, runner)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestPackageStep_Apply_WithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("pip", []string{"install", "--user", "black==23.1.0"}, ports.CommandResult{
		Stdout:   "Successfully installed black-23.1.0",
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "black", Version: "23.1.0"}, runner)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestPackageStep_Plan(t *testing.T) {
	step := NewPackageStep(Package{Name: "black", Version: "23.1.0"}, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Resource() != "pip-package" {
		t.Errorf("Plan().Resource() = %q, want %q", diff.Resource(), "pip-package")
	}
	if diff.Name() != "black" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "black")
	}
	if diff.NewValue() != "23.1.0" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "23.1.0")
	}
}

func TestPackageStep_Explain(t *testing.T) {
	step := NewPackageStep(Package{Name: "black", Version: "23.1.0"}, nil)
	explainCtx := compiler.NewExplainContext()

	explanation := step.Explain(explainCtx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
	if explanation.Detail() == "" {
		t.Error("Explain().Detail() should not be empty")
	}
}

func TestParsePackage_Simple(t *testing.T) {
	pkg := parsePackageString("black")
	if pkg.Name != "black" {
		t.Errorf("Name = %q, want %q", pkg.Name, "black")
	}
	if pkg.Version != "" {
		t.Errorf("Version = %q, want %q", pkg.Version, "")
	}
}

func TestParsePackage_WithExactVersion(t *testing.T) {
	pkg := parsePackageString("black==23.1.0")
	if pkg.Name != "black" {
		t.Errorf("Name = %q, want %q", pkg.Name, "black")
	}
	// Version includes the specifier (==) for pip to preserve intent
	if pkg.Version != "==23.1.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "==23.1.0")
	}
}

func TestParsePackage_WithMinVersion(t *testing.T) {
	pkg := parsePackageString("black>=23.0.0")
	if pkg.Name != "black" {
		t.Errorf("Name = %q, want %q", pkg.Name, "black")
	}
	if pkg.Version != ">=23.0.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, ">=23.0.0")
	}
}

func TestParsePackage_WithCompatibleVersion(t *testing.T) {
	pkg := parsePackageString("black~=23.0")
	if pkg.Name != "black" {
		t.Errorf("Name = %q, want %q", pkg.Name, "black")
	}
	if pkg.Version != "~=23.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "~=23.0")
	}
}
