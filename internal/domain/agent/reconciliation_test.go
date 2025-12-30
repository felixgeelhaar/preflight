package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewReconciliationResult(t *testing.T) {
	before := time.Now()
	result := NewReconciliationResult()
	after := time.Now()

	assert.False(t, result.StartedAt.IsZero())
	assert.True(t, result.StartedAt.After(before) || result.StartedAt.Equal(before))
	assert.True(t, result.StartedAt.Before(after) || result.StartedAt.Equal(after))
	assert.True(t, result.CompletedAt.IsZero())
}

func TestReconciliationResult_Complete(t *testing.T) {
	result := NewReconciliationResult()
	time.Sleep(10 * time.Millisecond) // Small delay
	result.Complete()

	assert.False(t, result.CompletedAt.IsZero())
	assert.True(t, result.CompletedAt.After(result.StartedAt))
	assert.Positive(t, result.Duration)
}

func TestReconciliationResult_AddDrift(t *testing.T) {
	result := NewReconciliationResult()

	// Initially no drift
	assert.False(t, result.DriftDetected)
	assert.Equal(t, 0, result.DriftCount)

	// Add first drift
	result.AddDrift(DriftItem{
		ID:       "drift-1",
		Type:     "package",
		Name:     "ripgrep",
		Expected: "14.1.0",
		Actual:   "14.0.0",
		Severity: DriftSeverityLow,
	})

	assert.True(t, result.DriftDetected)
	assert.Equal(t, 1, result.DriftCount)
	assert.Len(t, result.DriftItems, 1)

	// Add second drift
	result.AddDrift(DriftItem{
		ID:       "drift-2",
		Type:     "file",
		Name:     "~/.gitconfig",
		Expected: "expected content",
		Actual:   "modified content",
		Severity: DriftSeverityMedium,
	})

	assert.Equal(t, 2, result.DriftCount)
	assert.Len(t, result.DriftItems, 2)
}

func TestReconciliationResult_AddRemediation(t *testing.T) {
	result := NewReconciliationResult()

	// Initially no remediation
	assert.False(t, result.RemediationApplied)
	assert.Equal(t, 0, result.RemediationCount)

	// Add failed remediation
	result.AddRemediation(RemediationItem{
		ID:      "rem-1",
		DriftID: "drift-1",
		Action:  "upgrade package",
		Success: false,
		Message: "network error",
	})

	assert.False(t, result.RemediationApplied) // Still false - no success
	assert.Equal(t, 1, result.RemediationCount)

	// Add successful remediation
	result.AddRemediation(RemediationItem{
		ID:      "rem-2",
		DriftID: "drift-2",
		Action:  "restore file",
		Success: true,
		Message: "file restored from template",
	})

	assert.True(t, result.RemediationApplied)
	assert.Equal(t, 2, result.RemediationCount)
}

func TestReconciliationResult_AddError(t *testing.T) {
	result := NewReconciliationResult()

	assert.False(t, result.HasErrors())

	result.AddError("detection", "failed to scan packages", true)

	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "detection", result.Errors[0].Phase)
	assert.Equal(t, "failed to scan packages", result.Errors[0].Message)
	assert.True(t, result.Errors[0].Recoverable)
}

func TestReconciliationResult_AddPendingApproval(t *testing.T) {
	result := NewReconciliationResult()

	assert.False(t, result.HasPendingApprovals())

	result.AddPendingApproval(ApprovalRequest{
		ID:        "approval-1",
		DriftID:   "drift-1",
		Action:    "upgrade node to v20",
		Risk:      "high",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	assert.True(t, result.HasPendingApprovals())
	assert.Len(t, result.PendingApprovals, 1)
}

func TestReconciliationResult_Summary(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*ReconciliationResult)
		expected string
	}{
		{
			name:     "no drift",
			setup:    func(_ *ReconciliationResult) {},
			expected: "no drift detected",
		},
		{
			name: "drift detected",
			setup: func(r *ReconciliationResult) {
				r.AddDrift(DriftItem{ID: "d1"})
			},
			expected: "drift detected",
		},
		{
			name: "drift remediated",
			setup: func(r *ReconciliationResult) {
				r.AddDrift(DriftItem{ID: "d1"})
				r.AddRemediation(RemediationItem{ID: "r1", Success: true})
			},
			expected: "drift detected and remediated",
		},
		{
			name: "drift awaiting approval",
			setup: func(r *ReconciliationResult) {
				r.AddDrift(DriftItem{ID: "d1"})
				r.AddPendingApproval(ApprovalRequest{ID: "a1"})
			},
			expected: "drift detected, awaiting approval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewReconciliationResult()
			tt.setup(result)

			assert.Equal(t, tt.expected, result.Summary())
		})
	}
}

func TestDriftSeverity_Constants(t *testing.T) {
	assert.Equal(t, DriftSeverityLow, DriftSeverity("low"))
	assert.Equal(t, DriftSeverityMedium, DriftSeverity("medium"))
	assert.Equal(t, DriftSeverityHigh, DriftSeverity("high"))
	assert.Equal(t, DriftSeverityCritical, DriftSeverity("critical"))
}
