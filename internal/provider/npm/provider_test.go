package npm

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
	if got := provider.Name(); got != "npm" {
		t.Errorf("Name() = %q, want %q", got, "npm")
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

func TestProvider_Compile_NoNpmSection(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{},
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
		"npm": map[string]interface{}{
			"packages": []interface{}{"typescript", "eslint"},
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
	if !ids["npm:package:typescript"] {
		t.Error("Missing npm:package:typescript step")
	}
	if !ids["npm:package:eslint"] {
		t.Error("Missing npm:package:eslint step")
	}
}

func TestProvider_Compile_ScopedPackage(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"npm": map[string]interface{}{
			"packages": []interface{}{"@anthropic-ai/claude-code"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}
	// Step ID strips leading @ from scoped packages for valid ID format
	if steps[0].ID().String() != "npm:package:anthropic-ai/claude-code" {
		t.Errorf("ID() = %q, want %q", steps[0].ID().String(), "npm:package:anthropic-ai/claude-code")
	}
}

func TestProvider_Compile_PackageWithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"npm": map[string]interface{}{
			"packages": []interface{}{"typescript@5.0.0"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}
	// Step ID uses name only, not version
	if steps[0].ID().String() != "npm:package:typescript" {
		t.Errorf("ID() = %q, want %q", steps[0].ID().String(), "npm:package:typescript")
	}
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"npm": map[string]interface{}{
			"packages": "not-a-list", // Invalid: should be a list
		},
	})
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Error("Compile() should return error for invalid config")
	}
}

func TestPackageStep_Check_Installed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	// Check calls npm list -g --depth=0 --json and parses JSON output
	runner.AddResult("npm", []string{"list", "-g", "--depth=0", "--json"}, ports.CommandResult{
		Stdout:   `{"dependencies":{"typescript":{"version":"5.0.0"}}}`,
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
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
	// Check calls npm list -g --depth=0 --json and parses JSON output
	runner.AddResult("npm", []string{"list", "-g", "--depth=0", "--json"}, ports.CommandResult{
		Stdout:   `{"dependencies":{"typescript":{"version":"5.0.0"}}}`,
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "nonexistent"}, runner, nil)
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
	runner.AddResult("npm", []string{"install", "-g", "typescript"}, ports.CommandResult{
		Stdout:   "added 1 package",
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestPackageStep_Apply_WithVersion(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("npm", []string{"install", "-g", "typescript@5.0.0"}, ports.CommandResult{
		Stdout:   "added 1 package",
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "typescript", Version: "5.0.0"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestPackageStep_Plan(t *testing.T) {
	step := NewPackageStep(Package{Name: "typescript", Version: "5.0.0"}, nil, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Resource() != "npm-package" {
		t.Errorf("Plan().Resource() = %q, want %q", diff.Resource(), "npm-package")
	}
	if diff.Name() != "typescript" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "typescript")
	}
	if diff.NewValue() != "5.0.0" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "5.0.0")
	}
}

func TestPackageStep_Explain(t *testing.T) {
	step := NewPackageStep(Package{Name: "typescript", Version: "5.0.0"}, nil, nil)
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
	pkg := parsePackageString("typescript")
	if pkg.Name != "typescript" {
		t.Errorf("Name = %q, want %q", pkg.Name, "typescript")
	}
	if pkg.Version != "" {
		t.Errorf("Version = %q, want %q", pkg.Version, "")
	}
}

func TestParsePackage_WithVersion(t *testing.T) {
	pkg := parsePackageString("typescript@5.0.0")
	if pkg.Name != "typescript" {
		t.Errorf("Name = %q, want %q", pkg.Name, "typescript")
	}
	if pkg.Version != "5.0.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "5.0.0")
	}
}

func TestParsePackage_Scoped(t *testing.T) {
	pkg := parsePackageString("@anthropic-ai/claude-code")
	if pkg.Name != "@anthropic-ai/claude-code" {
		t.Errorf("Name = %q, want %q", pkg.Name, "@anthropic-ai/claude-code")
	}
	if pkg.Version != "" {
		t.Errorf("Version = %q, want %q", pkg.Version, "")
	}
}

func TestParsePackage_ScopedWithVersion(t *testing.T) {
	pkg := parsePackageString("@anthropic-ai/claude-code@1.0.0")
	if pkg.Name != "@anthropic-ai/claude-code" {
		t.Errorf("Name = %q, want %q", pkg.Name, "@anthropic-ai/claude-code")
	}
	if pkg.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "1.0.0")
	}
}

func TestPackageStep_DependsOn(t *testing.T) {
	deps := []compiler.StepID{compiler.MustNewStepID("brew:formula:node")}
	step := NewPackageStep(Package{Name: "typescript"}, nil, deps)

	got := step.DependsOn()
	if len(got) != 1 {
		t.Fatalf("DependsOn() len = %d, want 1", len(got))
	}
	if got[0].String() != "brew:formula:node" {
		t.Errorf("DependsOn()[0] = %q, want %q", got[0].String(), "brew:formula:node")
	}
}

func TestPackageStep_DependsOn_Empty(t *testing.T) {
	step := NewPackageStep(Package{Name: "typescript"}, nil, nil)

	got := step.DependsOn()
	if got != nil {
		t.Errorf("DependsOn() = %v, want nil", got)
	}
}

func TestPackageStep_LockInfo(t *testing.T) {
	step := NewPackageStep(Package{Name: "typescript", Version: "5.0.0"}, nil, nil)

	info, ok := step.LockInfo()
	if !ok {
		t.Fatal("LockInfo() ok = false, want true")
	}
	if info.Provider != "npm" {
		t.Errorf("LockInfo().Provider = %q, want %q", info.Provider, "npm")
	}
	if info.Name != "typescript" {
		t.Errorf("LockInfo().Name = %q, want %q", info.Name, "typescript")
	}
	if info.Version != "5.0.0" {
		t.Errorf("LockInfo().Version = %q, want %q", info.Version, "5.0.0")
	}
}

func TestPackageStep_InstalledVersion_Found(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("npm", []string{"list", "-g", "--depth=0", "--json"}, ports.CommandResult{
		Stdout:   `{"dependencies":{"typescript":{"version":"5.0.0"}}}`,
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if !found {
		t.Error("InstalledVersion() found = false, want true")
	}
	if version != "5.0.0" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "5.0.0")
	}
}

func TestPackageStep_InstalledVersion_NotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("npm", []string{"list", "-g", "--depth=0", "--json"}, ports.CommandResult{
		Stdout:   `{"dependencies":{}}`,
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
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

func TestPackageStep_InstalledVersion_NpmNotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("npm", []string{"list", "-g", "--depth=0", "--json"}, &commandNotFoundError{cmd: "npm"})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
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

func TestPackageStep_InstalledVersion_InvalidJSON(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("npm", []string{"list", "-g", "--depth=0", "--json"}, ports.CommandResult{
		Stdout:   `{invalid json`,
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	_, _, err := step.InstalledVersion(runCtx)
	if err == nil {
		t.Error("InstalledVersion() error = nil, want error for invalid JSON")
	}
}

func TestPackageStep_Check_NpmNotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("npm", []string{"list", "-g", "--depth=0", "--json"}, &commandNotFoundError{cmd: "npm"})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err == nil {
		t.Fatal("Check() error = nil, want error for npm not found with no deps")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusUnknown)
	}
}

func TestPackageStep_Check_NpmNotFound_WithDeps(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("npm", []string{"list", "-g", "--depth=0", "--json"}, &commandNotFoundError{cmd: "npm"})

	deps := []compiler.StepID{compiler.MustNewStepID("brew:formula:node")}
	step := NewPackageStep(Package{Name: "typescript"}, runner, deps)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestPackageStep_Check_InvalidJSON(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("npm", []string{"list", "-g", "--depth=0", "--json"}, ports.CommandResult{
		Stdout:   `{invalid json`,
		ExitCode: 0,
	})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err == nil {
		t.Error("Check() error = nil, want error for invalid JSON")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusUnknown)
	}
}

func TestPackageStep_Apply_NpmNotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("npm", []string{"install", "-g", "typescript"}, &commandNotFoundError{cmd: "npm"})

	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err == nil {
		t.Error("Apply() error = nil, want error for npm not found")
	}
}

func TestPackageStep_Apply_Failure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("npm", []string{"install", "-g", "nonexistent"}, ports.CommandResult{
		Stderr:   "npm ERR! 404 Not Found",
		ExitCode: 1,
	})

	step := NewPackageStep(Package{Name: "nonexistent"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err == nil {
		t.Error("Apply() error = nil, want error for failed install")
	}
}

func TestPackageStep_Plan_NoVersion(t *testing.T) {
	step := NewPackageStep(Package{Name: "typescript"}, nil, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.NewValue() != "latest" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "latest")
	}
}

func TestParsePackage_Map(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    Package
		wantErr bool
	}{
		{
			name:  "map with name only",
			input: map[string]interface{}{"name": "typescript"},
			want:  Package{Name: "typescript"},
		},
		{
			name:  "map with name and version",
			input: map[string]interface{}{"name": "typescript", "version": "5.0.0"},
			want:  Package{Name: "typescript", Version: "5.0.0"},
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
			got, err := parsePackage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePackage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parsePackage() = %+v, want %+v", got, tt.want)
			}
		})
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
