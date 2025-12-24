package policy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgViolation_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		violation OrgViolation
		want      string
	}{
		{
			name: "missing required without message",
			violation: OrgViolation{
				Type:    "missing_required",
				Pattern: "git:*",
			},
			want: "missing required: git:*",
		},
		{
			name: "missing required with message",
			violation: OrgViolation{
				Type:    "missing_required",
				Pattern: "layer:base",
				Message: "base layer is required for all workstations",
			},
			want: "missing required: layer:base (base layer is required for all workstations)",
		},
		{
			name: "forbidden present without message",
			violation: OrgViolation{
				Type:    "forbidden_present",
				Pattern: "brew:*-nightly",
				Value:   "brew:rust-nightly",
			},
			want: "forbidden: brew:rust-nightly matches brew:*-nightly",
		},
		{
			name: "forbidden present with message",
			violation: OrgViolation{
				Type:    "forbidden_present",
				Pattern: "vscode:extension:*-unofficial",
				Value:   "vscode:extension:python-unofficial",
				Message: "only official extensions allowed",
			},
			want: "forbidden: vscode:extension:python-unofficial matches vscode:extension:*-unofficial (only official extensions allowed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.violation.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOrgResult_HasViolations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result OrgResult
		want   bool
	}{
		{
			name: "no violations block mode",
			result: OrgResult{
				Violations:  []OrgViolation{},
				Enforcement: EnforcementBlock,
			},
			want: false,
		},
		{
			name: "violations in block mode",
			result: OrgResult{
				Violations:  []OrgViolation{{Type: "missing_required", Pattern: "git:*"}},
				Enforcement: EnforcementBlock,
			},
			want: true,
		},
		{
			name: "violations in warn mode",
			result: OrgResult{
				Violations:  []OrgViolation{{Type: "missing_required", Pattern: "git:*"}},
				Enforcement: EnforcementWarn,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.result.HasViolations())
		})
	}
}

func TestOrgResult_HasWarnings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result OrgResult
		want   bool
	}{
		{
			name: "no warnings",
			result: OrgResult{
				Warnings: []OrgViolation{},
			},
			want: false,
		},
		{
			name: "has warnings",
			result: OrgResult{
				Warnings: []OrgViolation{{Type: "missing_required", Pattern: "git:*"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.result.HasWarnings())
		})
	}
}

func TestOrgResult_AllIssues(t *testing.T) {
	t.Parallel()

	result := OrgResult{
		Violations: []OrgViolation{
			{Type: "missing_required", Pattern: "git:*"},
		},
		Warnings: []OrgViolation{
			{Type: "forbidden_present", Pattern: "brew:*-nightly", Value: "brew:rust-nightly"},
		},
	}

	all := result.AllIssues()
	assert.Len(t, all, 2)
	assert.Equal(t, "missing_required", all[0].Type)
	assert.Equal(t, "forbidden_present", all[1].Type)
}

func TestOrgResult_Errors(t *testing.T) {
	t.Parallel()

	result := OrgResult{
		Violations: []OrgViolation{
			{Type: "missing_required", Pattern: "git:*"},
			{Type: "forbidden_present", Pattern: "brew:*-nightly", Value: "brew:rust-nightly"},
		},
	}

	errs := result.Errors()
	require.Len(t, errs, 2)
	assert.Contains(t, errs[0].Error(), "missing required: git:*")
	assert.Contains(t, errs[1].Error(), "forbidden: brew:rust-nightly")
}

func TestOrgEvaluator_Evaluate_NilPolicy(t *testing.T) {
	t.Parallel()

	evaluator := &OrgEvaluator{policy: nil}
	result := evaluator.Evaluate([]string{"git:user.name", "brew:git"})

	assert.Empty(t, result.Violations)
	assert.Empty(t, result.Warnings)
}

func TestOrgEvaluator_Evaluate_Required(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		policy         *OrgPolicy
		values         []string
		wantViolations int
	}{
		{
			name: "required pattern present",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Required: []Requirement{
					{Pattern: "git:*"},
				},
			},
			values:         []string{"git:user.name", "git:user.email"},
			wantViolations: 0,
		},
		{
			name: "required pattern missing",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Required: []Requirement{
					{Pattern: "git:*"},
				},
			},
			values:         []string{"brew:ripgrep"},
			wantViolations: 1,
		},
		{
			name: "multiple required patterns",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Required: []Requirement{
					{Pattern: "git:*"},
					{Pattern: "ssh:*"},
					{Pattern: "brew:git"},
				},
			},
			values:         []string{"git:user.name", "brew:git"},
			wantViolations: 1, // ssh:* missing
		},
		{
			name: "required with scope - matches",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Required: []Requirement{
					{Pattern: "git:*", Scope: "git"},
				},
			},
			values:         []string{"git:user.name"},
			wantViolations: 0,
		},
		{
			name: "required with scope - no matching scope",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Required: []Requirement{
					{Pattern: "git:*", Scope: "git"},
				},
			},
			values:         []string{"brew:git"},
			wantViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evaluator := NewOrgEvaluator(tt.policy)
			result := evaluator.Evaluate(tt.values)

			assert.Len(t, result.Violations, tt.wantViolations)
		})
	}
}

func TestOrgEvaluator_Evaluate_Forbidden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		policy         *OrgPolicy
		values         []string
		wantViolations int
	}{
		{
			name: "forbidden pattern not present",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "brew:*-nightly"},
				},
			},
			values:         []string{"brew:git", "brew:ripgrep"},
			wantViolations: 0,
		},
		{
			name: "forbidden pattern present",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "brew:*-nightly"},
				},
			},
			values:         []string{"brew:git", "brew:rust-nightly"},
			wantViolations: 1,
		},
		{
			name: "multiple forbidden matches",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "brew:*-nightly"},
				},
			},
			values:         []string{"brew:rust-nightly", "brew:go-nightly"},
			wantViolations: 2,
		},
		{
			name: "forbidden with scope - matches",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "*-nightly", Scope: "brew"},
				},
			},
			values:         []string{"brew:rust-nightly"},
			wantViolations: 1,
		},
		{
			name: "forbidden with scope - different scope",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "*-nightly", Scope: "brew"},
				},
			},
			values:         []string{"npm:webpack-nightly"},
			wantViolations: 0, // npm scope doesn't match brew scope
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evaluator := NewOrgEvaluator(tt.policy)
			result := evaluator.Evaluate(tt.values)

			assert.Len(t, result.Violations, tt.wantViolations)
		})
	}
}

func TestOrgEvaluator_Evaluate_WarnMode(t *testing.T) {
	t.Parallel()

	policy := &OrgPolicy{
		Name:        "test",
		Enforcement: EnforcementWarn,
		Required: []Requirement{
			{Pattern: "git:*"},
		},
		Forbidden: []Forbidden{
			{Pattern: "brew:*-nightly"},
		},
	}

	evaluator := NewOrgEvaluator(policy)
	result := evaluator.Evaluate([]string{"brew:rust-nightly"})

	// In warn mode, violations go to warnings
	assert.Empty(t, result.Violations)
	assert.Len(t, result.Warnings, 2) // missing required + forbidden present
	assert.False(t, result.HasViolations())
	assert.True(t, result.HasWarnings())
}

func TestOrgEvaluator_Evaluate_Overrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		policy            *OrgPolicy
		values            []string
		wantViolations    int
		wantOverridesUsed int
	}{
		{
			name: "override allows forbidden",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "brew:*-nightly"},
				},
				Overrides: []Override{
					{
						Pattern:       "brew:rust-nightly",
						Justification: "needed for testing new features",
					},
				},
			},
			values:            []string{"brew:rust-nightly"},
			wantViolations:    0,
			wantOverridesUsed: 1,
		},
		{
			name: "override with future expiration",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "brew:*-nightly"},
				},
				Overrides: []Override{
					{
						Pattern:       "brew:rust-nightly",
						Justification: "temporary for testing",
						ExpiresAt:     time.Now().Add(24 * time.Hour).Format(time.RFC3339),
					},
				},
			},
			values:            []string{"brew:rust-nightly"},
			wantViolations:    0,
			wantOverridesUsed: 1,
		},
		{
			name: "override expired",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "brew:*-nightly"},
				},
				Overrides: []Override{
					{
						Pattern:       "brew:rust-nightly",
						Justification: "was temporary",
						ExpiresAt:     time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
					},
				},
			},
			values:            []string{"brew:rust-nightly"},
			wantViolations:    1, // Override expired, so violation stands
			wantOverridesUsed: 0,
		},
		{
			name: "override with invalid date format",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "brew:*-nightly"},
				},
				Overrides: []Override{
					{
						Pattern:       "brew:rust-nightly",
						Justification: "bad date",
						ExpiresAt:     "not-a-date",
					},
				},
			},
			values:            []string{"brew:rust-nightly"},
			wantViolations:    1, // Invalid date = expired
			wantOverridesUsed: 0,
		},
		{
			name: "override pattern with glob",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden: []Forbidden{
					{Pattern: "brew:*-nightly"},
				},
				Overrides: []Override{
					{
						Pattern:       "brew:*-nightly",
						Justification: "allow all nightly packages",
					},
				},
			},
			values:            []string{"brew:rust-nightly", "brew:go-nightly"},
			wantViolations:    0,
			wantOverridesUsed: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evaluator := NewOrgEvaluator(tt.policy)
			result := evaluator.Evaluate(tt.values)

			assert.Len(t, result.Violations, tt.wantViolations)
			assert.Len(t, result.OverridesApplied, tt.wantOverridesUsed)
		})
	}
}

func TestSplitFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"git:user.name", ":", []string{"git", "user.name"}},
		{"brew:git", ":", []string{"brew", "git"}},
		{"no-separator", ":", []string{"no-separator"}},
		{"multi:colon:value", ":", []string{"multi", "colon:value"}},
		{"empty:", ":", []string{"empty", ""}},
		{":leading", ":", []string{"", "leading"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := splitFirst(tt.input, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value   string
		pattern string
		want    bool
	}{
		{"git:user.name", "git:*", true},
		{"git:user.name", "git:user.*", true},
		{"git:user.name", "git:user.name", true},
		{"git:user.name", "brew:*", false},
		{"brew:rust-nightly", "brew:*-nightly", true},
		{"brew:rust-stable", "brew:*-nightly", false},
		{"brew:rust", "*", true},
		{"vscode:extension:python", "vscode:extension:*", true},
	}

	for _, tt := range tests {
		t.Run(tt.value+"_"+tt.pattern, func(t *testing.T) {
			t.Parallel()
			result := matchGlob(tt.value, tt.pattern)
			assert.Equal(t, tt.want, result)
		})
	}
}
