package runtime

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

func TestToolVersionStep_ID(t *testing.T) {
	cfg := &Config{
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	step := NewToolVersionStep(cfg, ports.NewMockFileSystem())

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
	step := NewToolVersionStep(cfg, ports.NewMockFileSystem())

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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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

func TestToolVersionStep_ProjectScope_WritesToProjectPath(t *testing.T) {
	cfg := &Config{
		Scope: "project",
		Tools: []ToolConfig{
			{Name: "node", Version: "20.10.0"},
		},
	}
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
	step := NewToolVersionStep(cfg, fs)

	explanation := step.Explain(compiler.ExplainContext{})
	if explanation.Summary() == "" {
		t.Error("Explain() returned empty summary")
	}
}
