package brew

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestFormulaStep_Idempotent verifies the central guarantee for the brew
// formula installer: running Apply twice is safe.
//
// Uses StatefulCommandRunner so `brew install ripgrep` actually mutates the
// fake's installed-set; the subsequent Check (`brew list --formula`) reflects
// the install. The second Apply is a no-op at the brew level (formula is
// already in the set) yet must still succeed.
func TestFormulaStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewFormulaStep(Formula{Name: "ripgrep"}, runner)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}

// TestTapStep_Idempotent: same contract, but for a Homebrew tap.
func TestTapStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewTapStep("homebrew/cask-fonts", runner)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
