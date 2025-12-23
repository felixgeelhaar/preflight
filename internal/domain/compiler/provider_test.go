package compiler

import (
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
)

// mockProvider is a test double for Provider interface.
type mockProvider struct {
	name      string
	compileFn func(CompileContext) ([]Step, error)
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name: name,
		compileFn: func(_ CompileContext) ([]Step, error) {
			return []Step{}, nil
		},
	}
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Compile(ctx CompileContext) ([]Step, error) {
	return m.compileFn(ctx)
}

func TestProvider_Name(t *testing.T) {
	provider := newMockProvider("brew")
	if provider.Name() != "brew" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "brew")
	}
}

func TestProvider_Compile_EmptySteps(t *testing.T) {
	provider := newMockProvider("brew")
	ctx := NewCompileContext(nil)

	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestProvider_Compile_WithSteps(t *testing.T) {
	provider := newMockProvider("brew")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		step1 := newMockStep("brew:install:git")
		step2 := newMockStep("brew:install:curl")
		return []Step{step1, step2}, nil
	}

	ctx := NewCompileContext(nil)
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Compile() len = %d, want 2", len(steps))
	}
}

func TestProvider_Compile_Error(t *testing.T) {
	provider := newMockProvider("brew")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		return nil, errors.New("compilation failed")
	}

	ctx := NewCompileContext(nil)
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Fatal("expected error from Compile()")
	}
}

func TestCompileContext_Config(t *testing.T) {
	config := map[string]interface{}{
		"packages": []string{"git", "curl"},
	}
	ctx := NewCompileContext(config)

	if ctx.Config() == nil {
		t.Error("Config() should not be nil")
	}

	packages, ok := ctx.Config()["packages"].([]string)
	if !ok {
		t.Fatal("Config() should contain packages")
	}
	if len(packages) != 2 {
		t.Errorf("packages len = %d, want 2", len(packages))
	}
}

func TestCompileContext_GetSection(t *testing.T) {
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"packages": []string{"git", "curl"},
		},
	}
	ctx := NewCompileContext(config)

	section := ctx.GetSection("brew")
	if section == nil {
		t.Fatal("GetSection(brew) should not be nil")
	}

	packages, ok := section["packages"].([]string)
	if !ok {
		t.Fatal("section should contain packages")
	}
	if len(packages) != 2 {
		t.Errorf("packages len = %d, want 2", len(packages))
	}
}

func TestCompileContext_GetSection_Missing(t *testing.T) {
	ctx := NewCompileContext(nil)

	section := ctx.GetSection("nonexistent")
	if section != nil {
		t.Error("GetSection for missing key should return nil")
	}
}

func TestCompileContext_Provenance(t *testing.T) {
	ctx := NewCompileContext(nil).WithProvenance("layers/base.yaml")

	if ctx.Provenance() != "layers/base.yaml" {
		t.Errorf("Provenance() = %q, want %q", ctx.Provenance(), "layers/base.yaml")
	}
}

func createTestMachineInfo(t *testing.T) lock.MachineInfo {
	t.Helper()
	info, err := lock.NewMachineInfo("darwin", "arm64", "macbook.local", time.Now())
	if err != nil {
		t.Fatalf("failed to create machine info: %v", err)
	}
	return info
}

func createTestResolver(t *testing.T, mode config.ReproducibilityMode) *lock.Resolver {
	t.Helper()
	lockfile := lock.NewLockfile(mode, createTestMachineInfo(t))
	return lock.NewResolver(lockfile)
}

func TestCompileContext_Resolver_Nil(t *testing.T) {
	ctx := NewCompileContext(nil)

	if ctx.Resolver() != nil {
		t.Error("Resolver() should be nil by default")
	}
}

func TestCompileContext_WithResolver(t *testing.T) {
	resolver := createTestResolver(t, config.ModeLocked)
	ctx := NewCompileContext(nil).WithResolver(resolver)

	if ctx.Resolver() != resolver {
		t.Error("Resolver() should return the set resolver")
	}
}

func TestCompileContext_WithResolver_PreservesOtherFields(t *testing.T) {
	cfg := map[string]interface{}{"key": "value"}
	resolver := createTestResolver(t, config.ModeLocked)

	ctx := NewCompileContext(cfg).
		WithProvenance("layers/base.yaml").
		WithResolver(resolver)

	if ctx.Config()["key"] != "value" {
		t.Error("Config should be preserved")
	}
	if ctx.Provenance() != "layers/base.yaml" {
		t.Error("Provenance should be preserved")
	}
	if ctx.Resolver() != resolver {
		t.Error("Resolver should be set")
	}
}

func TestCompileContext_ResolveVersion_NoResolver(t *testing.T) {
	ctx := NewCompileContext(nil)

	resolution := ctx.ResolveVersion("brew", "ripgrep", "14.1.0")

	if resolution.Version != "14.1.0" {
		t.Errorf("Version = %q, want %q", resolution.Version, "14.1.0")
	}
	if resolution.Source != lock.ResolutionSourceLatest {
		t.Errorf("Source = %q, want %q", resolution.Source, lock.ResolutionSourceLatest)
	}
}

func TestCompileContext_ResolveVersion_WithResolver_Locked(t *testing.T) {
	resolver := createTestResolver(t, config.ModeLocked)

	// Add a locked package
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := lock.NewIntegrity("sha256", hash)
	_ = resolver.Lock("brew", "ripgrep", "14.0.0", integrity)

	ctx := NewCompileContext(nil).WithResolver(resolver)
	resolution := ctx.ResolveVersion("brew", "ripgrep", "14.1.0")

	// Should use locked version in locked mode
	if resolution.Version != "14.0.0" {
		t.Errorf("Version = %q, want %q", resolution.Version, "14.0.0")
	}
	if resolution.Source != lock.ResolutionSourceLockfile {
		t.Errorf("Source = %q, want %q", resolution.Source, lock.ResolutionSourceLockfile)
	}
	if !resolution.Locked {
		t.Error("Locked should be true")
	}
}

func TestCompileContext_ResolveVersion_WithResolver_Intent(t *testing.T) {
	resolver := createTestResolver(t, config.ModeIntent)

	// Add a locked package
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, _ := lock.NewIntegrity("sha256", hash)
	_ = resolver.Lock("brew", "ripgrep", "14.0.0", integrity)

	ctx := NewCompileContext(nil).WithResolver(resolver)
	resolution := ctx.ResolveVersion("brew", "ripgrep", "14.1.0")

	// Intent mode should use latest version
	if resolution.Version != "14.1.0" {
		t.Errorf("Version = %q, want %q", resolution.Version, "14.1.0")
	}
	if resolution.Source != lock.ResolutionSourceLatest {
		t.Errorf("Source = %q, want %q", resolution.Source, lock.ResolutionSourceLatest)
	}
}
