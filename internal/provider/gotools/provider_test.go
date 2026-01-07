package gotools

import (
	"context"
	"os"
	"os/exec"
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

func TestToolStep_DependsOn(t *testing.T) {
	deps := []compiler.StepID{compiler.MustNewStepID("brew:formula:go")}
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, nil, deps)

	got := step.DependsOn()
	if len(got) != 1 {
		t.Fatalf("DependsOn() len = %d, want 1", len(got))
	}
	if got[0].String() != "brew:formula:go" {
		t.Errorf("DependsOn()[0] = %q, want %q", got[0].String(), "brew:formula:go")
	}
}

func TestToolStep_DependsOn_Empty(t *testing.T) {
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, nil, nil)

	got := step.DependsOn()
	if got != nil {
		t.Errorf("DependsOn() = %v, want nil", got)
	}
}

func TestToolStep_LockInfo(t *testing.T) {
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "v0.14.0"}, nil, nil)

	info, ok := step.LockInfo()
	if !ok {
		t.Fatal("LockInfo() ok = false, want true")
	}
	if info.Provider != "go" {
		t.Errorf("LockInfo().Provider = %q, want %q", info.Provider, "go")
	}
	if info.Name != "golang.org/x/tools/gopls" {
		t.Errorf("LockInfo().Name = %q, want %q", info.Name, "golang.org/x/tools/gopls")
	}
	if info.Version != "v0.14.0" {
		t.Errorf("LockInfo().Version = %q, want %q", info.Version, "v0.14.0")
	}
}

func TestToolStep_InstalledVersion_Found(t *testing.T) {
	// Create a temp directory and fake binary
	tempDir := t.TempDir()
	t.Setenv("GOBIN", tempDir)

	fakeBinary := filepath.Join(tempDir, "gopls")
	if err := os.WriteFile(fakeBinary, []byte("fake"), 0755); err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}

	runner := mocks.NewCommandRunner()
	runner.AddResult("go", []string{"version", "-m", fakeBinary}, ports.CommandResult{
		Stdout:   "gopls: go1.21.0\n\tpath\tgolang.org/x/tools/gopls\n\tmod\tgolang.org/x/tools/gopls\tv0.14.2\n",
		ExitCode: 0,
	})

	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	version, found, err := step.InstalledVersion(runCtx)
	if err != nil {
		t.Fatalf("InstalledVersion() error = %v", err)
	}
	if !found {
		t.Error("InstalledVersion() found = false, want true")
	}
	if version != "v0.14.2" {
		t.Errorf("InstalledVersion() = %q, want %q", version, "v0.14.2")
	}
}

func TestToolStep_InstalledVersion_NotFound(t *testing.T) {
	// Create an empty temp directory as GOBIN
	tempDir := t.TempDir()
	t.Setenv("GOBIN", tempDir)

	runner := mocks.NewCommandRunner()
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, runner, nil)
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

func TestToolStep_InstalledVersion_GoNotFound(t *testing.T) {
	// Create a temp directory and fake binary
	tempDir := t.TempDir()
	t.Setenv("GOBIN", tempDir)

	fakeBinary := filepath.Join(tempDir, "gopls")
	if err := os.WriteFile(fakeBinary, []byte("fake"), 0755); err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}

	runner := mocks.NewCommandRunner()
	runner.AddError("go", []string{"version", "-m", fakeBinary}, &commandNotFoundError{cmd: "go"})

	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, runner, nil)
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

func TestToolStep_InstalledVersion_CommandFailed(t *testing.T) {
	// Create a temp directory and fake binary
	tempDir := t.TempDir()
	t.Setenv("GOBIN", tempDir)

	fakeBinary := filepath.Join(tempDir, "gopls")
	if err := os.WriteFile(fakeBinary, []byte("fake"), 0755); err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}

	runner := mocks.NewCommandRunner()
	runner.AddResult("go", []string{"version", "-m", fakeBinary}, ports.CommandResult{
		Stderr:   "error reading binary",
		ExitCode: 1,
	})

	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, runner, nil)
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

func TestToolStep_Apply_GoNotFound(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddError("go", []string{"install", "golang.org/x/tools/gopls@latest"}, &commandNotFoundError{cmd: "go"})

	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls", Version: "latest"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err == nil {
		t.Error("Apply() error = nil, want error for go not found")
	}
}

func TestToolStep_Apply_Failure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("go", []string{"install", "invalid/module@latest"}, ports.CommandResult{
		Stderr:   "go: invalid/module@latest: module not found",
		ExitCode: 1,
	})

	step := NewToolStep(Tool{Module: "invalid/module", Version: "latest"}, runner, nil)
	runCtx := compiler.NewRunContext(context.Background())

	err := step.Apply(runCtx)
	if err == nil {
		t.Error("Apply() error = nil, want error for failed install")
	}
}

func TestToolStep_Plan_NoVersion(t *testing.T) {
	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls"}, nil, nil)
	runCtx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(runCtx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.NewValue() != "latest" {
		t.Errorf("Plan().NewValue() = %q, want %q", diff.NewValue(), "latest")
	}
}

func TestParseTool_Map(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    Tool
		wantErr bool
	}{
		{
			name:  "map with module only",
			input: map[string]interface{}{"module": "golang.org/x/tools/gopls"},
			want:  Tool{Module: "golang.org/x/tools/gopls"},
		},
		{
			name:  "map with module and version",
			input: map[string]interface{}{"module": "golang.org/x/tools/gopls", "version": "v0.14.0"},
			want:  Tool{Module: "golang.org/x/tools/gopls", Version: "v0.14.0"},
		},
		{
			name:    "map missing module",
			input:   map[string]interface{}{"version": "latest"},
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
			got, err := parseTool(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseTool() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTool_FullName(t *testing.T) {
	tests := []struct {
		name     string
		tool     Tool
		expected string
	}{
		{
			name:     "with version",
			tool:     Tool{Module: "golang.org/x/tools/gopls", Version: "v0.14.0"},
			expected: "golang.org/x/tools/gopls@v0.14.0",
		},
		{
			name:     "without version defaults to latest",
			tool:     Tool{Module: "golang.org/x/tools/gopls"},
			expected: "golang.org/x/tools/gopls@latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tool.FullName()
			if got != tt.expected {
				t.Errorf("FullName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetGoBin_GOPATH(t *testing.T) {
	// Clear GOBIN and set GOPATH
	t.Setenv("GOBIN", "")
	t.Setenv("GOPATH", "/custom/gopath")

	got := getGoBin()
	want := "/custom/gopath/bin"
	if got != want {
		t.Errorf("getGoBin() = %q, want %q", got, want)
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
