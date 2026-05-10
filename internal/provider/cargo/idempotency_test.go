package cargo

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestCrateStep_Idempotent verifies running Apply twice on a cargo CrateStep
// is safe. StatefulCommandRunner reproduces the `cargo install --list` text
// output ("<name> v<version>:" + indented binary names) so the post-install
// Check finds the crate.
func TestCrateStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewCrateStep(Crate{Name: "ripgrep"}, runner, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
