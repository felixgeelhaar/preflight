package execution

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet/transport"
)

// RemoteStep wraps command execution for remote hosts.
type RemoteStep struct {
	id          string
	command     string
	checkCmd    string
	description string
	dependsOn   []string
}

// NewRemoteStep creates a new remote step.
func NewRemoteStep(id, command string) *RemoteStep {
	return &RemoteStep{
		id:      id,
		command: command,
	}
}

// WithCheck sets the check command.
func (s *RemoteStep) WithCheck(cmd string) *RemoteStep {
	s.checkCmd = cmd
	return s
}

// WithDescription sets the step description.
func (s *RemoteStep) WithDescription(desc string) *RemoteStep {
	s.description = desc
	return s
}

// WithDependsOn sets the dependencies.
func (s *RemoteStep) WithDependsOn(deps ...string) *RemoteStep {
	s.dependsOn = deps
	return s
}

// ID returns the step identifier.
func (s *RemoteStep) ID() string {
	return s.id
}

// Command returns the command to execute.
func (s *RemoteStep) Command() string {
	return s.command
}

// CheckCommand returns the check command.
func (s *RemoteStep) CheckCommand() string {
	return s.checkCmd
}

// Description returns the step description.
func (s *RemoteStep) Description() string {
	return s.description
}

// DependsOn returns the step dependencies.
func (s *RemoteStep) DependsOn() []string {
	return s.dependsOn
}

// Check runs the check command on the remote host.
func (s *RemoteStep) Check(ctx context.Context, conn transport.Connection) (StepStatus, error) {
	if s.checkCmd == "" {
		return StepStatusNeeds, nil
	}

	result, err := conn.Run(ctx, s.checkCmd)
	if err != nil {
		return StepStatusUnknown, fmt.Errorf("check command failed: %w", err)
	}

	if result.Success() {
		return StepStatusSatisfied, nil
	}

	return StepStatusNeeds, nil
}

// Apply runs the command on the remote host.
func (s *RemoteStep) Apply(ctx context.Context, conn transport.Connection) error {
	result, err := conn.Run(ctx, s.command)
	if err != nil {
		return fmt.Errorf("apply command failed: %w", err)
	}

	if !result.Success() {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr == "" {
			stderr = strings.TrimSpace(string(result.Stdout))
		}
		return fmt.Errorf("command exited with code %d: %s", result.ExitCode, stderr)
	}

	return nil
}

// RemoteStepBuilder helps build remote steps from config.
type RemoteStepBuilder struct {
	prefix string
}

// NewRemoteStepBuilder creates a new builder.
func NewRemoteStepBuilder(prefix string) *RemoteStepBuilder {
	return &RemoteStepBuilder{prefix: prefix}
}

// Build creates a remote step with the given ID suffix.
func (b *RemoteStepBuilder) Build(suffix, command string) *RemoteStep {
	id := suffix
	if b.prefix != "" {
		id = b.prefix + ":" + suffix
	}
	return NewRemoteStep(id, command)
}

// PackageInstallStep creates a step for installing a package.
type PackageInstallStep struct {
	*RemoteStep
	packageName string
	packageMgr  string
}

// NewPackageInstallStep creates a package installation step.
func NewPackageInstallStep(packageMgr, packageName string) *PackageInstallStep {
	id := fmt.Sprintf("%s:install:%s", packageMgr, packageName)

	var installCmd, checkCmd string

	switch packageMgr {
	case "apt":
		installCmd = fmt.Sprintf("apt-get install -y %s", packageName)
		checkCmd = fmt.Sprintf("dpkg -l %s 2>/dev/null | grep -q '^ii'", packageName)
	case "brew":
		installCmd = fmt.Sprintf("brew install %s", packageName)
		checkCmd = fmt.Sprintf("brew list %s >/dev/null 2>&1", packageName)
	case "dnf", "yum":
		installCmd = fmt.Sprintf("%s install -y %s", packageMgr, packageName)
		checkCmd = fmt.Sprintf("rpm -q %s >/dev/null 2>&1", packageName)
	default:
		installCmd = fmt.Sprintf("%s install %s", packageMgr, packageName)
		checkCmd = ""
	}

	step := NewRemoteStep(id, installCmd).
		WithCheck(checkCmd).
		WithDescription(fmt.Sprintf("Install %s via %s", packageName, packageMgr))

	return &PackageInstallStep{
		RemoteStep:  step,
		packageName: packageName,
		packageMgr:  packageMgr,
	}
}

// PackageName returns the package name.
func (s *PackageInstallStep) PackageName() string {
	return s.packageName
}

// PackageManager returns the package manager.
func (s *PackageInstallStep) PackageManager() string {
	return s.packageMgr
}

// FileStep creates a step for file operations.
type FileStep struct {
	*RemoteStep
	path     string
	content  string
	mode     string
	stepType string
}

// NewFileWriteStep creates a file write step.
func NewFileWriteStep(path, content string, mode string) *FileStep {
	id := fmt.Sprintf("file:write:%s", path)

	// Use printf for content to handle special characters
	installCmd := fmt.Sprintf("mkdir -p $(dirname %s) && printf '%%s' %q > %s && chmod %s %s",
		path, content, path, mode, path)

	// Check if file exists with correct content
	checkCmd := fmt.Sprintf("test -f %s && test \"$(cat %s)\" = %q", path, path, content)

	step := NewRemoteStep(id, installCmd).
		WithCheck(checkCmd).
		WithDescription(fmt.Sprintf("Write file %s", path))

	return &FileStep{
		RemoteStep: step,
		path:       path,
		content:    content,
		mode:       mode,
		stepType:   "write",
	}
}

// NewFileLinkStep creates a symlink step.
func NewFileLinkStep(source, target string) *FileStep {
	id := fmt.Sprintf("file:link:%s", target)

	installCmd := fmt.Sprintf("mkdir -p $(dirname %s) && ln -sf %s %s", target, source, target)
	checkCmd := fmt.Sprintf("test -L %s && test \"$(readlink %s)\" = %q", target, target, source)

	step := NewRemoteStep(id, installCmd).
		WithCheck(checkCmd).
		WithDescription(fmt.Sprintf("Link %s -> %s", target, source))

	return &FileStep{
		RemoteStep: step,
		path:       target,
		content:    source,
		stepType:   "link",
	}
}

// Path returns the file path.
func (s *FileStep) Path() string {
	return s.path
}

// Content returns the content (or source for links).
func (s *FileStep) Content() string {
	return s.content
}

// StepType returns the type of file operation.
func (s *FileStep) StepType() string {
	return s.stepType
}
