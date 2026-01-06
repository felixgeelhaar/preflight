package gotools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(nil)
	if got := provider.Name(); got != "go" {
		t.Errorf("Name() = %q, want %q", got, "go")
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

func TestProvider_Compile_NoGoSection(t *testing.T) {
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

func TestProvider_Compile_Tools(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"go": map[string]interface{}{
			"tools": []interface{}{
				"golang.org/x/tools/gopls@latest",
				"github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
			},
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
	if !ids["go:tool:gopls"] {
		t.Error("Missing go:tool:gopls step")
	}
	if !ids["go:tool:golangci-lint"] {
		t.Error("Missing go:tool:golangci-lint step")
	}
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"go": map[string]interface{}{
			"tools": "not-a-list", // Invalid: should be a list
		},
	})
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Error("Compile() should return error for invalid config")
	}
}

func TestToolStep_Check_Installed(t *testing.T) {
	// Create a temp directory to act as GOBIN with a fake binary
	tempDir := t.TempDir()
	t.Setenv("GOBIN", tempDir)

	// Create a fake binary file
	fakeBinary := filepath.Join(tempDir, "gopls")
	if err := os.WriteFile(fakeBinary, []byte("fake"), 0755); err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}

	runner := mocks.NewCommandRunner()
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestToolStep_Check_NotInstalled(t *testing.T) {
	// Create an empty temp directory as GOBIN (no binaries)
	tempDir := t.TempDir()
	t.Setenv("GOBIN", tempDir)

	runner := mocks.NewCommandRunner()
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	status, err := step.Check(runCtx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestToolStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("go", []string{"install", "golang.org/x/tools/gopls@latest"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

func TestToolStep_Plan(t *testing.T) {
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, nil, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
	if diff.Resource() != "go-tool" {
		t.Errorf("Plan().Resource() = %q, want %q", diff.Resource(), "go-tool")
	}
	if diff.Name() != "gopls" {
		t.Errorf("Plan().Name() = %q, want %q", diff.Name(), "gopls")
	}
	if diff.NewValue() != "latest" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "latest")
	}
}

func TestToolStep_Explain(t *testing.T) {
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, nil, nil)
	explainCtx := compiler.NewExplainContext()

	explanation := step.Explain(explainCtx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
	if explanation.Detail() == "" {
		t.Error("Explain().Detail() should not be empty")
	}
}

func TestParseTool_WithVersion(t *testing.T) {
	tool := parseToolString("golang.org/x/tools/gopls@latest")
	if tool.Module != "golang.org/x/tools/gopls" {
		t.Errorf("Module = %q, want %q", tool.Module, "golang.org/x/tools/gopls")
	}
	if tool.Version != "latest" {
		t.Errorf("Version = %q, want %q", tool.Version, "latest")
	}
}

func TestParseTool_WithoutVersion(t *testing.T) {
	tool := parseToolString("golang.org/x/tools/gopls")
	if tool.Module != "golang.org/x/tools/gopls" {
		t.Errorf("Module = %q, want %q", tool.Module, "golang.org/x/tools/gopls")
	}
	if tool.Version != "" {
		t.Errorf("Version = %q, want %q", tool.Version, "")
	}
}

func TestExtractBinaryName(t *testing.T) {
	tests := []struct {
		module string
		want   string
	}{
		{"golang.org/x/tools/gopls", "gopls"},
		{"github.com/golangci/golangci-lint/cmd/golangci-lint", "golangci-lint"},
		{"simple", "simple"},
		{"github.com/user/repo", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			tool := Tool{Module: tt.module}
			got := tool.BinaryName()
			if got != tt.want {
				t.Errorf("BinaryName() = %q, want %q", got, tt.want)
			}
		})
	}
}
