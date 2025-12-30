package config

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHookRunner(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner("/tmp")
	require.NotNil(t, runner)
	assert.Equal(t, "/tmp", runner.workDir)
	assert.NotNil(t, runner.evaluator)
}

func TestHookRunner_filterHooks(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner("/tmp")

	hooks := []Hook{
		{Name: "pre-apply", Phase: HookPhasePre, Type: HookTypeApply, Command: "echo pre"},
		{Name: "post-apply", Phase: HookPhasePost, Type: HookTypeApply, Command: "echo post"},
		{Name: "pre-plan", Phase: HookPhasePre, Type: HookTypePlan, Command: "echo plan"},
		{Name: "conditional", Phase: HookPhasePre, Type: HookTypeApply, Command: "echo cond", When: Condition{OS: "nonexistent_os"}},
	}

	tests := []struct {
		name     string
		phase    HookPhase
		hookType HookType
		expect   int
	}{
		{"pre-apply", HookPhasePre, HookTypeApply, 1}, // conditional is filtered out
		{"post-apply", HookPhasePost, HookTypeApply, 1},
		{"pre-plan", HookPhasePre, HookTypePlan, 1},
		{"post-plan", HookPhasePost, HookTypePlan, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := runner.filterHooks(hooks, tt.phase, tt.hookType)
			assert.Len(t, result, tt.expect)
		})
	}
}

func TestHookRunner_buildEnv(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner("/tmp")
	hook := Hook{
		Environment: map[string]string{
			"CUSTOM_VAR": "custom_value",
		},
	}
	hookCtx := HookContext{
		Target:       "default",
		LayerCount:   3,
		StepCount:    10,
		AppliedSteps: 5,
		SkippedSteps: 2,
		FailedSteps:  1,
		DryRun:       true,
	}

	env := runner.buildEnv(hook, hookCtx)

	assert.Contains(t, env, "PREFLIGHT_TARGET=default")
	assert.Contains(t, env, "PREFLIGHT_LAYER_COUNT=3")
	assert.Contains(t, env, "PREFLIGHT_STEP_COUNT=10")
	assert.Contains(t, env, "PREFLIGHT_APPLIED_STEPS=5")
	assert.Contains(t, env, "PREFLIGHT_SKIPPED_STEPS=2")
	assert.Contains(t, env, "PREFLIGHT_FAILED_STEPS=1")
	assert.Contains(t, env, "PREFLIGHT_DRY_RUN=true")
	assert.Contains(t, env, "CUSTOM_VAR=custom_value")
}

func TestHookRunner_RunHooks_Empty(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner("/tmp")
	ctx := context.Background()
	hookCtx := HookContext{}

	err := runner.RunHooks(ctx, nil, HookPhasePre, HookTypeApply, hookCtx)
	assert.NoError(t, err)
}

func TestHookRunner_RunHooks_Success(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner(t.TempDir())
	ctx := context.Background()

	hooks := []Hook{
		{Name: "test", Phase: HookPhasePre, Type: HookTypeApply, Command: "true"},
	}

	err := runner.RunHooks(ctx, hooks, HookPhasePre, HookTypeApply, HookContext{})
	assert.NoError(t, err)
}

func TestHookRunner_RunHooks_Timeout(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner(t.TempDir())
	ctx := context.Background()

	hooks := []Hook{
		{Name: "slow", Phase: HookPhasePre, Type: HookTypeApply, Command: "sleep 10", Timeout: "100ms"},
	}

	err := runner.RunHooks(ctx, hooks, HookPhasePre, HookTypeApply, HookContext{})
	assert.Error(t, err)
}

func TestHookRunner_RunHooks_OnErrorContinue(t *testing.T) {
	t.Parallel()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	runner := NewHookRunner(t.TempDir())
	ctx := context.Background()

	hooks := []Hook{
		{Name: "fail", Phase: HookPhasePre, Type: HookTypeApply, Command: "false", OnError: "continue"},
		{Name: "succeed", Phase: HookPhasePre, Type: HookTypeApply, Command: "true"},
	}

	err := runner.RunHooks(ctx, hooks, HookPhasePre, HookTypeApply, HookContext{})

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = oldStderr

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "fail")
}

func TestHookRunner_RunHooks_OnErrorFail(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner(t.TempDir())
	ctx := context.Background()

	hooks := []Hook{
		{Name: "fail", Phase: HookPhasePre, Type: HookTypeApply, Command: "false"},
	}

	err := runner.RunHooks(ctx, hooks, HookPhasePre, HookTypeApply, HookContext{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fail")
}

func TestHookRunner_runHook_NoCommandOrScript(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner(t.TempDir())
	ctx := context.Background()

	hook := Hook{Name: "empty"}
	err := runner.runHook(ctx, hook, HookContext{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have command or script")
}

func TestHookRunner_runHook_Script(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner(t.TempDir())
	ctx := context.Background()

	hook := Hook{
		Name:   "script",
		Script: "echo hello",
		Shell:  "bash",
	}

	err := runner.runHook(ctx, hook, HookContext{})
	assert.NoError(t, err)
}

func TestHookRunner_runHook_CustomTimeout(t *testing.T) {
	t.Parallel()

	runner := NewHookRunner(t.TempDir())
	ctx := context.Background()

	hook := Hook{
		Name:    "quick",
		Command: "true",
		Timeout: "1s",
	}

	start := time.Now()
	err := runner.runHook(ctx, hook, HookContext{})
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed, time.Second)
}

func TestValidateHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		hook      Hook
		expectErr string
	}{
		{
			name:      "valid hook",
			hook:      Hook{Phase: HookPhasePre, Type: HookTypeApply, Command: "echo test"},
			expectErr: "",
		},
		{
			name:      "invalid phase",
			hook:      Hook{Phase: "invalid", Type: HookTypeApply, Command: "echo test"},
			expectErr: "invalid hook phase",
		},
		{
			name:      "invalid type",
			hook:      Hook{Phase: HookPhasePre, Type: "invalid", Command: "echo test"},
			expectErr: "invalid hook type",
		},
		{
			name:      "missing command and script",
			hook:      Hook{Phase: HookPhasePre, Type: HookTypeApply},
			expectErr: "must have command or script",
		},
		{
			name:      "invalid on_error",
			hook:      Hook{Phase: HookPhasePre, Type: HookTypeApply, Command: "echo", OnError: "invalid"},
			expectErr: "invalid on_error value",
		},
		{
			name:      "valid with script",
			hook:      Hook{Phase: HookPhasePost, Type: HookTypePlan, Script: "echo test"},
			expectErr: "",
		},
		{
			name:      "valid on_error continue",
			hook:      Hook{Phase: HookPhasePre, Type: HookTypeDoctor, Command: "echo", OnError: "continue"},
			expectErr: "",
		},
		{
			name:      "valid on_error fail",
			hook:      Hook{Phase: HookPhasePre, Type: HookTypeCapture, Command: "echo", OnError: "fail"},
			expectErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateHook(tt.hook)
			if tt.expectErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			}
		})
	}
}

func TestParseHooksFromManifest_Empty(t *testing.T) {
	t.Parallel()

	hooks, err := ParseHooksFromManifest([]byte("version: 1\ntarget: default\n"))
	assert.NoError(t, err)
	assert.Nil(t, hooks)
}

func TestParseHooksFromManifest_NoHooksSection(t *testing.T) {
	t.Parallel()

	data := []byte(`
version: 1
target: default
layers:
  - base
`)
	hooks, err := ParseHooksFromManifest(data)
	assert.NoError(t, err)
	assert.Nil(t, hooks)
}

func TestHookContext_Fields(t *testing.T) {
	t.Parallel()

	ctx := HookContext{
		Target:       "production",
		LayerCount:   5,
		StepCount:    20,
		AppliedSteps: 15,
		SkippedSteps: 3,
		FailedSteps:  2,
		DryRun:       false,
	}

	assert.Equal(t, "production", ctx.Target)
	assert.Equal(t, 5, ctx.LayerCount)
	assert.Equal(t, 20, ctx.StepCount)
	assert.Equal(t, 15, ctx.AppliedSteps)
	assert.Equal(t, 3, ctx.SkippedSteps)
	assert.Equal(t, 2, ctx.FailedSteps)
	assert.False(t, ctx.DryRun)
}

func TestHookPhase_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, HookPhasePre, HookPhase("pre"))
	assert.Equal(t, HookPhasePost, HookPhase("post"))
}

func TestHookType_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, HookTypeApply, HookType("apply"))
	assert.Equal(t, HookTypePlan, HookType("plan"))
	assert.Equal(t, HookTypeDoctor, HookType("doctor"))
	assert.Equal(t, HookTypeCapture, HookType("capture"))
}
