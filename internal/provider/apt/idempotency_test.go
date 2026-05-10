package apt

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestPackageStep_Idempotent verifies running Apply twice on an apt
// PackageStep is safe. Uses StatefulCommandRunner so `apt-get install curl`
// transitions the fake's installed-set; the subsequent Check via dpkg-query
// reports "installed".
func TestPackageStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewPackageStep(Package{Name: "curl"}, runner)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
