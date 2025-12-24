package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile_NotExists(t *testing.T) {
	policies, err := LoadFromFile("/nonexistent/path/policy.yaml")
	assert.NoError(t, err)
	assert.Nil(t, policies)
}

func TestLoadFromFile_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.yaml")

	content := `
policies:
  - name: security-baseline
    description: Security policy
    rules:
      - pattern: "*:telnet"
        action: deny
        message: telnet is insecure
      - pattern: "*"
        action: allow
`
	err := os.WriteFile(policyPath, []byte(content), 0644)
	require.NoError(t, err)

	policies, err := LoadFromFile(policyPath)
	require.NoError(t, err)
	require.Len(t, policies, 1)

	assert.Equal(t, "security-baseline", policies[0].Name)
	assert.Equal(t, "Security policy", policies[0].Description)
	require.Len(t, policies[0].Rules, 2)
	assert.Equal(t, "*:telnet", policies[0].Rules[0].Pattern)
	assert.Equal(t, "deny", policies[0].Rules[0].Action)
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.yaml")

	err := os.WriteFile(policyPath, []byte("invalid: [yaml"), 0644)
	require.NoError(t, err)

	_, err = LoadFromFile(policyPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse policy YAML")
}

func TestParseYAML_Valid(t *testing.T) {
	data := []byte(`
policies:
  - name: test-policy
    rules:
      - pattern: brew:*
        action: deny
        scope: packages
`)

	policies, err := ParseYAML(data)
	require.NoError(t, err)
	require.Len(t, policies, 1)
	assert.Equal(t, "test-policy", policies[0].Name)
	require.Len(t, policies[0].Rules, 1)
	assert.Equal(t, "packages", policies[0].Rules[0].Scope)
}

func TestParseYAML_MultiplePolicies(t *testing.T) {
	data := []byte(`
policies:
  - name: policy-1
    rules:
      - pattern: a:*
        action: deny
  - name: policy-2
    rules:
      - pattern: b:*
        action: deny
`)

	policies, err := ParseYAML(data)
	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, "policy-1", policies[0].Name)
	assert.Equal(t, "policy-2", policies[1].Name)
}

func TestParseFromConfig_NoPolicies(t *testing.T) {
	config := map[string]interface{}{
		"packages": map[string]interface{}{
			"brew": []string{"git"},
		},
	}

	policies, err := ParseFromConfig(config)
	assert.NoError(t, err)
	assert.Nil(t, policies)
}

func TestParseFromConfig_ValidPolicies(t *testing.T) {
	config := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name":        "test-policy",
				"description": "A test policy",
				"rules": []interface{}{
					map[string]interface{}{
						"pattern": "brew:*",
						"action":  "deny",
						"message": "no brew",
					},
				},
			},
		},
	}

	policies, err := ParseFromConfig(config)
	require.NoError(t, err)
	require.Len(t, policies, 1)

	assert.Equal(t, "test-policy", policies[0].Name)
	assert.Equal(t, "A test policy", policies[0].Description)
	require.Len(t, policies[0].Rules, 1)
	assert.Equal(t, "brew:*", policies[0].Rules[0].Pattern)
	assert.Equal(t, "deny", policies[0].Rules[0].Action)
	assert.Equal(t, "no brew", policies[0].Rules[0].Message)
}

func TestParseFromConfig_InvalidPoliciesType(t *testing.T) {
	config := map[string]interface{}{
		"policies": "not-a-list",
	}

	_, err := ParseFromConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policies must be a list")
}

func TestParseFromConfig_InvalidPolicyType(t *testing.T) {
	config := map[string]interface{}{
		"policies": []interface{}{
			"not-a-map",
		},
	}

	_, err := ParseFromConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy 0 must be a map")
}

func TestParseFromConfig_InvalidRuleType(t *testing.T) {
	config := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name": "test",
				"rules": []interface{}{
					"not-a-map",
				},
			},
		},
	}

	_, err := ParseFromConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rule 0 must be a map")
}

func TestParseFromConfig_RuleWithScope(t *testing.T) {
	config := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name": "scoped",
				"rules": []interface{}{
					map[string]interface{}{
						"pattern": "*:telnet",
						"action":  "deny",
						"scope":   "brew",
					},
				},
			},
		},
	}

	policies, err := ParseFromConfig(config)
	require.NoError(t, err)
	require.Len(t, policies, 1)
	require.Len(t, policies[0].Rules, 1)
	assert.Equal(t, "brew", policies[0].Rules[0].Scope)
}
