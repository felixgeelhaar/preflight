package git

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestConfigStep_Idempotent verifies the central product guarantee: re-running
// Apply on a ConfigStep is safe. After the first Apply the gitconfig file is
// present with expected contents; the second Apply must succeed without
// changing anything (Check still reports StatusSatisfied).
func TestConfigStep_Idempotent(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Path: "/home/user/.gitconfig",
		User: UserConfig{
			Name:  "Alice",
			Email: "alice@example.com",
		},
		Core: CoreConfig{
			Editor: "nvim",
		},
		Aliases: map[string]string{
			"co": "checkout",
		},
	}
	step := NewConfigStep(cfg, fs)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
