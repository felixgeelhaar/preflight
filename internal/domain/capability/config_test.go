package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDefaultSecurityConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultSecurityConfig()
	assert.Empty(t, cfg.BlockedCapabilities)
	assert.Empty(t, cfg.CSPDeny)
	assert.Empty(t, cfg.CSPWarn)
	assert.NotNil(t, cfg.RequireApproval)
	assert.True(t, *cfg.RequireApproval)
}

func TestSecurityConfig_ToPolicy(t *testing.T) {
	t.Parallel()

	t.Run("empty config", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultSecurityConfig()
		policy, err := cfg.ToPolicy()
		require.NoError(t, err)
		assert.NotNil(t, policy)
		assert.True(t, policy.RequiresApproval())
	})

	t.Run("with blocked capabilities", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			BlockedCapabilities: []string{"shell:execute", "secrets:read"},
		}

		policy, err := cfg.ToPolicy()
		require.NoError(t, err)

		err = policy.Check(CapShellExecute)
		assert.ErrorIs(t, err, ErrCapabilityDenied)

		err = policy.Check(CapSecretsRead)
		assert.ErrorIs(t, err, ErrCapabilityDenied)
	})

	t.Run("with require approval false", func(t *testing.T) {
		t.Parallel()

		requireApproval := false
		cfg := SecurityConfig{
			RequireApproval: &requireApproval,
		}

		policy, err := cfg.ToPolicy()
		require.NoError(t, err)
		assert.False(t, policy.RequiresApproval())
	})

	t.Run("invalid capability", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			BlockedCapabilities: []string{"invalid"},
		}

		_, err := cfg.ToPolicy()
		assert.Error(t, err)
	})
}

func TestSecurityConfig_ToCSP(t *testing.T) {
	t.Parallel()

	t.Run("empty config uses defaults", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultSecurityConfig()
		csp, err := cfg.ToCSP()
		require.NoError(t, err)
		assert.Positive(t, csp.RuleCount())
	})

	t.Run("with custom rules", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			CSPDeny: []CSPEntry{{Pattern: `custom-deny`, Reason: "Custom deny"}},
			CSPWarn: []CSPEntry{{Pattern: `custom-warn`, Reason: "Custom warn"}},
		}

		csp, err := cfg.ToCSP()
		require.NoError(t, err)

		result := csp.Validate("custom-deny command")
		assert.False(t, result.IsAllowed())

		result = csp.Validate("custom-warn command")
		assert.True(t, result.IsAllowed())
		assert.True(t, result.HasWarnings())
	})

	t.Run("invalid pattern", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			CSPDeny: []CSPEntry{{Pattern: `[invalid`, Reason: "Bad"}},
		}

		_, err := cfg.ToCSP()
		assert.Error(t, err)
	})
}

func TestSecurityConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			BlockedCapabilities: []string{"shell:execute"},
			CSPDeny:             []CSPEntry{{Pattern: `sudo\s+`, Reason: "No sudo"}},
			CSPWarn:             []CSPEntry{{Pattern: `eval\s+`, Reason: "Eval warning"}},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid capability", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			BlockedCapabilities: []string{"invalid"},
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid blocked capability")
	})

	t.Run("invalid CSP deny pattern", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			CSPDeny: []CSPEntry{{Pattern: `[invalid`, Reason: "Bad"}},
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CSP deny pattern")
	})

	t.Run("invalid CSP warn pattern", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			CSPWarn: []CSPEntry{{Pattern: `[invalid`, Reason: "Bad"}},
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CSP warn pattern")
	})
}

func TestSecurityConfig_Merge(t *testing.T) {
	t.Parallel()

	t.Run("nil other", func(t *testing.T) {
		t.Parallel()

		cfg := &SecurityConfig{
			BlockedCapabilities: []string{"shell:execute"},
		}

		merged := cfg.Merge(nil)
		assert.Equal(t, cfg, merged)
	})

	t.Run("merge capabilities", func(t *testing.T) {
		t.Parallel()

		cfg1 := &SecurityConfig{
			BlockedCapabilities: []string{"shell:execute"},
		}
		cfg2 := &SecurityConfig{
			BlockedCapabilities: []string{"secrets:read", "shell:execute"},
		}

		merged := cfg1.Merge(cfg2)
		assert.Len(t, merged.BlockedCapabilities, 2) // No duplicates
	})

	t.Run("merge CSP rules", func(t *testing.T) {
		t.Parallel()

		cfg1 := &SecurityConfig{
			CSPDeny: []CSPEntry{{Pattern: "a", Reason: "A"}},
		}
		cfg2 := &SecurityConfig{
			CSPDeny: []CSPEntry{{Pattern: "b", Reason: "B"}},
		}

		merged := cfg1.Merge(cfg2)
		assert.Len(t, merged.CSPDeny, 2)
	})

	t.Run("require approval precedence", func(t *testing.T) {
		t.Parallel()

		require1 := true
		require2 := false

		cfg1 := &SecurityConfig{RequireApproval: &require1}
		cfg2 := &SecurityConfig{RequireApproval: &require2}

		merged := cfg1.Merge(cfg2)
		assert.NotNil(t, merged.RequireApproval)
		assert.False(t, *merged.RequireApproval)
	})
}

func TestSecurityConfig_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{}
		assert.True(t, cfg.IsEmpty())
	})

	t.Run("with blocked", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			BlockedCapabilities: []string{"shell:execute"},
		}
		assert.False(t, cfg.IsEmpty())
	})

	t.Run("with CSP", func(t *testing.T) {
		t.Parallel()

		cfg := SecurityConfig{
			CSPDeny: []CSPEntry{{Pattern: "a", Reason: "A"}},
		}
		assert.False(t, cfg.IsEmpty())
	})

	t.Run("with require approval", func(t *testing.T) {
		t.Parallel()

		require := true
		cfg := SecurityConfig{RequireApproval: &require}
		assert.False(t, cfg.IsEmpty())
	})
}

func TestSecurityConfig_YAML(t *testing.T) {
	t.Parallel()

	yamlData := `
blocked_capabilities:
  - shell:execute
  - secrets:read
csp_deny:
  - pattern: "sudo\\s+"
    reason: "No sudo"
csp_warn:
  - pattern: "eval\\s+"
    reason: "Eval warning"
require_approval: true
`

	var cfg SecurityConfig
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	require.NoError(t, err)

	assert.Len(t, cfg.BlockedCapabilities, 2)
	assert.Len(t, cfg.CSPDeny, 1)
	assert.Len(t, cfg.CSPWarn, 1)
	assert.NotNil(t, cfg.RequireApproval)
	assert.True(t, *cfg.RequireApproval)
}
