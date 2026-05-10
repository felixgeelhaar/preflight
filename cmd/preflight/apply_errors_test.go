package main

import (
	"errors"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
)

func TestFailedStepIDs(t *testing.T) {
	step1 := newDummyStep("step:ok")
	step2 := newDummyStep("step:bad")

	results := []execution.StepResult{
		execution.NewStepResult(step1.ID(), compiler.StatusSatisfied, nil),
		execution.NewStepResult(step2.ID(), compiler.StatusFailed, errors.New("boom")),
	}

	got := failedStepIDs(results)
	if len(got) != 1 || got[0] != "step:bad" {
		t.Fatalf("failedStepIDs() = %v, want [step:bad]", got)
	}
}

func TestNewApplyFailedUserError(t *testing.T) {
	err := errors.New("apply failed")
	userErr := newApplyFailedUserError("apply", []string{"step:x"}, err)

	if userErr.Code != "APPLY_FAILED" {
		t.Fatalf("Code = %q, want APPLY_FAILED", userErr.Code)
	}
	if userErr.Message == "" || userErr.Suggestion == "" {
		t.Fatal("Message and Suggestion should be non-empty")
	}
	if userErr.Underlying == nil {
		t.Fatal("Underlying should be preserved")
	}

	var typed *config.UserError
	if !errors.As(userErr, &typed) {
		t.Fatal("expected config.UserError type")
	}
}
