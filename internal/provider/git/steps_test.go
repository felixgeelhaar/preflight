package git

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestGitConfigStep_ID(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	step := NewConfigStep(cfg, fs)
	id := step.ID()

	if id.String() != "git:config" {
		t.Errorf("ID() = %q, want %q", id.String(), "git:config")
	}
}

func TestGitConfigStep_DependsOn(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{}
	step := NewConfigStep(cfg, fs)

	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestGitConfigStep_Check_NotExists(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	step := NewConfigStep(cfg, fs)
	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestGitConfigStep_Check_ExistsWithSameContent(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	// Write expected content first
	step := NewConfigStep(cfg, fs)
	content := step.generateConfig()
	fs.SetFileContent(ports.ExpandPath("~/.gitconfig"), content)

	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestGitConfigStep_Check_ExistsWithDifferentContent(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	// Write different content
	fs.SetFileContent(ports.ExpandPath("~/.gitconfig"), []byte("different content"))

	step := NewConfigStep(cfg, fs)
	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestGitConfigStep_Plan(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	step := NewConfigStep(cfg, fs)
	diff, err := step.Plan(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.IsEmpty() {
		t.Error("Plan() returned empty diff")
	}
}

func TestGitConfigStep_Apply(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify file was written
	path := ports.ExpandPath("~/.gitconfig")
	if !fs.Exists(path) {
		t.Error("Apply() did not create file")
	}

	content, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Check content contains expected values
	contentStr := string(content)
	if !contains(contentStr, "[user]") {
		t.Error("config should contain [user] section")
	}
	if !contains(contentStr, "name = John Doe") {
		t.Error("config should contain name")
	}
	if !contains(contentStr, "email = john@example.com") {
		t.Error("config should contain email")
	}
}

func TestGitConfigStep_Apply_WithAliases(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
		Aliases: map[string]string{
			"co": "checkout",
			"st": "status",
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.gitconfig")
	content, _ := fs.ReadFile(path)
	contentStr := string(content)

	if !contains(contentStr, "[alias]") {
		t.Error("config should contain [alias] section")
	}
	if !contains(contentStr, "co = checkout") {
		t.Error("config should contain co alias")
	}
}

func TestGitConfigStep_Apply_WithIncludes(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
		Includes: []Include{
			{Path: "~/.gitconfig.work", IfConfig: "gitdir:~/work/"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.gitconfig")
	content, _ := fs.ReadFile(path)
	contentStr := string(content)

	if !contains(contentStr, "[includeIf") {
		t.Error("config should contain includeIf section")
	}
	if !contains(contentStr, "path = ~/.gitconfig.work") {
		t.Error("config should contain include path")
	}
}

func TestGitConfigStep_Apply_WithGPGSigning(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:       "John Doe",
			Email:      "john@example.com",
			SigningKey: "ABCD1234",
		},
		Commit: CommitConfig{
			GPGSign: true,
		},
		GPG: GPGConfig{
			Format: "openpgp",
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.gitconfig")
	content, _ := fs.ReadFile(path)
	contentStr := string(content)

	if !contains(contentStr, "signingkey = ABCD1234") {
		t.Error("config should contain signing key")
	}
	if !contains(contentStr, "[commit]") {
		t.Error("config should contain [commit] section")
	}
	if !contains(contentStr, "gpgsign = true") {
		t.Error("config should contain gpgsign setting")
	}
}

func TestGitConfigStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		User: UserConfig{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	step := NewConfigStep(cfg, fs)
	explanation := step.Explain(compiler.ExplainContext{})

	if explanation.Summary() == "" {
		t.Error("Explain() returned empty summary")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
