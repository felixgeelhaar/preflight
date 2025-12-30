package aws_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/aws"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := aws.NewProvider(runner)

	assert.Equal(t, "aws", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := aws.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := aws.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"aws": map[string]interface{}{},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithProfiles(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := aws.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"aws": map[string]interface{}{
			"profiles": []interface{}{
				map[string]interface{}{
					"name":   "dev",
					"region": "us-east-1",
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "aws:profile:dev", steps[0].ID().String())
}

func TestProvider_Compile_WithSSO(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := aws.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"aws": map[string]interface{}{
			"sso": []interface{}{
				map[string]interface{}{
					"profile_name":  "sso-dev",
					"sso_start_url": "https://mycompany.awsapps.com/start",
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "aws:sso:sso-dev", steps[0].ID().String())
}

func TestProvider_Compile_WithDefaultProfile(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := aws.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"aws": map[string]interface{}{
			"default_profile": "dev",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "aws:default-profile", steps[0].ID().String())
}

func TestProvider_Compile_WithDefaultRegion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := aws.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"aws": map[string]interface{}{
			"default_region": "us-east-1",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "aws:default-region", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := aws.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"aws": map[string]interface{}{
			"profiles": []interface{}{
				map[string]interface{}{
					"name": "dev",
				},
			},
			"sso": []interface{}{
				map[string]interface{}{
					"profile_name": "sso-dev",
				},
			},
			"default_profile": "dev",
			"default_region":  "us-east-1",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 4)
}
