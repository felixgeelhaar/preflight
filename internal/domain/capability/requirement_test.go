package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequirement(t *testing.T) {
	t.Parallel()

	req := NewRequirement(CapFilesRead, "Need to read config files")

	assert.Equal(t, CapFilesRead, req.Capability)
	assert.Equal(t, "Need to read config files", req.Justification)
	assert.False(t, req.Optional)
}

func TestNewOptionalRequirement(t *testing.T) {
	t.Parallel()

	req := NewOptionalRequirement(CapNetworkFetch, "Optional network access")

	assert.Equal(t, CapNetworkFetch, req.Capability)
	assert.True(t, req.Optional)
}

func TestRequirements_Add(t *testing.T) {
	t.Parallel()

	r := NewRequirements()
	r.AddCapability(CapFilesRead, "Read files")
	r.AddOptional(CapNetworkFetch, "Fetch remote")

	assert.Equal(t, 2, r.Count())
	assert.False(t, r.IsEmpty())
}

func TestRequirements_Required(t *testing.T) {
	t.Parallel()

	r := NewRequirements()
	r.AddCapability(CapFilesRead, "")
	r.AddCapability(CapFilesWrite, "")
	r.AddOptional(CapNetworkFetch, "")

	required := r.Required()
	assert.Len(t, required, 2)

	optional := r.Optional()
	assert.Len(t, optional, 1)
}

func TestRequirements_Dangerous(t *testing.T) {
	t.Parallel()

	r := NewRequirements()
	r.AddCapability(CapFilesRead, "")
	r.AddCapability(CapShellExecute, "Run setup script")
	r.AddCapability(CapSecretsRead, "Access SSH keys")

	dangerous := r.Dangerous()
	assert.Len(t, dangerous, 2)
}

func TestRequirements_ToSet(t *testing.T) {
	t.Parallel()

	r := NewRequirements()
	r.AddCapability(CapFilesRead, "")
	r.AddCapability(CapFilesWrite, "")
	r.AddOptional(CapNetworkFetch, "")

	s := r.ToSet()
	assert.Equal(t, 2, s.Count()) // Only required

	full := r.FullSet()
	assert.Equal(t, 3, full.Count()) // All
}

func TestRequirements_ValidateAgainst(t *testing.T) {
	t.Parallel()

	t.Run("all allowed", func(t *testing.T) {
		t.Parallel()

		r := NewRequirements()
		r.AddCapability(CapFilesRead, "")
		r.AddCapability(CapFilesWrite, "")

		policy := NewPolicyBuilder().
			Grant(CapFilesRead, CapFilesWrite).
			Build()

		result := r.ValidateAgainst(policy)
		assert.True(t, result.IsValid())
		assert.False(t, result.NeedsApproval())
		assert.Len(t, result.Allowed, 2)
		assert.Empty(t, result.Denied)
	})

	t.Run("some denied", func(t *testing.T) {
		t.Parallel()

		r := NewRequirements()
		r.AddCapability(CapFilesRead, "")
		r.AddCapability(CapPackagesBrew, "")

		policy := NewPolicyBuilder().
			Grant(CapFilesRead).
			Build()

		result := r.ValidateAgainst(policy)
		assert.False(t, result.IsValid())
		assert.Len(t, result.Allowed, 1)
		assert.Len(t, result.Denied, 1)
	})

	t.Run("dangerous pending approval", func(t *testing.T) {
		t.Parallel()

		r := NewRequirements()
		r.AddCapability(CapFilesRead, "")
		r.AddCapability(CapShellExecute, "Run setup")

		policy := NewPolicyBuilder().
			Grant(CapFilesRead, CapShellExecute).
			RequireApproval(true).
			Build()

		result := r.ValidateAgainst(policy)
		assert.True(t, result.IsValid())
		assert.True(t, result.NeedsApproval())
		assert.Len(t, result.Allowed, 1)
		assert.Len(t, result.Pending, 1)
	})

	t.Run("blocked capability", func(t *testing.T) {
		t.Parallel()

		r := NewRequirements()
		r.AddCapability(CapShellExecute, "")

		policy := NewPolicyBuilder().
			Grant(CapShellExecute).
			Block(CapShellExecute).
			Build()

		result := r.ValidateAgainst(policy)
		assert.False(t, result.IsValid())
		assert.Len(t, result.Denied, 1)
		assert.Contains(t, result.Denied[0].Reason, "blocked")
	})
}

func TestValidationResult_Summary(t *testing.T) {
	t.Parallel()

	t.Run("all allowed", func(t *testing.T) {
		t.Parallel()

		result := &ValidationResult{
			Allowed: []Requirement{{Capability: CapFilesRead}},
		}

		assert.Contains(t, result.Summary(), "1 capabilities allowed")
	})

	t.Run("some denied", func(t *testing.T) {
		t.Parallel()

		result := &ValidationResult{
			Allowed: []Requirement{{Capability: CapFilesRead}},
			Denied:  []DeniedRequirement{{Requirement: Requirement{Capability: CapShellExecute}}},
		}

		assert.Contains(t, result.Summary(), "1 allowed")
		assert.Contains(t, result.Summary(), "1 denied")
	})

	t.Run("pending approval", func(t *testing.T) {
		t.Parallel()

		result := &ValidationResult{
			Allowed: []Requirement{{Capability: CapFilesRead}},
			Pending: []Requirement{{Capability: CapShellExecute}},
		}

		assert.Contains(t, result.Summary(), "1 pending approval")
	})
}

func TestParseRequirements(t *testing.T) {
	t.Parallel()

	t.Run("valid capabilities", func(t *testing.T) {
		t.Parallel()

		r, err := ParseRequirements([]string{"files:read", "packages:brew"})
		require.NoError(t, err)
		assert.Equal(t, 2, r.Count())
	})

	t.Run("invalid capability", func(t *testing.T) {
		t.Parallel()

		_, err := ParseRequirements([]string{"files:read", "invalid"})
		assert.Error(t, err)
	})
}
