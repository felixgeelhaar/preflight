package integration

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type coreFlowHarness struct {
	*TestHarness
}

func newCoreFlowHarness(t *testing.T) *coreFlowHarness {
	t.Helper()
	return &coreFlowHarness{TestHarness: NewHarness(t)}
}

func (h *coreFlowHarness) createDeterministicConfig() string {
	manifest := `
targets:
  default:
    - base
`
	layer := `
name: base
`
	return h.CreateConfig(manifest, layer)
}

func TestCoreFlowHarness_PlanApplyDoctor_Deterministic(t *testing.T) {
	t.Parallel()

	h := newCoreFlowHarness(t)
	configPath := h.createDeterministicConfig()

	ctx := context.Background()

	plan1, err := h.Preflight().Plan(ctx, configPath, "default")
	require.NoError(t, err)
	plan2, err := h.Preflight().Plan(ctx, configPath, "default")
	require.NoError(t, err)

	assert.Equal(t, plan1.HasChanges(), plan2.HasChanges())
	assert.Len(t, plan1.Entries(), len(plan2.Entries()))

	results1, err := h.Preflight().Apply(ctx, plan1, true)
	require.NoError(t, err)
	results2, err := h.Preflight().Apply(ctx, plan2, true)
	require.NoError(t, err)

	assert.Equal(t, summarizeResults(results1), summarizeResults(results2))

	report, err := h.Doctor(configPath, "default")
	require.NoError(t, err)
	require.NotNil(t, report)
}

func TestCoreFlowHarness_ApplyCancellationRegression(t *testing.T) {
	t.Parallel()

	h := newCoreFlowHarness(t)

	plan := execution.NewExecutionPlan()
	step := newBlockingStep(t, "step:block")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, compiler.Diff{}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh := make(chan struct {
		results []execution.StepResult
		err     error
	}, 1)

	go func() {
		results, err := h.Preflight().Apply(ctx, plan, false)
		doneCh <- struct {
			results []execution.StepResult
			err     error
		}{results: results, err: err}
	}()

	<-step.started
	cancel()

	select {
	case got := <-doneCh:
		require.Len(t, got.results, 1)
		assert.Equal(t, compiler.StatusFailed, got.results[0].Status())
		assert.ErrorIs(t, got.results[0].Error(), context.Canceled)
		assert.Error(t, got.err)
	case <-time.After(2 * time.Second):
		t.Fatal("apply did not return within 2s after cancellation")
	}

	close(step.release)
}

func summarizeResults(results []execution.StepResult) []string {
	out := make([]string, 0, len(results))
	for i := range results {
		out = append(out, results[i].StepID().String()+":"+results[i].Status().String())
	}
	return out
}

type blockingStep struct {
	id      compiler.StepID
	started chan struct{}
	release chan struct{}
}

func newBlockingStep(t *testing.T, id string) *blockingStep {
	t.Helper()
	stepID, err := compiler.NewStepID(id)
	require.NoError(t, err)
	return &blockingStep{
		id:      stepID,
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (s *blockingStep) ID() compiler.StepID { return s.id }

func (s *blockingStep) DependsOn() []compiler.StepID { return nil }

func (s *blockingStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return compiler.StatusNeedsApply, nil
}

func (s *blockingStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.Diff{}, nil
}

func (s *blockingStep) Apply(_ compiler.RunContext) error {
	close(s.started)
	<-s.release
	return nil
}

func (s *blockingStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation("blocking step for cancellation regression", "", nil)
}

var _ compiler.Step = (*blockingStep)(nil)
