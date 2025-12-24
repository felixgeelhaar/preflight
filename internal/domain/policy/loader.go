package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// File represents the structure of a policy YAML file.
type File struct {
	Policies []Policy `yaml:"policies"`
}

// LoadFromFile loads policies from a YAML file.
func LoadFromFile(path string) ([]Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No policy file is OK
		}
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	return ParseYAML(data)
}

// ParseYAML parses policies from YAML bytes.
func ParseYAML(data []byte) ([]Policy, error) {
	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse policy YAML: %w", err)
	}

	return file.Policies, nil
}

// ParseFromConfig extracts policies from a config map.
// This allows policies to be defined inline in preflight.yaml.
func ParseFromConfig(config map[string]interface{}) ([]Policy, error) {
	policiesRaw, ok := config["policies"]
	if !ok {
		return nil, nil // No policies defined
	}

	policiesSlice, ok := policiesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("policies must be a list")
	}

	policies := make([]Policy, 0, len(policiesSlice))
	for i, pRaw := range policiesSlice {
		pMap, ok := pRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("policy %d must be a map", i)
		}

		policy := Policy{}

		if name, ok := pMap["name"].(string); ok {
			policy.Name = name
		}
		if desc, ok := pMap["description"].(string); ok {
			policy.Description = desc
		}

		if rulesRaw, ok := pMap["rules"].([]interface{}); ok {
			for j, rRaw := range rulesRaw {
				rMap, ok := rRaw.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("policy %d rule %d must be a map", i, j)
				}

				rule := Rule{}
				if pattern, ok := rMap["pattern"].(string); ok {
					rule.Pattern = pattern
				}
				if action, ok := rMap["action"].(string); ok {
					rule.Action = action
				}
				if message, ok := rMap["message"].(string); ok {
					rule.Message = message
				}
				if scope, ok := rMap["scope"].(string); ok {
					rule.Scope = scope
				}

				policy.Rules = append(policy.Rules, rule)
			}
		}

		policies = append(policies, policy)
	}

	return policies, nil
}
