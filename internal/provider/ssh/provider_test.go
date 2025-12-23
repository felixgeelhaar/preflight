package ssh

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestProvider_Name(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	if p.Name() != "ssh" {
		t.Errorf("Name() = %q, want %q", p.Name(), "ssh")
	}
}

func TestProvider_Compile_NoConfig_ReturnsNil(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if steps != nil {
		t.Errorf("Compile() = %v, want nil", steps)
	}
}

func TestProvider_Compile_WithHosts_ReturnsConfigStep(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"ssh": map[string]interface{}{
			"hosts": []interface{}{
				map[string]interface{}{
					"host":         "github.com",
					"hostname":     "github.com",
					"user":         "git",
					"identityfile": "~/.ssh/id_ed25519",
				},
			},
		},
	})

	steps, err := p.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("Compile() len = %d, want 1", len(steps))
	}

	if steps[0].ID().String() != "ssh:config" {
		t.Errorf("step ID = %q, want %q", steps[0].ID().String(), "ssh:config")
	}
}

func TestProvider_Compile_WithDefaults_ReturnsConfigStep(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"ssh": map[string]interface{}{
			"defaults": map[string]interface{}{
				"addkeystoagent": true,
				"identitiesonly": true,
			},
		},
	})

	steps, err := p.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("Compile() len = %d, want 1", len(steps))
	}
}

func TestProvider_Compile_EmptySSHSection_ReturnsNil(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"ssh": map[string]interface{}{},
	})

	steps, err := p.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if steps != nil {
		t.Errorf("Compile() = %v, want nil for empty ssh config", steps)
	}
}
