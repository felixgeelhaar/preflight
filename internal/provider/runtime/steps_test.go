package runtime

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestToolVersionStep_ID(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	step := NewToolVersionStep(cfg, mocks.NewFileSystem())

	if step.ID().String() != "runtime:tool-versions" {
		t.Errorf("ID() = %q, want %q", step.ID().String(), "runtime:tool-versions")
	}
}

func TestToolVersionStep_DependsOn_Empty(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	step := NewToolVersionStep(cfg, mocks.NewFileSystem())

	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestToolVersionStep_Check_FileNotExists_ReturnsNeedsApply(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	fs := mocks.NewFileSystem()
	step := NewToolVersionStep(cfg, fs)

	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestToolVersionStep_Check_FileExistsWithCorrectContent_ReturnsSatisfied(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
			{Name: "python", Version: "3.12.0"},
		},
	}
	fs := mocks.NewFileSystem()
	path := ports.ExpandPath("~/.tool-versions")
	fs.SetFileContent(path, []byte("node 20.10.0\npython 3.12.0\n"))

	step := NewToolVersionStep(cfg, fs)

	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestToolVersionStep_Check_FileExistsWithDifferentContent_ReturnsNeedsApply(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	fs := mocks.NewFileSystem()
	path := ports.ExpandPath("~/.tool-versions")
	fs.SetFileContent(path, []byte("node 18.0.0\n"))

	step := NewToolVersionStep(cfg, fs)

	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestToolVersionStep_Apply_WritesToolVersionsFile(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
			{Name: "python", Version: "3.12.0"},
			{Name: "golang", Version: "1.21.5"},
		},
	}
	fs := mocks.NewFileSystem()
	step := NewToolVersionStep(cfg, fs)

	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.tool-versions")
	content, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	expected := "node 20.10.0\npython 3.12.0\ngolang 1.21.5\n"
	if string(content) != expected {
		t.Errorf("Apply() wrote:\n%s\nwant:\n%s", string(content), expected)
	}
}

func TestToolVersionStep_Plan_ReturnsDiff(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	fs := mocks.NewFileSystem()
	step := NewToolVersionStep(cfg, fs)

	diff, err := step.Plan(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if diff.IsEmpty() {
		t.Error("Plan() returned empty diff")
	}
}

func TestPluginStep_ID(t *testing.T) {
	plugin := PluginConfig{
		Name: "golang",
		URL:  "https://github.com/asdf-community/asdf-golang.git",
	}
	step := NewPluginStep(plugin)

	expected := "runtime:plugin:golang"
	if step.ID().String() != expected {
		t.Errorf("ID() = %q, want %q", step.ID().String(), expected)
	}
}

func TestPluginStep_DependsOn_Empty(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	step := NewPluginStep(plugin)

	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestPluginStep_Check_Installed(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	runner := mocks.NewCommandRunner()
	runner.AddResult("asdf", []string{"plugin", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "golang\nnode\npython\n",
	})

	step := NewPluginStepWith(plugin, "", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestPluginStep_Check_NotInstalled(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	runner := mocks.NewCommandRunner()
	runner.AddResult("asdf", []string{"plugin", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "node\npython\n",
	})

	step := NewPluginStepWith(plugin, "", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestPluginStep_Check_NoRunner(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	step := NewPluginStep(plugin)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestPluginStep_Apply_WithoutURL(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	runner := mocks.NewCommandRunner()
	runner.AddResult("asdf", []string{"plugin", "add", "golang"}, ports.CommandResult{ExitCode: 0})

	step := NewPluginStepWith(plugin, "", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	calls := runner.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Command != "asdf" {
		t.Errorf("command = %q, want %q", calls[0].Command, "asdf")
	}
}

func TestPluginStep_Apply_WithURL(t *testing.T) {
	plugin := PluginConfig{
		Name: "golang",
		URL:  "https://github.com/asdf-community/asdf-golang.git",
	}
	runner := mocks.NewCommandRunner()
	runner.AddResult("asdf", []string{"plugin", "add", "golang", "https://github.com/asdf-community/asdf-golang.git"}, ports.CommandResult{ExitCode: 0})

	step := NewPluginStepWith(plugin, "", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestPluginStep_Apply_Fails(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	runner := mocks.NewCommandRunner()
	runner.AddResult("asdf", []string{"plugin", "add", "golang"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "plugin not found",
	})

	step := NewPluginStepWith(plugin, "", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	if err == nil {
		t.Fatal("Apply() expected error, got nil")
	}
	if !contains(err.Error(), "plugin add golang failed") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "plugin add golang failed")
	}
}

func TestPluginStep_Apply_NoRunner(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	step := NewPluginStep(plugin)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	if err == nil {
		t.Fatal("Apply() expected error, got nil")
	}
	if !contains(err.Error(), "command runner not configured") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "command runner not configured")
	}
}

func TestPluginStep_BackendBinary_Default(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	runner := mocks.NewCommandRunner()
	runner.AddResult("asdf", []string{"plugin", "add", "golang"}, ports.CommandResult{ExitCode: 0})

	step := NewPluginStepWith(plugin, "", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	calls := runner.Calls()
	if calls[0].Command != "asdf" {
		t.Errorf("command = %q, want %q", calls[0].Command, "asdf")
	}
}

func TestPluginStep_BackendBinary_Mise(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	runner := mocks.NewCommandRunner()
	runner.AddResult("mise", []string{"plugin", "add", "golang"}, ports.CommandResult{ExitCode: 0})

	step := NewPluginStepWith(plugin, "mise", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	calls := runner.Calls()
	if calls[0].Command != "mise" {
		t.Errorf("command = %q, want %q", calls[0].Command, "mise")
	}
}

func TestPluginStep_BackendBinary_Rtx(t *testing.T) {
	plugin := PluginConfig{Name: "golang"}
	runner := mocks.NewCommandRunner()
	runner.AddResult("mise", []string{"plugin", "add", "golang"}, ports.CommandResult{ExitCode: 0})

	step := NewPluginStepWith(plugin, "rtx", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	calls := runner.Calls()
	if calls[0].Command != "mise" {
		t.Errorf("command = %q, want %q (rtx maps to mise)", calls[0].Command, "mise")
	}
}

func TestToolVersionStep_ProjectScope_WritesToProjectPath(t *testing.T) {
	cfg := &Config{
		Scope: "project",
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	fs := mocks.NewFileSystem()
	step := NewToolVersionStep(cfg, fs)

	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Project scope writes to .tool-versions (current directory)
	if !fs.Exists(".tool-versions") {
		t.Error("Apply() did not write .tool-versions in project directory")
	}
}

func TestToolVersionStep_Explain(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	fs := mocks.NewFileSystem()
	step := NewToolVersionStep(cfg, fs)

	explanation := step.Explain(compiler.ExplainContext{})
	if explanation.Summary() == "" {
		t.Error("Explain() returned empty summary")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
