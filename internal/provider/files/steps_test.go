package files

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()

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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
	fs := ports.NewMockFileSystem()
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
