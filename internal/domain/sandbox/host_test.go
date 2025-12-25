package sandbox

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/felixgeelhaar/preflight/internal/domain/capability"
)

func TestHostFunctions(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, HostFunctions)

	// Check that known functions exist
	funcNames := make(map[string]bool)
	for _, f := range HostFunctions {
		funcNames[f.Name] = true
		assert.Equal(t, "preflight", f.Module)
	}

	assert.True(t, funcNames["read_file"])
	assert.True(t, funcNames["write_file"])
	assert.True(t, funcNames["brew_install"])
	assert.True(t, funcNames["shell_exec"])
	assert.True(t, funcNames["http_get"])
	assert.True(t, funcNames["log_info"])
}

func TestHostServices_CheckCapability(t *testing.T) {
	t.Parallel()

	t.Run("nil policy allows all", func(t *testing.T) {
		t.Parallel()

		services := &HostServices{}
		err := services.CheckCapability(capability.CapShellExecute)
		assert.NoError(t, err)
	})

	t.Run("policy allows granted capability", func(t *testing.T) {
		t.Parallel()

		policy := capability.NewPolicyBuilder().
			Grant(capability.CapFilesRead).
			Build()

		services := &HostServices{Policy: policy}
		err := services.CheckCapability(capability.CapFilesRead)
		assert.NoError(t, err)
	})

	t.Run("policy denies blocked capability", func(t *testing.T) {
		t.Parallel()

		policy := capability.NewPolicyBuilder().
			Grant(capability.CapFilesRead).
			Block(capability.CapFilesRead).
			Build()

		services := &HostServices{Policy: policy}
		err := services.CheckCapability(capability.CapFilesRead)
		assert.Error(t, err)
	})
}

func TestNullFileSystem(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fs := NullFileSystem{}

	t.Run("ReadFile returns error", func(t *testing.T) {
		t.Parallel()

		_, err := fs.ReadFile(ctx, "/any/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("WriteFile returns error", func(t *testing.T) {
		t.Parallel()

		err := fs.WriteFile(ctx, "/any/path", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("Exists returns false", func(t *testing.T) {
		t.Parallel()

		exists, err := fs.Exists(ctx, "/any/path")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Remove returns error", func(t *testing.T) {
		t.Parallel()

		err := fs.Remove(ctx, "/any/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})
}

func TestNullPackageManager(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pm := NullPackageManager{}

	t.Run("Install returns error", func(t *testing.T) {
		t.Parallel()

		err := pm.Install(ctx, "any-package")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("List returns empty", func(t *testing.T) {
		t.Parallel()

		list, err := pm.List(ctx)
		assert.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("IsInstalled returns false", func(t *testing.T) {
		t.Parallel()

		installed, err := pm.IsInstalled(ctx, "any-package")
		assert.NoError(t, err)
		assert.False(t, installed)
	})
}

func TestNullShell(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sh := NullShell{}

	t.Run("Exec returns error", func(t *testing.T) {
		t.Parallel()

		_, err := sh.Exec(ctx, "any", "command")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("ExecWithInput returns error", func(t *testing.T) {
		t.Parallel()

		_, err := sh.ExecWithInput(ctx, nil, "any", "command")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})
}

func TestNullHTTPClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := NullHTTPClient{}

	t.Run("Get returns error", func(t *testing.T) {
		t.Parallel()

		_, _, err := client.Get(ctx, "http://example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("Post returns error", func(t *testing.T) {
		t.Parallel()

		_, _, err := client.Post(ctx, "http://example.com", "application/json", []byte("{}"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})
}

func TestNullLogger(t *testing.T) {
	t.Parallel()

	logger := NullLogger{}

	// These should not panic
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
}

func TestNewIsolatedServices(t *testing.T) {
	t.Parallel()

	policy := capability.RestrictedPolicy()
	services := NewIsolatedServices(policy)

	assert.NotNil(t, services.FileSystem)
	assert.NotNil(t, services.PackageManager)
	assert.NotNil(t, services.Shell)
	assert.NotNil(t, services.HTTP)
	assert.NotNil(t, services.Logger)
	assert.Equal(t, policy, services.Policy)

	// Verify all services are null implementations
	ctx := context.Background()

	_, err := services.FileSystem.ReadFile(ctx, "/test")
	assert.Error(t, err)

	err = services.PackageManager.Install(ctx, "test")
	assert.Error(t, err)

	_, err = services.Shell.Exec(ctx, "test")
	assert.Error(t, err)

	_, _, err = services.HTTP.Get(ctx, "http://test.com")
	assert.Error(t, err)
}
