package aws

import (
	"fmt"
)

// Config represents the aws section of the configuration.
type Config struct {
	Profiles       []Profile
	SSO            []SSOConfig
	DefaultProfile string
	DefaultRegion  string
}

// Profile represents an AWS CLI profile.
type Profile struct {
	Name            string
	Region          string
	Output          string
	AccessKeyRef    string // Reference to secret, e.g., "secret://aws/access_key"
	SecretKeyRef    string // Reference to secret
	RoleArn         string
	SourceProfile   string
	MFASerial       string
	ExternalID      string
	DurationSeconds int
}

// SSOConfig represents AWS SSO configuration.
type SSOConfig struct {
	ProfileName  string
	SSOStartURL  string
	SSORegion    string
	SSOAccountID string
	SSORoleName  string
	Region       string
	Output       string
}

// ParseConfig parses the aws configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Profiles: make([]Profile, 0),
		SSO:      make([]SSOConfig, 0),
	}

	// Parse profiles
	if profiles, ok := raw["profiles"].([]interface{}); ok {
		for _, p := range profiles {
			profile, err := parseProfile(p)
			if err != nil {
				return nil, err
			}
			cfg.Profiles = append(cfg.Profiles, profile)
		}
	}

	// Parse SSO configurations
	if sso, ok := raw["sso"].([]interface{}); ok {
		for _, s := range sso {
			ssoConfig, err := parseSSOConfig(s)
			if err != nil {
				return nil, err
			}
			cfg.SSO = append(cfg.SSO, ssoConfig)
		}
	}

	// Parse default profile
	if defaultProfile, ok := raw["default_profile"].(string); ok {
		cfg.DefaultProfile = defaultProfile
	}

	// Parse default region
	if defaultRegion, ok := raw["default_region"].(string); ok {
		cfg.DefaultRegion = defaultRegion
	}

	return cfg, nil
}

func parseProfile(raw interface{}) (Profile, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return Profile{}, fmt.Errorf("profile must be an object")
	}

	profile := Profile{}

	if name, ok := m["name"].(string); ok {
		profile.Name = name
	} else {
		return Profile{}, fmt.Errorf("profile must have a name")
	}

	if region, ok := m["region"].(string); ok {
		profile.Region = region
	}
	if output, ok := m["output"].(string); ok {
		profile.Output = output
	}
	if accessKeyRef, ok := m["access_key_ref"].(string); ok {
		profile.AccessKeyRef = accessKeyRef
	}
	if secretKeyRef, ok := m["secret_key_ref"].(string); ok {
		profile.SecretKeyRef = secretKeyRef
	}
	if roleArn, ok := m["role_arn"].(string); ok {
		profile.RoleArn = roleArn
	}
	if sourceProfile, ok := m["source_profile"].(string); ok {
		profile.SourceProfile = sourceProfile
	}
	if mfaSerial, ok := m["mfa_serial"].(string); ok {
		profile.MFASerial = mfaSerial
	}
	if externalID, ok := m["external_id"].(string); ok {
		profile.ExternalID = externalID
	}
	if durationSeconds, ok := m["duration_seconds"].(int); ok {
		profile.DurationSeconds = durationSeconds
	}

	return profile, nil
}

func parseSSOConfig(raw interface{}) (SSOConfig, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return SSOConfig{}, fmt.Errorf("sso config must be an object")
	}

	sso := SSOConfig{}

	if name, ok := m["profile_name"].(string); ok {
		sso.ProfileName = name
	} else {
		return SSOConfig{}, fmt.Errorf("sso config must have a profile_name")
	}

	if startURL, ok := m["sso_start_url"].(string); ok {
		sso.SSOStartURL = startURL
	}
	if ssoRegion, ok := m["sso_region"].(string); ok {
		sso.SSORegion = ssoRegion
	}
	if accountID, ok := m["sso_account_id"].(string); ok {
		sso.SSOAccountID = accountID
	}
	if roleName, ok := m["sso_role_name"].(string); ok {
		sso.SSORoleName = roleName
	}
	if region, ok := m["region"].(string); ok {
		sso.Region = region
	}
	if output, ok := m["output"].(string); ok {
		sso.Output = output
	}

	return sso, nil
}
