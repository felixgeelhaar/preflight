package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPolicy(t *testing.T) {
	t.Parallel()

	p := NewPolicy()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Granted())
	assert.NotNil(t, p.Blocked())
	assert.NotNil(t, p.Approved())
	assert.True(t, p.RequiresApproval())
}

func TestPolicyBuilder(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapFilesRead, CapFilesWrite).
		Block(CapShellExecute).
		Approve(CapSecretsRead).
		RequireApproval(true).
		Build()

	assert.True(t, p.Granted().Has(CapFilesRead))
	assert.True(t, p.Granted().Has(CapFilesWrite))
	assert.True(t, p.Blocked().Has(CapShellExecute))
	assert.True(t, p.Approved().Has(CapSecretsRead))
	assert.True(t, p.RequiresApproval())
}

func TestPolicyBuilder_GrantStrings(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		GrantStrings("files:read", "packages:brew", "invalid").
		Build()

	assert.True(t, p.Granted().Has(CapFilesRead))
	assert.True(t, p.Granted().Has(CapPackagesBrew))
	assert.Equal(t, 2, p.Granted().Count()) // Invalid not added
}

func TestPolicyBuilder_BlockStrings(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		BlockStrings("shell:execute", "invalid").
		Build()

	assert.True(t, p.Blocked().Has(CapShellExecute))
	assert.Equal(t, 1, p.Blocked().Count()) // Invalid not added
}

func TestPolicy_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  *Policy
		cap     Capability
		wantErr error
	}{
		{
			name: "granted capability",
			policy: NewPolicyBuilder().
				Grant(CapFilesRead).
				Build(),
			cap:     CapFilesRead,
			wantErr: nil,
		},
		{
			name: "blocked capability",
			policy: NewPolicyBuilder().
				Grant(CapFilesRead).
				Block(CapFilesRead).
				Build(),
			cap:     CapFilesRead,
			wantErr: ErrCapabilityDenied,
		},
		{
			name: "not granted capability",
			policy: NewPolicyBuilder().
				Grant(CapFilesRead).
				Build(),
			cap:     CapFilesWrite,
			wantErr: ErrCapabilityNotGranted,
		},
		{
			name: "dangerous without approval",
			policy: NewPolicyBuilder().
				Grant(CapShellExecute).
				RequireApproval(true).
				Build(),
			cap:     CapShellExecute,
			wantErr: ErrDangerousCapability,
		},
		{
			name: "dangerous with approval",
			policy: NewPolicyBuilder().
				Grant(CapShellExecute).
				Approve(CapShellExecute).
				RequireApproval(true).
				Build(),
			cap:     CapShellExecute,
			wantErr: nil,
		},
		{
			name: "dangerous without require approval",
			policy: NewPolicyBuilder().
				Grant(CapShellExecute).
				RequireApproval(false).
				Build(),
			cap:     CapShellExecute,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.policy.Check(tt.cap)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPolicy_CheckAll(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapFilesRead, CapFilesWrite).
		Build()

	err := p.CheckAll(CapFilesRead, CapFilesWrite)
	assert.NoError(t, err)

	err = p.CheckAll(CapFilesRead, CapPackagesBrew)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCapabilityNotGranted)
}

func TestPolicy_IsAllowed(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapFilesRead).
		Build()

	assert.True(t, p.IsAllowed(CapFilesRead))
	assert.False(t, p.IsAllowed(CapFilesWrite))
}

func TestPolicy_Effective(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapFilesRead, CapFilesWrite, CapPackagesBrew).
		Block(CapFilesWrite).
		Build()

	effective := p.Effective()
	assert.Equal(t, 2, effective.Count())
	assert.True(t, effective.Has(CapFilesRead))
	assert.True(t, effective.Has(CapPackagesBrew))
	assert.False(t, effective.Has(CapFilesWrite))
}

func TestPolicy_PendingApproval(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapFilesRead, CapShellExecute, CapSecretsRead).
		Approve(CapShellExecute).
		RequireApproval(true).
		Build()

	pending := p.PendingApproval()
	assert.Len(t, pending, 1)
	assert.Equal(t, CapSecretsRead, pending[0])
}

func TestPolicy_PendingApproval_NoApprovalRequired(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapShellExecute, CapSecretsRead).
		RequireApproval(false).
		Build()

	pending := p.PendingApproval()
	assert.Nil(t, pending)
}

func TestPolicy_PendingApproval_BlockedNotIncluded(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapShellExecute, CapSecretsRead).
		Block(CapShellExecute).
		RequireApproval(true).
		Build()

	pending := p.PendingApproval()
	assert.Len(t, pending, 1)
	assert.Equal(t, CapSecretsRead, pending[0])
}

func TestPolicy_NeedsApproval(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapShellExecute).
		RequireApproval(true).
		Build()

	assert.True(t, p.NeedsApproval())

	p.ApproveAll()
	assert.False(t, p.NeedsApproval())
}

func TestPolicy_ApproveAll(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapShellExecute, CapSecretsRead, CapSecretsWrite).
		RequireApproval(true).
		Build()

	assert.Len(t, p.PendingApproval(), 3)

	p.ApproveAll()

	assert.Empty(t, p.PendingApproval())
	assert.True(t, p.Approved().Has(CapShellExecute))
	assert.True(t, p.Approved().Has(CapSecretsRead))
	assert.True(t, p.Approved().Has(CapSecretsWrite))
}

func TestPolicy_Validate(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapFilesRead, CapFilesWrite).
		Block(CapShellExecute).
		Build()

	requested := NewSetFrom([]Capability{
		CapFilesRead,
		CapPackagesBrew,
		CapShellExecute,
	})

	violations := p.Validate(requested)
	assert.Len(t, violations, 2)

	// Check blocked violation
	var blocked, notGranted *Violation
	for i := range violations {
		if violations[i].Blocked {
			blocked = &violations[i]
		} else {
			notGranted = &violations[i]
		}
	}

	require.NotNil(t, blocked)
	assert.Equal(t, CapShellExecute, blocked.Capability)
	assert.Equal(t, "blocked by policy", blocked.Reason)

	require.NotNil(t, notGranted)
	assert.Equal(t, CapPackagesBrew, notGranted.Capability)
	assert.Equal(t, "not granted", notGranted.Reason)
}

func TestPolicy_Summary(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		Grant(CapFilesRead, CapFilesWrite, CapShellExecute, CapSecretsRead).
		Block(CapSecretsRead).
		Approve(CapShellExecute).
		RequireApproval(true).
		Build()

	summary := p.Summary()

	assert.Equal(t, 4, summary.GrantedCount)
	assert.Equal(t, 1, summary.BlockedCount)
	assert.Equal(t, 3, summary.EffectiveCount) // 4 granted - 1 blocked
	assert.Equal(t, 1, summary.DangerousCount) // shell:execute (secrets:read blocked)
	assert.Equal(t, 0, summary.PendingCount)   // shell:execute approved
}

func TestDefaultPolicy(t *testing.T) {
	t.Parallel()

	p := DefaultPolicy()

	// Safe capabilities granted
	assert.True(t, p.IsAllowed(CapFilesRead))
	assert.True(t, p.IsAllowed(CapFilesWrite))
	assert.True(t, p.IsAllowed(CapPackagesBrew))
	assert.True(t, p.IsAllowed(CapPackagesApt))
	assert.True(t, p.IsAllowed(CapNetworkFetch))

	// Dangerous capabilities not granted
	assert.False(t, p.IsAllowed(CapShellExecute))
	assert.False(t, p.IsAllowed(CapSecretsRead))
	assert.False(t, p.IsAllowed(CapSecretsWrite))
	assert.False(t, p.IsAllowed(CapSystemModify))
}

func TestFullAccessPolicy(t *testing.T) {
	t.Parallel()

	p := FullAccessPolicy()

	// All capabilities granted
	for _, info := range AllCapabilities() {
		assert.True(t, p.IsAllowed(info.Capability), "expected %s to be allowed", info.Capability)
	}

	// No approval required
	assert.False(t, p.RequiresApproval())
}

func TestRestrictedPolicy(t *testing.T) {
	t.Parallel()

	p := RestrictedPolicy()

	// Safe capabilities granted
	assert.True(t, p.IsAllowed(CapFilesRead))
	assert.True(t, p.IsAllowed(CapNetworkFetch))

	// Write not granted
	assert.False(t, p.IsAllowed(CapFilesWrite))

	// Dangerous capabilities blocked
	err := p.Check(CapShellExecute)
	assert.ErrorIs(t, err, ErrCapabilityDenied)

	err = p.Check(CapSecretsRead)
	assert.ErrorIs(t, err, ErrCapabilityDenied)

	err = p.Check(CapSystemModify)
	assert.ErrorIs(t, err, ErrCapabilityDenied)
}

func TestPolicy_WildcardMatching(t *testing.T) {
	t.Parallel()

	p := NewPolicyBuilder().
		GrantStrings("files:*").
		Build()

	assert.True(t, p.IsAllowed(CapFilesRead))
	assert.True(t, p.IsAllowed(CapFilesWrite))
	assert.False(t, p.IsAllowed(CapPackagesBrew))
}
