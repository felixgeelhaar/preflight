package gem

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestStep_Idempotent verifies running Apply twice on a gem Step is safe.
// StatefulCommandRunner models `gem install rails` -> `gem list -i rails`
// transition end-to-end.
func TestStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewStep(Gem{Name: "rails"}, runner, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
