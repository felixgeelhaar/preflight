package nvim

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestConfigSourceStep_Idempotent: Apply creates a symlink from
// dest to source; second Apply must succeed without error and Check must
// continue to report StatusSatisfied.
func TestConfigSourceStep_Idempotent(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewConfigSourceStep(
		"/preflight/dotfiles/nvim",
		"/home/user/.config/nvim",
		fs,
	)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
