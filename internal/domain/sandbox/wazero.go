package sandbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WazeroRuntime implements Runtime using Wazero.
type WazeroRuntime struct {
	runtime              wazero.Runtime
	hostFunctionsEnabled bool
	mu                   sync.Mutex
	closed               bool
}

// NewWazeroRuntime creates a new Wazero-based runtime.
func NewWazeroRuntime(ctx context.Context) (*WazeroRuntime, error) {
	cfg := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true)

	r := wazero.NewRuntimeWithConfig(ctx, cfg)

	// Instantiate WASI for standard I/O
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	return &WazeroRuntime{
		runtime: r,
	}, nil
}

// NewSandbox creates a new sandbox with the given config.
func (r *WazeroRuntime) NewSandbox(config Config) (Sandbox, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, ErrSandboxUnavailable
	}

	return &WazeroSandbox{
		runtime:       r.runtime,
		parentRuntime: r,
		config:        config,
	}, nil
}

// IsAvailable returns true if the runtime is available.
func (r *WazeroRuntime) IsAvailable() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.closed
}

// Version returns the runtime version.
func (r *WazeroRuntime) Version() string {
	return "wazero-1.8"
}

// Close releases runtime resources.
func (r *WazeroRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true
	return r.runtime.Close(context.Background())
}

// WazeroSandbox implements Sandbox using Wazero.
type WazeroSandbox struct {
	runtime       wazero.Runtime
	parentRuntime *WazeroRuntime
	config        Config
	services      *HostServices
	mu            sync.Mutex
}

// SetServices sets the host services for this sandbox.
func (s *WazeroSandbox) SetServices(services *HostServices) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services = services
}

// Execute runs a plugin with the given input.
func (s *WazeroSandbox) Execute(ctx context.Context, plugin *Plugin, _ []byte) (*ExecutionResult, error) {
	if err := plugin.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPluginInvalid, err)
	}

	// Apply timeout
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	start := time.Now()
	result := &ExecutionResult{}

	// Register host functions for plugin access (only once per runtime)
	s.parentRuntime.mu.Lock()
	if !s.parentRuntime.hostFunctionsEnabled {
		if err := s.registerHostFunctions(ctx); err != nil {
			s.parentRuntime.mu.Unlock()
			result.Error = fmt.Errorf("failed to register host functions: %w", err)
			result.Duration = time.Since(start)
			return result, nil
		}
		s.parentRuntime.hostFunctionsEnabled = true
	}
	s.parentRuntime.mu.Unlock()

	// Compile the module
	compiled, err := s.runtime.CompileModule(ctx, plugin.Module)
	if err != nil {
		result.Error = fmt.Errorf("failed to compile module: %w", err)
		result.Duration = time.Since(start)
		return result, nil
	}
	defer func() { _ = compiled.Close(ctx) }()

	// Create module config with resource limits
	modConfig := wazero.NewModuleConfig().
		WithName(plugin.ID).
		WithStartFunctions("_start", "_initialize")

	// Note: Memory limits are enforced via the WASM module's defined limits
	// The host cannot externally cap memory beyond what the module declares
	_ = s.config.Limits.MaxMemoryBytes // Reserved for future use

	// Instantiate and run
	instance, err := s.runtime.InstantiateModule(ctx, compiled, modConfig)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Error = ErrSandboxTimeout
		} else {
			result.Error = fmt.Errorf("failed to instantiate module: %w", err)
		}
		result.Duration = time.Since(start)
		return result, nil
	}
	defer func() { _ = instance.Close(ctx) }()

	// Call the main function if it exists
	mainFn := instance.ExportedFunction("main")
	if mainFn == nil {
		mainFn = instance.ExportedFunction("run")
	}

	if mainFn != nil {
		_, err = mainFn.Call(ctx)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				result.Error = ErrSandboxTimeout
			} else {
				result.Error = fmt.Errorf("plugin execution failed: %w", err)
			}
		}
	}

	result.Duration = time.Since(start)
	result.Success = result.Error == nil

	return result, nil
}

// Validate checks if a plugin can be loaded.
func (s *WazeroSandbox) Validate(ctx context.Context, plugin *Plugin) error {
	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrPluginInvalid, err)
	}

	// Check capabilities against policy
	if s.config.Policy != nil && plugin.Capabilities != nil {
		result := plugin.Capabilities.ValidateAgainst(s.config.Policy)
		if !result.IsValid() {
			denied := result.AllDenied()
			if len(denied) > 0 {
				return fmt.Errorf("%w: %s", ErrCapabilityDenied, denied[0].Reason)
			}
		}
	}

	// Try to compile the module
	compiled, err := s.runtime.CompileModule(ctx, plugin.Module)
	if err != nil {
		return fmt.Errorf("%w: failed to compile: %w", ErrPluginInvalid, err)
	}
	defer func() { _ = compiled.Close(ctx) }()

	return nil
}

// Close releases sandbox resources.
func (s *WazeroSandbox) Close() error {
	return nil
}

// registerHostFunctions adds preflight host functions to the runtime.
func (s *WazeroSandbox) registerHostFunctions(ctx context.Context) error {
	builder := s.runtime.NewHostModuleBuilder("preflight")

	// Log functions (always available)
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) {
			if s.services != nil && s.services.Logger != nil {
				msg := readString(m, ptr, length)
				s.services.Logger.Info(msg)
			}
		}).
		Export("log_info")

	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) {
			if s.services != nil && s.services.Logger != nil {
				msg := readString(m, ptr, length)
				s.services.Logger.Warn(msg)
			}
		}).
		Export("log_warn")

	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) {
			if s.services != nil && s.services.Logger != nil {
				msg := readString(m, ptr, length)
				s.services.Logger.Error(msg)
			}
		}).
		Export("log_error")

	_, err := builder.Instantiate(ctx)
	return err
}

// readString reads a string from WASM memory.
func readString(m api.Module, ptr, length uint32) string {
	if m == nil {
		return ""
	}
	mem := m.Memory()
	if mem == nil {
		return ""
	}
	data, ok := mem.Read(ptr, length)
	if !ok {
		return ""
	}
	return string(data)
}
