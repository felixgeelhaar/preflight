package files

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// LinkStep represents a symlink creation step.
type LinkStep struct {
	link      Link
	id        compiler.StepID
	fs        ports.FileSystem
	lifecycle ports.FileLifecycle
}

// NewLinkStep creates a new LinkStep.
func NewLinkStep(link Link, fs ports.FileSystem, lifecycle ports.FileLifecycle) *LinkStep {
	if lifecycle == nil {
		lifecycle = &ports.NoopLifecycle{}
	}
	id := compiler.MustNewStepID("files:link:" + link.ID())
	return &LinkStep{
		link:      link,
		id:        id,
		fs:        fs,
		lifecycle: lifecycle,
	}
}

// ID returns the step identifier.
func (s *LinkStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *LinkStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the symlink is already correct.
func (s *LinkStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	dest := ports.ExpandPath(s.link.Dest)

	isLink, target := s.fs.IsSymlink(dest)
	if isLink && target == s.link.Src {
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *LinkStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "symlink", s.link.Dest, "", s.link.Src), nil
}

// Apply creates the symlink.
func (s *LinkStep) Apply(_ compiler.RunContext) error {
	// Validate paths to prevent path traversal attacks
	if err := validation.ValidatePath(s.link.Src); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := validation.ValidatePath(s.link.Dest); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	dest := ports.ExpandPath(s.link.Dest)
	ctx := context.Background()

	// Snapshot before modification
	if err := s.lifecycle.BeforeModify(ctx, dest); err != nil {
		return fmt.Errorf("failed to snapshot before modify: %w", err)
	}

	// Handle existing file
	if s.fs.Exists(dest) {
		switch {
		case s.link.Backup:
			if err := s.fs.Rename(dest, dest+".bak"); err != nil {
				return fmt.Errorf("failed to backup %s: %w", dest, err)
			}
		case s.link.Force:
			if err := s.fs.Remove(dest); err != nil {
				return fmt.Errorf("failed to remove %s: %w", dest, err)
			}
		default:
			return fmt.Errorf("destination exists: %s (use force or backup)", dest)
		}
	}

	// Create symlink
	if err := s.fs.CreateSymlink(s.link.Src, dest); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Record for drift tracking
	if err := s.lifecycle.AfterApply(ctx, dest, "files"); err != nil {
		return fmt.Errorf("failed to record apply: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *LinkStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Create Symlink",
		fmt.Sprintf("Creates a symbolic link from %s to %s.", s.link.Dest, s.link.Src),
		nil,
	)
}

// CopyStep represents a file copy step.
type CopyStep struct {
	cp        Copy
	id        compiler.StepID
	fs        ports.FileSystem
	lifecycle ports.FileLifecycle
}

// NewCopyStep creates a new CopyStep.
func NewCopyStep(cp Copy, fs ports.FileSystem, lifecycle ports.FileLifecycle) *CopyStep {
	if lifecycle == nil {
		lifecycle = &ports.NoopLifecycle{}
	}
	id := compiler.MustNewStepID("files:copy:" + cp.ID())
	return &CopyStep{
		cp:        cp,
		id:        id,
		fs:        fs,
		lifecycle: lifecycle,
	}
}

// ID returns the step identifier.
func (s *CopyStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *CopyStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the file needs to be copied.
func (s *CopyStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	src := ports.ExpandPath(s.cp.Src)
	dest := ports.ExpandPath(s.cp.Dest)

	if !s.fs.Exists(dest) {
		return compiler.StatusNeedsApply, nil
	}

	// Compare file hashes
	srcHash, err := s.fs.FileHash(src)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	destHash, err := s.fs.FileHash(dest)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	if srcHash == destHash {
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *CopyStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "file", s.cp.Dest, "", s.cp.Src), nil
}

// Apply copies the file.
func (s *CopyStep) Apply(_ compiler.RunContext) error {
	// Validate paths to prevent path traversal attacks
	if err := validation.ValidatePath(s.cp.Src); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := validation.ValidatePath(s.cp.Dest); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	src := ports.ExpandPath(s.cp.Src)
	dest := ports.ExpandPath(s.cp.Dest)
	ctx := context.Background()

	// Snapshot before modification
	if err := s.lifecycle.BeforeModify(ctx, dest); err != nil {
		return fmt.Errorf("failed to snapshot before modify: %w", err)
	}

	content, err := s.fs.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	mode := parseFileMode(s.cp.Mode, 0o644)
	if err := s.fs.WriteFile(dest, content, mode); err != nil {
		return fmt.Errorf("failed to write destination: %w", err)
	}

	// Record for drift tracking
	if err := s.lifecycle.AfterApply(ctx, dest, "files"); err != nil {
		return fmt.Errorf("failed to record apply: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *CopyStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Copy File",
		fmt.Sprintf("Copies %s to %s.", s.cp.Src, s.cp.Dest),
		nil,
	)
}

// TemplateStep represents a template rendering step.
type TemplateStep struct {
	tmpl      Template
	id        compiler.StepID
	fs        ports.FileSystem
	lifecycle ports.FileLifecycle
}

// NewTemplateStep creates a new TemplateStep.
func NewTemplateStep(tmpl Template, fs ports.FileSystem, lifecycle ports.FileLifecycle) *TemplateStep {
	if lifecycle == nil {
		lifecycle = &ports.NoopLifecycle{}
	}
	id := compiler.MustNewStepID("files:template:" + tmpl.ID())
	return &TemplateStep{
		tmpl:      tmpl,
		id:        id,
		fs:        fs,
		lifecycle: lifecycle,
	}
}

// ID returns the step identifier.
func (s *TemplateStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *TemplateStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the template needs to be rendered.
func (s *TemplateStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	dest := ports.ExpandPath(s.tmpl.Dest)

	if !s.fs.Exists(dest) {
		return compiler.StatusNeedsApply, nil
	}

	// Render the template and compare with existing file
	src := ports.ExpandPath(s.tmpl.Src)
	templateContent, err := s.fs.ReadFile(src)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	tmpl, err := template.New("file").Parse(string(templateContent))
	if err != nil {
		return compiler.StatusUnknown, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, s.tmpl.Vars); err != nil {
		return compiler.StatusUnknown, err
	}

	existingContent, err := s.fs.ReadFile(dest)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	if bytes.Equal(buf.Bytes(), existingContent) {
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *TemplateStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "template", s.tmpl.Dest, "", s.tmpl.Src), nil
}

// Apply renders the template.
func (s *TemplateStep) Apply(_ compiler.RunContext) error {
	// Validate paths to prevent path traversal attacks
	if err := validation.ValidatePath(s.tmpl.Src); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := validation.ValidatePath(s.tmpl.Dest); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	src := ports.ExpandPath(s.tmpl.Src)
	dest := ports.ExpandPath(s.tmpl.Dest)
	ctx := context.Background()

	// Snapshot before modification
	if err := s.lifecycle.BeforeModify(ctx, dest); err != nil {
		return fmt.Errorf("failed to snapshot before modify: %w", err)
	}

	content, err := s.fs.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("file").Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, s.tmpl.Vars); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	mode := parseFileMode(s.tmpl.Mode, 0o644)
	if err := s.fs.WriteFile(dest, buf.Bytes(), mode); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	// Record for drift tracking
	if err := s.lifecycle.AfterApply(ctx, dest, "files"); err != nil {
		return fmt.Errorf("failed to record apply: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *TemplateStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Render Template",
		fmt.Sprintf("Renders template %s to %s.", s.tmpl.Src, s.tmpl.Dest),
		nil,
	)
}

// parseFileMode parses a file mode string or returns the default.
func parseFileMode(modeStr string, defaultMode os.FileMode) os.FileMode {
	if modeStr == "" {
		return defaultMode
	}
	var mode uint32
	if _, err := fmt.Sscanf(modeStr, "%o", &mode); err != nil {
		return defaultMode
	}
	return os.FileMode(mode)
}
