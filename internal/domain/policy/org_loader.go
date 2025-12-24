package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// OrgPolicyFile represents the structure of an org-policy.yaml file.
type OrgPolicyFile struct {
	// Version is the schema version
	Version string `yaml:"version"`
	// Policy is the org policy configuration
	Policy OrgPolicy `yaml:"policy"`
}

// LoadOrgPolicyFromFile loads an org policy from a YAML file.
func LoadOrgPolicyFromFile(path string) (*OrgPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No org policy file is OK
		}
		return nil, fmt.Errorf("failed to read org policy file: %w", err)
	}

	return ParseOrgPolicyYAML(data)
}

// ParseOrgPolicyYAML parses an org policy from YAML bytes.
func ParseOrgPolicyYAML(data []byte) (*OrgPolicy, error) {
	var file OrgPolicyFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse org policy YAML: %w", err)
	}

	// Validate the policy
	if err := ValidateOrgPolicy(&file.Policy); err != nil {
		return nil, err
	}

	// Default enforcement to block if not specified
	if file.Policy.Enforcement == "" {
		file.Policy.Enforcement = EnforcementBlock
	}

	return &file.Policy, nil
}

// ValidateOrgPolicy validates an org policy for correctness.
func ValidateOrgPolicy(policy *OrgPolicy) error {
	if policy.Name == "" {
		return fmt.Errorf("org policy must have a name")
	}

	// Validate enforcement mode
	if policy.Enforcement != "" &&
		policy.Enforcement != EnforcementWarn &&
		policy.Enforcement != EnforcementBlock {
		return fmt.Errorf("invalid enforcement mode: %s (must be 'warn' or 'block')", policy.Enforcement)
	}

	// Validate required patterns
	for i, req := range policy.Required {
		if req.Pattern == "" {
			return fmt.Errorf("required rule %d must have a pattern", i)
		}
	}

	// Validate forbidden patterns
	for i, forbidden := range policy.Forbidden {
		if forbidden.Pattern == "" {
			return fmt.Errorf("forbidden rule %d must have a pattern", i)
		}
	}

	// Validate overrides
	for i, override := range policy.Overrides {
		if override.Pattern == "" {
			return fmt.Errorf("override %d must have a pattern", i)
		}
		if override.Justification == "" {
			return fmt.Errorf("override %d for pattern %q must have a justification", i, override.Pattern)
		}
	}

	return nil
}

// ParseOrgPolicyFromConfig extracts an org policy from a config map.
// This allows org policies to be defined inline in preflight.yaml.
func ParseOrgPolicyFromConfig(config map[string]interface{}) (*OrgPolicy, error) {
	orgPolicyRaw, ok := config["org_policy"]
	if !ok {
		return nil, nil // No org policy defined
	}

	orgPolicyMap, ok := orgPolicyRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("org_policy must be a map")
	}

	policy := &OrgPolicy{}

	// Parse name
	if name, ok := orgPolicyMap["name"].(string); ok {
		policy.Name = name
	}

	// Parse description
	if desc, ok := orgPolicyMap["description"].(string); ok {
		policy.Description = desc
	}

	// Parse enforcement
	if enforcement, ok := orgPolicyMap["enforcement"].(string); ok {
		policy.Enforcement = EnforcementMode(enforcement)
	} else {
		policy.Enforcement = EnforcementBlock // Default
	}

	// Parse required
	if requiredRaw, ok := orgPolicyMap["required"].([]interface{}); ok {
		for i, rRaw := range requiredRaw {
			rMap, ok := rRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("required rule %d must be a map", i)
			}
			req := Requirement{}
			if pattern, ok := rMap["pattern"].(string); ok {
				req.Pattern = pattern
			}
			if message, ok := rMap["message"].(string); ok {
				req.Message = message
			}
			if scope, ok := rMap["scope"].(string); ok {
				req.Scope = scope
			}
			policy.Required = append(policy.Required, req)
		}
	}

	// Parse forbidden
	if forbiddenRaw, ok := orgPolicyMap["forbidden"].([]interface{}); ok {
		for i, fRaw := range forbiddenRaw {
			fMap, ok := fRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("forbidden rule %d must be a map", i)
			}
			forbidden := Forbidden{}
			if pattern, ok := fMap["pattern"].(string); ok {
				forbidden.Pattern = pattern
			}
			if message, ok := fMap["message"].(string); ok {
				forbidden.Message = message
			}
			if scope, ok := fMap["scope"].(string); ok {
				forbidden.Scope = scope
			}
			policy.Forbidden = append(policy.Forbidden, forbidden)
		}
	}

	// Parse overrides
	if overridesRaw, ok := orgPolicyMap["overrides"].([]interface{}); ok {
		for i, oRaw := range overridesRaw {
			oMap, ok := oRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("override %d must be a map", i)
			}
			override := Override{}
			if pattern, ok := oMap["pattern"].(string); ok {
				override.Pattern = pattern
			}
			if justification, ok := oMap["justification"].(string); ok {
				override.Justification = justification
			}
			if approvedBy, ok := oMap["approved_by"].(string); ok {
				override.ApprovedBy = approvedBy
			}
			if expiresAt, ok := oMap["expires_at"].(string); ok {
				override.ExpiresAt = expiresAt
			}
			policy.Overrides = append(policy.Overrides, override)
		}
	}

	// Validate
	if err := ValidateOrgPolicy(policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// MergeOrgPolicies merges multiple org policies into one.
// Later policies override earlier ones for conflicts.
func MergeOrgPolicies(policies ...*OrgPolicy) *OrgPolicy {
	if len(policies) == 0 {
		return nil
	}

	merged := &OrgPolicy{
		Name:        "merged",
		Description: "Merged org policy",
		Enforcement: EnforcementWarn, // Start with least restrictive
		Required:    []Requirement{},
		Forbidden:   []Forbidden{},
		Overrides:   []Override{},
	}

	for _, policy := range policies {
		if policy == nil {
			continue
		}

		// Use the most restrictive enforcement (block wins over warn)
		if policy.Enforcement == EnforcementBlock || merged.Enforcement == EnforcementBlock {
			merged.Enforcement = EnforcementBlock
		} else if policy.Enforcement == EnforcementWarn {
			merged.Enforcement = EnforcementWarn
		}

		// Merge required (add all)
		merged.Required = append(merged.Required, policy.Required...)

		// Merge forbidden (add all)
		merged.Forbidden = append(merged.Forbidden, policy.Forbidden...)

		// Merge overrides (add all)
		merged.Overrides = append(merged.Overrides, policy.Overrides...)
	}

	return merged
}
