package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViolation_Error(t *testing.T) {
	tests := []struct {
		name      string
		violation Violation
		want      string
	}{
		{
			name: "basic violation",
			violation: Violation{
				StepID: "brew:install:curl",
				Rule:   Rule{Pattern: "brew:*", Action: "deny"},
				Value:  "brew:install:curl",
			},
			want: `policy violation: brew:install:curl is denied by rule "brew:*"`,
		},
		{
			name: "violation with message",
			violation: Violation{
				StepID: "apt:install:telnet",
				Rule:   Rule{Pattern: "apt:install:telnet", Action: "deny", Message: "insecure protocol"},
				Value:  "apt:install:telnet",
			},
			want: `policy violation: apt:install:telnet is denied by rule "apt:install:telnet" (insecure protocol)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.violation.Error())
		})
	}
}

func TestResult_HasViolations(t *testing.T) {
	t.Run("no violations", func(t *testing.T) {
		result := &Result{Violations: []Violation{}}
		assert.False(t, result.HasViolations())
	})

	t.Run("has violations", func(t *testing.T) {
		result := &Result{Violations: []Violation{{}}}
		assert.True(t, result.HasViolations())
	})
}

func TestResult_Errors(t *testing.T) {
	result := &Result{
		Violations: []Violation{
			{StepID: "step1", Rule: Rule{Pattern: "deny1", Action: "deny"}},
			{StepID: "step2", Rule: Rule{Pattern: "deny2", Action: "deny"}},
		},
	}

	errs := result.Errors()
	require.Len(t, errs, 2)
	assert.Contains(t, errs[0].Error(), "step1")
	assert.Contains(t, errs[1].Error(), "step2")
}

func TestEvaluator_Evaluate_AllowAll(t *testing.T) {
	evaluator := NewEvaluator()
	result := evaluator.Evaluate([]string{"brew:install:git", "apt:install:curl"})

	assert.False(t, result.HasViolations())
	assert.Len(t, result.Allowed, 2)
}

func TestEvaluator_Evaluate_DenyPattern(t *testing.T) {
	policy := Policy{
		Name: "no-brew",
		Rules: []Rule{
			{Pattern: "brew:*", Action: "deny", Message: "Homebrew not allowed"},
		},
	}

	evaluator := NewEvaluator(policy)
	result := evaluator.Evaluate([]string{"brew:install:git", "apt:install:curl"})

	assert.True(t, result.HasViolations())
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "brew:install:git", result.Violations[0].StepID)
	assert.Equal(t, "Homebrew not allowed", result.Violations[0].Rule.Message)
	assert.Len(t, result.Allowed, 1)
}

func TestEvaluator_Evaluate_AllowBeforeDeny(t *testing.T) {
	policy := Policy{
		Name: "allow-git-only",
		Rules: []Rule{
			{Pattern: "brew:install:git", Action: "allow", Message: "git is allowed"},
			{Pattern: "brew:*", Action: "deny", Message: "other brew packages not allowed"},
		},
	}

	evaluator := NewEvaluator(policy)
	result := evaluator.Evaluate([]string{
		"brew:install:git",
		"brew:install:curl",
		"apt:install:vim",
	})

	assert.True(t, result.HasViolations())
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "brew:install:curl", result.Violations[0].StepID)
	assert.Len(t, result.Allowed, 2)
}

func TestEvaluator_Evaluate_ExactMatch(t *testing.T) {
	policy := Policy{
		Name: "deny-specific",
		Rules: []Rule{
			{Pattern: "brew:install:telnet", Action: "deny", Message: "telnet is insecure"},
		},
	}

	evaluator := NewEvaluator(policy)
	result := evaluator.Evaluate([]string{
		"brew:install:telnet",
		"brew:install:ssh",
	})

	assert.True(t, result.HasViolations())
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "brew:install:telnet", result.Violations[0].StepID)
}

func TestEvaluator_Evaluate_ScopedRule(t *testing.T) {
	policy := Policy{
		Name: "scoped-deny",
		Rules: []Rule{
			{Pattern: "*:telnet", Action: "deny", Scope: "brew", Message: "no telnet via brew"},
		},
	}

	evaluator := NewEvaluator(policy)
	result := evaluator.Evaluate([]string{
		"brew:install:telnet",
		"apt:install:telnet",
	})

	// Only brew:install:telnet should be denied
	assert.True(t, result.HasViolations())
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "brew:install:telnet", result.Violations[0].StepID)
}

func TestEvaluator_EvaluateSteps(t *testing.T) {
	policy := Policy{
		Name: "test",
		Rules: []Rule{
			{Pattern: "banned:*", Action: "deny"},
		},
	}

	evaluator := NewEvaluator(policy)
	result := evaluator.EvaluateSteps([]string{"banned:step1", "allowed:step2"})

	assert.True(t, result.HasViolations())
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "banned:step1", result.Violations[0].StepID)
}

func TestEvaluator_MultiplePolicies(t *testing.T) {
	policy1 := Policy{
		Name: "no-games",
		Rules: []Rule{
			{Pattern: "*:steam", Action: "deny", Message: "no games"},
		},
	}
	policy2 := Policy{
		Name: "no-social",
		Rules: []Rule{
			{Pattern: "*:discord", Action: "deny", Message: "no social apps"},
		},
	}

	evaluator := NewEvaluator(policy1, policy2)
	result := evaluator.Evaluate([]string{
		"brew:cask:steam",
		"brew:cask:discord",
		"brew:cask:vscode",
	})

	assert.True(t, result.HasViolations())
	require.Len(t, result.Violations, 2)
	assert.Len(t, result.Allowed, 1)
}

func TestDefaultDenyAll(t *testing.T) {
	policy := DefaultDenyAll("brew:install:git", "apt:*")

	evaluator := NewEvaluator(policy)
	result := evaluator.Evaluate([]string{
		"brew:install:git",  // allowed by exception
		"apt:install:vim",   // allowed by apt:* exception
		"brew:install:curl", // denied by default
	})

	assert.True(t, result.HasViolations())
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "brew:install:curl", result.Violations[0].StepID)
	assert.Len(t, result.Allowed, 2)
}

func TestDefaultAllowAll(t *testing.T) {
	policy := DefaultAllowAll("brew:cask:steam", "apt:install:telnet")

	evaluator := NewEvaluator(policy)
	result := evaluator.Evaluate([]string{
		"brew:install:git",    // allowed by default
		"brew:cask:steam",     // explicitly denied
		"apt:install:telnet",  // explicitly denied
		"apt:install:openssh", // allowed by default
	})

	assert.True(t, result.HasViolations())
	require.Len(t, result.Violations, 2)
	assert.Len(t, result.Allowed, 2)
}

func TestMatchGlob_Patterns(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		pattern string
		want    bool
	}{
		{"exact match", "brew:install:git", "brew:install:git", true},
		{"wildcard suffix", "brew:install:git", "brew:*", true},
		{"wildcard prefix", "brew:install:git", "*:git", true},
		{"single char wildcard", "brew:install:git", "brew:instal?:git", true},
		{"no match", "apt:install:vim", "brew:*", false},
		{"partial match fails", "brew:install:git", "brew:install", false},
		{"nested wildcard", "brew:install:git", "brew:install:*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchGlob(tt.value, tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPolicy_Structure(t *testing.T) {
	policy := Policy{
		Name:        "security-baseline",
		Description: "Baseline security policy for all workstations",
		Rules: []Rule{
			{Pattern: "*:telnet", Action: "deny", Message: "telnet is insecure"},
			{Pattern: "*:ftp", Action: "deny", Message: "use sftp instead"},
			{Pattern: "*", Action: "allow"},
		},
	}

	assert.Equal(t, "security-baseline", policy.Name)
	assert.Equal(t, "Baseline security policy for all workstations", policy.Description)
	require.Len(t, policy.Rules, 3)
	assert.Equal(t, "deny", policy.Rules[0].Action)
	assert.Equal(t, "allow", policy.Rules[2].Action)
}
