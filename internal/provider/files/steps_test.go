package files

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestLinkStep_ID(t *testing.T) {
	link := Link{Src: "dotfiles/.zshrc", Dest: "~/.zshrc"}
	step := NewLinkStep(link, nil)
	if step.ID().String() == "" {
		t.Error("ID() should not be empty")
	}
}

func TestLinkStep_DependsOn(t *testing.T) {
	link := Link{Src: "dotfiles/.zshrc", Dest: "~/.zshrc"}
	step := NewLinkStep(link, nil)
	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestLinkStep_Check_AlreadyLinked(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddSymlink("/home/user/.zshrc", "/dotfiles/.zshrc")

	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc"}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestLinkStep_Check_NotLinked(t *testing.T) {
	fs := mocks.NewFileSystem()

	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc"}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestLinkStep_Check_WrongTarget(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddSymlink("/home/user/.zshrc", "/wrong/target")

	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc"}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestLinkStep_Plan(t *testing.T) {
	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc"}
	step := NewLinkStep(link, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
}

func TestLinkStep_Apply(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/dotfiles/.zshrc", "export PATH=$PATH")

	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc"}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	isLink, target := fs.IsSymlink("/home/user/.zshrc")
	if !isLink {
		t.Error("Apply() should create symlink")
	}
	if target != "/dotfiles/.zshrc" {
		t.Errorf("Apply() target = %q, want %q", target, "/dotfiles/.zshrc")
	}
}

func TestLinkStep_Apply_WithBackup(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/dotfiles/.zshrc", "new content")
	fs.AddFile("/home/user/.zshrc", "old content")

	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc", Backup: true}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if !fs.Exists("/home/user/.zshrc.bak") {
		t.Error("Apply() should create backup")
	}
}

func TestLinkStep_Apply_WithForce(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/dotfiles/.zshrc", "new content")
	fs.AddFile("/home/user/.zshrc", "old content")

	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc", Force: true}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	isLink, _ := fs.IsSymlink("/home/user/.zshrc")
	if !isLink {
		t.Error("Apply() should create symlink with force")
	}
}

func TestLinkStep_Explain(t *testing.T) {
	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc"}
	step := NewLinkStep(link, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
}

func TestCopyStep_ID(t *testing.T) {
	cp := Copy{Src: "files/script.sh", Dest: "~/.local/bin/script.sh"}
	step := NewCopyStep(cp, nil)
	if step.ID().String() == "" {
		t.Error("ID() should not be empty")
	}
}

func TestCopyStep_Check_FileExists_SameContent(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/src/script.sh", "#!/bin/bash\necho hello")
	fs.AddFile("/dest/script.sh", "#!/bin/bash\necho hello")

	cp := Copy{Src: "/src/script.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestCopyStep_Check_FileExists_DifferentContent(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/src/script.sh", "#!/bin/bash\necho hello")
	fs.AddFile("/dest/script.sh", "#!/bin/bash\necho goodbye")

	cp := Copy{Src: "/src/script.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestCopyStep_Check_FileNotExists(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/src/script.sh", "#!/bin/bash\necho hello")

	cp := Copy{Src: "/src/script.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestCopyStep_Apply(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/src/script.sh", "#!/bin/bash\necho hello")

	cp := Copy{Src: "/src/script.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	content, _ := fs.ReadFile("/dest/script.sh")
	if string(content) != "#!/bin/bash\necho hello" {
		t.Errorf("Apply() content = %q, want %q", string(content), "#!/bin/bash\necho hello")
	}
}

func TestCopyStep_Plan(t *testing.T) {
	cp := Copy{Src: "/src/script.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
}

func TestCopyStep_Explain(t *testing.T) {
	cp := Copy{Src: "/src/script.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
}

func TestTemplateStep_ID(t *testing.T) {
	tmpl := Template{Src: "templates/config.tmpl", Dest: "~/.config/app/config"}
	step := NewTemplateStep(tmpl, nil)
	if step.ID().String() == "" {
		t.Error("ID() should not be empty")
	}
}

func TestTemplateStep_Check_NotExists(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/templates/config.tmpl", "name = {{ .name }}")

	tmpl := Template{Src: "/templates/config.tmpl", Dest: "/home/user/.config"}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestTemplateStep_Apply(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/templates/config.tmpl", "name = {{ .name }}")

	tmpl := Template{
		Src:  "/templates/config.tmpl",
		Dest: "/home/user/.config",
		Vars: map[string]string{"name": "John"},
	}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	content, _ := fs.ReadFile("/home/user/.config")
	if string(content) != "name = John" {
		t.Errorf("Apply() content = %q, want %q", string(content), "name = John")
	}
}

func TestTemplateStep_Plan(t *testing.T) {
	tmpl := Template{Src: "/templates/config.tmpl", Dest: "/home/user/.config"}
	step := NewTemplateStep(tmpl, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != compiler.DiffTypeAdd {
		t.Errorf("Plan().Type() = %v, want %v", diff.Type(), compiler.DiffTypeAdd)
	}
}

func TestTemplateStep_Explain(t *testing.T) {
	tmpl := Template{Src: "/templates/config.tmpl", Dest: "/home/user/.config"}
	step := NewTemplateStep(tmpl, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)
	if explanation.Summary() == "" {
		t.Error("Explain().Summary() should not be empty")
	}
}

// DependsOn tests

func TestCopyStep_DependsOn(t *testing.T) {
	cp := Copy{Src: "/src/script.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, nil)
	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestTemplateStep_DependsOn(t *testing.T) {
	tmpl := Template{Src: "/templates/config.tmpl", Dest: "/home/user/.config"}
	step := NewTemplateStep(tmpl, nil)
	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

// TemplateStep.Check additional tests

func TestTemplateStep_Check_ExistsWithSameContent(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/templates/config.tmpl", "name = {{ .name }}")
	// Pre-render the expected content
	fs.AddFile("/home/user/.config", "name = John")

	tmpl := Template{
		Src:  "/templates/config.tmpl",
		Dest: "/home/user/.config",
		Vars: map[string]string{"name": "John"},
	}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestTemplateStep_Check_ExistsWithDifferentContent(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/templates/config.tmpl", "name = {{ .name }}")
	fs.AddFile("/home/user/.config", "name = OldValue")

	tmpl := Template{
		Src:  "/templates/config.tmpl",
		Dest: "/home/user/.config",
		Vars: map[string]string{"name": "NewValue"},
	}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestTemplateStep_Check_TemplateReadError(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Dest exists but template source doesn't
	fs.AddFile("/home/user/.config", "existing content")

	tmpl := Template{
		Src:  "/templates/nonexistent.tmpl",
		Dest: "/home/user/.config",
	}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err == nil {
		t.Error("Check() should return error for missing template")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusUnknown)
	}
}

func TestTemplateStep_Check_TemplateParseError(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/templates/bad.tmpl", "name = {{ .name }") // Missing closing braces
	fs.AddFile("/home/user/.config", "existing content")

	tmpl := Template{
		Src:  "/templates/bad.tmpl",
		Dest: "/home/user/.config",
	}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err == nil {
		t.Error("Check() should return error for invalid template")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusUnknown)
	}
}

// LinkStep.Apply validation and error tests

func TestLinkStep_Apply_InvalidSrcPath(t *testing.T) {
	fs := mocks.NewFileSystem()
	link := Link{Src: "../../../etc/passwd", Dest: "/home/user/.zshrc"}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error for invalid source path")
	}
}

func TestLinkStep_Apply_InvalidDestPath(t *testing.T) {
	fs := mocks.NewFileSystem()
	link := Link{Src: "/dotfiles/.zshrc", Dest: "../../../etc/shadow"}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error for invalid destination path")
	}
}

func TestLinkStep_Apply_DestExistsNoForceNoBackup(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/dotfiles/.zshrc", "new content")
	fs.AddFile("/home/user/.zshrc", "old content")

	link := Link{Src: "/dotfiles/.zshrc", Dest: "/home/user/.zshrc", Force: false, Backup: false}
	step := NewLinkStep(link, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error when dest exists without force/backup")
	}
}

// CopyStep.Apply validation and error tests

func TestCopyStep_Apply_InvalidSrcPath(t *testing.T) {
	fs := mocks.NewFileSystem()
	cp := Copy{Src: "../../../etc/passwd", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error for invalid source path")
	}
}

func TestCopyStep_Apply_InvalidDestPath(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/src/script.sh", "#!/bin/bash")
	cp := Copy{Src: "/src/script.sh", Dest: "../../../etc/shadow"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error for invalid destination path")
	}
}

func TestCopyStep_Apply_SrcReadError(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Source file doesn't exist
	cp := Copy{Src: "/src/nonexistent.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error when source doesn't exist")
	}
}

func TestCopyStep_Apply_WithMode(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/src/script.sh", "#!/bin/bash\necho hello")

	cp := Copy{Src: "/src/script.sh", Dest: "/dest/script.sh", Mode: "0755"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify file was created
	if !fs.Exists("/dest/script.sh") {
		t.Error("Apply() should create destination file")
	}
}

// TemplateStep.Apply validation and error tests

func TestTemplateStep_Apply_InvalidSrcPath(t *testing.T) {
	fs := mocks.NewFileSystem()
	tmpl := Template{Src: "../../../etc/passwd", Dest: "/home/user/.config"}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error for invalid source path")
	}
}

func TestTemplateStep_Apply_InvalidDestPath(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/templates/config.tmpl", "content")
	tmpl := Template{Src: "/templates/config.tmpl", Dest: "../../../etc/shadow"}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error for invalid destination path")
	}
}

func TestTemplateStep_Apply_SrcReadError(t *testing.T) {
	fs := mocks.NewFileSystem()
	tmpl := Template{Src: "/templates/nonexistent.tmpl", Dest: "/home/user/.config"}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error when template doesn't exist")
	}
}

func TestTemplateStep_Apply_ParseError(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/templates/bad.tmpl", "name = {{ .name }") // Invalid template

	tmpl := Template{Src: "/templates/bad.tmpl", Dest: "/home/user/.config"}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error for invalid template syntax")
	}
}

func TestTemplateStep_Apply_WithMode(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/templates/config.tmpl", "name = {{ .name }}")

	tmpl := Template{
		Src:  "/templates/config.tmpl",
		Dest: "/home/user/.config",
		Vars: map[string]string{"name": "John"},
		Mode: "0600",
	}
	step := NewTemplateStep(tmpl, fs)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify file was created
	if !fs.Exists("/home/user/.config") {
		t.Error("Apply() should create destination file")
	}
}

// CopyStep.Check error tests

func TestCopyStep_Check_SrcHashError(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Dest exists but source doesn't (will cause hash error)
	fs.AddFile("/dest/script.sh", "content")

	cp := Copy{Src: "/src/nonexistent.sh", Dest: "/dest/script.sh"}
	step := NewCopyStep(cp, fs)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err == nil {
		t.Error("Check() should return error when source hash fails")
	}
	if status != compiler.StatusUnknown {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusUnknown)
	}
}
