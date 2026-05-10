package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

// AssertStepIsIdempotent verifies the central product guarantee that running
// Apply more than once is safe. It applies the step once, then a second time,
// and asserts that:
//
//  1. After the first Apply, Check() returns StatusSatisfied.
//  2. The second Apply succeeds (returns nil).
//  3. After the second Apply, Check() still returns StatusSatisfied.
//
// The caller must supply a RunContext whose CommandRunner / filesystem fake
// allows repeated invocations without external side effects. Use this helper
// in provider-level contract tests.
func AssertStepIsIdempotent(t testing.TB, step compiler.Step, ctx compiler.RunContext) {
	t.Helper()

	require.NotNil(t, step, "step must not be nil")

	require.NoError(t, step.Apply(ctx), "first Apply must succeed")

	status, err := step.Check(ctx)
	require.NoError(t, err, "Check after first Apply must not error")
	assert.Equal(t, compiler.StatusSatisfied, status,
		"after first Apply, Check() must report StatusSatisfied")

	require.NoError(t, step.Apply(ctx), "second Apply must succeed (idempotent)")

	status, err = step.Check(ctx)
	require.NoError(t, err, "Check after second Apply must not error")
	assert.Equal(t, compiler.StatusSatisfied, status,
		"after second Apply, Check() must still report StatusSatisfied")
}

// AssertStepsAreIdempotent runs AssertStepIsIdempotent against a batch of
// steps using a fresh background RunContext for each. Convenience wrapper for
// table-driven tests.
func AssertStepsAreIdempotent(t *testing.T, steps []compiler.Step, ctxFn func() compiler.RunContext) {
	t.Helper()

	for _, step := range steps {
		ctx := ctxFn()
		t.Run(step.ID().String(), func(t *testing.T) {
			AssertStepIsIdempotent(t, step, ctx)
		})
	}
}

// DefaultIdempotencyContext returns a RunContext rooted in context.Background()
// with default settings. Suitable when the step under test reads nothing from
// the context beyond cancellation.
func DefaultIdempotencyContext() compiler.RunContext {
	return compiler.NewRunContext(context.Background())
}
