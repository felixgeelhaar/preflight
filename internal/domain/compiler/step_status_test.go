package compiler

import (
	"testing"
)

func TestStepStatus_Values(t *testing.T) {
	// Verify all expected status values exist
	statuses := []StepStatus{
		StatusSatisfied,
		StatusNeedsApply,
		StatusUnknown,
		StatusFailed,
		StatusSkipped,
	}

	expected := []string{
		"satisfied",
		"needs-apply",
		"unknown",
		"failed",
		"skipped",
	}

	for i, status := range statuses {
		if status.String() != expected[i] {
			t.Errorf("status %d: got %q, want %q", i, status.String(), expected[i])
		}
	}
}

func TestStepStatus_NeedsAction(t *testing.T) {
	tests := []struct {
		status StepStatus
		want   bool
	}{
		{StatusSatisfied, false},
		{StatusNeedsApply, true},
		{StatusUnknown, true},
		{StatusFailed, true},
		{StatusSkipped, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.NeedsAction(); got != tt.want {
				t.Errorf("StepStatus(%q).NeedsAction() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStepStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status StepStatus
		want   bool
	}{
		{StatusSatisfied, true},
		{StatusNeedsApply, false},
		{StatusUnknown, false},
		{StatusFailed, true},
		{StatusSkipped, true},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Errorf("StepStatus(%q).IsTerminal() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
