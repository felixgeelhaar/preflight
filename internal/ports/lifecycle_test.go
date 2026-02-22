package ports

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoopLifecycle_BeforeModify(t *testing.T) {
	t.Parallel()

	lc := &NoopLifecycle{}
	err := lc.BeforeModify(context.Background(), "/some/path")
	assert.NoError(t, err)
}

func TestNoopLifecycle_AfterApply(t *testing.T) {
	t.Parallel()

	lc := &NoopLifecycle{}
	err := lc.AfterApply(context.Background(), "/some/path", "base")
	assert.NoError(t, err)
}

func TestNoopLifecycle_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Compile-time check is already in lifecycle.go, but this verifies it at test time.
	var lc FileLifecycle = &NoopLifecycle{}
	assert.NotNil(t, lc)
}
