package sandbox

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/capability"
)

func TestNewWazeroRuntime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	require.NotNil(t, runtime)

	defer func() { _ = runtime.Close() }()

	assert.True(t, runtime.IsAvailable())
	assert.Equal(t, "wazero-1.8", runtime.Version())
}

func TestWazeroRuntime_Close(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)

	// First close should succeed
	err = runtime.Close()
	assert.NoError(t, err)
	assert.False(t, runtime.IsAvailable())

	// Second close should be idempotent
	err = runtime.Close()
	assert.NoError(t, err)
}

func TestWazeroRuntime_NewSandbox(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Close() })

	t.Run("creates sandbox with default config", func(t *testing.T) {
		t.Parallel()
		sandbox, err := runtime.NewSandbox(DefaultConfig())
		require.NoError(t, err)
		assert.NotNil(t, sandbox)
		defer func() { _ = sandbox.Close() }()
	})

	t.Run("creates sandbox with full isolation", func(t *testing.T) {
		t.Parallel()
		sandbox, err := runtime.NewSandbox(FullIsolationConfig())
		require.NoError(t, err)
		assert.NotNil(t, sandbox)
		defer func() { _ = sandbox.Close() }()
	})

	t.Run("creates sandbox with trusted config", func(t *testing.T) {
		t.Parallel()
		sandbox, err := runtime.NewSandbox(TrustedConfig())
		require.NoError(t, err)
		assert.NotNil(t, sandbox)
		defer func() { _ = sandbox.Close() }()
	})

	t.Run("fails after runtime closed", func(t *testing.T) {
		t.Parallel()
		runtime2, err := NewWazeroRuntime(ctx)
		require.NoError(t, err)

		err = runtime2.Close()
		require.NoError(t, err)

		_, err = runtime2.NewSandbox(DefaultConfig())
		assert.ErrorIs(t, err, ErrSandboxUnavailable)
	})
}

func TestWazeroSandbox_SetServices(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	// Cast to WazeroSandbox to access SetServices
	wazeroSandbox, ok := sandbox.(*WazeroSandbox)
	require.True(t, ok)

	services := NewIsolatedServices(capability.RestrictedPolicy())
	wazeroSandbox.SetServices(services)

	// Verify services are set (indirectly via the struct)
	assert.NotNil(t, wazeroSandbox.services)
}

func TestWazeroSandbox_Close(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)

	err = sandbox.Close()
	assert.NoError(t, err)
}

func TestWazeroSandbox_Validate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Close() })

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	t.Run("rejects plugin with empty ID", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			Name:   "Test",
			Module: []byte{0x00, 0x61, 0x73, 0x6d},
		}
		err := sandbox.Validate(ctx, plugin)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPluginInvalid)
	})

	t.Run("rejects plugin with empty name", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			ID:     "test",
			Module: []byte{0x00, 0x61, 0x73, 0x6d},
		}
		err := sandbox.Validate(ctx, plugin)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPluginInvalid)
	})

	t.Run("rejects plugin with empty module", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			ID:   "test",
			Name: "Test",
		}
		err := sandbox.Validate(ctx, plugin)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPluginInvalid)
	})

	t.Run("rejects invalid WASM module", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			ID:     "test",
			Name:   "Test",
			Module: []byte("not valid wasm"),
		}
		err := sandbox.Validate(ctx, plugin)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPluginInvalid)
	})

	t.Run("rejects plugin with denied capability", func(t *testing.T) {
		t.Parallel()
		// Create a policy that blocks files:write
		policy := capability.NewPolicyBuilder().
			Block(capability.CapFilesWrite).
			Build()

		// Create sandbox with this policy
		cfg := Config{
			Mode:    ModeRestricted,
			Timeout: 30 * time.Second,
			Policy:  policy,
		}
		restrictedSandbox, err := runtime.NewSandbox(cfg)
		require.NoError(t, err)
		defer func() { _ = restrictedSandbox.Close() }()

		// Create plugin that requires files:write
		caps := capability.NewRequirements()
		caps.AddCapability(capability.CapFilesWrite, "Write output files")

		plugin := &Plugin{
			ID:           "test",
			Name:         "Test",
			Version:      "1.0.0",
			Module:       validWASMModule(),
			Capabilities: caps,
		}

		err = restrictedSandbox.Validate(ctx, plugin)
		assert.ErrorIs(t, err, ErrCapabilityDenied)
	})
}

func TestWazeroSandbox_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Close() })

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	t.Run("rejects invalid plugin", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			Name: "Test",
			// Missing ID and Module
		}
		result, err := sandbox.Execute(ctx, plugin, nil)
		assert.ErrorIs(t, err, ErrPluginInvalid)
		assert.Nil(t, result)
	})

	t.Run("handles invalid WASM module", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			ID:     "test",
			Name:   "Test",
			Module: []byte("not valid wasm"),
		}
		result, err := sandbox.Execute(ctx, plugin, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error.Error(), "compile")
	})

	t.Run("respects timeout", func(t *testing.T) {
		t.Parallel()
		// Create sandbox with very short timeout
		cfg := Config{
			Mode:    ModeRestricted,
			Timeout: 1 * time.Nanosecond, // Impossibly short
			Policy:  capability.DefaultPolicy(),
		}
		timeoutSandbox, err := runtime.NewSandbox(cfg)
		require.NoError(t, err)
		defer func() { _ = timeoutSandbox.Close() }()

		plugin := &Plugin{
			ID:     "test",
			Name:   "Test",
			Module: validWASMModule(),
		}

		// Use a context that's already done
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		result, err := timeoutSandbox.Execute(canceledCtx, plugin, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// The operation should fail due to context being canceled
		assert.False(t, result.Success)
	})

	t.Run("tracks execution duration", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			ID:     "test",
			Name:   "Test",
			Module: validWASMModule(),
		}
		result, err := sandbox.Execute(ctx, plugin, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Positive(t, result.Duration.Nanoseconds())
	})
}

func TestWazeroSandbox_Execute_WithServices(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	// Cast to WazeroSandbox to set services
	wazeroSandbox, ok := sandbox.(*WazeroSandbox)
	require.True(t, ok)

	// Create mock logger to capture logs
	logger := &testLogger{}
	services := &HostServices{
		FileSystem:     NullFileSystem{},
		PackageManager: NullPackageManager{},
		Shell:          NullShell{},
		HTTP:           NullHTTPClient{},
		Logger:         logger,
		Policy:         capability.DefaultPolicy(),
	}
	wazeroSandbox.SetServices(services)

	plugin := &Plugin{
		ID:     "test",
		Name:   "Test",
		Module: validWASMModule(),
	}

	result, err := sandbox.Execute(ctx, plugin, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// The module is a simple one, should execute without errors
	assert.Positive(t, result.Duration.Nanoseconds())
}

// testLogger captures log messages for testing.
type testLogger struct {
	infoMsgs  []string
	warnMsgs  []string
	errorMsgs []string
}

func (l *testLogger) Info(msg string) {
	l.infoMsgs = append(l.infoMsgs, msg)
}

func (l *testLogger) Warn(msg string) {
	l.warnMsgs = append(l.warnMsgs, msg)
}

func (l *testLogger) Error(msg string) {
	l.errorMsgs = append(l.errorMsgs, msg)
}

// validWASMModule returns a minimal valid WASM module.
// This is a simple module that exports an empty function.
func validWASMModule() []byte {
	// Minimal valid WASM module:
	// (module
	//   (func (export "main"))
	// )
	return []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // WASM version 1
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Export section
		0x07, 0x08, 0x01, 0x04, 0x6d, 0x61, 0x69, 0x6e, 0x00, 0x00,
		// Code section
		0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b,
	}
}

func TestReadString(t *testing.T) {
	t.Parallel()

	t.Run("returns empty for nil module", func(t *testing.T) {
		t.Parallel()
		result := readString(nil, 0, 0)
		assert.Empty(t, result)
	})

	t.Run("returns empty for zero length", func(t *testing.T) {
		t.Parallel()
		result := readString(nil, 100, 0)
		assert.Empty(t, result)
	})
}

func TestWazeroSandbox_Validate_ValidModule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	plugin := &Plugin{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
		Module:  validWASMModule(),
	}

	err = sandbox.Validate(ctx, plugin)
	assert.NoError(t, err)
}

func TestWazeroSandbox_Validate_WithCapabilities(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	// Create sandbox with a policy that grants some capabilities
	policy := capability.NewPolicyBuilder().
		Grant(capability.CapFilesRead).
		Grant(capability.CapFilesWrite).
		Build()

	cfg := Config{
		Mode:    ModeRestricted,
		Timeout: 30 * time.Second,
		Policy:  policy,
	}

	sandbox, err := runtime.NewSandbox(cfg)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	// Plugin with allowed capabilities should validate
	caps := capability.NewRequirements()
	caps.AddCapability(capability.CapFilesRead, "Read config")

	plugin := &Plugin{
		ID:           "test-plugin",
		Name:         "Test Plugin",
		Version:      "1.0.0",
		Module:       validWASMModule(),
		Capabilities: caps,
	}

	err = sandbox.Validate(ctx, plugin)
	assert.NoError(t, err)
}

func TestWazeroSandbox_Execute_ValidModule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	plugin := &Plugin{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
		Module:  validWASMModule(),
	}

	result, err := sandbox.Execute(ctx, plugin, nil)
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
	assert.Positive(t, result.Duration.Nanoseconds())
}

func TestWazeroSandbox_Execute_NoTimeout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	// Create sandbox with zero timeout
	cfg := Config{
		Mode:    ModeRestricted,
		Timeout: 0, // No timeout
		Policy:  capability.DefaultPolicy(),
	}

	sandbox, err := runtime.NewSandbox(cfg)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	plugin := &Plugin{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
		Module:  validWASMModule(),
	}

	result, err := sandbox.Execute(ctx, plugin, nil)
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestWazeroSandbox_Validate_NilPolicy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	// Create sandbox with nil policy
	cfg := Config{
		Mode:    ModeRestricted,
		Timeout: 30 * time.Second,
		Policy:  nil,
	}

	sandbox, err := runtime.NewSandbox(cfg)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	// Plugin with capabilities should still validate (no policy to check against)
	caps := capability.NewRequirements()
	caps.AddCapability(capability.CapFilesWrite, "Write files")

	plugin := &Plugin{
		ID:           "test-plugin",
		Name:         "Test Plugin",
		Version:      "1.0.0",
		Module:       validWASMModule(),
		Capabilities: caps,
	}

	err = sandbox.Validate(ctx, plugin)
	assert.NoError(t, err)
}

func TestWazeroSandbox_Validate_NilCapabilities(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	// Plugin without capabilities
	plugin := &Plugin{
		ID:           "test-plugin",
		Name:         "Test Plugin",
		Version:      "1.0.0",
		Module:       validWASMModule(),
		Capabilities: nil, // Explicitly nil
	}

	err = sandbox.Validate(ctx, plugin)
	assert.NoError(t, err)
}

func TestWazeroSandbox_Execute_ModuleWithRunFunc(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	// Module that exports "run" instead of "main"
	plugin := &Plugin{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
		Module:  moduleWithRunFunc(),
	}

	result, err := sandbox.Execute(ctx, plugin, nil)
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestWazeroSandbox_Execute_ModuleWithNoEntrypoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	// Module without main or run exports
	plugin := &Plugin{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
		Module:  moduleWithNoEntrypoint(),
	}

	result, err := sandbox.Execute(ctx, plugin, nil)
	assert.NoError(t, err)
	require.NotNil(t, result)
	// Should succeed even without a main function (just no-op)
	assert.True(t, result.Success)
}

// moduleWithRunFunc returns a WASM module that exports "run" function.
func moduleWithRunFunc() []byte {
	// (module
	//   (func (export "run"))
	// )
	return []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // WASM version 1
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Export section - exports "run"
		0x07, 0x07, 0x01, 0x03, 0x72, 0x75, 0x6e, 0x00, 0x00,
		// Code section
		0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b,
	}
}

func TestWazeroSandbox_Execute_MultiplePlugins(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Close() })

	sandbox, err := runtime.NewSandbox(DefaultConfig())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	// Execute multiple plugins sequentially to test host function caching
	for i := 0; i < 3; i++ {
		plugin := &Plugin{
			ID:      fmt.Sprintf("test-plugin-%d", i),
			Name:    "Test Plugin",
			Version: "1.0.0",
			Module:  validWASMModule(),
		}

		result, err := sandbox.Execute(ctx, plugin, nil)
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	}
}

// moduleWithNoEntrypoint returns a WASM module with no main/run export.
func moduleWithNoEntrypoint() []byte {
	// (module
	//   (func (export "helper"))
	// )
	return []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // WASM version 1
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Export section - exports "helper" (not main/run)
		0x07, 0x0a, 0x01, 0x06, 0x68, 0x65, 0x6c, 0x70, 0x65, 0x72, 0x00, 0x00,
		// Code section
		0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b,
	}
}
