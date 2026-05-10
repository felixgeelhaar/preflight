package main

import (
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
)

func failedStepIDs(results []execution.StepResult) []string {
	failedIDs := make([]string, 0, len(results))
	for i := range results {
		if results[i].Error() != nil {
			failedIDs = append(failedIDs, results[i].StepID().String())
		}
	}
	return failedIDs
}

func newApplyFailedUserError(action string, failedIDs []string, underlying error) *config.UserError {
	suggestion := "Inspect the failing step output above. Run 'preflight rollback' to restore previous state, or 'preflight doctor' to diagnose the underlying issue."
	if len(failedIDs) == 1 {
		suggestion = fmt.Sprintf("Step %q failed. Run 'preflight rollback' to restore previous state, or 'preflight doctor --verbose' to diagnose.", failedIDs[0])
	}

	msg := fmt.Sprintf("%s failed: some steps failed", action)
	if len(failedIDs) > 0 {
		msg = fmt.Sprintf("%s failed: %d step(s) did not complete", action, len(failedIDs))
	}

	return &config.UserError{
		Code:       "APPLY_FAILED",
		Message:    msg,
		Suggestion: suggestion,
		Underlying: underlying,
	}
}
