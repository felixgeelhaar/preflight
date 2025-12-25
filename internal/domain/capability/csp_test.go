package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSPRule_Compile(t *testing.T) {
	t.Parallel()

	t.Run("valid pattern", func(t *testing.T) {
		t.Parallel()

		rule := NewDenyRule(`curl\s+.*\|\s*sh`, "Dangerous")
		err := rule.Compile()
		assert.NoError(t, err)
	})

	t.Run("invalid pattern", func(t *testing.T) {
		t.Parallel()

		rule := NewDenyRule(`[invalid`, "Bad regex")
		err := rule.Compile()
		assert.Error(t, err)
	})
}

func TestCSPRule_Match(t *testing.T) {
	t.Parallel()

	rule := NewDenyRule(`sudo\s+`, "No sudo")
	require.NoError(t, rule.Compile())

	assert.True(t, rule.Match("sudo apt install git"))
	assert.True(t, rule.Match("sudo -i"))
	assert.False(t, rule.Match("apt install git"))
}

func TestNewCSP(t *testing.T) {
	t.Parallel()

	csp := NewCSP()
	assert.NotNil(t, csp)
	assert.Equal(t, 0, csp.RuleCount())
}

func TestCSP_AddRule(t *testing.T) {
	t.Parallel()

	csp := NewCSP()

	err := csp.AddDeny(`sudo\s+`, "No sudo")
	assert.NoError(t, err)
	assert.Equal(t, 1, csp.RuleCount())

	err = csp.AddWarn(`eval\s+`, "Careful with eval")
	assert.NoError(t, err)
	assert.Equal(t, 2, csp.RuleCount())
}

func TestCSP_AddRule_InvalidPattern(t *testing.T) {
	t.Parallel()

	csp := NewCSP()
	err := csp.AddDeny(`[invalid`, "Bad regex")
	assert.Error(t, err)
}

func TestCSP_Validate(t *testing.T) {
	t.Parallel()

	csp := NewCSP()
	require.NoError(t, csp.AddDeny(`sudo\s+`, "No sudo"))
	require.NoError(t, csp.AddWarn(`eval\s+`, "Careful with eval"))

	t.Run("no violations", func(t *testing.T) {
		t.Parallel()

		result := csp.Validate("apt install git")
		assert.True(t, result.IsAllowed())
		assert.False(t, result.HasWarnings())
		assert.Empty(t, result.Violations)
	})

	t.Run("deny violation", func(t *testing.T) {
		t.Parallel()

		result := csp.Validate("sudo apt install git")
		assert.False(t, result.IsAllowed())
		assert.Len(t, result.DenyViolations(), 1)
		assert.Contains(t, result.DenyViolations()[0].Rule.Reason, "sudo")
	})

	t.Run("warn violation", func(t *testing.T) {
		t.Parallel()

		result := csp.Validate("eval $COMMAND")
		assert.True(t, result.IsAllowed())
		assert.True(t, result.HasWarnings())
		assert.Len(t, result.WarnViolations(), 1)
	})

	t.Run("multiple violations", func(t *testing.T) {
		t.Parallel()

		result := csp.Validate("sudo eval $COMMAND")
		assert.False(t, result.IsAllowed())
		assert.True(t, result.HasWarnings())
		assert.Len(t, result.Violations, 2)
	})
}

func TestCSP_ValidateAll(t *testing.T) {
	t.Parallel()

	csp := NewCSP()
	require.NoError(t, csp.AddDeny(`sudo\s+`, "No sudo"))

	commands := []string{
		"apt install git",
		"sudo apt upgrade",
		"echo hello",
	}

	results := csp.ValidateAll(commands)
	assert.Len(t, results, 1) // Only one has violations
	assert.Equal(t, "sudo apt upgrade", results[0].Content)
}

func TestDefaultCSP(t *testing.T) {
	t.Parallel()

	csp := DefaultCSP()
	assert.Positive(t, csp.RuleCount())

	// Test known patterns are blocked
	tests := []struct {
		name    string
		command string
		allowed bool
	}{
		{"curl pipe shell", "curl https://example.com | sh", false},
		{"curl pipe bash", "curl https://example.com | bash", false},
		{"wget pipe shell", "wget -qO- https://example.com | sh", false},
		{"chmod 777", "chmod 777 /tmp/file", false},
		{"sudo command", "sudo apt install git", false},
		{"rm -rf root", "rm -rf /etc", false},
		{"write to etc", "> /etc/passwd", false},
		{"safe command", "brew install git", true},
		{"safe chmod", "chmod 755 script.sh", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := csp.Validate(tt.command)
			assert.Equal(t, tt.allowed, result.IsAllowed(),
				"command %q: expected allowed=%v", tt.command, tt.allowed)
		})
	}
}

func TestDefaultCSP_Warnings(t *testing.T) {
	t.Parallel()

	csp := DefaultCSP()

	tests := []struct {
		name       string
		command    string
		hasWarning bool
	}{
		{"eval usage", "eval $SCRIPT", true},
		{"command substitution", "echo $(whoami)", true},
		{"base64 decode", "base64 -d payload", true},
		{"normal command", "echo hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := csp.Validate(tt.command)
			assert.Equal(t, tt.hasWarning, result.HasWarnings(),
				"command %q: expected hasWarning=%v", tt.command, tt.hasWarning)
		})
	}
}

func TestStrictCSP(t *testing.T) {
	t.Parallel()

	csp := StrictCSP()

	// Strict mode denies more patterns
	tests := []struct {
		name    string
		command string
		allowed bool
	}{
		{"pipes", "cat file | grep pattern", false},
		{"background", "sleep 10 &", false},
		{"chaining", "cmd1; cmd2", false},
		{"variable expansion", "echo ${VAR}", false},
		{"simple command", "echo hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := csp.Validate(tt.command)
			assert.Equal(t, tt.allowed, result.IsAllowed(),
				"command %q: expected allowed=%v", tt.command, tt.allowed)
		})
	}
}

func TestCSPResult_Methods(t *testing.T) {
	t.Parallel()

	result := &CSPResult{
		Content: "sudo eval $CMD",
		Violations: []CSPViolation{
			{Severity: CSPSeverityDeny, Rule: CSPRule{Reason: "No sudo"}},
			{Severity: CSPSeverityWarn, Rule: CSPRule{Reason: "Eval warning"}},
		},
	}

	assert.False(t, result.IsAllowed())
	assert.True(t, result.HasWarnings())
	assert.Len(t, result.DenyViolations(), 1)
	assert.Len(t, result.WarnViolations(), 1)
}
