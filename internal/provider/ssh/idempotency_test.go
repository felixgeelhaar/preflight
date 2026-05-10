package ssh

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestConfigStep_Idempotent runs the central idempotency contract on the SSH
// ConfigStep: Apply -> Check -> Apply -> Check must all succeed and the
// system must report StatusSatisfied after each Apply.
func TestConfigStep_Idempotent(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Defaults: DefaultsConfig{
			AddKeysToAgent: true,
			ForwardAgent:   false,
		},
		Hosts: []HostConfig{
			{Host: "github.com", User: "git", IdentityFile: "~/.ssh/id_ed25519"},
		},
	}
	step := NewConfigStep(cfg, fs)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
