package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOrgPolicyFromFile(t *testing.T) {
	t.Parallel()

	t.Run("file not found returns nil", func(t *testing.T) {
		t.Parallel()
		policy, err := LoadOrgPolicyFromFile("/nonexistent/org-policy.yaml")
		assert.NoError(t, err)
		assert.Nil(t, policy)
	})

	t.Run("valid policy file", func(t *testing.T) {
		t.Parallel()

		content := `
version: "1"
policy:
  name: "test-org-policy"
  description: "Test org policy"
  enforcement: block
  required:
    - pattern: "git:*"
      message: "Git configuration required"
  forbidden:
    - pattern: "brew:*-nightly"
      message: "Nightly packages not allowed"
`
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "org-policy.yaml")
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		policy, err := LoadOrgPolicyFromFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, policy)

		assert.Equal(t, "test-org-policy", policy.Name)
		assert.Equal(t, "Test org policy", policy.Description)
		assert.Equal(t, EnforcementBlock, policy.Enforcement)
		assert.Len(t, policy.Required, 1)
		assert.Len(t, policy.Forbidden, 1)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		t.Parallel()

		content := `
invalid: [yaml
`
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "org-policy.yaml")
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		policy, err := LoadOrgPolicyFromFile(filePath)
		assert.Error(t, err)
		assert.Nil(t, policy)
		assert.Contains(t, err.Error(), "failed to parse org policy YAML")
	})
}

func TestParseOrgPolicyYAML(t *testing.T) {
	t.Parallel()

	t.Run("complete policy", func(t *testing.T) {
		t.Parallel()

		yaml := `
version: "1"
policy:
  name: "acme-corp"
  description: "ACME Corporation policy"
  enforcement: warn
  required:
    - pattern: "git:user.email"
      message: "Git email must be configured"
      scope: "git"
    - pattern: "ssh:*"
  forbidden:
    - pattern: "brew:*-nightly"
      message: "No nightly packages"
      scope: "brew"
  overrides:
    - pattern: "brew:rust-nightly"
      justification: "Needed for testing"
      approved_by: "security@acme.com"
      expires_at: "2025-12-31T23:59:59Z"
`
		policy, err := ParseOrgPolicyYAML([]byte(yaml))
		require.NoError(t, err)
		require.NotNil(t, policy)

		assert.Equal(t, "acme-corp", policy.Name)
		assert.Equal(t, "ACME Corporation policy", policy.Description)
		assert.Equal(t, EnforcementWarn, policy.Enforcement)

		require.Len(t, policy.Required, 2)
		assert.Equal(t, "git:user.email", policy.Required[0].Pattern)
		assert.Equal(t, "Git email must be configured", policy.Required[0].Message)
		assert.Equal(t, "git", policy.Required[0].Scope)

		require.Len(t, policy.Forbidden, 1)
		assert.Equal(t, "brew:*-nightly", policy.Forbidden[0].Pattern)
		assert.Equal(t, "brew", policy.Forbidden[0].Scope)

		require.Len(t, policy.Overrides, 1)
		assert.Equal(t, "brew:rust-nightly", policy.Overrides[0].Pattern)
		assert.Equal(t, "Needed for testing", policy.Overrides[0].Justification)
		assert.Equal(t, "security@acme.com", policy.Overrides[0].ApprovedBy)
		assert.Equal(t, "2025-12-31T23:59:59Z", policy.Overrides[0].ExpiresAt)
	})

	t.Run("defaults enforcement to block", func(t *testing.T) {
		t.Parallel()

		yaml := `
version: "1"
policy:
  name: "minimal"
`
		policy, err := ParseOrgPolicyYAML([]byte(yaml))
		require.NoError(t, err)
		assert.Equal(t, EnforcementBlock, policy.Enforcement)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		t.Parallel()

		yaml := `invalid: [yaml`
		_, err := ParseOrgPolicyYAML([]byte(yaml))
		assert.Error(t, err)
	})
}

func TestValidateOrgPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  *OrgPolicy
		wantErr string
	}{
		{
			name:    "missing name",
			policy:  &OrgPolicy{},
			wantErr: "org policy must have a name",
		},
		{
			name: "invalid enforcement mode",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: "invalid",
			},
			wantErr: "invalid enforcement mode: invalid",
		},
		{
			name: "required rule without pattern",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Required:    []Requirement{{Message: "needs pattern"}},
			},
			wantErr: "required rule 0 must have a pattern",
		},
		{
			name: "forbidden rule without pattern",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Forbidden:   []Forbidden{{Message: "needs pattern"}},
			},
			wantErr: "forbidden rule 0 must have a pattern",
		},
		{
			name: "override without pattern",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Overrides:   []Override{{Justification: "no pattern"}},
			},
			wantErr: "override 0 must have a pattern",
		},
		{
			name: "override without justification",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Overrides:   []Override{{Pattern: "brew:*"}},
			},
			wantErr: "override 0 for pattern \"brew:*\" must have a justification",
		},
		{
			name: "valid policy",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementBlock,
				Required:    []Requirement{{Pattern: "git:*"}},
				Forbidden:   []Forbidden{{Pattern: "brew:*-nightly"}},
				Overrides: []Override{{
					Pattern:       "brew:rust-nightly",
					Justification: "needed for testing",
				}},
			},
			wantErr: "",
		},
		{
			name: "valid with warn enforcement",
			policy: &OrgPolicy{
				Name:        "test",
				Enforcement: EnforcementWarn,
			},
			wantErr: "",
		},
		{
			name: "valid with empty enforcement",
			policy: &OrgPolicy{
				Name: "test",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateOrgPolicy(tt.policy)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseOrgPolicyFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("no org_policy key", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"packages": map[string]interface{}{
				"brew": []interface{}{"git"},
			},
		}

		policy, err := ParseOrgPolicyFromConfig(config)
		assert.NoError(t, err)
		assert.Nil(t, policy)
	})

	t.Run("org_policy not a map", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"org_policy": "not a map",
		}

		_, err := ParseOrgPolicyFromConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "org_policy must be a map")
	})

	t.Run("complete inline policy", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"org_policy": map[string]interface{}{
				"name":        "inline-policy",
				"description": "Inline org policy",
				"enforcement": "warn",
				"required": []interface{}{
					map[string]interface{}{
						"pattern": "git:*",
						"message": "Git required",
						"scope":   "git",
					},
				},
				"forbidden": []interface{}{
					map[string]interface{}{
						"pattern": "brew:*-nightly",
						"message": "No nightly",
						"scope":   "brew",
					},
				},
				"overrides": []interface{}{
					map[string]interface{}{
						"pattern":       "brew:rust-nightly",
						"justification": "Testing",
						"approved_by":   "admin",
						"expires_at":    "2025-12-31T23:59:59Z",
					},
				},
			},
		}

		policy, err := ParseOrgPolicyFromConfig(config)
		require.NoError(t, err)
		require.NotNil(t, policy)

		assert.Equal(t, "inline-policy", policy.Name)
		assert.Equal(t, "Inline org policy", policy.Description)
		assert.Equal(t, EnforcementWarn, policy.Enforcement)

		require.Len(t, policy.Required, 1)
		assert.Equal(t, "git:*", policy.Required[0].Pattern)
		assert.Equal(t, "Git required", policy.Required[0].Message)
		assert.Equal(t, "git", policy.Required[0].Scope)

		require.Len(t, policy.Forbidden, 1)
		assert.Equal(t, "brew:*-nightly", policy.Forbidden[0].Pattern)

		require.Len(t, policy.Overrides, 1)
		assert.Equal(t, "brew:rust-nightly", policy.Overrides[0].Pattern)
		assert.Equal(t, "Testing", policy.Overrides[0].Justification)
	})

	t.Run("defaults enforcement to block", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"org_policy": map[string]interface{}{
				"name": "minimal",
			},
		}

		policy, err := ParseOrgPolicyFromConfig(config)
		require.NoError(t, err)
		assert.Equal(t, EnforcementBlock, policy.Enforcement)
	})

	t.Run("required not a slice", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"org_policy": map[string]interface{}{
				"name": "test",
				"required": []interface{}{
					"not-a-map",
				},
			},
		}

		_, err := ParseOrgPolicyFromConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required rule 0 must be a map")
	})

	t.Run("forbidden not a slice", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"org_policy": map[string]interface{}{
				"name": "test",
				"forbidden": []interface{}{
					"not-a-map",
				},
			},
		}

		_, err := ParseOrgPolicyFromConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden rule 0 must be a map")
	})

	t.Run("override not a slice", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"org_policy": map[string]interface{}{
				"name": "test",
				"overrides": []interface{}{
					"not-a-map",
				},
			},
		}

		_, err := ParseOrgPolicyFromConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "override 0 must be a map")
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"org_policy": map[string]interface{}{
				"enforcement": "block",
				// missing name
			},
		}

		_, err := ParseOrgPolicyFromConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "org policy must have a name")
	})
}

func TestMergeOrgPolicies(t *testing.T) {
	t.Parallel()

	t.Run("empty slice returns nil", func(t *testing.T) {
		t.Parallel()

		result := MergeOrgPolicies()
		assert.Nil(t, result)
	})

	t.Run("single policy", func(t *testing.T) {
		t.Parallel()

		policy := &OrgPolicy{
			Name:        "single",
			Enforcement: EnforcementWarn,
			Required:    []Requirement{{Pattern: "git:*"}},
		}

		result := MergeOrgPolicies(policy)
		assert.Equal(t, "merged", result.Name)
		assert.Equal(t, EnforcementWarn, result.Enforcement) // Inherited from single policy
		assert.Len(t, result.Required, 1)
	})

	t.Run("skips nil policies", func(t *testing.T) {
		t.Parallel()

		policy := &OrgPolicy{
			Name:     "valid",
			Required: []Requirement{{Pattern: "git:*"}},
		}

		result := MergeOrgPolicies(nil, policy, nil)
		assert.NotNil(t, result)
		assert.Len(t, result.Required, 1)
	})

	t.Run("merges required patterns", func(t *testing.T) {
		t.Parallel()

		policy1 := &OrgPolicy{
			Name:     "policy1",
			Required: []Requirement{{Pattern: "git:*"}},
		}
		policy2 := &OrgPolicy{
			Name:     "policy2",
			Required: []Requirement{{Pattern: "ssh:*"}},
		}

		result := MergeOrgPolicies(policy1, policy2)
		assert.Len(t, result.Required, 2)
		assert.Equal(t, "git:*", result.Required[0].Pattern)
		assert.Equal(t, "ssh:*", result.Required[1].Pattern)
	})

	t.Run("merges forbidden patterns", func(t *testing.T) {
		t.Parallel()

		policy1 := &OrgPolicy{
			Name:      "policy1",
			Forbidden: []Forbidden{{Pattern: "brew:*-nightly"}},
		}
		policy2 := &OrgPolicy{
			Name:      "policy2",
			Forbidden: []Forbidden{{Pattern: "npm:*-beta"}},
		}

		result := MergeOrgPolicies(policy1, policy2)
		assert.Len(t, result.Forbidden, 2)
	})

	t.Run("merges overrides", func(t *testing.T) {
		t.Parallel()

		policy1 := &OrgPolicy{
			Name: "policy1",
			Overrides: []Override{{
				Pattern:       "brew:rust-nightly",
				Justification: "testing",
			}},
		}
		policy2 := &OrgPolicy{
			Name: "policy2",
			Overrides: []Override{{
				Pattern:       "npm:webpack-beta",
				Justification: "migration",
			}},
		}

		result := MergeOrgPolicies(policy1, policy2)
		assert.Len(t, result.Overrides, 2)
	})

	t.Run("block enforcement wins", func(t *testing.T) {
		t.Parallel()

		policy1 := &OrgPolicy{
			Name:        "policy1",
			Enforcement: EnforcementWarn,
		}
		policy2 := &OrgPolicy{
			Name:        "policy2",
			Enforcement: EnforcementBlock,
		}

		result := MergeOrgPolicies(policy1, policy2)
		assert.Equal(t, EnforcementBlock, result.Enforcement)
	})

	t.Run("block from earlier policy preserved", func(t *testing.T) {
		t.Parallel()

		policy1 := &OrgPolicy{
			Name:        "policy1",
			Enforcement: EnforcementBlock,
		}
		policy2 := &OrgPolicy{
			Name:        "policy2",
			Enforcement: EnforcementWarn,
		}

		result := MergeOrgPolicies(policy1, policy2)
		assert.Equal(t, EnforcementBlock, result.Enforcement)
	})
}
