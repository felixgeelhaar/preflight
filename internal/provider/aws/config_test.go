package aws_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := aws.ParseConfig(map[string]interface{}{})

	require.NoError(t, err)
	assert.Empty(t, cfg.Profiles)
	assert.Empty(t, cfg.SSO)
	assert.Empty(t, cfg.DefaultProfile)
	assert.Empty(t, cfg.DefaultRegion)
}

func TestParseConfig_WithProfiles(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"profiles": []interface{}{
			map[string]interface{}{
				"name":           "dev",
				"region":         "us-east-1",
				"output":         "json",
				"access_key_ref": "secret://aws/dev_access_key",
				"secret_key_ref": "secret://aws/dev_secret_key",
			},
			map[string]interface{}{
				"name":           "prod",
				"region":         "us-west-2",
				"role_arn":       "arn:aws:iam::123456789:role/Admin",
				"source_profile": "dev",
				"mfa_serial":     "arn:aws:iam::123456789:mfa/user",
			},
		},
	}

	cfg, err := aws.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 2)
	assert.Equal(t, "dev", cfg.Profiles[0].Name)
	assert.Equal(t, "us-east-1", cfg.Profiles[0].Region)
	assert.Equal(t, "json", cfg.Profiles[0].Output)
	assert.Equal(t, "secret://aws/dev_access_key", cfg.Profiles[0].AccessKeyRef)
	assert.Equal(t, "prod", cfg.Profiles[1].Name)
	assert.Equal(t, "dev", cfg.Profiles[1].SourceProfile)
}

func TestParseConfig_WithSSO(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sso": []interface{}{
			map[string]interface{}{
				"profile_name":   "sso-dev",
				"sso_start_url":  "https://mycompany.awsapps.com/start",
				"sso_region":     "us-east-1",
				"sso_account_id": "123456789012",
				"sso_role_name":  "Developer",
				"region":         "us-east-1",
				"output":         "json",
			},
		},
	}

	cfg, err := aws.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.SSO, 1)
	assert.Equal(t, "sso-dev", cfg.SSO[0].ProfileName)
	assert.Equal(t, "https://mycompany.awsapps.com/start", cfg.SSO[0].SSOStartURL)
	assert.Equal(t, "123456789012", cfg.SSO[0].SSOAccountID)
	assert.Equal(t, "Developer", cfg.SSO[0].SSORoleName)
}

func TestParseConfig_WithDefaultProfile(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"default_profile": "dev",
	}

	cfg, err := aws.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "dev", cfg.DefaultProfile)
}

func TestParseConfig_WithDefaultRegion(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"default_region": "us-west-2",
	}

	cfg, err := aws.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "us-west-2", cfg.DefaultRegion)
}

func TestParseConfig_InvalidProfile_NotObject(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"profiles": []interface{}{
			"not-an-object",
		},
	}

	cfg, err := aws.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile must be an object")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidProfile_NoName(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"profiles": []interface{}{
			map[string]interface{}{
				"region": "us-east-1",
			},
		},
	}

	cfg, err := aws.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile must have a name")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidSSO_NotObject(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sso": []interface{}{
			"not-an-object",
		},
	}

	cfg, err := aws.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sso config must be an object")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidSSO_NoProfileName(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"sso": []interface{}{
			map[string]interface{}{
				"sso_start_url": "https://mycompany.awsapps.com/start",
			},
		},
	}

	cfg, err := aws.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sso config must have a profile_name")
	assert.Nil(t, cfg)
}
