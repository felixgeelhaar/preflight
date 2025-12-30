package execution

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/felixgeelhaar/preflight/internal/domain/fleet/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRemoteStep(t *testing.T) {
	t.Parallel()

	step := NewRemoteStep("test:step", "echo hello")

	assert.Equal(t, "test:step", step.ID())
	assert.Equal(t, "echo hello", step.Command())
	assert.Empty(t, step.CheckCommand())
	assert.Empty(t, step.Description())
	assert.Empty(t, step.DependsOn())
}

func TestRemoteStep_Builders(t *testing.T) {
	t.Parallel()

	step := NewRemoteStep("test:step", "echo hello").
		WithCheck("test -f /tmp/hello").
		WithDescription("Test step").
		WithDependsOn("dep1", "dep2")

	assert.Equal(t, "test -f /tmp/hello", step.CheckCommand())
	assert.Equal(t, "Test step", step.Description())
	assert.Equal(t, []string{"dep1", "dep2"}, step.DependsOn())
}

func TestRemoteStep_Check(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	id, _ := fleet.NewHostID("test-host")
	host, _ := fleet.NewHost(id, fleet.SSHConfig{Hostname: "localhost"})
	conn, _ := tr.Connect(context.Background(), host)
	t.Cleanup(func() { _ = conn.Close() })

	t.Run("no check command returns needs", func(t *testing.T) {
		t.Parallel()
		step := NewRemoteStep("test", "echo hello")
		status, err := step.Check(context.Background(), conn)
		require.NoError(t, err)
		assert.Equal(t, StepStatusNeeds, status)
	})

	t.Run("check passes returns satisfied", func(t *testing.T) {
		t.Parallel()
		step := NewRemoteStep("test", "echo hello").
			WithCheck("true")
		status, err := step.Check(context.Background(), conn)
		require.NoError(t, err)
		assert.Equal(t, StepStatusSatisfied, status)
	})

	t.Run("check fails returns needs", func(t *testing.T) {
		t.Parallel()
		step := NewRemoteStep("test", "echo hello").
			WithCheck("false")
		status, err := step.Check(context.Background(), conn)
		require.NoError(t, err)
		assert.Equal(t, StepStatusNeeds, status)
	})
}

func TestRemoteStep_Apply(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	id, _ := fleet.NewHostID("test-host")
	host, _ := fleet.NewHost(id, fleet.SSHConfig{Hostname: "localhost"})
	conn, _ := tr.Connect(context.Background(), host)
	t.Cleanup(func() { _ = conn.Close() })

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()
		step := NewRemoteStep("test", "true")
		err := step.Apply(context.Background(), conn)
		assert.NoError(t, err)
	})

	t.Run("failing command", func(t *testing.T) {
		t.Parallel()
		step := NewRemoteStep("test", "exit 1")
		err := step.Apply(context.Background(), conn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "code 1")
	})
}

func TestRemoteStepBuilder(t *testing.T) {
	t.Parallel()

	t.Run("with prefix", func(t *testing.T) {
		t.Parallel()
		builder := NewRemoteStepBuilder("brew")
		step := builder.Build("install", "brew install ripgrep")
		assert.Equal(t, "brew:install", step.ID())
	})

	t.Run("without prefix", func(t *testing.T) {
		t.Parallel()
		builder := NewRemoteStepBuilder("")
		step := builder.Build("install", "brew install ripgrep")
		assert.Equal(t, "install", step.ID())
	})
}

func TestPackageInstallStep(t *testing.T) {
	t.Parallel()

	t.Run("apt package", func(t *testing.T) {
		t.Parallel()
		step := NewPackageInstallStep("apt", "ripgrep")
		assert.Equal(t, "apt:install:ripgrep", step.ID())
		assert.Equal(t, "apt", step.PackageManager())
		assert.Equal(t, "ripgrep", step.PackageName())
		assert.Contains(t, step.Command(), "apt-get install -y ripgrep")
		assert.Contains(t, step.CheckCommand(), "dpkg -l ripgrep")
	})

	t.Run("brew package", func(t *testing.T) {
		t.Parallel()
		step := NewPackageInstallStep("brew", "ripgrep")
		assert.Equal(t, "brew:install:ripgrep", step.ID())
		assert.Contains(t, step.Command(), "brew install ripgrep")
		assert.Contains(t, step.CheckCommand(), "brew list ripgrep")
	})

	t.Run("dnf package", func(t *testing.T) {
		t.Parallel()
		step := NewPackageInstallStep("dnf", "ripgrep")
		assert.Contains(t, step.Command(), "dnf install -y ripgrep")
		assert.Contains(t, step.CheckCommand(), "rpm -q ripgrep")
	})

	t.Run("yum package", func(t *testing.T) {
		t.Parallel()
		step := NewPackageInstallStep("yum", "ripgrep")
		assert.Contains(t, step.Command(), "yum install -y ripgrep")
	})

	t.Run("unknown package manager", func(t *testing.T) {
		t.Parallel()
		step := NewPackageInstallStep("unknown", "ripgrep")
		assert.Contains(t, step.Command(), "unknown install ripgrep")
		assert.Empty(t, step.CheckCommand())
	})
}

func TestFileStep(t *testing.T) {
	t.Parallel()

	t.Run("write step", func(t *testing.T) {
		t.Parallel()
		step := NewFileWriteStep("/tmp/test.txt", "hello", "0644")
		assert.Equal(t, "file:write:/tmp/test.txt", step.ID())
		assert.Equal(t, "/tmp/test.txt", step.Path())
		assert.Equal(t, "hello", step.Content())
		assert.Equal(t, "write", step.StepType())
		assert.Contains(t, step.Description(), "Write file")
	})

	t.Run("link step", func(t *testing.T) {
		t.Parallel()
		step := NewFileLinkStep("/source/file", "/target/file")
		assert.Equal(t, "file:link:/target/file", step.ID())
		assert.Equal(t, "/target/file", step.Path())
		assert.Equal(t, "/source/file", step.Content())
		assert.Equal(t, "link", step.StepType())
		assert.Contains(t, step.Description(), "Link")
	})
}
